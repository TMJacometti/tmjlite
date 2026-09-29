package tmjlite

// #include "include/tmjlite.h"
// #include <stdlib.h>
import "C"

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// TMJLiteError represents an error returned by the TMJLite engine.
type TMJLiteError struct {
	Message string
}

func (e *TMJLiteError) Error() string {
	return fmt.Sprintf("tmjlite: %s", e.Message)
}

// Connection represents an open TMJLite database connection.
type Connection struct {
	db *C.TmjDb
}

// QueryResult holds the columns and rows returned by a SELECT query.
type QueryResult struct {
	Columns []string
	Rows    []Row
}

// Len returns the number of rows in the result.
func (r *QueryResult) Len() int {
	return len(r.Rows)
}

// Row holds a single result row with map-based access.
type Row struct {
	columns []string
	values  []interface{}
}

// Get returns the value for the given column name, or nil if not found.
func (r *Row) Get(key string) interface{} {
	for i, col := range r.columns {
		if col == key {
			return r.values[i]
		}
	}
	return nil
}

// GetIndex returns the value at the given column index, or nil if out of bounds.
func (r *Row) GetIndex(i int) interface{} {
	if i < 0 || i >= len(r.values) {
		return nil
	}
	return r.values[i]
}

// GetString returns the value for the given column as a string.
// Returns "" if the column is not found or the value is nil.
func (r *Row) GetString(key string) string {
	v := r.Get(key)
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// GetInt returns the value for the given column as int64.
// Returns 0 if the column is not found, nil, or not numeric.
func (r *Row) GetInt(key string) int64 {
	v := r.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	default:
		return 0
	}
}

// GetFloat returns the value for the given column as float64.
// Returns 0 if the column is not found, nil, or not numeric.
func (r *Row) GetFloat(key string) float64 {
	v := r.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// GetBool returns the value for the given column as bool.
// Returns false if the column is not found or nil.
func (r *Row) GetBool(key string) bool {
	v := r.Get(key)
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		b, _ := strconv.ParseBool(val)
		return b
	default:
		return false
	}
}

// IsNull returns true if the value for the given column is nil.
func (r *Row) IsNull(key string) bool {
	for i, col := range r.columns {
		if col == key {
			return r.values[i] == nil
		}
	}
	return true
}

// Len returns the number of columns in the row.
func (r *Row) Len() int {
	return len(r.values)
}

// ToMap returns all columns and values as a map.
func (r *Row) ToMap() map[string]interface{} {
	m := make(map[string]interface{}, len(r.columns))
	for i, col := range r.columns {
		m[col] = r.values[i]
	}
	return m
}

// errFromDB extracts the last error message from the database handle.
func errFromDB(db *C.TmjDb) error {
	msg := C.tmjlite_errmsg(db)
	if msg == nil {
		return &TMJLiteError{Message: "unknown error"}
	}
	return &TMJLiteError{Message: C.GoString(msg)}
}

// Open opens a file-backed database at the given path.
// The file is created if it does not exist.
func Open(path string) (*Connection, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	db := C.tmjlite_open(cPath)
	if db == nil {
		return nil, &TMJLiteError{Message: "failed to open database: " + path}
	}
	return &Connection{db: db}, nil
}

// OpenMemory creates an in-memory database. Data is lost when closed.
func OpenMemory() *Connection {
	db := C.tmjlite_open_memory()
	return &Connection{db: db}
}

// Execute runs a non-query SQL statement (CREATE, INSERT, UPDATE, DELETE, etc.).
func (c *Connection) Execute(sql string) error {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	rc := C.tmjlite_exec(c.db, cSQL)
	if rc != C.TMJLITE_OK {
		return errFromDB(c.db)
	}
	return nil
}

// parseValue converts a C result string into a typed Go value.
// "NULL" -> nil, "true"/"false" -> bool, integers -> int64, floats -> float64, else string.
func parseValue(s string) interface{} {
	if s == "NULL" {
		return nil
	}

	lower := strings.ToLower(s)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}

	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return s
}

