# TMJLite — Go driver

O pacote `tmjlite-lib-go-<versão>.zip` traz o módulo Go **com o header e as
libs nativas embutidos** (`include/` e `lib/`): as diretivas `#cgo` do pacote
já apontam para `lib/`, então o `go build` funciona direto, sem `CGO_LDFLAGS`.

## Instalação

1. Baixe e extraia `tmjlite-lib-go-<versão>.zip`.
2. Coloque a pasta `tmjlite-go/` no seu projeto e referencie no `go.mod`:

```
require github.com/TMJacometti/tmjlite v0.0.0
replace github.com/TMJacometti/tmjlite => ./tmjlite-go
```

3. `go build` — pronto.

Requer cgo habilitado (`CGO_ENABLED=1`) e um compilador C (no Windows, MinGW
— a lib deve ser o build `x86_64-pc-windows-gnu`, já incluído no pacote).

Em runtime no Windows, copie `tmjlite-go/lib/tmjlite_ffi.dll` para a pasta do
seu executável (Linux/macOS usam rpath e não precisam de nada).

## Uso básico

```go
import tmjlite "github.com/TMJacometti/tmjlite"

db, err := tmjlite.Open("app.tmjp")
if err != nil { panic(err) }
defer db.Close()

db.Execute("CREATE TABLE users (id PK, name TEXT, age INT);")
db.Execute("INSERT INTO users VALUES ('Alice', 30);")

result, _ := db.Query("SELECT name, age FROM users;")
for _, row := range result.Rows {
    fmt.Println(row.GetString("name"), row.GetInt("age"))
}

db.Begin()
db.Execute("UPDATE users SET age = 31 WHERE name = 'Alice';")
db.Commit()

db.Save()
```

Memória: `tmjlite.OpenMemory()`.

## Índices compostos e EXPLAIN

```go
db.Execute("CREATE INDEX idx ON orders (user_id, status);")
plan, _ := db.Query("EXPLAIN SELECT * FROM orders WHERE user_id = 1 AND status = 'open';")
for _, row := range plan.Rows {
    fmt.Println(row.GetIndex(0))
}
```

## Requisitos

- Go 1.21+ com cgo (compilador C; MinGW no Windows)

## Desenvolvimento no repositório

```bash
cargo build --release -p tmjlite-ffi
cd drivers/go
go test ./...
go run ./example
```

No repo, o link usa `cgo_link_dev_*.go` (aponta para `../../target/release`).

## Solução de problemas

| Erro | Ação |
|------|------|
| `cannot find -ltmjlite_ffi` | Confira se `lib/` veio junto na cópia do módulo |
| runtime loader error (Windows) | Copie `lib/tmjlite_ffi.dll` para a pasta do executável |
| `.tmjdb` legado | Migra para `.tmjp` no primeiro open; backup `.bak` |

Compatibilidade e migração: https://tmjlite.com.br/manual.html (seção "Compatibilidade e migração") · repositório: `docs/EndUserManual.md`

## SharedConnection — concorrência (goroutine-safe)

```go
db, err := tmjlite.OpenShared("app.tmjp")
if err != nil { log.Fatal(err) }
defer db.Close()

var wg sync.WaitGroup
for w := 0; w < 8; w++ {
    wg.Add(1)
    go func(w int) {
        defer wg.Done()
        db.Execute(fmt.Sprintf("INSERT INTO t VALUES (%d);", w))
        rows, _ := db.Query("SELECT * FROM t;")
        _ = rows
    }(w)
}
wg.Wait()
```

SELECTs em snapshot MVCC paralelos; escritas serializadas e duráveis.
Auto-commit por statement (use `Open()` para transações).
**Multi-processo é exclusivo**: segundo processo falha com
`database is locked by another process`.

### Transações e prepared statements na SharedConnection (0.2.9+)

