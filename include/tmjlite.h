/*
 * TMJLite C API
 * Cross-platform embedded database engine
 *
 * Usage:
 *   TmjDb *db = tmjlite_open("mydb.tmjdb");  // or tmjlite_open_memory()
 *   tmjlite_exec(db, "CREATE TABLE t (x INT);");
 *   TmjResult *r = tmjlite_query(db, "SELECT * FROM t;");
 *   // ... read results ...
 *   tmjlite_result_free(r);
 *   tmjlite_close(db);
 */

#ifndef TMJLITE_H
#define TMJLITE_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Error codes */
#define TMJLITE_OK    0
#define TMJLITE_ERROR 1

/* ---- Stable error codes (ABI 2+) ----
 *
 * Every place that holds an error message also holds a numeric code:
 *
 *   message                    code
 *   tmjlite_errmsg             tmjlite_errcode
 *   tmjlite_errmsg_shared      tmjlite_errcode_shared
 *   tmjlite_shadow_errmsg      tmjlite_shadow_errcode
 *   tmjlite_cursor_errmsg      tmjlite_cursor_errcode
 *   tmjlite_open_error         tmjlite_open_errcode
 *
 * Message and code are set and cleared together: 0 means "no error".
 * (Prepared statements report through the handle they run on.)
 *
 * The contract:
 *  - a code's numeric value never changes and is never reused;
 *  - new codes may be added in any release, so treat a code you do not know
 *    by its category, TMJLITE_ERR_CATEGORY(code) (code / 100), or as a
 *    generic error;
 *  - the message text is for people and is NOT part of the contract.
 *
 * Do not confuse with the int returned by tmjlite_exec & co, which stays
 * TMJLITE_OK / TMJLITE_ERROR. */
#define TMJLITE_ERR_CATEGORY(code) ((code) / 100)