// Query executes a SELECT statement and returns the result set.
func (c *Connection) Query(sql string) (*QueryResult, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	result := C.tmjlite_query(c.db, cSQL)
	if result == nil {
		return nil, errFromDB(c.db)
	}
	defer C.tmjlite_result_free(result)

	colCount := int(C.tmjlite_result_column_count(result))
	rowCount := int(C.tmjlite_result_row_count(result))

	columns := make([]string, colCount)
	for i := 0; i < colCount; i++ {
		cName := C.tmjlite_result_column_name(result, C.int(i))
		if cName != nil {
			columns[i] = C.GoString(cName)
		}
	}

	rows := make([]Row, rowCount)
	for r := 0; r < rowCount; r++ {
		values := make([]interface{}, colCount)
		for col := 0; col < colCount; col++ {
			cVal := C.tmjlite_result_value(result, C.int(r), C.int(col))
			if cVal != nil {
				values[col] = parseValue(C.GoString(cVal))
			}
		}
		rows[r] = Row{
			columns: columns,
			values:  values,
		}
	}

	return &QueryResult{
		Columns: columns,
		Rows:    rows,
	}, nil
}

// Begin starts a new transaction.
func (c *Connection) Begin() error {
	return c.Execute("BEGIN")
}

// Commit commits the current transaction.
func (c *Connection) Commit() error {
	return c.Execute("COMMIT")
}

// Rollback rolls back the current transaction.
func (c *Connection) Rollback() error {
	return c.Execute("ROLLBACK")
}

// Save explicitly saves the database to its file.
// Returns an error if no file path is associated with this connection.
func (c *Connection) Save() error {
	rc := C.tmjlite_save(c.db)
	if rc != C.TMJLITE_OK {
		return errFromDB(c.db)
	}
	return nil
}

// SaveAs saves the database to a new file path and sets it as the default path.
func (c *Connection) SaveAs(path string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.tmjlite_save_as(c.db, cPath)
	if rc != C.TMJLITE_OK {
		return errFromDB(c.db)
	}
	return nil
}

// Close closes the database connection and frees all resources.
// It is safe to call Close multiple times.
func (c *Connection) Close() {
	if c.db != nil {
		C.tmjlite_close(c.db)
		c.db = nil
	}
}

// Version returns the TMJLite library version string.
func Version() string {
	return C.GoString(C.tmjlite_version())
}

// ---------------------------------------------------------------------------
// SharedConnection — handle thread-safe (MVCC)
// ---------------------------------------------------------------------------

// SharedConnection is a thread-safe handle over one .tmjp file: SELECTs run on
// MVCC snapshots in parallel with each other and with the writer; writes
// serialize internally and are durable when the call returns. Each statement
// auto-commits (BEGIN/COMMIT are rejected — use Connection for transactions).
//
// Share one *SharedConnection across goroutines freely:
//
//	db, _ := tmjlite.OpenShared("app.tmjp")
//	var wg sync.WaitGroup
//	for w := 0; w < 8; w++ {
//	    wg.Add(1)
//	    go func() { defer wg.Done(); db.Execute("INSERT ...") }()
//	}
//
// Cross-process access stays exclusive: opening a file another process holds
// fails with "database is locked by another process".
type SharedConnection struct {
	db *C.TmjSharedDb
}

// OpenShared opens (or creates) a file-backed database for shared use.
func OpenShared(path string) (*SharedConnection, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	db := C.tmjlite_open_shared(cPath)
	if db == nil {
		return nil, &TMJLiteError{Message: "failed to open database: " + path + " (locked by another process?)"}
	}
	return &SharedConnection{db: db}, nil
}

func errFromShared(db *C.TmjSharedDb) error {
	msg := C.tmjlite_errmsg_shared(db)
	if msg == nil {
		return &TMJLiteError{Message: "unknown error"}
	}
	return &TMJLiteError{Message: C.GoString(msg)}
}

// Execute runs a non-query SQL statement. Safe to call from any goroutine.
func (c *SharedConnection) Execute(sql string) error {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	rc := C.tmjlite_exec_shared(c.db, cSQL)
	if rc != C.TMJLITE_OK {
		return errFromShared(c.db)
	}
	return nil
}

