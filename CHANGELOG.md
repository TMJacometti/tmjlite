# Changelog

Todas as mudanças relevantes do TMJLite (engine, CLI, drivers) ficam aqui.
Formato: [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/);
versões seguem [SemVer](https://semver.org/lang/pt-BR/).

A seção da versão publicada é extraída por `scripts/changelog-section.sh` e
vai para a GitHub Release, o NuGet (*Release Notes*) e os READMEs dos pacotes.
O TMJStudio tem ciclo próprio (`studio-vX.Y.Z`) e é anotado na seção da
versão do engine em que saiu.

## [Unreleased]

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

[Unreleased]: https://github.com/TMJacometti/TMJPlatform/compare/v0.3.5...HEAD
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