/* GENERIC */
#define TMJLITE_ERR_ERROR                    1  /* Error with no more specific classification */
/* SQL */
#define TMJLITE_ERR_SYNTAX                 100  /* SQL the parser rejects */
#define TMJLITE_ERR_SEMANTIC               101  /* Well-formed SQL but invalid query (GROUP BY, ORDER BY not in SELECT, value count, multi-row scalar subquery) */
/* SCHEMA */
#define TMJLITE_ERR_SCHEMA                 200  /* Invalid DDL, or schema object state forbids the operation */
#define TMJLITE_ERR_NO_SUCH_TABLE          201  /* Table does not exist */
#define TMJLITE_ERR_NO_SUCH_COLUMN         202  /* Column does not exist */
#define TMJLITE_ERR_NO_SUCH_INDEX          203  /* Index does not exist */
#define TMJLITE_ERR_TABLE_EXISTS           204  /* Table already exists */
#define TMJLITE_ERR_COLUMN_EXISTS          205  /* Column already exists */
#define TMJLITE_ERR_INDEX_EXISTS           206  /* Index already exists (by name or by columns) */
#define TMJLITE_ERR_AMBIGUOUS_COLUMN       207  /* Ambiguous column: qualify it with table or alias */
/* TYPE */
#define TMJLITE_ERR_TYPE_MISMATCH          300  /* Value of a type the column does not accept */
#define TMJLITE_ERR_INVALID_VALUE          301  /* Right type, unacceptable value (longer than STRING(n), invalid date, NUL byte) */
/* CONSTRAINT */
#define TMJLITE_ERR_CONSTRAINT             400  /* Constraint violation with no more specific code */
#define TMJLITE_ERR_UNIQUE_VIOLATION       401  /* Duplicate key in a UNIQUE index */
#define TMJLITE_ERR_PRIMARY_KEY_VIOLATION  402  /* Attempt to set or change the PK (the engine generates it) */
#define TMJLITE_ERR_NOT_NULL_VIOLATION     403  /* NULL in a NOT NULL column */
#define TMJLITE_ERR_FOREIGN_KEY_VIOLATION  404  /* Foreign key without parent, or parent still referenced */
/* ACCESS */
#define TMJLITE_ERR_LOCKED                 500  /* File locked by another process */
#define TMJLITE_ERR_BUSY                   501  /* Operation needs exclusive access that is unavailable now (e.g. VACUUM on a shared connection) */
#define TMJLITE_ERR_READ_ONLY              502  /* Write on a read-only connection, snapshot or medium */
/* OPEN */
#define TMJLITE_ERR_CANNOT_OPEN            600  /* Could not open (reason not covered below) */
#define TMJLITE_ERR_FILE_NOT_FOUND         601  /* File or directory does not exist */
#define TMJLITE_ERR_PERMISSION_DENIED      602  /* Permission denied by the operating system */
#define TMJLITE_ERR_NOT_A_DATABASE         603  /* File exists but is not a TMJLite database */
#define TMJLITE_ERR_UNSUPPORTED_FORMAT     604  /* TMJLite file of a version/format this library cannot read */
#define TMJLITE_ERR_FILE_EXISTS            605  /* File already exists */
/* CORRUPT */
#define TMJLITE_ERR_CORRUPT                700  /* Invalid on-disk structure */
#define TMJLITE_ERR_CORRUPT_CHECKSUM       701  /* Page or file checksum mismatch */
#define TMJLITE_ERR_CORRUPT_WAL            702  /* WAL invalid, incomplete or inconsistent with the main file */
/* IO */
#define TMJLITE_ERR_IO                     800  /* Read/write/storage failure with no more specific code */
#define TMJLITE_ERR_IO_FSYNC               801  /* fsync failed: durability is not guaranteed */
#define TMJLITE_ERR_DISK_FULL              802  /* Disk full or quota exceeded */
/* TRANSACTION */
#define TMJLITE_ERR_TRANSACTION            900  /* Transaction state error with no more specific code */
#define TMJLITE_ERR_NO_ACTIVE_TRANSACTION  901  /* COMMIT/ROLLBACK without an active transaction */
#define TMJLITE_ERR_TRANSACTION_ACTIVE     902  /* A transaction is already active (nested BEGIN, VACUUM inside a transaction) */
#define TMJLITE_ERR_TRANSACTION_ABORTED    903  /* Transaction recovery failed: close and reopen the database */
/* MERGE */
#define TMJLITE_ERR_MERGE                 1000  /* Shadow session/merge error with no more specific code */
#define TMJLITE_ERR_MERGE_CONFLICT        1001  /* Merge conflict */
#define TMJLITE_ERR_SCHEMA_DIVERGENCE     1002  /* The shadow was written against a different schema */
#define TMJLITE_ERR_NO_SUCH_BRANCH        1003  /* Branch does not exist */
#define TMJLITE_ERR_BRANCH_EXISTS         1004  /* Branch already exists */
#define TMJLITE_ERR_UNSEALED_OPS          1005  /* The branch has unsealed operations */
/* LIMIT */
#define TMJLITE_ERR_LIMIT_EXCEEDED        1100  /* Engine limit exceeded (key size, LIMIT too large) */
#define TMJLITE_ERR_NESTING_TOO_DEEP      1101  /* Parentheses/subqueries nested beyond the limit */
/* UNSUPPORTED */
#define TMJLITE_ERR_UNSUPPORTED           1200  /* Unsupported feature */
#define TMJLITE_ERR_MULTIPLE_STATEMENTS   1201  /* More than one statement in a single call */
/* MISUSE */
#define TMJLITE_ERR_MISUSE                1300  /* API misuse (invalid argument, null statement, cursor over a non-SELECT) */
#define TMJLITE_ERR_PARAM_COUNT           1301  /* Parameter count differs from the number of ? */
/* INTERNAL */
#define TMJLITE_ERR_INTERNAL              1400  /* Internal error (broken invariant or captured panic): a TMJLite bug */
/* 405 is reserved for CHECK. */