// Query executes a SELECT and returns the result set. Safe from any goroutine.
func (c *SharedConnection) Query(sql string) (*QueryResult, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	result := C.tmjlite_query_shared(c.db, cSQL)
	if result == nil {
		return nil, errFromShared(c.db)
	}
	defer C.tmjlite_result_free(result)

	colCount := int(C.tmjlite_result_column_count(result))
	rowCount := int(C.tmjlite_result_row_count(result))

	columns := make([]string, colCount)
	for i := 0; i < colCount; i++ {
		cName := C.tmjlite_result_column_name(result, C.int(i))
		if cName != nil {
			columns[i] = C.GoString(cName)
		}
	}

	rows := make([]Row, rowCount)
	for r := 0; r < rowCount; r++ {
		values := make([]interface{}, colCount)
		for col := 0; col < colCount; col++ {
			cVal := C.tmjlite_result_value(result, C.int(r), C.int(col))
			if cVal != nil {
				values[col] = parseValue(C.GoString(cVal))
			}
		}
		rows[r] = Row{columns: columns, values: values}
	}

	return &QueryResult{Columns: columns, Rows: rows}, nil
}

// Close releases the handle and the cross-process file lock.
func (c *SharedConnection) Close() {
	if c.db != nil {
		C.tmjlite_close_shared(c.db)
		c.db = nil
	}
}

// ---------------------------------------------------------------------------
// ShadowConnection — escrita multi-processo por merge + branches (git-merge)
// ---------------------------------------------------------------------------

// ShadowConnection is a shadow session: plain SQL over a snapshot of the
// database plus this session's own writes. Every INSERT/UPDATE/DELETE becomes
// an op in a `.tmjv` next to the file; Commit merges those ops by primary key
// into the main file under a millisecond lock. N processes coexist without
// blocking each other; what neighbours wrote shows up after their merge and
// a Refresh. DML-only (DDL and BEGIN/COMMIT are rejected).
//
// Opened with OpenBranch it is a persistent named branch (`.tmjb`): Seal
// records a commit (author/message); only sealed ops are merged by Commit,
// which then consumes the branch.
//
//	sess, _ := tmjlite.OpenShadow("app.tmjp")
//	sess.Execute("INSERT INTO listing (agency_id, external_id) VALUES ('ag', 'e1');")
//	applied, _ := sess.Commit()          // merge into main
//
//	br, _ := tmjlite.OpenBranch("app.tmjp", "import-2026-09")
//	br.Execute("UPDATE listing SET seen = 1 WHERE external_id = 'e1';")
//	br.Seal("first batch", "thiago")
//	br.Commit()
type ShadowConnection struct {
	db *C.TmjShadow
}

// OpenShadow opens an anonymous shadow session on a .tmjp file.
func OpenShadow(path string) (*ShadowConnection, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	db := C.tmjlite_shadow_open(cPath)
	if db == nil {
		return nil, &TMJLiteError{Message: "failed to open shadow session on: " + path}
	}
	return &ShadowConnection{db: db}, nil
}

// OpenBranch opens (or creates) the named branch `<path>.shadows/<name>.tmjb`.
func OpenBranch(path, name string) (*ShadowConnection, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	db := C.tmjlite_shadow_open_branch(cPath, cName)
	if db == nil {
		return nil, &TMJLiteError{Message: "failed to open branch '" + name + "' on: " + path}
	}
	return &ShadowConnection{db: db}, nil
}

func errFromShadow(db *C.TmjShadow) error {
	msg := C.tmjlite_shadow_errmsg(db)
	if msg == nil {
		return &TMJLiteError{Message: "unknown error"}
	}
	return &TMJLiteError{Message: C.GoString(msg)}
}

// Execute runs INSERT/UPDATE/DELETE (recorded as ops) or any other DML.
func (c *ShadowConnection) Execute(sql string) error {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))
	if C.tmjlite_shadow_exec(c.db, cSQL) != C.TMJLITE_OK {
		return errFromShadow(c.db)
	}
	return nil
}

// Query runs a SELECT over snapshot + own writes.
func (c *ShadowConnection) Query(sql string) (*QueryResult, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))
	result := C.tmjlite_shadow_query(c.db, cSQL)
	if result == nil {
		return nil, errFromShadow(c.db)
	}
	defer C.tmjlite_result_free(result)
	return readResult(result), nil
}