```go
db, _ := tmjlite.OpenShared("app.tmjp")

// Transação presa à THREAD do SO que deu Begin: prenda a goroutine com
// runtime.LockOSThread() entre Begin e Commit/Rollback. Outras goroutines
// bloqueiam só em escritas até o fim; leituras seguem em snapshot do último COMMIT (0.3.3+).
runtime.LockOSThread()
db.Begin()
for _, it := range lote { db.Execute(fmt.Sprintf("UPDATE rs SET samples = %d WHERE ...", it.Samples)) }
db.Commit()
runtime.UnlockOSThread()

// Prepared statement: parse uma vez, executa N vezes; parâmetros tipados,
// sem escape manual (string → TEXT, int → INT, float64 → FLOAT, bool, nil → NULL).
st, _ := db.Prepare("INSERT INTO rs (ns, k, samples) VALUES (?, ?, ?) ON CONFLICT (ns, k) DO UPDATE SET samples = excluded.samples;")
defer st.Close()
for _, it := range lote { st.Exec(it.NS, it.K, it.Samples) }
sel, _ := db.Prepare("SELECT samples FROM rs WHERE ns = ? AND k = ?;")
rows, _ := sel.Query("ns3", "k1")
```

`Prepare()` também existe na `Connection` clássica.

### Bootstrap idempotente e a PK do INSERT (0.3.6+)

```go
db.Execute("CREATE TABLE IF NOT EXISTS clientes (id PK, nome TEXT, email TEXT);") // no-op se já existe
r, err := db.Exec("INSERT INTO clientes (nome, email) VALUES ('Ana', 'ana@x');")
fmt.Println(r.RowsAffected, r.LastPk) // 1 <uuid gerado>; LastPk vazio em multi-linha/UPDATE/DELETE
```

### Cursor — leitura em streaming (0.3.1+)

```go
// uma linha por vez, direto do snapshot: memória O(1 linha) para N linhas
cur, err := db.Cursor("SELECT samples, cpu_hist FROM rs WHERE ns = ?", "ns3")
if err != nil { return err }
defer cur.Close()   // enquanto o cursor vive, o checkpoint do WAL é adiado
for cur.Next() {
    row := cur.Row()
    total += row.Get(0).(int64)
}
if err := cur.Err(); err != nil { return err }
```

### Memória: teto, stats e modo estrito (0.3.2+)

A `SharedConnection` mantém em RAM as tabelas em que **escreveu** (linhas +
índices) para que escritas repetidas sejam baratas; leituras por
`query`/`cursor` não entram nesse cache. Dê um teto para o limite ficar
previsível: acima dele, depois de uma escrita durável (autocommit ou COMMIT),
o cache é descartado e a memória devolvida ao SO; a próxima escrita recarrega
a tabela. `memory_stats` mostra o que está residente; o modo estrito faz um
cursor que não consegue streamar (JOIN, DISTINCT, subquery, window function,
ORDER BY/LIMIT sobre agregação) falhar com o motivo em vez de materializar em
silêncio.

```go
db.SetMemoryBudget(256 << 20)            // 256 MiB; 0 desliga
st, _ := db.MemoryStats()                // st.TotalBytes, st.Rows, st.Tables[i].Name/Rows/Bytes
db.ReleaseMemory()                       // descarta o cache agora (0 em transação)
db.SetStrictStreaming(true)              // Cursor() não-streamável → erro com o motivo
db.SetSnapshotCachePages(512)            // cache por cursor/query (páginas de 4 KiB; padrão 2048 = 8 MiB)
fsyncs, commits, _ := db.GroupCommitStats() // razão < 1 = group commit coalescendo escritores
```

### Tipos, chaves estrangeiras e VACUUM (0.5.0+)

- **Os valores voltam com o tipo da coluna.** `'007'` numa coluna `TEXT` volta
  texto (antes virava o número `7`) e `SUM` de coluna `INT` volta inteiro. Quem
  dependia da conversão implícita de texto para número passa a converter no código.
- **Chaves estrangeiras**: `cliente TEXT REFERENCES clientes ON DELETE CASCADE`
  (ou `RESTRICT` / `SET NULL`). INSERT e UPDATE no filho exigem o pai; crie um
  índice na coluna da FK para o `DELETE` do pai não varrer a tabela filha.
- **`VACUUM;`** compacta o arquivo (ele nunca encolhia sozinho). Exige a
  conexão clássica, exclusiva, fora de transação.
- **Apelido de coluna**: `SELECT COUNT(*) AS total FROM clientes;`.
- **Formato de arquivo v3**: o `.tmjp` é atualizado sozinho na primeira
  abertura e versões ≤ 0.4.0 deixam de abri-lo. Atualize CLI, drivers e Studio juntos.