/* Internal errors: a panic inside the engine never crosses this boundary.
 * It is caught and reported through the function's normal error return
 * (NULL, TMJLITE_ERROR, -1...), with the message "internal error (panic): ..."
 * in the usual place (tmjlite_errmsg*, tmjlite_cursor_errmsg,
 * tmjlite_shadow_errmsg, or tmjlite_open_error for the open functions)
 * and the code TMJLITE_ERR_INTERNAL in the matching *_errcode function.
 * The handle stays valid for reading the error and closing; treat it as a
 * fatal error for that handle, since the interrupted operation may be
 * half done. */

/* Opaque handles */
typedef struct TmjDb TmjDb;
typedef struct TmjResult TmjResult;

/* ---- Lifecycle ---- */

/* Open a database file. Creates the file if it doesn't exist (a 0-byte file
 * counts as a new database). Returns NULL on failure — call
 * tmjlite_open_error() for the reason. */
TmjDb *tmjlite_open(const char *path);

/* Why the last tmjlite_open* / tmjlite_shadow_open* call on THIS thread
 * returned NULL: locked by another process, not a TMJLite database,
 * permission denied, missing directory... NULL if that open succeeded.
 * The pointer is valid until the next open call on the same thread. (0.5.1+) */
const char *tmjlite_open_error(void);

/* Code (TMJLITE_ERR_*) of that same failed open; 0 if the last open on this
 * thread succeeded. (ABI 2+) */
int32_t tmjlite_open_errcode(void);

/* Create an in-memory database (no file, data lost on close). */
TmjDb *tmjlite_open_memory(void);

/* Close the database and free all resources. */
void tmjlite_close(TmjDb *db);

/* ---- Execute ---- */

/* Execute a non-query SQL statement (CREATE, INSERT, UPDATE, DELETE, ALTER, BEGIN, COMMIT, ROLLBACK).
 * Returns TMJLITE_OK on success, TMJLITE_ERROR on failure.
 * On error, call tmjlite_errmsg() for details. */
int tmjlite_exec(TmjDb *db, const char *sql);

/* Execute any SQL statement and get the result.
 * For SELECT: returns a result with columns and rows.
 * For non-SELECT: returns a result with a message (0 columns, 0 rows).
 * Returns NULL on error — call tmjlite_errmsg() for details.
 * Caller must free the result with tmjlite_result_free(). */
TmjResult *tmjlite_query(TmjDb *db, const char *sql);

/* ---- Result accessors ---- */

/* Number of columns in the result. */
int tmjlite_result_column_count(const TmjResult *result);

/* Column name at the given index. Returns NULL if out of bounds.
 * The returned pointer is valid until tmjlite_result_free(). */
const char *tmjlite_result_column_name(const TmjResult *result, int index);

/* Number of rows in the result. */
int tmjlite_result_row_count(const TmjResult *result);

/* Value at (row, col) as a string. Returns NULL if out of bounds.
 * NULL values are returned as the string "NULL" (use tmjlite_result_is_null
 * to tell them apart). The returned pointer is BORROWED: it stays valid until
 * tmjlite_result_free(), so callers may keep `const char*` views instead of
 * copying every cell. The text is built lazily on first access; typed
 * accessors below never build it. */
const char *tmjlite_result_value(const TmjResult *result, int row, int col);

/* Typed accessors (0.3.1+): no string, no parse on the caller side.
 * is_null: 1 = SQL NULL, 0 = value, -1 = out of bounds.
 * i64/f64/bool: 0 = ok (written to *out), -1 = not convertible / out of
 * bounds. INT/FLOAT/BOOL columns convert directly; TEXT is parsed. */
int tmjlite_result_is_null(const TmjResult *result, int row, int col);
/* Stored type of the value (0.4.1+): 0 NULL, 1 INT, 2 FLOAT, 3 BOOL, 4 TEXT,
 * 5 UUID (PK), 6 DATE, 7 DATETIME; -1 = out of bounds. Drivers must use this
 * instead of guessing from the text: a TEXT cell holding '007' is text. */
int tmjlite_result_type(const TmjResult *result, int row, int col);
int tmjlite_result_i64(const TmjResult *result, int row, int col, int64_t *out);
int tmjlite_result_f64(const TmjResult *result, int row, int col, double *out);
int tmjlite_result_bool(const TmjResult *result, int row, int col, int *out);