// Commit merges this session's ops into the main file and returns how many
// were applied. On a named branch only sealed ops are merged (unsealed ops
// after the last Seal are an error) and the branch is consumed.
func (c *ShadowConnection) Commit() (int64, error) {
	n := int64(C.tmjlite_shadow_commit(c.db))
	if n < 0 {
		return 0, errFromShadow(c.db)
	}
	return n, nil
}

// Refresh re-snapshots the main file (sees neighbours' merges) keeping own ops.
func (c *ShadowConnection) Refresh() error {
	if C.tmjlite_shadow_refresh(c.db) != C.TMJLITE_OK {
		return errFromShadow(c.db)
	}
	return nil
}

// PendingOps is the number of ops logged and not yet merged.
func (c *ShadowConnection) PendingOps() int64 {
	return int64(C.tmjlite_shadow_pending(c.db))
}

// Seal records a commit on a named branch (author/message) covering the
// unsealed ops; returns how many ops were sealed.
func (c *ShadowConnection) Seal(message, author string) (int64, error) {
	if author == "" {
		author = "go"
	}
	cAuthor := C.CString(author)
	defer C.free(unsafe.Pointer(cAuthor))
	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cMsg))
	n := int64(C.tmjlite_shadow_seal(c.db, cAuthor, cMsg))
	if n < 0 {
		return 0, errFromShadow(c.db)
	}
	return n, nil
}

// UnsealedOps is the number of ops logged after the last Seal (0 on anonymous sessions).
func (c *ShadowConnection) UnsealedOps() int64 {
	return int64(C.tmjlite_shadow_unsealed(c.db))
}

// SetMergePolicy sets the conflict policy used by Commit: "lww", "error",
// "field_merge" or "manual". An empty table sets the session default;
// otherwise only that table.
func (c *ShadowConnection) SetMergePolicy(policy, table string) error {
	cPolicy := C.CString(policy)
	defer C.free(unsafe.Pointer(cPolicy))
	var cTable *C.char
	if table != "" {
		cTable = C.CString(table)
		defer C.free(unsafe.Pointer(cTable))
	}
	if C.tmjlite_shadow_set_merge_policy(c.db, cTable, cPolicy) != C.TMJLITE_OK {
		return errFromShadow(c.db)
	}
	return nil
}

// Conflicts is a dry-run of Commit: one description per conflicting row.
func (c *ShadowConnection) Conflicts() ([]string, error) {
	n := int64(C.tmjlite_shadow_conflicts(c.db))
	if n < 0 {
		return nil, errFromShadow(c.db)
	}
	if n == 0 {
		return []string{}, nil
	}
	msg := C.tmjlite_shadow_errmsg(c.db)
	if msg == nil {
		return []string{}, nil
	}
	return strings.Split(C.GoString(msg), "\n"), nil
}

// Close ends the session. Unmerged ops stay on disk (mergeable later).
func (c *ShadowConnection) Close() {
	if c.db != nil {
		C.tmjlite_shadow_close(c.db)
		c.db = nil
	}
}

// readResult materializes a TmjResult into a QueryResult (caller frees the pointer).
func readResult(result *C.TmjResult) *QueryResult {
	colCount := int(C.tmjlite_result_column_count(result))
	rowCount := int(C.tmjlite_result_row_count(result))
	columns := make([]string, colCount)
	for i := 0; i < colCount; i++ {
		if cName := C.tmjlite_result_column_name(result, C.int(i)); cName != nil {
			columns[i] = C.GoString(cName)
		}
	}
	rows := make([]Row, rowCount)
	for r := 0; r < rowCount; r++ {
		values := make([]interface{}, colCount)
		for col := 0; col < colCount; col++ {
			if cVal := C.tmjlite_result_value(result, C.int(r), C.int(col)); cVal != nil {
				values[col] = parseValue(C.GoString(cVal))
			}
		}
		rows[r] = Row{columns: columns, values: values}
	}
	return &QueryResult{Columns: columns, Rows: rows}
}

// ===================== Prepared statements (6.4) =====================

