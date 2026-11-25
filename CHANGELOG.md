# Changelog

Todas as mudanças relevantes do TMJLite (engine, CLI, drivers) ficam aqui.
Formato: [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/);
versões seguem [SemVer](https://semver.org/lang/pt-BR/).

A seção da versão publicada é extraída por `scripts/changelog-section.sh` e
vai para a GitHub Release, o NuGet (*Release Notes*) e os READMEs dos pacotes.
O TMJStudio tem ciclo próprio (`studio-vX.Y.Z`) e é anotado na seção da
versão do engine em que saiu.

## [Unreleased]

## [0.6.3] - 2026-11-25

Só distribuição: nenhuma mudança no engine, no CLI ou nos drivers.

### Adicionado
- **Publicação do TMJStudio num botão.** O workflow "Publicar Studio" espelha
  a release validada no repositório público, atualiza o manifest, publica o
  `tmjstudio` no apt, o cask no Homebrew e a landing. Não há mais passo na
  máquina de ninguém.
- **A landing se publica sozinha** no fim de cada release do CLI e de cada
  publicação do Studio (e por um botão, quando só o conteúdo muda).

### Corrigido
- **A página de drivers da landing estava parada na 0.3.7.** O guia era
  gerado por um script que ninguém chamava; agora faz parte do deploy, e a
  "versão corrente" vem do código.
- **Token do apt/Homebrew com espaço ou quebra de linha** fazia a publicação
  falhar com "Malformed input to a URL function". O token é limpo antes de
  usar, e um valor malformado vira erro claro.

## [0.6.2] - 2026-11-25

> **Sobre a 0.6.1:** a tag `v0.6.1` foi criada por engano num commit da
> 0.6.0. O que saiu com esse número (release do GitHub, pacote NuGet) tem o
> código da 0.6.0, sem nada do que está listado abaixo. Use a 0.6.2.

### Adicionado
- **Repositório apt para Debian e Ubuntu:** `sudo apt install tmjlite
  tmjstudio`, com atualização pelo `apt upgrade`. Os índices são assinados; a
  página <https://tmjacometti.github.io/apt> traz os dois comandos de
  configuração. O pacote `tmjstudio` substitui o antigo `tmj-studio`, que
  deixava o app duplicado no menu.
- **TMJStudio pelo Homebrew no macOS:**
  `brew install --cask tmjacometti/tap/tmjstudio` (e `brew upgrade --cask
  tmjstudio` para atualizar), Apple Silicon e Intel. O cask é publicado junto
  com o espelho da release do Studio.

### Alterado
- **`ALTER TABLE` não carrega mais a tabela para a memória.** `ADD`, `DROP` e
  `MODIFY COLUMN` conferem as linhas numa passada de leitura e as reescrevem
  em pedaços; `ADD FOREIGN KEY` confere as referências pelo índice de PK do
  pai. A memória usada é a de um pedaço (4096 linhas), não a da tabela, e os
  rowids não mudam mais.

### Corrigido
- **TMJStudio no Linux: instalar por cima de uma versão antiga duplicava o app
  no menu e os dois abriam a versão velha.** O pacote trazia um atalho com
  `Exec=tmjstudio`, que abre o primeiro `tmjstudio` do `PATH`. O `.tar.gz`
  agora traz um `install.sh`: copia o binário para `~/.local/bin`, grava o
  atalho com o caminho completo, remove atalhos antigos e, se achar uma
  versão antiga instalada no sistema (o `.deb` `tmj-studio`), pergunta se
  pode desinstalar. `./install.sh --uninstall` desfaz.
- **O CLI 0.6.0 (e o 0.6.1) abria dizendo "TMJLite 0.5 - Capivara"** (o `--version` estava
  certo). O texto da abertura agora sai da versão do pacote, e o manual volta
  a mostrar a versão corrente.
- **`DROP COLUMN` de coluna indexada deixava o índice no catálogo** apontando
  para a posição de outra coluna. Agora os índices que usam a coluna saem
  junto com ela.

## [0.6.0] - 2026-11-25

### Adicionado
- **Abrir um banco em uso, somente para leitura e sem lock.** Dá para observar
  um banco que outro processo segura: `tmjlite --read-only <arquivo>` no CLI,
  `ReadOnlyDb` no engine e, no TMJStudio, a abertura automática em modo
  "somente leitura" quando o arquivo está em uso (sem a espera de 10 s), além
  do botão "Abrir somente leitura". Só `SELECT` e `EXPLAIN` rodam; cada
  instrução vê o último commit do dono; nada é criado ou alterado no disco
  (nem `.lock`, nem recovery do WAL). Se o dono fizer checkpoint no meio da
  leitura, a instrução é repetida sozinha e, no limite, devolve o erro
  repetível `BUSY` (501), nunca um resultado misturado. Arquivo legado ou
  anterior à 0.5.0 é recusado neste modo: abra uma vez do jeito normal.
- **Várias instruções na mesma linha do CLI** (`INSERT …; SELECT …;`) rodam em
  ordem. O corte é feito pelo `split_statements` do engine, o mesmo que o
  TMJStudio passou a usar.
- **Resultado dos testes publicado a cada versão.** A release passa a trazer
  `TEST-REPORT.md` e `test-report.json`: contagem por suíte (engine, FFI,
  Studio), torture SQL completo, `kill -9` real com o número de repetições,
  harness de durabilidade e benchmark. Falhas e
  testes ignorados aparecem por nome, e uma tentativa repetida fica
  registrada. Só nomes, resultados e durações — o fonte do engine continua
  privado. O torture completo também roda toda noite no CI.
- **Instalador do CLI para macOS e Linux:**
  `curl -fsSL https://tmjlite.com.br/install.sh | bash` baixa o `.tar.gz` da
  release, confere o SHA-256 e instala o `tmjlite` em `/usr/local/bin` ou
  `~/.local/bin`, sem `sudo`. O CLI de macOS/Linux sai também em `.tar.gz`
  (o `.zip` continua), e a release traz um `SHA256SUMS`.
- **Fórmula do Homebrew** (`brew install tmjacometti/tap/tmjlite`), gerada a
  cada versão quando o tap está configurado.
- **Códigos de erro estáveis.** Todo erro passa a ter, além da mensagem, um
  código numérico que não muda entre versões: violação de `UNIQUE` (401),
  `NOT NULL` (403), chave estrangeira (404), banco travado por outro processo
  (500), arquivo que não é um banco (603), erro de sintaxe (100), tabela
  inexistente (201), corrupção (7xx), E/S (8xx), transação (9xx), merge
  (10xx)… A centena é a categoria. A aplicação não precisa mais procurar
  trechos da mensagem em inglês. A tabela completa está no manual ("Códigos de
  erro") e em `tmjlite.h` (`TMJLITE_ERR_*`). As mensagens não mudaram.
  - **Engine:** `TMJError::code()` devolve um `ErrorCode` (`#[non_exhaustive]`,
    `#[repr(i32)]`), com `category()`, `class()` e `message()`.
  - **API C:** `tmjlite_errcode`, `tmjlite_errcode_shared`,
    `tmjlite_shadow_errcode`, `tmjlite_cursor_errcode` e `tmjlite_open_errcode`
    acompanham cada função de mensagem; `tmjlite_errcode_name` dá o nome de um
    código. Um panic capturado na FFI vem como `TMJLITE_ERR_INTERNAL` (1400).
  - **Drivers:** Python `e.code` / `e.code_name` / `e.category`
    (`tmjlite.ErrorCode`; `TMJLiteError` e `ErrorCode` agora são exportados
    pelo pacote); Node `e.code` (nome), `e.errno` (número) e `e.category`;
    C# `e.Code` (`TMJLiteErrorCode`), `e.ErrorNumber`, `e.Category`; Java
    `e.getCode()` (`ErrorCode`), `e.getErrorNumber()`, `e.getCategory()`; Go
    `errors.Is(err, tmjlite.ErrUniqueViolation)` /
    `errors.Is(err, tmjlite.CategoryConstraint)` e o campo `Code`; Rust
    `e.code()`, `e.raw_code()`, `e.category()`.
- **Versão da ABI:** `tmjlite_abi_version()` devolve um inteiro que só sobe
  quando a API C muda, separado de `tmjlite_version()`. Esta é a ABI 2 (uma
  biblioteca sem o símbolo é a ABI 1). Os seis drivers carregam os símbolos
  novos de forma opcional: driver novo com biblioteca nativa antiga continua
  funcionando, com código de erro "desconhecido" (-1). Nos drivers:
  `abi_version()` / `abiVersion()` / `AbiVersion()` / `ABIVersion()`.

### Corrigido
- **Subquery correlacionada por FK devolvia vazio logo depois de abrir o
  banco.** `… WHERE EXISTS (SELECT … FROM pedidos p WHERE p.cliente_id = c.id)`
  com índice em `cliente_id` não achava nada enquanto a tabela não tivesse
  recebido uma escrita na sessão: o seek comparava o UUID da PK com a coluna
  de texto pelo tipo errado. Agora acha.
- **`INSERT` com apóstrofo gravava o texto errado.** `INSERT INTO t VALUES
  ('it''s')` guardava `it''s` (com as duas aspas) quando passava pelo caminho
  rápido de insert. Agora grava `it's`.
- **SQL muito aninhado não derruba mais o processo.** Parênteses ou
  subqueries com milhares de níveis, centenas de milhares de linhas de
  comentário ou cadeias enormes de `AND`/`OR` estouravam a pilha. O parser
  limita o aninhamento a 32 níveis (erro `NESTING_TOO_DEEP`, 1101) e trata os
  outros casos sem recursão.
- **Panic do engine não atravessa mais a FFI.** Toda função da biblioteca
  nativa captura o panic e devolve erro (`INTERNAL`, 1400) pelo canal que já
  usava, em vez de encerrar o processo da aplicação.
- **macOS não mata mais o `tmjlite` baixado pelo navegador.** O CLI e a
  `libtmjlite_ffi.dylib` que vai dentro dos drivers (PyPI, npm, NuGet, Maven,
  crates.io, módulo Go e zips) passam a sair assinados com Developer ID e
  notarizados pela Apple, como o Studio já saía. Antes o binário só tinha a
  assinatura ad-hoc do linker e o Gatekeeper encerrava o processo na primeira
  execução (`zsh: killed ./tmjlite`); o contorno com `xattr` deixa de ser
  necessário.

### Alterado
- **DDL dentro de transação agora é desfeito pelo `ROLLBACK`.** `CREATE INDEX`,
  `DROP INDEX` e `ALTER TABLE … SET MERGE POLICY` dentro de `BEGIN` eram
  aplicados na hora e sobreviviam ao `ROLLBACK`; agora fazem parte da
  transação, como `CREATE TABLE` já fazia. Quem contava com o índice criado
  mesmo depois de um `ROLLBACK` precisa criá-lo fora da transação.
- **`INSERT`, `UPDATE` e `DELETE` não carregam mais a tabela para a memória.**
  A instrução lê do arquivo só as linhas que o `WHERE` alcança (pelo índice,
  quando há) e grava direto. Um banco de vários GB deixa de ocupar RAM do
  tamanho da tabela na primeira escrita. Vale também para tabelas com FK
  (o pai é conferido pelo índice de PK; `RESTRICT`, `CASCADE` e `SET NULL`
  buscam os filhos pelo índice da FK, quando existe) e para `INSERT … ON
  CONFLICT`. `CREATE INDEX`, `DROP TABLE` e `TRUNCATE` também não carregam
  nada (o índice é montado lendo a tabela em pedaços). Só `ALTER TABLE` que
  mexe em coluna ainda carrega a tabela, e a solta no fim da instrução.
  `DELETE` linha a linha ficou ~4× mais rápido, `UPDATE` ~2×; `INSERT`, igual.
  Para quem usa o crate direto: `db.tables` só tem o que foi materializado
  com `load_table`; o schema se lê com `table_names()` e o novo
  `table_columns(nome)` / `table_row_count(nome)`.
- **TMJStudio: a lista de tabelas não carrega mais as linhas.** O explorer lia
  todas as tabelas inteiras para mostrar nome, colunas e contagem; agora vem
  do catálogo do arquivo.
- **A transação passou a viver no arquivo, não na memória.** Cada instrução de
  um `BEGIN` é gravada na hora e fica pendente até o `COMMIT`; o `ROLLBACK`
  descarta o pendente. Some a cópia da tabela inteira que DDL e `TRUNCATE`
  guardavam para o `ROLLBACK`, e o `COMMIT` deixa de reescrever a tabela.
  `INSERT`s seguidos continuam indo em lote (100 mil numa transação: igual à
  versão anterior). Se o disco falhar no meio de uma transação, ela é desfeita
  inteira e o erro diz isso.
- **Licença.** O código-fonte do engine, da FFI, do CLI e do TMJStudio é
  proprietário da TMJ Sistemas de Informação (arquivo `LICENSE`). Os drivers
  continuam sob MIT, cada um com o seu `LICENSE`. Os binários oficiais podem
  ser usados e redistribuídos sem modificação.
- **Uma linha só com `;` no CLI não faz nada** (antes dava erro de parse).
- **Coluna `HASH`: o formato SHA-256 antigo não é mais verificado.** As
  funções de verificação não tinham uso no engine e foram removidas; uma linha
  gravada antes de 2026-06 só casa com `WHERE col = '<texto>'` depois de um
  `UPDATE` com o texto, como já acontecia.
- **CLI em crate próprio** (`tmjlite-cli`); a biblioteca do engine não depende
  mais de `rustyline` e expõe só a fachada pública do `lib.rs`.
- **Driver Rust: `Error::Database` guarda `DatabaseError`** (mensagem +
  código) no lugar de `String`. O tipo se comporta como a mensagem (`Deref`
  para `str`, `Display`, `Debug` igual, `==` com `&str`/`String`,
  `From<&str>`/`From<String>`), então `Error::Database(m) if m.contains("…")`
  continua compilando. Precisa de ajuste só quem movia o conteúdo para uma
  `String` (use `m.into()` ou `m.into_string()`) ou construía a variante com
  uma `String` já tipada (acrescente `.into()`).
- **Driver Rust carrega biblioteca nativa mais antiga:** os símbolos da ABI 2
  são opcionais, em vez de recusar a biblioteca inteira por um símbolo ausente.
- **Driver Go liga com `-ldl` no Linux** (os símbolos da ABI 2 são resolvidos
  com `dlsym` em tempo de execução).

## [0.5.1] - 2026-11-24

### Corrigido
- **Texto depois do statement não é mais descartado em silêncio.** O parser
  parava no fim do primeiro statement e ignorava o resto: `DELETE … LIMIT 1`
  apagava todas as linhas do `WHERE`, `OFFSET` e `RETURNING` eram ignorados,
  `UPDATE … FROM` atualizava a tabela inteira e, em `a; b`, só `a` rodava.
  Agora qualquer coisa depois do statement (e do `;` opcional) é `Parse
  error`, e nada é executado. Vale para `execute`, `query` e statements
  preparados, em todos os drivers.
  - **Muda comportamento:** `OFFSET`, `RETURNING`, `DELETE … LIMIT` e
    `UPDATE … FROM` passam a dar erro de sintaxe (antes "funcionavam" com
    resultado errado). Vários statements numa chamada são recusados com
    `multiple statements not supported` — mande um por chamada. No CLI, um
    script com dois statements na mesma linha (`INSERT …; INSERT …;`) passa
    a dar erro: um statement por linha. O Studio já divide o SQL sozinho.
- **`UPDATE` que viola índice `UNIQUE` não altera mais nada.** A linha era
  gravada em memória antes da checagem e a chave antiga saía do índice antes
  de a nova ser testada: o erro voltava, mas tabela e índice ficavam
  discordando na sessão, e o statement válido seguinte levava as linhas
  duplicadas ao disco (em transação, o `COMMIT` gravava a parte do `UPDATE`
  que tinha "passado"). Agora o `UPDATE` inteiro é validado contra o estado
  final antes de tocar em qualquer linha. O ramo `DO UPDATE` do upsert tinha
  o mesmo defeito no índice e foi corrigido junto.
- **Upsert de várias linhas é tudo ou nada.** `INSERT … VALUES (…), (…) ON
  CONFLICT …` aplica as linhas uma a uma: um erro na segunda (outro índice
  único, `NOT NULL`, FK) deixava a primeira gravada, e ela ia para o disco no
  statement válido seguinte ou no `COMMIT`. Agora um erro desfaz tudo o que o
  statement fez.
- **Texto com byte NUL (`\0`) é recusado na entrada**, em literal e em
  parâmetro (INSERT, UPDATE, upsert). O engine gravava o texto inteiro, mas a
  FFI devolve texto como C string e ele voltava vazio. O erro agora sai na
  gravação, em todos os drivers (antes só o driver Rust recusava, e só o
  literal). Dado binário fica para o tipo `BLOB`.
- **`CREATE INDEX` na mesma transação do `CREATE TABLE`** falhava com `table
  not found` (o índice ia ao engine na hora, a tabela só no `COMMIT`) e ainda
  deixava o nome do índice "ocupado" até fechar o banco. Agora funciona, o
  `ROLLBACK` desfaz os dois, e um `CREATE INDEX` recusado não deixa rastro.
- **Erro de abertura diz o motivo.** `tmjlite_open*` devolvia NULL sem
  explicação e os drivers respondiam "em uso por outro processo?" para
  qualquer falha. Agora a mensagem distingue arquivo travado, arquivo que não
  é um banco TMJLite, permissão negada e diretório inexistente. Um arquivo de
  **0 bytes** é aceito como banco novo (permite pré-criar o arquivo com a
  permissão desejada). FFI: função nova `tmjlite_open_error()` — só adição;
  os seis drivers passam a usá-la e exigem a lib nativa desta versão.
- **Arquivo principal vazio com `.pwal` não vira banco novo.** Um `.tmjp` de
  0 bytes só é inicializado se não houver WAL com dados ao lado. Com WAL
  completo (banco que nunca passou por checkpoint) o banco é recuperado
  inteiro; com WAL parcial a abertura é recusada (`does not hold the whole
  database`) e nada é gravado — antes esse caso abria como um banco sem
  tabelas e o checkpoint da recuperação gravava por cima.
- **Texto com NUL já gravado** (banco de versão anterior) volta pela FFI com
  `U+FFFD` no lugar de cada NUL, em vez de uma string vazia: dá para achar e
  regravar essas linhas.
- **Upsert devolve a PK** (`last_pk`): a da linha inserida ou a da linha
  atualizada, quando o statement afeta uma linha. Vinha sempre vazia.

### Alterado
- **Arquivos novos nascem com permissão `0600`** (Unix): `.tmjp`, `.pwal`,
  `.lock`, temporários e os oplogs de shadow/branch (`.tmjv`, `.tmjb`,
  `.tmjc`; o diretório `.shadows/` nasce `0700`). Antes valia o padrão do sistema
  (`0644` com a umask usual), legível por outros usuários da máquina. Os
  auxiliares herdam o modo do `.tmjp` quando ele já existe, e o `VACUUM`
  preserva o modo. **Arquivos existentes não são alterados**: para fechar um
  banco antigo, `chmod 600` nele. Para um banco novo mais aberto, crie o
  arquivo vazio com o modo desejado antes de abrir. Consequência a conhecer:
  um `.tmjp` de 0 bytes **sem** `.pwal` ao lado abre como banco novo e vazio,
  sem aviso — se um arquivo que tinha dados aparecer com 0 bytes (disco,
  cópia interrompida), restaure o backup antes de abrir. O `.lock` vazio continua
  ao lado do banco depois de fechar: apagá-lo abriria uma corrida entre
  processos com versões diferentes do engine.

## [0.5.0] - 2026-11-23

### Adicionado
- **Chaves estrangeiras** (`REFERENCES` / `FOREIGN KEY`). Até aqui o
  relacionamento era uma coluna `STRING(36)` com o UUID do pai e a integridade
  ficava por conta da aplicação (com corrida entre o "pai existe?" e o
  INSERT). Agora o engine garante:
  - `coluna TEXT|STRING(36+) REFERENCES pai [ON DELETE RESTRICT | NO ACTION |
    CASCADE | SET NULL]`, também na forma de tabela `FOREIGN KEY (col)
    REFERENCES pai (pk)`. A FK aponta sempre para a `PK` do pai, uma coluna
    só; auto-referência é aceita. Sem `ON UPDATE` (a PK é imutável).
  - INSERT / UPDATE / upsert no filho exigem o pai; `DELETE` no pai aplica a
    ação — planejado inteiro antes de tocar qualquer linha (statement
    atômico, cascata multi-nível, desfeita por `ROLLBACK`); `DROP TABLE` /
    `TRUNCATE` do pai são recusados enquanto houver filho.
  - `ALTER TABLE t ADD FOREIGN KEY (col) REFERENCES pai …` em tabela já
    populada (valida as linhas existentes e diz quantas estão órfãs) e
    `ALTER TABLE t DROP FOREIGN KEY col`; `ADD COLUMN … REFERENCES`;
    `MODIFY COLUMN` preserva a FK.
  - Sessões shadow / branches: as FKs valem dentro da sessão, a cascata vira
    ops no oplog, e o merge revalida o estado final contra o main — recusa
    (`merge conflict: foreign key …`) em vez de criar órfão.
  - A FK mora no catálogo do `.tmjp` (campo novo, opcional, na coluna).
    `.schema` e o DDL do Studio mostram o `REFERENCES`. As palavras novas não são reservadas (`key`, `foreign` etc.
    continuam nomes de coluna válidos). Sem mudança na FFI nem nos drivers.
  - O engine não cria índice na coluna da FK: crie um (`CREATE INDEX`) para o
    `DELETE` do pai achar os filhos por seek em vez de varrer a tabela filha.
- **Índice de PK no arquivo.** Toda tabela ganha um índice automático na
  coluna `PK`, gravado junto com os dados (UUID em 16 bytes, não nos 36 do
  texto). Até aqui o índice de PK existia só em RAM, depois de a tabela ser
  carregada por uma escrita: num arquivo recém-aberto e nas leituras em
  snapshot da conexão compartilhada, `WHERE id = ?` **varria a tabela
  inteira**. Agora é uma descida na B+tree em qualquer conexão (num teste com
  5 mil linhas: até 8 páginas lidas, contra mais de 100 da varredura). Custo:
  35 a 47 bytes por linha em cada tabela e ~4% no tempo de INSERT em
  autocommit (a PK é aleatória: uma página a mais por INSERT).
- **`ORDER BY … DESC LIMIT n`, `MAX()` e `MIN()` pelo índice.** A B+tree
  ganhou varredura reversa. `ORDER BY col DESC LIMIT n` lê só as `n` últimas
  entradas (`IndexSeek(…, order=desc)`), inclusive com prefixo de índice
  composto (`WHERE a = ? ORDER BY b DESC LIMIT n` num índice `(a, b)`) e com
  intervalo na mesma coluna. `SELECT MAX(c), MIN(c) FROM t [WHERE a = ?]` vira
  uma descida por agregação, sem ler a tabela (`IndexMinMax`). Com dois
  índices servindo ao `WHERE`, o planner agora prefere o que já entrega na
  ordem pedida.
- **JOIN lendo só as linhas necessárias.** Quando o `WHERE` tem condição
  indexada só sobre a tabela do `FROM` e cada `JOIN` (`INNER`/`LEFT`, `ON` de
  igualdade) chega numa coluna indexada (a PK sempre é), a base vem por seek
  e cada tabela juntada tem só as chaves encontradas buscadas no índice
  (`IndexLookupJoin`). Na conexão compartilhada, todo JOIN carregava as
  tabelas inteiras do snapshot a cada consulta; num teste de 3 tabelas
  (12 mil linhas) um JOIN pontual lê 9 páginas e um de 40 pedidos vizinhos
  lê 17, contra 296 do JOIN inteiro. Fora dessas condições nada muda.
- **`VACUUM`**: reescreve o banco num arquivo novo e compacto e troca pelo
  atual (o arquivo nunca encolhia: `DELETE`/`DROP TABLE` só devolviam páginas
  à lista de livres). Arquivo novo gravado e sincronizado por inteiro antes de
  um `rename` atômico; exige conexão exclusiva, fora de transação (a conexão
  compartilhada recusa com explicação). Num teste que apaga 90% das linhas e
  uma tabela inteira: 3,5 MB → 205 KB.
- **Formato de arquivo v3**, com atualização automática na primeira abertura
  de um arquivo ≤ 0.4.0: completa os índices com as linhas NULL (ver
  *Corrigido*) e cria o índice de PK de cada tabela — uma leitura de cada
  tabela, um commit só, idempotente. **Sem volta**: a versão do arquivo sobe
  de 2 para 3 e um engine ≤ 0.4.0 passa a recusá-lo (`unsupported page file
  version: 3`) em vez de devolver resultado errado ou deixar entradas órfãs
  nos índices. Atualizar CLI, drivers e Studio juntos.
- **TMJStudio: diagrama de tabelas e chaves estrangeiras** (botão *Diagrama*).
  Uma caixa por tabela, uma linha por FK. Arrastar uma coluna até outra tabela
  cria a FK (nos dois sentidos: coluna da filha → pai, ou PK do pai → coluna da
  filha; soltar a PK no cabeçalho da filha cria a coluna junto); um diálogo
  mostra o SQL e o `ON DELETE` antes de executar. Clicar numa linha edita ou
  remove a FK. Só leitura em branch/snapshot. Erros de FK ganharam mensagem
  amigável e o plano do `EXPLAIN` descreve `IndexMinMax`, `IndexLookupJoin` e
  `order=desc`.
- **Apelido de coluna**: `SELECT COUNT(*) AS total`, `SELECT col AS nome` —
  antes era erro de parse. O apelido vira o nome da coluna de saída (em todos
  os caminhos: executor, snapshot, cursor, atalho do `COUNT(*)`) e pode ser
  usado em `ORDER BY` e `GROUP BY`. O `AS` é obrigatório.
- FFI: `tmjlite_result_type` e `tmjlite_cursor_type` devolvem o tipo que o
  engine guardou no valor (0 NULL, 1 INT, 2 FLOAT, 3 BOOL, 4 TEXT, 5 UUID,
  6 DATE, 7 DATETIME). Só adição — a ABI existente não muda.

### Corrigido
- **Índices não guardavam linhas com NULL — duas consultas davam resultado
  errado** (existia desde os índices compostos). (1) Seek por prefixo:
  `WHERE a = 1` usando um índice `(a, b)` não devolvia as linhas com
  `b IS NULL`. (2) `ORDER BY col LIMIT n` atendido pelo índice não devolvia as
  linhas com `col IS NULL` — o mesmo `SELECT` sem `LIMIT` devolvia. Agora toda
  linha tem entrada em todo índice (NULL ordena antes de qualquer valor):
  o seek por prefixo é completo e a ordem do índice é a mesma do sort (NULL
  primeiro no ASC, por último no DESC). Intervalos e igualdades continuam
  nunca casando com NULL, e `UNIQUE` continua aceitando vários NULL. Arquivos
  existentes são corrigidos na primeira abertura (formato v3).
- `ALTER TABLE … ADD/DROP COLUMN` não reposicionava as colunas dos índices no
  engine (as posições antigas valiam até reabrir o arquivo).
- **Drivers devolviam texto só de dígitos como número** (Python, Node, Go, C#,
  Java e Rust). O tipo de cada célula era *adivinhado pelo texto*: numa coluna
  `TEXT`/`STRING`, `'593'` voltava `593` e `'007'` voltava `7` (zero à esquerda
  perdido); `'true'` virava booleano, `'1e5'` virava float, o texto `'NULL'`
  virava nulo, e um `FLOAT` `2.0` voltava inteiro. Agora o driver pergunta o
  tipo ao engine (`tmjlite_*_type`) em `query()`, prepared e cursor.
  **Muda o que a aplicação recebe**: quem dependia da conversão implícita de
  uma coluna de texto para número passa a receber `str`/`string` — o tipo
  declarado na coluna. `PK`, `DATE` e `DATETIME` continuam texto. Exige a lib
  nativa desta versão (os drivers já a trazem embutida).
- **`SUM` de coluna `INT` devolvia `FLOAT`** (somava em `f64`, com perda acima
  de 2^53). Agora a soma de inteiros é exata e devolve `INT`; com `FLOAT` no
  meio (ou estouro de `i64`) devolve `FLOAT`. Vale para o `SUM` agregado e o
  `SUM(...) OVER`. `AVG` continua `FLOAT`.
- **`WHERE fk = ? ORDER BY outra_coluna LIMIT n` varria a tabela.** Quando a
  coluna do `ORDER BY` também tinha índice, o planejador tentava o atalho
  "índice já entrega em ordem", via que o `WHERE` era em outra coluna e
  desistia de *qualquer* seek — plano `TableScan → Filter → ExternalSort`.
  Agora segue para o seek pelo índice do `WHERE` e ordena só as linhas que
  sobraram (`IndexSeek → ExternalSort → Limit`). Mesmo plano no executor, no
  snapshot e no cursor.
- **Arquivo ~2x maior em autocommit do que em transação** (≈416 vs ≈221
  bytes/linha no mesmo dado). O split da B+tree era sempre ao meio; com chaves
  crescentes (o rowid de todo INSERT) cada folha ficava pela metade para
  sempre. Agora a inserção no fim da última folha faz *split de append* (a
  folha cheia fica cheia, a célula nova abre a seguinte), também nos nós
  internos. Autocommit passa a ocupar o mesmo que o lote. Sem mudança de
  formato: arquivos existentes abrem igual e passam a crescer mais devagar
  (as páginas já pela metade não são recompactadas).

## [0.4.0] - 2026-11-10

### Corrigido
- **Cursor aberto dentro de transação não streamava** (relatório do tmjLens,
  2026-11-10; afetava 0.3.1–0.3.9). Com a thread em `BEGIN`, `cursor()`
  executava o SELECT pelo executor da transação e devolvia o resultado inteiro
  materializado, `is_streaming = 0` — mesmo num walk de 12 linhas por índice.
  E o modo estrito não valia ali (passava com a flag em 0, enquanto fora da
  transação uma forma não streamável falhava): dois contratos para o mesmo
  conceito. No tmjLens 0.7.4 isso derrubou todo `POST /api/ingest/rollups`
  desde 16/10. Agora o cursor em transação **streama da visão da própria
  transação**: o planejamento é o mesmo do snapshot (`plan_streaming`), só a
  lista de rowids candidatos é montada na abertura (seek por índice quando há)
  e cada linha é buscada no passo — do overlay em RAM, com as escritas não
  commitadas, ou do engine para tabelas intocadas. Contrato único: forma
  streamável streama dentro e fora; forma não streamável materializa, ou falha
  em modo estrito, dentro e fora. O cursor vale até o COMMIT/ROLLBACK (avançar
  depois é erro, e não lê a transação seguinte); linhas inseridas depois da
  abertura não entram na iteração, apagadas são puladas. Sem mudança na FFI
  nem nos drivers.

## [0.3.9] - 2026-11-06

Origem: dossiê do loadTest, resultado pós-0.3.8 (P1 "escrita serializada: sem
mudança", 98 writes/s) — a causa era CPU dentro do lock, não fsync.

### Corrigido / desempenho
- **Argon2 das colunas `HASH` fora do lock do writer (P1)** — o hash
  (argon2id OWASP, ~11 ms de CPU por valor) era calculado DEPOIS de pegar o
  lock global de escrita: enquanto uma thread hasheava, as outras esperavam,
  e o group commit nunca coalescia (fsyncs/commits = 1). Agora
  `SharedConnection` pré-hasheia os literais de colunas HASH antes do lock —
  INSERT (valores e `ON CONFLICT DO UPDATE SET`), UPDATE (atribuições e
  WHERE), DELETE (WHERE), inclusive INSERT/UPDATE preparados — com o schema
  vindo do snapshot (cache por tabela, limpo em DDL). O executor reconhece
  valores já em PHC `$argon2id$` e não hasheia de novo; formato gravado,
  `hash_login` e arquivos existentes não mudam. Bench (`INSERT` com senha,
  M5 10 núcleos): 8 threads **97,5 → 356 writes/s**, 32 threads 373/s
  (fsyncs/commits 0,64 — o group commit voltou a trabalhar); com 16 leitores
  concorrentes 90 → 286/s. 1 thread: igual (71/s; o argon2 é o custo).
  Sem coluna HASH nada muda (1.520/s com 8 threads).
- `gate.py`: cenário novo — razão de paralelismo da escrita com coluna HASH (8 threads ÷ 1; mín. 1,5; com o hash dentro do lock dá ~1,0).

## [0.3.8] - 2026-11-06

Origem: dossiê de ajustes do loadTest (bench de 05/11/2026, rodado na 0.3.5) —
itens P5, P2, P3, P4; o P1 (group commit) já saiu na 0.3.7.

### Corrigido / desempenho
- **Custo de escrita que crescia com a tabela (P2)** — no `data_scale` do
  harness o seed ia de 95 → 142 → 197 µs por cliente (1k → 50k). Perfil
  (`sample` no caminho de pedidos): 44% fsync (ticket, WAL, checkpoint e
  **fsync de diretório** ao recriar o WAL), 35% checkpoint reescrevendo páginas
  uma a uma, 16% SHA-256 por frame, 11% inserção nos índices. Com chave
  aleatória (UUID de cliente no índice de pedidos) cada commit suja uma página
  por linha por índice; o checkpoint a cada 4 MiB virava custo fixo por commit.
  Mudanças no pager (**WAL v2**, arquivos existentes continuam válidos):
  - frames com **CRC-32 + salt de geração** em vez de SHA-256 (`crc32fast`,
    instruções da CPU; mesmo polinômio do checksum de página, cuja implementação
    bit a bit também foi trocada — valores idênticos, ver teste);
  - checkpoint **trunca e reaproveita** o WAL (salt novo) em vez de apagar e
    recriar: sem fsync de diretório por ciclo; frames da geração anterior que
    sobrevivam a uma queda são ignorados pelo salt (teste
    `frames_from_previous_wal_generation_are_ignored`); WAL v1 deixado por
    versão anterior ainda é recuperado (`recovers_legacy_v1_wal`);
  - limite de checkpoint 4 → **16 MiB** e páginas contíguas gravadas numa
    escrita só; o fold roda fora do lock dos leitores (só a troca de índice +
    reset é atômica);
  - recuperação: frame ilegível é rabo torto (para no último commit inteiro),
    inclusive o primeiro frame de uma geração reaproveitada.
  Resultado (`engine/bench/data_scale.py`, réplica do harness): seed **plano**
  em 115 / 104 / 119 µs por cliente; pedidos 41 → 93 µs/linha virou 35 → 42.
- **`SELECT COUNT(*) FROM t` em O(1) (P3)** — contador de linhas versionado no
  catálogo (`CatalogEntry.row_count`), mantido em INSERT/DELETE/TRUNCATE e
  gravado com o commit; o handle compartilhado responde do catálogo do
  snapshot sem ler página de dados. 50k linhas: 29,9 → 0,03 ms. Catálogos
  antigos (sem o campo) caem na varredura até a primeira reescrita da tabela.
- **Planner: igualdade fora do índice vira filtro residual** —
  `WHERE city = ? AND active = TRUE` com índice só em `city` fazia
  `TableScan`; agora `IndexSeek(city) → Filter → Limit`.
- **Faixas de índice lidas sob demanda no snapshot** — o seek materializava
  todos os rowids da faixa (10k da cidade) antes do `LIMIT`; agora lê em
  pedaços (64 → 2048) conforme a cauda puxa.
- **B+tree sem cópia de página no caminho quente** — `descend_to_leaf` lia cada
  página duas vezes (tipo + nó) copiando 4 KiB e parseando o nó inteiro com
  uma alocação por célula; `get` parseava a folha toda. Novo
  `PageRead::with_page` empresta a página do cache e a descida/lookup leem os
  bytes crus. No snapshot de 50k: 10k linhas via índice 387 → 67 ms,
  `COUNT(*) … WHERE city = ?` 368 → 46 ms, `LIMIT 50` via índice 4,4 → 1,6 ms;
  subqueries do gate 15 → 5 ms.

### Adicionado
- **Estatísticas do group commit nos drivers (P4)**: `tmjlite_group_commit_stats_shared`
  na FFI e `group_commit_stats()` / `groupCommitStats()` / `GroupCommitStats()`
  nos seis drivers — `(fsyncs, commits)` desde a abertura; razão < 1 = escritores
  concorrentes coalescidos. Permite ao harness separar espera por fsync de
  tempo de execução no cenário `isolation`.

### Testes
- **`tests/crash_kill9.rs`** (P5): durabilidade com processo real morto por
  `kill -9`. O binário de teste reexecuta como filho com 4 escritores via
  `SharedDb` (autocommit e transações de 3 statements), imprime o id de cada
  commit **depois** do retorno; o pai mata num instante determinístico por
  repetição, reabre (recuperação do WAL) e exige faltando = 0 e nenhuma
  transação pela metade. 8 repetições no CI; `TMJ_KILL9_REPS=50` para a
  rodada longa (50 reps, 7.475 commits confirmados, 0 faltando, 0 parciais).

### TMJStudio reescrito em Rust nativo (egui)
- O Studio deixou de ser Tauri 2 + React/Vite/CodeMirror e virou um binário
  único **Rust + eframe/egui 0.37** (`studio/`, crate `tmjstudio` no
  workspace), dependendo do engine por path: sem FFI, webview ou npm. Tema
  próprio (azul-noite, ciano e menta), fontes Outfit e JetBrains Mono embutidas.
- Paridade com o Studio anterior: explorer com menu de contexto, editor com
  abas/highlight/histórico, grid virtualizado com ordenação, filtro, edição
  por duplo clique e exclusão, EXPLAIN com dicas, painel git-native completo
  (branches, commit, log, merge com dry-run e conflitos), snapshot mode,
  importação CSV, detecção de escrita externa. Toda operação no banco roda
  em thread de trabalho — a janela não congela em SELECT grande ou import.
- **Removido**: suporte a SQLite (abrir `.db`, importar SQLite → `.tmjp`). O
  trait `DbBackend` fica como costura para plugins de outros engines (v2).
- **Autocomplete no editor**: tabelas, colunas (com `alias.` resolvido pelo
  FROM/JOIN da aba) e palavras-chave; ↑↓ navega, Tab/Enter aceita, Esc fecha.
- Empacotamento próprio: `scripts/bundle-studio.sh` gera `.app`/`.dmg`
  (assinado com Developer ID; notarização no CI), `.tar.gz` (Linux) e `.zip`
  (Windows). `studio-release.yml` reescrito; nomes de asset
  `TMJStudio_<ver>_<label>.<ext>` (manifest da landing reconhece os dois
  formatos).

## [0.3.7] - 2026-11-05

Origem: backlog pós-0.3.5 (achados do harness de carga), itens 1–6.

> **Aviso sobre a 0.3.6 em crates.io, npm e Maven Central:** esses três
> pacotes saíram **sem a lib nativa de macOS** (só linux-x64 e windows-x64).
> O build de macOS nunca começou durante o incidente do GitHub Actions de
> 2026-11-04 e os jobs de publish rodavam com `if: always()`. PyPI e NuGet
> 0.3.6 vieram do rerun completo e estão íntegros. A próxima release
> substitui os três; a 0.3.6 é retirada de circulação (`cargo yank`,
> `npm deprecate`; Maven é imutável).

### Release / CI
- Jobs de publish em npm, crates.io e Maven agora exigem `build` concluído
  com sucesso nas três plataformas (a republicação por `workflow_dispatch`,
  que não compila, continua permitida). Novo input `deprecate_version` no
  dispatch: `cargo yank` + `npm deprecate` de uma versão publicada quebrada.
- `scripts/sync-landing-artifacts.sh` também carimba a versão em
  `drivers/csharp/TMJLite/TMJLite.csproj` (estava em 0.1.6; o CI passava
  `-p:Version`, então o NuGet nunca saiu errado).
- Exemplo `landing/examples/cadastro/rust` portado para o crate público
  `tmjlite` (crates.io): prepared statements com `last_pk`, `IF NOT EXISTS`,
  cursor em streaming, branches/merge e `conflicts()`. Antes dependia do SDK
  interno removido.

### Corrigido
- **Subquery `IN`/`EXISTS`/escalar era O(N×M)** — a subquery rodava de novo
  para cada linha externa, varrendo a tabela interna (8 s com 2k clientes ×
  4k pedidos). Agora as referências ao escopo externo no WHERE da subquery
  viram literais da linha corrente antes de executar (o planner interno
  usa o índice: `customer_id = '<uuid>' AND status = 'open'` → seek), e o
  resultado fica em cache por (subquery, valores ligados) durante o
  statement: subquery não correlacionada roda uma vez e `IN` consulta um
  conjunto. Semântica tri-valorada de `IN`/`NOT IN` com NULL preservada.
  `engine/bench/subquery.py`, 2k × 4k: `IN` 7,9 s → 15 ms; `EXISTS` 8,2 s →
  16 ms; escalar 88 → 12 ms; `NOT IN` 12 ms.
- **Group commit na `SharedConnection`** — cada escrita autocommit (e cada
  COMMIT de transação) pagava o próprio `F_FULLFSYNC` dentro do lock do
  writer, serializando tudo: 90% do tempo do mix de 32 threads do loadTest.
  Agora o commit é anotado no WAL (`Pager::commit_nosync` → `CommitTicket`),
  o lock é solto e a thread espera a durabilidade fora dele; o primeiro a
  esperar vira líder, aguarda uma janela de 1 ms quando há outros
  escritores em voo e faz UM fsync que cobre todos os commits anotados. A
  publicação para leitores acontece só depois do fsync, em ordem de época,
  e cada escrita continua retornando apenas quando durável (teste de queda
  de energia com o VFS simulado: commit anotado sem esperar não sobrevive,
  esperado sobrevive). Quem escreve sozinho não paga a janela.
  16 threads × 50 INSERTs autocommit (Rust, Mac): 3,2 s → 0,6 s, 802
  commits em 108 fsyncs; mix Python de 32 threads: 285 → 407 ops/s, DML de
  ~150 para ~90–120 ms de média (o `transaction_batch` ainda segura o lock
  durante o corpo da transação). `group_commit_stats()` expõe
  (fsyncs, commits). Autocommit de uma thread fica igual (INSERT 3–4 ms).

- **Varredura reparseava a folha da B+tree a cada linha.** O cursor chamava
  `read_leaf` (parse da página inteira, clonando chave e payload de todas as
  células) em cada `next()`: O(células²) por folha. Agora a folha parseada
  fica no cursor e cada célula é movida uma única vez. Vale para tudo que
  varre: `COUNT(*)`, `GROUP BY`, cursores, joins materializados.
  `engine/bench/scan_cost.py`, 50k linhas: `COUNT(*)` 107 → 27 ms (2,1 →
  0,5 µs/linha); `GROUP BY` + agregações 114 → 29 ms; cursor de 1 coluna
  168 → 86 ms (o resto é FFI por linha).

- **`EXPLAIN` na `SharedConnection` materializava a tabela** para planejar
  (~17 ms numa tabela de 10k linhas). O plano só depende do schema e dos
  índices: o rascunho usa tabelas vazias e nada é lido do disco.
- **Planner**: `WHERE k = ? AND n > ?` com índice só em `k` ia de scan (o
  caminho prefixo + filtro residual exigia índice composto). Agora seek por
  `k` e filtro em `n`, em RAM, engine e snapshot.

### Adicionado
- **Gate de performance e memória no CI** (`engine/bench/gate.py` +
  `.github/workflows/perf-gate.yml`, roadmap 3.1): a cada push em `main`
  que toque engine ou driver Python, Linux compila a lib em release e roda
  13 cenários com limiares folgados (subqueries 2k×4k, varredura 50k,
  leitores sob 8 escritores em fsync, vazão de escrita concorrente, lote
  de 450 upserts, teto de memória). Falha = regressão catastrófica; os
  números ficam no artifact `gate.json`. Mac (1,6 s): tudo 10–100× abaixo
  do limiar.
- **Teto do cache de páginas por snapshot configurável**:
  `set_snapshot_cache_pages(n)` na `SharedConnection` (FFI
  `tmjlite_set_snapshot_cache_pages_shared`; nos seis drivers). Cada cursor
  ou query abre um snapshot com cache de até 2048 páginas de 4 KiB (8 MiB) —
  com 32 leitores simultâneos eram ~256 MiB de RSS com o engine em 7 MiB.
  Vale para snapshots abertos depois; 0 desliga; o resultado não muda.
- CLI: `tmjlite --version` (e `-V`/`version`) imprime a versão; qualquer
  outra flag desconhecida é erro de uso. Antes, o argumento virava nome de
  banco e o CLI **criava** `--version` e `--version.lock` no diretório atual.

## [0.3.6] - 2026-11-04

Origem: bootstrap do app .NET de Thiago (2026-11-04) — "não há CREATE TABLE
IF NOT EXISTS" e "o driver C# não devolve o UUID do INSERT".

### Adicionado
- **`CREATE TABLE IF NOT EXISTS` e `CREATE [UNIQUE] INDEX IF NOT EXISTS`**:
  objeto já existente (por nome, ou índice nas mesmas colunas) vira no-op com
  a mensagem `… already exists (IF NOT EXISTS: nothing done).` em vez de
  erro — bootstrap idempotente, inclusive em reabertura a frio.
- **Resultado estruturado de escrita**: `execute()` (e `execute` de prepared)
  nos seis drivers agora devolve `ExecResult` com `rows_affected`, `last_pk`
  (o UUID gerado por um INSERT de UMA linha em tabela com coluna `PK`; vazio
  em multi-linha/UPDATE/DELETE/DDL) e `message`. Nada de parsear
  `"Inserted 1 row … PK: …"`. FFI: `tmjlite_result_rows_affected` e
  `tmjlite_result_last_pk` no `TmjResult` de `tmjlite_query[_shared]` /
  `tmjlite_stmt_query[_shared]`. Por driver: C# `ExecResult Execute(…)`
  (`RowsAffected`, `LastPk`, `Message`); Python `ExecResult`
  (`rows_affected`, `last_pk`); Node `{ rowsAffected, lastPk, message }`;
  Java `ExecResult execute(…)` (`getLastPk()`); Go `Exec(sql)` /
  `PreparedStatement.ExecResult(…)` (`Execute`/`Exec` com erro só
  continuam); Rust `ExecResult { rows_affected, last_pk, message }`.
  Compatível: quem ignorava o retorno `void` continua compilando.

## [0.3.5] - 2026-10-16

Origem: report do loadTester de Thiago na 0.3.3 (2026-10-16): todas as
leituras caíram para poucos ms, menos `SELECT WHERE ?`, `WHERE ? ORDER BY
LIMIT` e `HASH login`, as três **preparadas** (300+ ms de média).

### Corrigido
- **Prepared SELECT/EXPLAIN na `SharedConnection` ainda passava pelo writer.**
  `execute_prepared` pegava o lock do writer para qualquer statement; agora
  SELECT e EXPLAIN preparados vão para o snapshot como os demais
  (`PreparedStatement::is_read`). Prepared de escrita segue no writer.
  Teste: prepared SELECT não espera transação alheia; p95 sob 8 escritores
  em fsync < 20 ms.

## [0.3.4] - 2026-10-16

Origem: `write-load --help` do loadTester de Thiago (2026-10-16) — "skips
known-broken subquery ops (engine table-not-found)".

### Corrigido
- **Subquery no handle compartilhado falhava com "table not found".** O
  caminho materializado do snapshot carregava só as tabelas de `FROM`/`JOIN`
  (`stmt_tables`); `WHERE id IN (SELECT … FROM outra)`, `EXISTS (…)` e
  subquery escalar no SELECT referenciam outra tabela e ela não estava no
  rascunho. Agora a coleta desce nas subqueries em qualquer profundidade
  (também para `EXPLAIN` e cursor). Teste com as três formas do loadTester.

### Nota de versão
- A **0.3.3** teve o workflow cancelado no meio: o crate `tmjlite` 0.3.3 e o
  Maven chegaram a sair (sem este fix); PyPI, NuGet, release do GitHub e
  espelho não. A 0.3.4 é a 0.3.3 completa + este fix, em todos os registros.

## [0.3.3] - 2026-10-15

Origem: loadTest de Thiago (2026-10-16, 32 threads × 24 formas de operação,
16.000 ops, zero erro) — `SELECT WHERE` com mínimo de 0,1 ms e média de
150 ms: leitura que não trabalha, espera.

### Corrigido
- **Leitores entravam na fila do writer.** `SharedDb::snapshot()` pegava o
  mutex do writer para clonar o índice do WAL; toda leitura esperava cada
  escrita autocommit (inclusive o `F_FULLFSYNC`) e transações inteiras de
  outras threads. Agora o pager **publica** o índice commitado ao fim de
  cada commit/checkpoint (`SnapshotHandle`, lock de microssegundos) e o
  leitor abre a view direto dele, sem o writer. Leitura durante transação
  alheia enxerga o último COMMIT (isolamento por snapshot, que era o
  prometido). O checkpoint decide "há leitor?" e dobra o WAL sob o mesmo
  lock, publicando o índice vazio antes de apagar o arquivo — sem janela
  para um snapshot apontar para um WAL removido.
  Bench `engine/bench/reader_writer.py` (32 threads, mix de escrita
  autocommit + transação + leitura, Mac): `SELECT WHERE` 72 → 0,4 ms de
  média e 134 → 1,8 ms de p95; `COUNT(*)` 71 → 4,6 ms; `ORDER BY LIMIT`
  93 → 26 ms; `EXPLAIN` 119 → 32 ms. Escritas continuam no teto do fsync.
- `EXPLAIN` na `SharedConnection` era tratado como escrita (fila do writer).
  Agora roda no snapshot, com os índices do catálogo recriados no rascunho
  para o plano sair igual ao do banco real.

### Adicionado
- `Pager::snapshot_handle()` / `Engine::snapshot_handle()` /
  `ReadEngine::from_view()`: snapshots MVCC sem referência ao writer.
- Testes: handle enxerga só o commitado; checkpoint adia com view viva e
  dobra depois; corrida de 4 leitores por handle × 150 commits/checkpoints;
  leitor não espera transação alheia; p95 de leitor sob 8 escritores em
  fsync < 20 ms; `EXPLAIN` por índice no snapshot com transação aberta.

## [0.3.2] - 2026-10-15

Origem: dossiê #2 do tmjLens em EKS (2026-10-15) — RSS ~19× o arquivo
(111 MiB → 2,1 GiB), crescimento em degrau por lote e depois plano.

### Corrigido
- **Raiz de índice secundário obsoleta no catálogo (silencioso, anterior a
  0.3.x).** Quando a raiz da B+tree de um índice mudava (split da raiz) no
  INSERT autocommit, no INSERT em lote ou no delta de UPDATE, só a meta em
  RAM era atualizada; o catálogo em disco só era regravado quando a raiz da
  *tabela* mudava ou o bloco de rowids estourava. Nessa janela, snapshots
  (`SharedConnection` lendo por cursor/`query`, 0.3.1) e reabertura a frio
  desciam por uma folha antiga e o seek por índice devolvia **vazio sem
  erro**. Agora toda mudança de raiz de índice regrava o catálogo da tabela.
  Teste de regressão: 600 INSERTs autocommit + 600 UPDATEs em transação com
  seek por cursor a cada passo e após reabrir.
- **Três cópias da tabela por lote em transação (H1 do dossiê).** Dentro de
  `BEGIN`, todo SELECT clonava a tabela inteira (`Arc<Vec<Row>>`) e a
  primeira mutação por tabela clonava outra vez como pré-imagem para o
  ROLLBACK; o glibc nunca devolvia o pico. Agora o SELECT em transação
  empresta `&table.rows` (zero cópia) e o ROLLBACK de tabela que só sofreu
  DML desfaz o log de deltas em ordem reversa (pré-imagem só para DDL/
  TRUNCATE, reconstruída a partir dos deltas se chegar depois de DML).
- **`compact_into` caía em scan.** `WHERE a = ? AND b = ? AND c = ? AND d = ?
  AND janela >= ? AND janela < ?` sobre índice `(a,b,c,d,janela)` não casava
  o planner (que exigia a chave inteira por igualdade). Agora: igualdades em
  qualquer ordem formam o prefixo, faixa opcional na coluna seguinte, e o
  seek vale também com `ORDER BY` (ordem do índice quando é a coluna
  seguinte ASC; senão sort só do resultado), com agregação/GROUP BY (só o
  WHERE dirige o seek) e com faixa em coluna fora do índice (filtro
  residual). RAM, engine e snapshot; `EXPLAIN` mostra
  `IndexSeek(prefix=[…], col: lower/upper)`.
- Reprodução (`engine/bench/memory_ingest.py`, 120 chaves × 240 janelas,
  131 MB, Mac): 0.3.1 = 395 MB no primeiro contato (3,0× o arquivo) e
  0,44 s/lote; agora 206 MB (1,6×) plano desde o primeiro lote e 0,05 s/lote,
  com asserção de que cada seek devolve as mesmas linhas que o scan.

### Adicionado
- **Driver Rust público: crate `tmjlite` no crates.io** (`cargo add tmjlite`).
  Até aqui o único driver Rust era o `tmjlite-rs`, que linka o engine em fonte
  e mora no repositório privado — ninguém de fora conseguia usar, embora a
  landing o listasse. O crate novo (`drivers/rust-ffi`) é só bindings
  `extern "C"` sobre a `libtmjlite_ffi` + carregador (`libloading`), com as
  nativas pré-compiladas dentro do pacote (`lib/<plataforma>/`, macOS arm64,
  Linux x64, Windows x64); nenhuma linha do engine sai do privado. Mesma API
  dos outros drivers: `Connection`, `SharedConnection` (prepared, transação
  por thread, cursor em streaming, `set_memory_budget`/`memory_stats`/
  `release_memory`/`set_strict_streaming`), `ShadowConnection` (branches,
  `seal`, `conflicts`, `commit`). Job `publish-crates` no release (input
  `crates_version` para republicar), zip `tmjlite-lib-rust-<ver>.zip`.
  O SDK `tmjlite-rs` (engine em fonte) **saiu do repositório** — quem o
  usava por `git`+tag continua funcionando nas tags ≤ v0.3.2; migre para
  `tmjlite` (mesma API; `tmjlite_rs::` → `tmjlite::`). O exemplo
  `cadastro/rust` ainda aponta para o SDK antigo e será portado depois.
- **Teto de memória e observabilidade na `SharedConnection` (7.1)**:
  `set_memory_budget(bytes)` — acima do teto, depois de cada escrita durável
  (autocommit ou COMMIT), o cache de tabelas é descartado e a memória
  devolvida ao SO (`malloc_trim` no glibc); nunca dentro de transação.
  `memory_stats()` (total, por tabela, índices, cache de linhas) e
  `release_memory()`. FFI: `tmjlite_set_memory_budget_shared`,
  `tmjlite_memory_stats_shared` (JSON) + `tmjlite_string_free`,
  `tmjlite_release_memory_shared`. Nos seis drivers.
- **Modo estrito de streaming (7.2)**: `set_strict_streaming(true)` faz um
  cursor cuja forma não streama (JOIN, DISTINCT, subquery, window function,
  ORDER BY/LIMIT sobre agregação) falhar com o motivo em vez de materializar
  em silêncio. FFI `tmjlite_set_strict_streaming_shared`; nos seis drivers.
- Header documenta o contrato de memória (7.3/7.4): o que fica residente,
  spill do sort a partir de 32k linhas, checkpoint adiado com cursor aberto
  mas WAL só em disco.

## [0.3.1] - 2026-10-15

Origem: dossiê do OOM em produção do tmjLens (2026-10-14/15) — "o formato do
result set como teto de memória".

### Corrigido
- **Leitura no handle compartilhado materializava a tabela inteira.** Toda
  `query_shared` carregava TODAS as linhas das tabelas referenciadas do
  snapshot para a RAM (mesmo com `WHERE` por índice), relia todas as páginas
  do disco a cada query (o snapshot não tinha cache) e só então montava o
  resultado — três cópias, e memória proporcional à tabela, não ao resultado.
  Agora SELECTs de tabela única rodam em **streaming direto da B+tree do
  snapshot** (seek por índice quando há; `LIMIT` para cedo; `GROUP BY` em
  O(grupos)), e o snapshot tem cache de páginas (8 MiB por cursor/query).
  Formas que não streamam (joins, subqueries, window functions, `DISTINCT`,
  `ORDER BY`/`LIMIT` sobre agregação) continuam pelo caminho materializado.
- `tmjlite_version()` devolvia `0.1.0-capivara`; agora é a versão do crate.

### Adicionado
- **Cursor na FFI (4.1)**: `tmjlite_cursor_open_shared`,
  `tmjlite_stmt_cursor_shared` (prepared + params), `tmjlite_cursor_step`
  (1/0/-1), `column_count/name`, `text` (emprestado até o próximo step),
  `is_null`, `i64`, `f64`, `bool`, `is_streaming`, `errmsg`, `close`.
  Memória por query = uma linha. Dentro de `BEGIN` na mesma thread o cursor
  lê a transação; fora, um snapshot consistente que vive até o `close`.
- **Acessores tipados no `TmjResult` (4.2)**: `tmjlite_result_is_null`,
  `_i64`, `_f64`, `_bool`. O `TmjResult` agora guarda os valores tipados e
  só constrói o texto de uma célula quando `tmjlite_result_value` é chamado —
  ponteiro emprestado, válido até `tmjlite_result_free` (documentado no
  header; 4.4).
- `SharedDb::query_cursor` / `query_cursor_prepared` no engine; `cursor()`
  nos drivers Python (`Cursor` iterável), Node (`for…of`), C#
  (`IEnumerable<Row>`), Java (`Iterable<Row>`, try-with-resources), Go
  (`Next()/Row()/Err()`) e Rust (`cursor()`/`cursor_prepared()` com `fold`).
- Driver Rust (`tmjlite-rs`) equiparado aos demais: `prepare` /
  `execute_prepared` / `query_prepared` na `Connection` e na
  `SharedConnection`; `begin/commit/rollback/transaction` na `SharedConnection`.
- `version()` em Python e Node (já existia em C#, Java e Go).
- Benchmark de leitura `engine/bench/read_path.py` (cenário do tmjLens: 1
  container × 2.160 janelas × histogramas de 200 buckets): caminho antigo
  32,6 MB de pico extra por query; cursor 9,2 MB (8 MiB é o cache de
  páginas, teto fixo), independente de N.

### Não feito (segue no backlog)
- Agregação custom registrada pelo cliente (`hist_merge`, 4.3): o cursor já
  leva a compactação a O(1 linha); um tipo/agg de histograma só se isso não
  bastar.

## [0.3.0] - 2026-10-07

### Corrigido
- Recuperação do WAL agora falha explicitamente diante de erros de abertura/leitura
  ou cabeçalho/checksum inválido antes de um commit recuperável, preservando o
  sidecar em vez de abrir dados principais possivelmente antigos.
- `COMMIT` só encerra a transação depois do sync durável; falhas tentam restaurar
  todas as tabelas e liberar o writer. Erros de `ROLLBACK` são propagados e, se
  a restauração falhar, o handle compartilhado exige fechar e reabrir.

## [0.2.9] - 2026-09-29

Origem: diagnóstico de desempenho de escrita do tmjLens (2026-09-29) — cada
escrita auto-commitada custava ~400 ms num banco de 13 MB.

### Corrigido
- **Write-through incremental (6.1)**: UPDATE/DELETE (e INSERT em tabela com
  índice UNIQUE) reescreviam a tabela inteira e reconstruíam todos os índices
  no engine a cada statement — O(N) por commit. Agora o executor registra
  deltas por linha (rowid, antes, depois) e o engine aplica só as páginas
  tocadas; fallback automático para o rewrite quando o engine diverge da RAM.
  UPDATE de 1 linha em 20k linhas: 76 ms → 4 ms (Mac, `.sync full`, o resto
  é o fsync); INSERT 111 → 4 ms; DELETE 84 → 4 ms; dentro de transação
  0,2 ms/stmt.
- Lado RAM: checagem de UNIQUE no INSERT era O(N²) (purge global de ghosts);
  DELETE reconstruía o índice de PK (agora remove chaves); `row_idx_for_rowid`
  era varredura linear (agora busca binária); UPDATE/DELETE usam os índices
  para achar as linhas candidatas em vez de varrer a tabela.
- `SELECT DISTINCT` não parseava ("expected From, got Identifier").
- Prepared INSERT ignorava tokens depois do `VALUES (...)`.

### Adicionado
- **Transações na SharedConnection (6.2)**: `BEGIN`/`COMMIT`/`ROLLBACK` são
  aceitos no handle shared e ficam presos à thread que abriu; outras threads
  bloqueiam em escritas/leituras até o fim. FFI (`tmjlite_exec_shared`) e os
  seis drivers (`transaction()`/`begin()`/`Transaction(...)`).
- **`INSERT … ON CONFLICT (cols) DO NOTHING | DO UPDATE SET col = valor |
  excluded.col` (6.3)**: upsert nativo em um commit, sem SELECT prévio; o alvo
  deve casar um índice UNIQUE ou a PK; funciona em transação e em prepared.
- **Prepared statements genéricos (6.4)**: `?` em INSERT, UPDATE, DELETE,
  SELECT e upsert; FFI `tmjlite_prepare[_shared]`, `tmjlite_stmt_exec/query
  [_shared]`, `tmjlite_stmt_param_count`, `tmjlite_stmt_free` — parâmetros
  como array JSON (tipos preservados, sem escape); `prepare()` em todos os
  drivers (Connection e SharedConnection).
- **`SELECT DISTINCT` (6.5)**, com ORDER BY/LIMIT (dedupe antes do LIMIT).
- Benchmark de referência: 450 upserts preparados em transação (lote do
  tmjLens) = 25 ms no total, contra minutos antes.

## [0.2.8] - 2026-09-03

### Adicionado
- Drivers publicados nos registros oficiais: `pip install tmjlite`,
  `npm install tmjlite`, Maven Central (`io.github.tmjacometti:tmjlite`) —
  além do NuGet que já existia. Os zips em tmjlite.com.br continuam como
  alternativa; o driver Rust segue como SDK sob licença (demo binário no zip).
- `CHANGELOG.md` e notas de versão por release (GitHub Release, NuGet).
- Módulo Go com caminho correto (`github.com/TMJacometti/TMJPlatform/drivers/go`)
  e header embutido, instalável por `go get`.
- Guia "Branches de dados e concorrência" + FAQ (`docs/Branches.md`,
  tmjlite.com.br/branches.html).
- TMJStudio 0.2.8: só TMJLite e SQLite; grid paginado e virtualizado com
  edição inline; editor CodeMirror com abas e histórico; painel git-native
  (branches, commit, log, merge com dry-run e conflitos base/main/branch);
  EXPLAIN; importação de CSV e de SQLite para `.tmjp`; mensagens de erro
  amigáveis; primeiro build macOS assinado e notarizado (Developer ID).

## [0.2.7] - 2026-09-03

### Adicionado
- **Branches de dados git-native (M4/M5)**: branches nomeadas (`.tmjb`) com
  commits e log; `tmjlite branch|checkout|commit|merge|log|status`; REPL de
  branch (`.commit -m`, `.merge`, `.status`, `.log`, `.refresh`).
- Detecção de conflito V2 com linha-base (`UpdateDiverged`, `UpdateOnDeleted`,
  `DeleteDiverged`, `InsertExists`, `FieldConflict`), políticas por tabela
  `lww` / `error` / `field_merge` / `manual`, quarentena `.tmjc`, dry-run.
- `ALTER TABLE t SET MERGE POLICY …` persistida no catálogo; hash de schema
  por tabela no oplog v3 (merge recusa schema divergente).
- Login por SQL em colunas `HASH` (`WHERE senha = 'texto'` compara com o hash).
- FK `TEXT` casa PK `UUID` em JOINs; agregação sobre tabela vazia devolve uma
  linha; coerção `INT` → `FLOAT` em INSERT/UPDATE.
- Sessões shadow e branches em todos os drivers (Rust, Python, Node, Go, Java,
  C#): `connect_shadow`/`open_branch`, `seal`, `set_merge_policy`, `conflicts`.
- Exemplos "cadastro" completos em Rust, Python, Java e .NET
  (tmjlite.com.br/examples.html), com `setup.sh` que baixa o driver.
- Docs consolidadas em `docs/`; README novo.

### Corrigido
- Merge join descartava o filtro `WHERE` empurrado para o índice.
- Alinhamento de colunas do REPL com valores largos.
- Mensagens do parser para datas sem aspas e aritmética em VALUES.

## [0.2.6] - 2026-07-28

### Adicionado
- Shadows M1–M3: sessões shadow anônimas (`.tmjv`) com merge automático por
  chave primária; Studio em *snapshot mode* quando o arquivo está em uso.

### Melhorado
- Performance: backport das otimizações #1–#4 (scan, ordenação com spill,
  índices) e página de evolução de performance na landing.

## [0.2.5] - 2026-07-28

### Adicionado
- Lock entre processos em todos os modos de abertura; `SharedConnection` em
  todos os drivers; erro tipado quando o WAL está ausente.

## [0.2.4] - 2026-07-27

### Corrigido
- Rebuild de índices após reabertura; versão dinâmica na home do Studio.

## [0.2.3] - 2026-07-27

### Corrigido
- **Crítico:** comparação entre tipos distintos avaliava como igual, e
  `UPDATE`/`DELETE … WHERE id = <uuid>` atingia a tabela inteira.

[Unreleased]: https://github.com/TMJacometti/TMJPlatform/compare/v0.6.3...HEAD
[0.6.3]: https://github.com/TMJacometti/TMJPlatform/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/TMJacometti/TMJPlatform/compare/v0.6.0...v0.6.2
[0.6.0]: https://github.com/TMJacometti/TMJPlatform/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/TMJacometti/TMJPlatform/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/TMJacometti/TMJPlatform/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.9...v0.4.0
[0.3.9]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.8...v0.3.9
[0.3.8]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.7...v0.3.8
[0.3.7]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.6...v0.3.7
[0.3.6]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/TMJacometti/TMJPlatform/compare/v0.2.9...v0.3.0
[0.2.9]: https://github.com/TMJacometti/TMJPlatform/compare/v0.2.8...v0.2.9
[0.2.8]: https://github.com/TMJacometti/TMJPlatform/compare/v0.2.7...v0.2.8
[0.2.7]: https://github.com/TMJacometti/TMJPlatform/compare/v0.2.6...v0.2.7
[0.2.6]: https://github.com/TMJacometti/TMJPlatform/compare/v0.2.5...v0.2.6
[0.2.5]: https://github.com/TMJacometti/TMJPlatform/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/TMJacometti/TMJPlatform/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/TMJacometti/TMJPlatform/releases/tag/v0.2.3