/* Message for non-SELECT results (e.g. "Table 'users' created.").
 * Returns NULL if the result is a SELECT query.
 * The returned pointer is valid until tmjlite_result_free(). */
const char *tmjlite_result_message(const TmjResult *result);

/* Structured outcome of a write (0.3.6+) — run INSERT/UPDATE/DELETE through
 * tmjlite_query() / tmjlite_query_shared() / tmjlite_stmt_query*() instead of
 * the exec variants when you need it:
 *   rows_affected: rows inserted/updated/deleted/upserted (0 for SELECT/DDL).
 *   last_pk:       the UUID generated for a single-row INSERT into a table with
 *                  a PK column; NULL otherwise (multi-row INSERT, UPDATE, …).
 *                  Borrowed: valid until tmjlite_result_free(). */
int64_t     tmjlite_result_rows_affected(const TmjResult *result);
const char *tmjlite_result_last_pk(const TmjResult *result);

/* Free a result returned by tmjlite_query(). */
void tmjlite_result_free(TmjResult *result);

/* ---- Error ---- */

/* Last error message, or NULL if no error.
 * The returned pointer is valid until the next call on this db. */
const char *tmjlite_errmsg(const TmjDb *db);

/* Last error code (TMJLITE_ERR_*), or 0 if no error. Set and cleared together
 * with tmjlite_errmsg(). A NULL handle returns TMJLITE_ERR_MISUSE. (ABI 2+) */
int32_t tmjlite_errcode(const TmjDb *db);

/* Stable name of a code ("UNIQUE_VIOLATION" for 401), or NULL if this
 * library does not know the number. Static string: do not free. (ABI 2+) */
const char *tmjlite_errcode_name(int32_t code);

/* ---- Save ---- */

/* Explicitly save the database to its file. Returns TMJLITE_ERROR if no path is set. */
int tmjlite_save(TmjDb *db);

/* Save the database to a new path (sets it as the default path). */
int tmjlite_save_as(TmjDb *db, const char *path);

/* ---- Shared (thread-safe) API ---- */

/* Opaque handle for the thread-safe connection (MVCC): one handle may be used
 * from N threads concurrently — SELECTs run on parallel snapshots, writes
 * serialize internally and are durable when the call returns. Each statement
 * auto-commits (BEGIN/COMMIT are rejected on this handle).
 * Cross-process access stays exclusive: opening a file another process holds
 * fails and returns NULL. */
typedef struct TmjSharedDb TmjSharedDb;

/* Open (or create) a database file for shared, thread-safe use.
 * Returns NULL on failure (e.g. locked by another process). */
TmjSharedDb *tmjlite_open_shared(const char *path);

/* Close the shared handle and release the file lock. */
void tmjlite_close_shared(TmjSharedDb *db);

/* Execute a non-query statement. Thread-safe.
 * Transactions: BEGIN opens an interactive transaction bound to the CALLING
 * THREAD; its statements (exec/query/stmt_*) run inside it until COMMIT or
 * ROLLBACK on that same thread. Other threads block on WRITES until it ends;
 * their reads (query/cursor/EXPLAIN) keep running on a snapshot of the last
 * COMMIT and never wait for the writer (0.3.3+; up to 0.3.2 they queued
 * behind every autocommit fsync and every open transaction). */
int tmjlite_exec_shared(const TmjSharedDb *db, const char *sql);

/* Execute any statement and get the result. Thread-safe.
 * Caller must free the result with tmjlite_result_free(). */
TmjResult *tmjlite_query_shared(const TmjSharedDb *db, const char *sql);

/* ---- Prepared statements (parse once, execute many) ----
 *
 * Any statement with `?` placeholders: INSERT, UPDATE, DELETE, SELECT,
 * INSERT … ON CONFLICT. Parameters are passed as ONE JSON array string,
 * e.g. "[\"ns3\", 14, 1.5, true, null]" — types are preserved (string,
 * integer, float, bool, null) and no SQL escaping is needed. NULL or ""
 * means no parameters. One TmjStmt works with both handle kinds.
 *
 *   TmjStmt *st = tmjlite_prepare(db, "UPDATE rs SET samples = ? WHERE ns = ? AND k = ?;");
 *   tmjlite_stmt_exec(db, st, "[14, \"ns3\", \"Deployment\"]");
 *   tmjlite_stmt_free(st);
 */
