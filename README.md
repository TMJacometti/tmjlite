# TMJLite

[![Release](https://img.shields.io/github/v/release/TMJacometti/tmjlite?label=release)](https://github.com/TMJacometti/tmjlite/releases/latest)
[![PyPI](https://img.shields.io/pypi/v/tmjlite?label=PyPI)](https://pypi.org/project/tmjlite/)
[![npm](https://img.shields.io/npm/v/tmjlite?label=npm)](https://www.npmjs.com/package/tmjlite)
[![NuGet](https://img.shields.io/nuget/v/TMJLite?label=NuGet)](https://www.nuget.org/packages/TMJLite)
[![Maven Central](https://img.shields.io/maven-central/v/io.github.tmjacometti/tmjlite?label=Maven%20Central)](https://central.sonatype.com/artifact/io.github.tmjacometti/tmjlite)

**TMJLite** is an embedded SQL database that lives in one `.tmjp` file — no
server, no setup — and treats your data the way git treats your code. Open a
branch, import or fix whatever you need, review it, and merge back into the
main file with row-level conflict detection and per-table policies, while
your application keeps running against an untouched database. Written in
Rust, with a real SQL engine (joins, group by, subqueries, window functions,
transactions, `EXPLAIN`), a CLI, the native **TMJStudio** IDE, and drivers
with the engine bundled for Python, Node, Java, C#, Go and Rust.

This repository hosts the **public releases**, the **Go module** and the
**issue tracker**. Downloads, manual and guides: **<https://tmjlite.com.br>**.

## Install

| | |
|---|---|
| Python | `pip install tmjlite` |
| Node.js | `npm install tmjlite` |
| C# / .NET | `dotnet add package TMJLite` |
| Java | `io.github.tmjacometti:tmjlite` on Maven Central |
| Go | `go get github.com/TMJacometti/tmjlite` (cgo; native libs ship in `lib/`) |
| Rust | SDK under license — binary demo in the release assets |
| CLI | [`tmjlite-cli-<os>-<version>.zip`](https://github.com/TMJacometti/tmjlite/releases/latest) |
| TMJStudio | [`.dmg` / `.msi` / `.AppImage`](https://github.com/TMJacometti/tmjlite/releases) (tags `studio-v*`) |

The native engine is inside every package: nothing to compile.

## 60 seconds

```bash
tmjlite app.tmjp
tmjlite> CREATE TABLE users (id PK, name TEXT, age INT);
tmjlite> INSERT INTO users (name, age) VALUES ('Alice', 30);
tmjlite> SELECT name, age FROM users WHERE age > 18;
```

Branches of data:

```bash
tmjlite branch   app.tmjp importacao        # create
tmjlite checkout app.tmjp importacao        # work in it: INSERT/UPDATE/DELETE, .commit -m "..."
tmjlite merge    app.tmjp importacao        # merge into app.tmjp, row by row, with conflict policies
```

Your application keeps seeing the untouched `app.tmjp` until the merge.
How that works with workers, the Studio and locks:
**[Branches & concurrency guide + FAQ](https://tmjlite.com.br/branches.html)**.

## Go module

```go
import tmjlite "github.com/TMJacometti/tmjlite"

db, err := tmjlite.Open("app.tmjp")
```

The module carries `include/tmjlite.h` and the native library for macOS
(Apple Silicon), Linux x86-64 and Windows x86-64 in `lib/`; the cgo
directives already point at `${SRCDIR}/lib`. See [HOWTO-GO.md](HOWTO-GO.md).

## Links

- Manual: <https://tmjlite.com.br/manual.html>
- Examples (full "cadastro" app in Rust, Python, Java, .NET): <https://tmjlite.com.br/examples.html>
- Changelog: [CHANGELOG.md](CHANGELOG.md)
- Issues and questions: [Issues](https://github.com/TMJacometti/tmjlite/issues)

MIT License · TMJ Sistemas de Informação · <https://www.tmjsistemas.com.br>