### Erro de abertura com o motivo, um statement por chamada, PK do upsert (0.5.1+)

```go
db, err := tmjlite.OpenShared("app.tmjp")
if err != nil {
    return err // ...: database 'app.tmjp' is locked by another process
               // ...: 'app.tmjp' is not a TMJLite database (unrecognized file format)
}

r, err := db.Exec("INSERT INTO clientes (nome, email) VALUES ('Ana', 'ana@x') " +
    "ON CONFLICT (email) DO UPDATE SET nome = excluded.nome;")
fmt.Println(r.RowsAffected, r.LastPk) // 1 <uuid da linha inserida OU atualizada>
```

- **Abertura**: o erro traz o motivo real (travado por outro processo, não é
  um banco TMJLite, permissão negada, diretório inexistente). Antes toda falha
  parecia "em uso por outro processo". Um arquivo de 0 bytes é aceito como
  banco novo.
- **Upsert devolve a PK** em `LastPk`, tanto quando insere quanto quando atualiza
  (uma linha; em lote de várias linhas continua vazio, como no INSERT).
- **Um statement por chamada.** `a; b` numa chamada só é erro (antes só `a`
  rodava, em silêncio). Texto sobrando depois do statement também:
  `OFFSET`, `RETURNING`, `DELETE … LIMIT` e `UPDATE … FROM` ainda não existem
  e agora dão erro de sintaxe em vez de serem ignorados.
- **`UPDATE` que viola um índice `UNIQUE`** falha sem alterar nenhuma linha.
- **Texto com byte NUL** (`\0`) é recusado, em literal e em parâmetro.
- **Permissões (Unix)**: arquivos novos nascem `0600`; os auxiliares (`.pwal`,
  `.lock`) herdam o modo do `.tmjp` que já existe. Para um banco mais aberto,
  crie o arquivo vazio com o modo desejado antes de abrir. Um banco criado
  por versão anterior mantém a permissão que tinha: `chmod 600` nele uma vez.
  O `.lock` vazio ao lado do banco é esperado e fica depois de fechar.
- **Upsert de várias linhas** é tudo ou nada: um erro em qualquer linha desfaz
  as anteriores.

## Tratando erros por código

Todo erro do engine é um `*tmjlite.TMJLiteError` com um código estável no campo `Code`. Use o código para decidir o que fazer; não é mais preciso procurar texto em inglês na mensagem.

```go
import (
    "errors"

    tmjlite "github.com/TMJacometti/tmjlite"
)

err := db.Execute("INSERT INTO clientes (nome, email) VALUES ('Ana', 'ana@x');")
switch {
case errors.Is(err, tmjlite.ErrUniqueViolation):   // 401: e-mail já cadastrado
case errors.Is(err, tmjlite.CategoryConstraint):   // qualquer 4xx
case err != nil:
    return err
}

db, err := tmjlite.Open("app.tmjp")
var te *tmjlite.TMJLiteError
if errors.As(err, &te) {
    switch te.Code {
    case tmjlite.ErrLocked:        // outro processo está com o arquivo
    case tmjlite.ErrNotADatabase:  // o arquivo não é um banco TMJLite
    }
    fmt.Println(int(te.Code), te.Code, te.Code.Category())  // 500 LOCKED ACCESS
}

fmt.Println(tmjlite.ABIVersion())                  // 2
```

As constantes são `tmjlite.Err` + o nome da tabela abaixo em CamelCase (`UNIQUE_VIOLATION` → `ErrUniqueViolation`), do tipo `ErrorCode`; as categorias são `tmjlite.Category…` (`CategoryConstraint`, `CategoryOpen`…). As duas implementam `error`, então servem de alvo para `errors.Is`, inclusive através de `fmt.Errorf("…: %w", err)`. `err.Error()` continua sendo `tmjlite: <mensagem>`. Um número que o driver ainda não conhece (biblioteca nativa mais nova) aparece como `CODE_<n>` e ainda casa com a categoria. Erros que nascem no driver têm `Code == tmjlite.ErrUnknown` e não casam com nenhum código.