typedef struct TmjStmt TmjStmt;

/* Prepare on a classic / shared handle. NULL on parse error (see errmsg). */
TmjStmt *tmjlite_prepare(TmjDb *db, const char *sql);
TmjStmt *tmjlite_prepare_shared(const TmjSharedDb *db, const char *sql);
void tmjlite_stmt_free(TmjStmt *stmt);
/* Number of `?` placeholders (-1 if stmt is NULL). */
int tmjlite_stmt_param_count(const TmjStmt *stmt);

int tmjlite_stmt_exec(TmjDb *db, const TmjStmt *stmt, const char *params_json);
TmjResult *tmjlite_stmt_query(TmjDb *db, const TmjStmt *stmt, const char *params_json);
int tmjlite_stmt_exec_shared(const TmjSharedDb *db, const TmjStmt *stmt, const char *params_json);
TmjResult *tmjlite_stmt_query_shared(const TmjSharedDb *db, const TmjStmt *stmt, const char *params_json);

/* ---- Memory: budget, stats, release (0.3.2+) ----
 *
 * A shared handle keeps every table it has WRITTEN to materialized in RAM
 * (rows + indexes) — the engine file stays the source of truth, the cache
 * makes repeated writes cheap. Reads via query/cursor do NOT populate it.
 * Give it a budget to make the ceiling predictable: when the estimate
 * exceeds the budget after a durable write (autocommit or COMMIT), the cache
 * is dropped and memory returned to the OS (glibc: malloc_trim). The next
 * write to a table reloads it.
 *
 * Also note: ORDER BY on a non-indexed column keeps up to 32k rows in memory
 * before spilling to disk; WAL checkpoints are deferred while a cursor is
 * open, but the WAL stays on disk (nothing accumulates in memory) and the
 * fold happens at the first commit with no open cursor. */
int   tmjlite_set_memory_budget_shared(const TmjSharedDb *db, uint64_t bytes);  /* 0 = off */
char *tmjlite_memory_stats_shared(const TmjSharedDb *db);                        /* JSON; free with tmjlite_string_free */
int   tmjlite_release_memory_shared(const TmjSharedDb *db);                      /* tables dropped, or 0 inside a txn */
/* Strict streaming: a cursor whose shape cannot stream (JOIN, DISTINCT,
 * subquery, window function, ORDER BY/LIMIT over aggregate) FAILS with the
 * reason instead of silently materializing. */
int   tmjlite_set_strict_streaming_shared(const TmjSharedDb *db, int on);
/* Page cache of each snapshot (cursor/query): pages of 4 KiB, default 2048
 * (8 MiB per reader). N concurrent readers cost N x this in RSS — the knob
 * for many-threaded services. 0 disables the cache. (0.3.7+) */
int   tmjlite_set_snapshot_cache_pages_shared(const TmjSharedDb *db, uint64_t pages);
/* Group commit counters since open: fsyncs performed and commits appended to
 * the WAL. fsyncs/commits < 1 means concurrent writers are being coalesced
 * into one fsync; ~1 means serial writes. NULL out-pointers are skipped. (0.3.8+) */
int   tmjlite_group_commit_stats_shared(const TmjSharedDb *db, uint64_t *fsyncs, uint64_t *commits);
void  tmjlite_string_free(char *s);