// PreparedStatement is a statement parsed ONCE and executed N times with `?`
// parameters. Works on Connection and SharedConnection (including inside a
// SharedConnection transaction on the same goroutine's OS thread).
//
//	st, _ := db.Prepare("INSERT INTO rs (ns, k, samples) VALUES (?, ?, ?) ON CONFLICT (ns, k) DO UPDATE SET samples = excluded.samples;")
//	defer st.Close()
//	for _, it := range lote { st.Exec(it.NS, it.K, it.Samples) }
//	rows, _ := sel.Query("ns3", "k1")
//
// No manual escaping: string → TEXT, int → INT, float64 → FLOAT, bool → BOOL,
// nil → NULL.
type PreparedStatement struct {
	stmt   *C.TmjStmt
	db     *C.TmjDb
	shared *C.TmjSharedDb
}

// Prepare parses a statement with `?` placeholders on a classic connection.
func (c *Connection) Prepare(sql string) (*PreparedStatement, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))
	st := C.tmjlite_prepare(c.db, cSQL)
	if st == nil {
		return nil, errFromDB(c.db)
	}
	return &PreparedStatement{stmt: st, db: c.db}, nil
}

// Prepare parses a statement with `?` placeholders on a shared connection.
func (c *SharedConnection) Prepare(sql string) (*PreparedStatement, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))
	st := C.tmjlite_prepare_shared(c.db, cSQL)
	if st == nil {
		return nil, errFromShared(c.db)
	}
	return &PreparedStatement{stmt: st, shared: c.db}, nil
}

// Begin opens a transaction bound to the calling OS thread (0.2.9+). Pin the
// goroutine with runtime.LockOSThread() between Begin and Commit/Rollback.
func (c *SharedConnection) Begin() error    { return c.Execute("BEGIN;") }
func (c *SharedConnection) Commit() error   { return c.Execute("COMMIT;") }
func (c *SharedConnection) Rollback() error { return c.Execute("ROLLBACK;") }

// ParamCount is the number of `?` placeholders.
func (s *PreparedStatement) ParamCount() int {
	return int(C.tmjlite_stmt_param_count(s.stmt))
}

func paramsJSON(params []interface{}) (*C.char, error) {
	if params == nil {
		params = []interface{}{}
	}
	b, err := json.Marshal(params)
	if err != nil {
		return nil, &TMJLiteError{Message: "params: " + err.Error()}
	}
	return C.CString(string(b)), nil
}

func (s *PreparedStatement) lastErr() error {
	if s.shared != nil {
		return errFromShared(s.shared)
	}
	return errFromDB(s.db)
}

// Exec runs the statement with the given parameters.
func (s *PreparedStatement) Exec(params ...interface{}) error {
	if s.stmt == nil {
		return &TMJLiteError{Message: "PreparedStatement is closed"}
	}
	cParams, err := paramsJSON(params)
	if err != nil {
		return err
	}
	defer C.free(unsafe.Pointer(cParams))
	var rc C.int
	if s.shared != nil {
		rc = C.tmjlite_stmt_exec_shared(s.shared, s.stmt, cParams)
	} else {
		rc = C.tmjlite_stmt_exec(s.db, s.stmt, cParams)
	}
	if rc != C.TMJLITE_OK {
		return s.lastErr()
	}
	return nil
}

// Query runs a SELECT with the given parameters.
func (s *PreparedStatement) Query(params ...interface{}) (*QueryResult, error) {
	if s.stmt == nil {
		return nil, &TMJLiteError{Message: "PreparedStatement is closed"}
	}
	cParams, err := paramsJSON(params)
	if err != nil {
		return nil, err
	}
	defer C.free(unsafe.Pointer(cParams))
	var result *C.TmjResult
	if s.shared != nil {
		result = C.tmjlite_stmt_query_shared(s.shared, s.stmt, cParams)
	} else {
		result = C.tmjlite_stmt_query(s.db, s.stmt, cParams)
	}
	if result == nil {
		return nil, s.lastErr()
	}
	defer C.tmjlite_result_free(result)
	return readResult(result), nil
}

// Close frees the statement.
func (s *PreparedStatement) Close() {
	if s.stmt != nil {
		C.tmjlite_stmt_free(s.stmt)
		s.stmt = nil
	}
}