Regras do contrato (valem para o engine, a API C e os seis drivers):

- o **número** de um código nunca muda nem é reaproveitado;
- códigos novos podem aparecer numa versão futura: para um código que o seu
  programa não conhece, use a **categoria** (a centena do número) ou trate
  como erro genérico;
- a **mensagem** continua existindo e continua igual, mas é texto para gente,
  não contrato: não procure trechos dela no seu código.

**Biblioteca nativa antiga.** Os símbolos de código são resolvidos em tempo de execução (`dlsym` / `GetProcAddress`), não pelo linker: um binário compilado com este driver ainda carrega uma `libtmjlite_ffi` anterior aos códigos (ABI 1), e nesse caso `Code` vale `tmjlite.ErrUnknown` (-1). `tmjlite.ABIVersion()` diz qual ABI foi carregada. No Linux o driver passa a ligar com `-ldl` (necessário só em glibc antiga).

| Código | Nome | Categoria | Quando |
|---:|---|---|---|
| 1 | `ERROR` | GENERIC | Erro sem classificação mais específica |
| 100 | `SYNTAX` | SQL | SQL que o parser não aceita |
| 101 | `SEMANTIC` | SQL | SQL bem formado, mas a consulta é inválida (GROUP BY, ORDER BY fora do SELECT, número de valores, subconsulta com mais de uma linha) |
| 200 | `SCHEMA` | SCHEMA | DDL inválido ou objeto em estado que não permite a operação |
| 201 | `NO_SUCH_TABLE` | SCHEMA | Tabela não existe |
| 202 | `NO_SUCH_COLUMN` | SCHEMA | Coluna não existe |
| 203 | `NO_SUCH_INDEX` | SCHEMA | Índice não existe |
| 204 | `TABLE_EXISTS` | SCHEMA | Tabela já existe |
| 205 | `COLUMN_EXISTS` | SCHEMA | Coluna já existe |
| 206 | `INDEX_EXISTS` | SCHEMA | Índice já existe (pelo nome ou pelas colunas) |
| 207 | `AMBIGUOUS_COLUMN` | SCHEMA | Coluna ambígua: qualifique com tabela ou alias |
| 300 | `TYPE_MISMATCH` | TYPE | Valor de um tipo que a coluna não aceita |
| 301 | `INVALID_VALUE` | TYPE | Tipo certo, valor inaceitável (maior que STRING(n), data inválida, byte NUL) |
| 400 | `CONSTRAINT` | CONSTRAINT | Violação de restrição sem código mais específico |
| 401 | `UNIQUE_VIOLATION` | CONSTRAINT | Chave duplicada em índice UNIQUE |
| 402 | `PRIMARY_KEY_VIOLATION` | CONSTRAINT | Tentativa de definir ou alterar a PK (ela é gerada pelo engine) |
| 403 | `NOT_NULL_VIOLATION` | CONSTRAINT | NULL em coluna NOT NULL |
| 404 | `FOREIGN_KEY_VIOLATION` | CONSTRAINT | Chave estrangeira sem pai, ou pai ainda referenciado |
| 500 | `LOCKED` | ACCESS | Arquivo travado por outro processo |
| 501 | `BUSY` | ACCESS | A operação precisa de acesso exclusivo indisponível agora (ex.: VACUUM em conexão shared) |
| 502 | `READ_ONLY` | ACCESS | Escrita em conexão, snapshot ou mídia só de leitura |
| 600 | `CANNOT_OPEN` | OPEN | Não foi possível abrir (motivo fora dos códigos abaixo) |
| 601 | `FILE_NOT_FOUND` | OPEN | Arquivo ou diretório não existe |
| 602 | `PERMISSION_DENIED` | OPEN | Permissão negada pelo sistema operacional |
| 603 | `NOT_A_DATABASE` | OPEN | O arquivo existe mas não é um banco TMJLite |
| 604 | `UNSUPPORTED_FORMAT` | OPEN | Arquivo TMJLite de versão/formato que esta biblioteca não lê |
| 605 | `FILE_EXISTS` | OPEN | O arquivo já existe |
| 700 | `CORRUPT` | CORRUPT | Estrutura em disco inválida |
| 701 | `CORRUPT_CHECKSUM` | CORRUPT | Checksum de página ou de arquivo não confere |
| 702 | `CORRUPT_WAL` | CORRUPT | WAL inválido, incompleto ou incoerente com o arquivo principal |
| 800 | `IO` | IO | Falha de leitura/escrita/armazenamento sem código mais específico |
| 801 | `IO_FSYNC` | IO | fsync falhou: a durabilidade não está garantida |
| 802 | `DISK_FULL` | IO | Disco cheio ou cota excedida |
| 900 | `TRANSACTION` | TRANSACTION | Erro de estado de transação sem código mais específico |
| 901 | `NO_ACTIVE_TRANSACTION` | TRANSACTION | COMMIT/ROLLBACK sem transação ativa |
| 902 | `TRANSACTION_ACTIVE` | TRANSACTION | Já existe transação ativa (BEGIN aninhado, VACUUM dentro de transação) |
| 903 | `TRANSACTION_ABORTED` | TRANSACTION | A recuperação de uma transação falhou: feche e reabra o banco |
| 1000 | `MERGE` | MERGE | Erro de sessão shadow/merge sem código mais específico |
| 1001 | `MERGE_CONFLICT` | MERGE | Conflito de merge |
| 1002 | `SCHEMA_DIVERGENCE` | MERGE | O shadow foi gravado contra um esquema diferente do atual |
| 1003 | `NO_SUCH_BRANCH` | MERGE | Branch não existe |
| 1004 | `BRANCH_EXISTS` | MERGE | Branch já existe |
| 1005 | `UNSEALED_OPS` | MERGE | Há operações não seladas no branch |
| 1100 | `LIMIT_EXCEEDED` | LIMIT | Limite do engine excedido (tamanho de chave, LIMIT grande demais) |
| 1101 | `NESTING_TOO_DEEP` | LIMIT | Parênteses/subconsultas aninhados além do limite |
| 1200 | `UNSUPPORTED` | UNSUPPORTED | Recurso não suportado |
| 1201 | `MULTIPLE_STATEMENTS` | UNSUPPORTED | Mais de uma instrução na mesma chamada |
| 1300 | `MISUSE` | MISUSE | Uso incorreto da API (argumento inválido, statement nulo, cursor sobre algo que não é SELECT) |
| 1301 | `PARAM_COUNT` | MISUSE | Número de parâmetros diferente do número de ? |
| 1400 | `INTERNAL` | INTERNAL | Erro interno (invariante quebrada ou panic capturado): bug do TMJLite |