/* ---- Cursor: one row at a time (0.3.1+) ----
 *
 * Reads straight from an MVCC snapshot of the shared handle without
 * materializing the table OR the result set: memory is O(1 row) for scans
 * and index seeks, O(k) for LIMIT k, O(groups) for GROUP BY. ORDER BY on a
 * non-indexed column sorts (spilling to disk past 32k rows). Shapes that
 * cannot stream (joins, subqueries, window functions, DISTINCT, ORDER BY or
 * LIMIT over aggregates) fall back to a materialized result behind the SAME
 * API — tmjlite_cursor_is_streaming() tells which.
 *
 * Inside a BEGIN transaction on the calling thread the cursor sees that
 * transaction's writes and streams the same way (0.4.0+; before it was always
 * materialized, is_streaming = 0). Such a cursor is valid until COMMIT/ROLLBACK:
 * stepping it afterwards is an error; rows inserted after open are not visited.
 * Strict mode applies equally inside and outside a transaction.
 * Outside, it holds a consistent snapshot: rows
 * committed by other threads after open are NOT visible. While a cursor is
 * open, WAL checkpoints are deferred — close it as soon as you are done.
 *
 *   TmjCursor *c = tmjlite_cursor_open_shared(db, "SELECT samples, cpu_hist FROM rs WHERE ...");
 *   while (tmjlite_cursor_step(c) == 1) {
 *       int64_t samples; tmjlite_cursor_i64(c, 0, &samples);
 *       const char *hist = tmjlite_cursor_text(c, 1);   // borrowed until next step
 *   }
 *   tmjlite_cursor_close(c);
 */
typedef struct TmjCursor TmjCursor;

TmjCursor *tmjlite_cursor_open_shared(const TmjSharedDb *db, const char *sql);
TmjCursor *tmjlite_stmt_cursor_shared(const TmjSharedDb *db, const TmjStmt *stmt, const char *params_json);
/* 1 = row available, 0 = end, -1 = error (see tmjlite_cursor_errmsg). */
int tmjlite_cursor_step(TmjCursor *c);
int tmjlite_cursor_column_count(const TmjCursor *c);
const char *tmjlite_cursor_column_name(const TmjCursor *c, int col);
/* Current row, borrowed: valid until the next step/close. SQL NULL is the
 * string "NULL" — use tmjlite_cursor_is_null. */
const char *tmjlite_cursor_text(const TmjCursor *c, int col);
int tmjlite_cursor_is_null(const TmjCursor *c, int col);
/* Stored type of the current row's value — same tags as tmjlite_result_type. */
int tmjlite_cursor_type(const TmjCursor *c, int col);
int tmjlite_cursor_i64(const TmjCursor *c, int col, int64_t *out);
int tmjlite_cursor_f64(const TmjCursor *c, int col, double *out);
int tmjlite_cursor_bool(const TmjCursor *c, int col, int *out);
int tmjlite_cursor_is_streaming(const TmjCursor *c);
const char *tmjlite_cursor_errmsg(const TmjCursor *c);
/* Last error code of the cursor (TMJLITE_ERR_*), or 0. (ABI 2+) */
int32_t tmjlite_cursor_errcode(const TmjCursor *c);
void tmjlite_cursor_close(TmjCursor *c);

/* Last error message for the calling thread, or NULL.
 * The pointer stays valid for this thread until its next call. */
const char *tmjlite_errmsg_shared(const TmjSharedDb *db);

/* Last error code of the shared handle (TMJLITE_ERR_*), or 0. The error slot
 * belongs to the handle: with several threads failing at once the most
 * recent error wins, so read it right after the call that failed. (ABI 2+) */
int32_t tmjlite_errcode_shared(const TmjSharedDb *db);

/* ---- Shadow (git-merge) API ---- */

/* Shadow session (RFC GIT_MERGE): snapshot of the main file + overlay of this
 * session's own writes; DML statements become row ops in a `.tmjv` sidecar;
 * commit merges them into the main file (LWW by PK, atomic per session).
 * N sessions/processes coexist on the same file. Handle is single-thread. */
typedef struct TmjShadow TmjShadow;

/* Open a shadow session. NULL on failure. */
TmjShadow *tmjlite_shadow_open(const char *path);

/* Close the session. Uncommitted ops stay in the .tmjv (mergeable later). */
void tmjlite_shadow_close(TmjShadow *db);

