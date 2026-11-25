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