`405` está reservado para `CHECK`. A lista vale como contrato e está também no
manual (`docs/EndUserManual.md`, "Códigos de erro") e no header `tmjlite.h`
(`TMJLITE_ERR_*`).

## ShadowConnection e branches — escrita multi-processo por merge (git-merge)

```go
// N processos no mesmo arquivo, sem lock: cada um trabalha num snapshot +
// suas próprias escritas e faz merge por chave primária no Commit().
sess, _ := tmjlite.OpenShadow("app.tmjp")
sess.Execute("INSERT INTO listing (agency_id, external_id) VALUES ('ag', 'e1');")
rows, _ := sess.Query("SELECT * FROM listing;")   // snapshot + próprias escritas
applied, _ := sess.Commit()                       // merge no main (lock de ms)
sess.Refresh()                                    // enxerga merges dos vizinhos
sess.Close()

// Branch nomeada: persiste entre sessões/processos; só ops seladas mergeiam.
br, _ := tmjlite.OpenBranch("app.tmjp", "import-2026-09")
br.Execute("UPDATE listing SET seen = 1 WHERE external_id = 'e1';")
br.Seal("primeira leva", "thiago")                // commit na branch
br.SetMergePolicy("field_merge", "listing")       // lww | error | field_merge | manual ("" = default)
conflicts, _ := br.Conflicts()                    // dry-run
br.Commit()                                       // merge + branch consumida
br.Close()
```

Conflito = a mesma linha alterada no main desde o snapshot; a política vem da
tabela (`ALTER TABLE t SET MERGE POLICY …`) ou do `SetMergePolicy` (que tem
precedência). DML-only: DDL e BEGIN/COMMIT rodam em `Open()`.
Ver `docs/engine/GIT_MERGE.md`.