/* Execute a DML/SELECT statement (DDL and transactions are rejected). */
int tmjlite_shadow_exec(TmjShadow *db, const char *sql);

/* Execute and get the result (sees main snapshot + own writes).
 * Caller must free with tmjlite_result_free(). */
TmjResult *tmjlite_shadow_query(TmjShadow *db, const char *sql);

/* Merge this session's ops into the main file. Returns the number of ops
 * applied (>= 0) or -1 on error/conflict (ops are preserved on conflict). */
long long tmjlite_shadow_commit(TmjShadow *db);

/* Re-snapshot the main file, keeping this session's uncommitted writes. */
int tmjlite_shadow_refresh(TmjShadow *db);

/* Number of ops logged and not yet merged (-1 on bad handle). */
long long tmjlite_shadow_pending(const TmjShadow *db);

/* ---- Named branches (RFC GIT_MERGE M5) ---- */

/* Open (or create) a persistent named branch: <db>.shadows/<name>.tmjb.
 * Same handle/API as a shadow session; SQL becomes branch ops. Only SEALED
 * ops are merged by tmjlite_shadow_commit(), which then consumes the branch
 * (unsealed ops after the last seal make commit fail). NULL on failure. */
TmjShadow *tmjlite_shadow_open_branch(const char *path, const char *name);

/* Seal the unsealed ops of a branch as a commit (author/message stored in
 * the branch log). Returns the number of ops sealed, or -1 on error. */
long long tmjlite_shadow_seal(TmjShadow *db, const char *author, const char *message);

/* Ops logged after the last seal (0 for anonymous shadows; -1 on bad handle). */
long long tmjlite_shadow_unsealed(const TmjShadow *db);

/* Conflict policy for commit (RFC GIT_MERGE M4). policy is one of
 * "lww" (default: shadow wins), "error" (any conflict aborts the whole
 * merge, .tmjv kept), "field_merge" (3-way per column; same column edited on
 * both sides aborts), "manual" (clean rows applied, conflicting ops
 * quarantined in a sibling .tmjc for human resolution).
 * table NULL sets the session default; otherwise overrides that table only. */
int tmjlite_shadow_set_merge_policy(TmjShadow *db, const char *table, const char *policy);

/* Dry-run: number of conflicts commit would hit right now (-1 on error).
 * When > 0, tmjlite_shadow_errmsg() lists them, one per line. */
long long tmjlite_shadow_conflicts(TmjShadow *db);

/* Last error message, or NULL. Valid until the next call on this handle. */
const char *tmjlite_shadow_errmsg(const TmjShadow *db);

/* Last error code (TMJLITE_ERR_*), or 0. After tmjlite_shadow_conflicts()
 * returned > 0 it is TMJLITE_ERR_MERGE_CONFLICT. (ABI 2+) */
int32_t tmjlite_shadow_errcode(const TmjShadow *db);

/* ---- Version ---- */

/* Returns the library version string (e.g. "0.1.0-capivara"). */
const char *tmjlite_version(void);

/* ABI number of this library: an integer, independent of tmjlite_version().
 *
 * Rule: it is bumped by exactly 1 whenever the C ABI changes — a function is
 * added, removed, or changes signature or meaning — and ONLY then. A release
 * that does not touch this header keeps the number. Additions are backward
 * compatible: a caller built for ABI N works with any library >= N unless a
 * release note says a function was removed.
 *
 *   1  everything that existed before stable error codes. Those libraries
 *      do NOT export this symbol: a missing tmjlite_abi_version IS ABI 1.
 *   2  tmjlite_abi_version, tmjlite_errcode, tmjlite_errcode_shared,
 *      tmjlite_shadow_errcode, tmjlite_cursor_errcode, tmjlite_open_errcode,
 *      tmjlite_errcode_name.
 *
 * Drivers resolve the ABI 2 symbols optionally, so a new driver still loads
 * an ABI 1 library (error codes are then reported as unknown, -1). */
#define TMJLITE_ABI_VERSION 2
uint32_t tmjlite_abi_version(void);

#ifdef __cplusplus
}
#endif

#endif /* TMJLITE_H */
