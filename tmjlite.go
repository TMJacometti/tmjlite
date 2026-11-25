// Copyright (c) 2026 TMJ Sistemas de Informação.
// SPDX-License-Identifier: MIT

package tmjlite

/*
#cgo linux LDFLAGS: -ldl
#ifndef _WIN32
#define _GNU_SOURCE
#endif
#include "include/tmjlite.h"
#include <stdlib.h>
#include <stdint.h>

// Símbolos da ABI 2 (códigos de erro) são resolvidos em tempo de execução, e
// não pelo linker: assim um binário compilado com este driver ainda carrega
// uma libtmjlite_ffi anterior aos códigos (o símbolo ausente vira NULL e o
// código do erro vira "desconhecido", -1).
#ifdef _WIN32
#include <windows.h>
static void *tmj_sym(const char *name) {
    HMODULE m = GetModuleHandleA("tmjlite_ffi.dll");
    return m ? (void *)GetProcAddress(m, name) : NULL;
}
#else
#include <dlfcn.h>
static void *tmj_sym(const char *name) { return dlsym(RTLD_DEFAULT, name); }
#endif

typedef int32_t (*tmj_handle_code_fn)(const void *);
typedef int32_t (*tmj_plain_code_fn)(void);
typedef uint32_t (*tmj_abi_fn)(void);

static int32_t tmj_handle_code(void *fn, const void *handle) {
    return fn ? ((tmj_handle_code_fn)fn)(handle) : -1;
}
static int32_t tmj_plain_code(void *fn) {
    return fn ? ((tmj_plain_code_fn)fn)() : -1;
}
static uint32_t tmj_abi(void *fn) {
    return fn ? ((tmj_abi_fn)fn)() : 1;
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// ErrorCode is a stable TMJLite error code. The numeric values never change
// and are never reused; the hundred is the category (see Category).
//
// An ErrorCode is itself an error, so it works as a sentinel:
//
//	if errors.Is(err, tmjlite.ErrUniqueViolation) { ... }
//
// New codes may appear in a future native library; a number this driver does
// not know still carries its category.
type ErrorCode int32

const (
	// ErrUnknown means "no code": the native library predates error codes
	// (ABI 1), or the error was produced by the driver itself.
	ErrUnknown ErrorCode = -1

	// GENERIC
	// Erro sem classificação mais específica.
	ErrError ErrorCode = 1

	// SQL
	// SQL que o parser não aceita.
	ErrSyntax ErrorCode = 100
	// SQL bem formado, mas a consulta é inválida (GROUP BY, ORDER BY fora do SELECT, número de valores, subconsulta com mais de uma linha).
	ErrSemantic ErrorCode = 101

	// SCHEMA
	// DDL inválido ou objeto em estado que não permite a operação.
	ErrSchema ErrorCode = 200
	// Tabela não existe.
	ErrNoSuchTable ErrorCode = 201
	// Coluna não existe.
	ErrNoSuchColumn ErrorCode = 202
	// Índice não existe.
	ErrNoSuchIndex ErrorCode = 203
	// Tabela já existe.
	ErrTableExists ErrorCode = 204
	// Coluna já existe.
	ErrColumnExists ErrorCode = 205
	// Índice já existe (pelo nome ou pelas colunas).
	ErrIndexExists ErrorCode = 206
	// Coluna ambígua: qualifique com tabela ou alias.
	ErrAmbiguousColumn ErrorCode = 207

	// TYPE
	// Valor de um tipo que a coluna não aceita.
	ErrTypeMismatch ErrorCode = 300
	// Tipo certo, valor inaceitável (maior que STRING(n), data inválida, byte NUL).
	ErrInvalidValue ErrorCode = 301

	// CONSTRAINT
	// Violação de restrição sem código mais específico.
	ErrConstraint ErrorCode = 400
	// Chave duplicada em índice UNIQUE.
	ErrUniqueViolation ErrorCode = 401
	// Tentativa de definir ou alterar a PK (ela é gerada pelo engine).
	ErrPrimaryKeyViolation ErrorCode = 402
	// NULL em coluna NOT NULL.
	ErrNotNullViolation ErrorCode = 403
	// Chave estrangeira sem pai, ou pai ainda referenciado.
	ErrForeignKeyViolation ErrorCode = 404

	// ACCESS
	// Arquivo travado por outro processo.
	ErrLocked ErrorCode = 500
	// A operação precisa de acesso exclusivo indisponível agora (ex.: VACUUM em conexão shared).
	ErrBusy ErrorCode = 501
	// Escrita em conexão, snapshot ou mídia só de leitura.
	ErrReadOnly ErrorCode = 502

	// OPEN
	// Não foi possível abrir (motivo fora dos códigos abaixo).
	ErrCannotOpen ErrorCode = 600
	// Arquivo ou diretório não existe.
	ErrFileNotFound ErrorCode = 601
	// Permissão negada pelo sistema operacional.
	ErrPermissionDenied ErrorCode = 602
	// O arquivo existe mas não é um banco TMJLite.
	ErrNotADatabase ErrorCode = 603
	// Arquivo TMJLite de versão/formato que esta biblioteca não lê.
	ErrUnsupportedFormat ErrorCode = 604
	// O arquivo já existe.
	ErrFileExists ErrorCode = 605

	// CORRUPT
	// Estrutura em disco inválida.
	ErrCorrupt ErrorCode = 700
	// Checksum de página ou de arquivo não confere.
	ErrCorruptChecksum ErrorCode = 701
	// WAL inválido, incompleto ou incoerente com o arquivo principal.
	ErrCorruptWal ErrorCode = 702

	// IO
	// Falha de leitura/escrita/armazenamento sem código mais específico.
	ErrIo ErrorCode = 800
	// fsync falhou: a durabilidade não está garantida.
	ErrIoFsync ErrorCode = 801
	// Disco cheio ou cota excedida.
	ErrDiskFull ErrorCode = 802

	// TRANSACTION
	// Erro de estado de transação sem código mais específico.
	ErrTransaction ErrorCode = 900
	// COMMIT/ROLLBACK sem transação ativa.
	ErrNoActiveTransaction ErrorCode = 901
	// Já existe transação ativa (BEGIN aninhado, VACUUM dentro de transação).
	ErrTransactionActive ErrorCode = 902
	// A recuperação de uma transação falhou: feche e reabra o banco.
	ErrTransactionAborted ErrorCode = 903

	// MERGE
	// Erro de sessão shadow/merge sem código mais específico.
	ErrMerge ErrorCode = 1000
	// Conflito de merge.
	ErrMergeConflict ErrorCode = 1001
	// O shadow foi gravado contra um esquema diferente do atual.
	ErrSchemaDivergence ErrorCode = 1002
	// Branch não existe.
	ErrNoSuchBranch ErrorCode = 1003
	// Branch já existe.
	ErrBranchExists ErrorCode = 1004
	// Há operações não seladas no branch.
	ErrUnsealedOps ErrorCode = 1005

	// LIMIT
	// Limite do engine excedido (tamanho de chave, LIMIT grande demais).
	ErrLimitExceeded ErrorCode = 1100
	// Parênteses/subconsultas aninhados além do limite.
	ErrNestingTooDeep ErrorCode = 1101

	// UNSUPPORTED
	// Recurso não suportado.
	ErrUnsupported ErrorCode = 1200
	// Mais de uma instrução na mesma chamada.
	ErrMultipleStatements ErrorCode = 1201

	// MISUSE
	// Uso incorreto da API (argumento inválido, statement nulo, cursor sobre algo que não é SELECT).
	ErrMisuse ErrorCode = 1300
	// Número de parâmetros diferente do número de ?.
	ErrParamCount ErrorCode = 1301

	// INTERNAL
	// Erro interno (invariante quebrada ou panic capturado): bug do TMJLite.
	ErrInternal ErrorCode = 1400
)

var errorCodeNames = map[ErrorCode]string{
	ErrUnknown:             "UNKNOWN",
	ErrError:               "ERROR",
	ErrSyntax:              "SYNTAX",
	ErrSemantic:            "SEMANTIC",
	ErrSchema:              "SCHEMA",
	ErrNoSuchTable:         "NO_SUCH_TABLE",
	ErrNoSuchColumn:        "NO_SUCH_COLUMN",
	ErrNoSuchIndex:         "NO_SUCH_INDEX",
	ErrTableExists:         "TABLE_EXISTS",
	ErrColumnExists:        "COLUMN_EXISTS",
	ErrIndexExists:         "INDEX_EXISTS",
	ErrAmbiguousColumn:     "AMBIGUOUS_COLUMN",
	ErrTypeMismatch:        "TYPE_MISMATCH",
	ErrInvalidValue:        "INVALID_VALUE",
	ErrConstraint:          "CONSTRAINT",
	ErrUniqueViolation:     "UNIQUE_VIOLATION",
	ErrPrimaryKeyViolation: "PRIMARY_KEY_VIOLATION",
	ErrNotNullViolation:    "NOT_NULL_VIOLATION",
	ErrForeignKeyViolation: "FOREIGN_KEY_VIOLATION",
	ErrLocked:              "LOCKED",
	ErrBusy:                "BUSY",
	ErrReadOnly:            "READ_ONLY",
	ErrCannotOpen:          "CANNOT_OPEN",
	ErrFileNotFound:        "FILE_NOT_FOUND",
	ErrPermissionDenied:    "PERMISSION_DENIED",
	ErrNotADatabase:        "NOT_A_DATABASE",
	ErrUnsupportedFormat:   "UNSUPPORTED_FORMAT",
	ErrFileExists:          "FILE_EXISTS",
	ErrCorrupt:             "CORRUPT",
	ErrCorruptChecksum:     "CORRUPT_CHECKSUM",
	ErrCorruptWal:          "CORRUPT_WAL",
	ErrIo:                  "IO",
	ErrIoFsync:             "IO_FSYNC",
	ErrDiskFull:            "DISK_FULL",
	ErrTransaction:         "TRANSACTION",
	ErrNoActiveTransaction: "NO_ACTIVE_TRANSACTION",
	ErrTransactionActive:   "TRANSACTION_ACTIVE",
	ErrTransactionAborted:  "TRANSACTION_ABORTED",
	ErrMerge:               "MERGE",
	ErrMergeConflict:       "MERGE_CONFLICT",
	ErrSchemaDivergence:    "SCHEMA_DIVERGENCE",
	ErrNoSuchBranch:        "NO_SUCH_BRANCH",
	ErrBranchExists:        "BRANCH_EXISTS",
	ErrUnsealedOps:         "UNSEALED_OPS",
	ErrLimitExceeded:       "LIMIT_EXCEEDED",
	ErrNestingTooDeep:      "NESTING_TOO_DEEP",
	ErrUnsupported:         "UNSUPPORTED",
	ErrMultipleStatements:  "MULTIPLE_STATEMENTS",
	ErrMisuse:              "MISUSE",
	ErrParamCount:          "PARAM_COUNT",
	ErrInternal:            "INTERNAL",
}

// String returns the stable name of the code ("UNIQUE_VIOLATION"), or
// "CODE_<n>" for a number this driver does not know.
func (c ErrorCode) String() string {
	if name, ok := errorCodeNames[c]; ok {
		return name
	}
	return "CODE_" + strconv.Itoa(int(c))
}

// Error makes ErrorCode usable as an errors.Is target.
func (c ErrorCode) Error() string {
	return "tmjlite: " + c.String()
}

// Category returns the category of the code (its hundred).
func (c ErrorCode) Category() ErrorCategory {
	if c <= 0 {
		return CategoryUnknown
	}
	if h := int(c) / 100; h < len(errorCategoryNames) {
		return ErrorCategory(h)
	}
	return CategoryGeneric
}

// ErrorCategory groups error codes. Like ErrorCode it is an error, so a whole
// category can be matched:
//
//	if errors.Is(err, tmjlite.CategoryConstraint) { ... } // any 4xx
type ErrorCategory int

const (
	CategoryGeneric ErrorCategory = iota
	CategorySql
	CategorySchema
	CategoryType
	CategoryConstraint
	CategoryAccess
	CategoryOpen
	CategoryCorrupt
	CategoryIo
	CategoryTransaction
	CategoryMerge
	CategoryLimit
	CategoryUnsupported
	CategoryMisuse
	CategoryInternal
	// CategoryUnknown: the error carries no code.
	CategoryUnknown ErrorCategory = -1
)

var errorCategoryNames = [...]string{"GENERIC", "SQL", "SCHEMA", "TYPE", "CONSTRAINT", "ACCESS", "OPEN", "CORRUPT", "IO", "TRANSACTION", "MERGE", "LIMIT", "UNSUPPORTED", "MISUSE", "INTERNAL"}

// String returns the stable name of the category ("CONSTRAINT").
func (c ErrorCategory) String() string {
	if c >= 0 && int(c) < len(errorCategoryNames) {
		return errorCategoryNames[c]
	}
	return "UNKNOWN"
}

// Error makes ErrorCategory usable as an errors.Is target.
func (c ErrorCategory) Error() string {
	return "tmjlite: " + c.String()
}

// TMJLiteError represents an error returned by the TMJLite engine.
//
// Message is the text the engine produced (unchanged by error codes). Code is
// the stable code: decide on it, not on the text. Use errors.As to reach the
// struct, or errors.Is with an ErrorCode / ErrorCategory:
//
//	var te *tmjlite.TMJLiteError
//	if errors.As(err, &te) && te.Code == tmjlite.ErrLocked { ... }
//	if errors.Is(err, tmjlite.ErrUniqueViolation) { ... }
//	if errors.Is(err, tmjlite.CategoryConstraint) { ... }
//
// Code is ErrUnknown (-1) when the native library predates error codes or the
// error was produced by the driver; a zero Code means the same.
type TMJLiteError struct {
	Message string
	Code    ErrorCode
}

func (e *TMJLiteError) Error() string {
	return fmt.Sprintf("tmjlite: %s", e.Message)
}

// Is reports whether target is the ErrorCode or the ErrorCategory of e.
func (e *TMJLiteError) Is(target error) bool {
	switch t := target.(type) {
	case ErrorCode:
		return e.Code > 0 && e.Code == t
	case ErrorCategory:
		return e.Code > 0 && e.Code.Category() == t
	}
	return false
}

// Símbolos opcionais da ABI 2, resolvidos uma vez.
var abi2 struct {
	once                                           sync.Once
	version, classic, shared, shadow, cursor, open unsafe.Pointer
}

func abi2Symbols() {
	abi2.once.Do(func() {
		lookup := func(name string) unsafe.Pointer {
			cname := C.CString(name)
			defer C.free(unsafe.Pointer(cname))
			return C.tmj_sym(cname)
		}
		abi2.version = lookup("tmjlite_abi_version")
		abi2.classic = lookup("tmjlite_errcode")
		abi2.shared = lookup("tmjlite_errcode_shared")
		abi2.shadow = lookup("tmjlite_shadow_errcode")
		abi2.cursor = lookup("tmjlite_cursor_errcode")
		abi2.open = lookup("tmjlite_open_errcode")
	})
}

// handleCode reads the last error code of a handle through an optional
// symbol; ErrUnknown when the native library does not have it.
func handleCode(fn unsafe.Pointer, handle unsafe.Pointer) ErrorCode {
	return ErrorCode(C.tmj_handle_code(fn, handle))
}

// plainCode is handleCode for an accessor without arguments (open error).
func plainCode(fn unsafe.Pointer) ErrorCode {
	return ErrorCode(C.tmj_plain_code(fn))
}

// abiOf calls tmjlite_abi_version through its optional symbol; 1 if absent.
func abiOf(fn unsafe.Pointer) int {
	return int(C.tmj_abi(fn))
}

// ABIVersion returns the ABI number of the loaded native library. It is 1 for
// a library that predates error codes (it does not export
// tmjlite_abi_version); every error then has Code == ErrUnknown.
func ABIVersion() int {
	abi2Symbols()
	return abiOf(abi2.version)
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
	abi2Symbols()
	code := handleCode(abi2.classic, unsafe.Pointer(db))
	if msg == nil {
		return &TMJLiteError{Message: "unknown error", Code: code}
	}
	return &TMJLiteError{Message: C.GoString(msg), Code: code}
}

// openFailure builds the error for a failed open with the reason the engine
// gave (locked by another process, not a TMJLite database, permission denied,
// missing directory...). Call it on the same OS thread as the failed open.
func openFailure(what string) *TMJLiteError {
	abi2Symbols()
	reason := C.tmjlite_open_error()
	code := plainCode(abi2.open)
	if reason != nil {
		return &TMJLiteError{Message: what + ": " + C.GoString(reason), Code: code}
	}
	return &TMJLiteError{Message: what, Code: code}
}

// Open opens a file-backed database at the given path.
// The file is created if it does not exist.
func Open(path string) (*Connection, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// O motivo da falha fica guardado por thread do SO: a goroutine não
	// pode trocar de thread entre abrir e ler o erro.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	db := C.tmjlite_open(cPath)
	if db == nil {
		return nil, openFailure("failed to open database: " + path)
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

// typedValue builds the Go value for a cell from the type the ENGINE stored
// (tmjlite_result_type / tmjlite_cursor_type) and the cell text. The type is
// never guessed from the text: a TEXT column holding "593" or "007" comes
// back as a string (it used to become int64 and lose the leading zero).
// Tags: 0 NULL -> nil, 1 INT -> int64, 2 FLOAT -> float64, 3 BOOL -> bool;
// everything else (TEXT, PK/UUID, DATE, DATETIME) -> string.
func typedValue(tag int, s string) interface{} {
	switch tag {
	case 0:
		return nil
	case 1:
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	case 2:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	case 3:
		return s == "true"
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
				values[col] = typedValue(int(C.tmjlite_result_type(result, C.int(r), C.int(col))), C.GoString(cVal))
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

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	db := C.tmjlite_open_shared(cPath)
	if db == nil {
		return nil, openFailure("failed to open database: " + path)
	}
	return &SharedConnection{db: db}, nil
}

func errFromShared(db *C.TmjSharedDb) error {
	msg := C.tmjlite_errmsg_shared(db)
	abi2Symbols()
	code := handleCode(abi2.shared, unsafe.Pointer(db))
	if msg == nil {
		return &TMJLiteError{Message: "unknown error", Code: code}
	}
	return &TMJLiteError{Message: C.GoString(msg), Code: code}
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
				values[col] = typedValue(int(C.tmjlite_result_type(result, C.int(r), C.int(col))), C.GoString(cVal))
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
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	db := C.tmjlite_shadow_open(cPath)
	if db == nil {
		return nil, openFailure("failed to open shadow session on: " + path)
	}
	return &ShadowConnection{db: db}, nil
}

// OpenBranch opens (or creates) the named branch `<path>.shadows/<name>.tmjb`.
func OpenBranch(path, name string) (*ShadowConnection, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	db := C.tmjlite_shadow_open_branch(cPath, cName)
	if db == nil {
		return nil, openFailure("failed to open branch '" + name + "' on: " + path)
	}
	return &ShadowConnection{db: db}, nil
}

func errFromShadow(db *C.TmjShadow) error {
	msg := C.tmjlite_shadow_errmsg(db)
	abi2Symbols()
	code := handleCode(abi2.shadow, unsafe.Pointer(db))
	if msg == nil {
		return &TMJLiteError{Message: "unknown error", Code: code}
	}
	return &TMJLiteError{Message: C.GoString(msg), Code: code}
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
// ExecResult is the structured outcome of INSERT/UPDATE/DELETE/DDL (0.3.6+).
// LastPk is the UUID generated by a single-row INSERT into a table with a PK
// column; empty otherwise (multi-row INSERT, UPDATE, DELETE, DDL).
type ExecResult struct {
	RowsAffected int64
	LastPk       string
	Message      string
}

func readExec(result *C.TmjResult) *ExecResult {
	defer C.tmjlite_result_free(result)
	r := &ExecResult{RowsAffected: int64(C.tmjlite_result_rows_affected(result))}
	if p := C.tmjlite_result_last_pk(result); p != nil {
		r.LastPk = C.GoString(p)
	}
	if m := C.tmjlite_result_message(result); m != nil {
		r.Message = C.GoString(m)
	}
	return r
}

// Exec runs a non-SELECT statement and returns rows affected and, for a
// single-row INSERT, the generated PK. Execute keeps the error-only form.
func (c *Connection) Exec(sql string) (*ExecResult, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))
	result := C.tmjlite_query(c.db, cSQL)
	if result == nil {
		return nil, errFromDB(c.db)
	}
	return readExec(result), nil
}

// Exec runs a non-SELECT statement on the shared handle and returns rows
// affected and, for a single-row INSERT, the generated PK.
func (c *SharedConnection) Exec(sql string) (*ExecResult, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))
	result := C.tmjlite_query_shared(c.db, cSQL)
	if result == nil {
		return nil, errFromShared(c.db)
	}
	return readExec(result), nil
}

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
				values[col] = typedValue(int(C.tmjlite_result_type(result, C.int(r), C.int(col))), C.GoString(cVal))
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

// MemoryStats is the estimate of what a SharedConnection holds in RAM (0.3.2+).
type MemoryStats struct {
	TotalBytes    int64         `json:"total_bytes"`
	TableBytes    int64         `json:"table_bytes"`
	IndexBytes    int64         `json:"index_bytes"`
	RowCacheBytes int64         `json:"row_cache_bytes"`
	Rows          int64         `json:"rows"`
	Tables        []TableMemory `json:"tables"`
}

// TableMemory is one materialized table inside MemoryStats.
type TableMemory struct {
	Name  string `json:"name"`
	Rows  int64  `json:"rows"`
	Bytes int64  `json:"bytes"`
}

// SetMemoryBudget caps (in bytes) the handle's RAM cache of tables (0.3.2+).
// Tables the handle has WRITTEN to stay resident (rows + indexes); reads do
// not populate it. When the estimate exceeds the budget after a durable
// write (autocommit or COMMIT) the cache is dropped and memory returned to
// the OS. 0 disables.
func (c *SharedConnection) SetMemoryBudget(bytes uint64) error {
	if C.tmjlite_set_memory_budget_shared(c.db, C.uint64_t(bytes)) != C.TMJLITE_OK {
		return errFromShared(c.db)
	}
	return nil
}

// MemoryStats returns what the handle holds in RAM; nil inside another
// goroutine's transaction.
func (c *SharedConnection) MemoryStats() (*MemoryStats, error) {
	p := C.tmjlite_memory_stats_shared(c.db)
	if p == nil {
		return nil, nil
	}
	defer C.tmjlite_string_free(p)
	var st MemoryStats
	if err := json.Unmarshal([]byte(C.GoString(p)), &st); err != nil {
		return nil, &TMJLiteError{Message: "memory_stats: " + err.Error(), Code: ErrUnknown}
	}
	return &st, nil
}

// ReleaseMemory drops the table cache now and returns how many tables were
// dropped (0 inside a transaction).
func (c *SharedConnection) ReleaseMemory() (int, error) {
	n := int(C.tmjlite_release_memory_shared(c.db))
	if n < 0 {
		return 0, errFromShared(c.db)
	}
	return n, nil
}

// SetSnapshotCachePages caps the page cache (4 KiB pages) of every snapshot
// (cursor/query) opened from now on. Default 2048 (8 MiB per reader); N
// concurrent readers cost N x this in RSS. 0 disables the cache.
func (c *SharedConnection) SetSnapshotCachePages(pages uint64) error {
	if C.tmjlite_set_snapshot_cache_pages_shared(c.db, C.uint64_t(pages)) != C.TMJLITE_OK {
		return errFromShared(c.db)
	}
	return nil
}

// GroupCommitStats returns (fsyncs, commits) since open. fsyncs/commits < 1
// means concurrent writers are being coalesced into one fsync; ~1 means
// serial writes (one fsync per commit).
func (c *SharedConnection) GroupCommitStats() (fsyncs, commits uint64, err error) {
	var f, n C.uint64_t
	if C.tmjlite_group_commit_stats_shared(c.db, &f, &n) != C.TMJLITE_OK {
		return 0, 0, errFromShared(c.db)
	}
	return uint64(f), uint64(n), nil
}

// SetStrictStreaming makes Cursor fail (with the reason) for shapes that
// cannot stream (JOIN, DISTINCT, subquery, window function, ORDER BY/LIMIT
// over aggregate) instead of silently materializing.
func (c *SharedConnection) SetStrictStreaming(on bool) error {
	v := C.int(0)
	if on {
		v = 1
	}
	if C.tmjlite_set_strict_streaming_shared(c.db, v) != C.TMJLITE_OK {
		return errFromShared(c.db)
	}
	return nil
}

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
		return nil, &TMJLiteError{Message: "params: " + err.Error(), Code: ErrUnknown}
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
		return &TMJLiteError{Message: "PreparedStatement is closed", Code: ErrUnknown}
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
// ExecResult executes with parameters and returns rows affected and, for a
// single-row INSERT, the generated PK (Exec keeps the error-only form).
func (s *PreparedStatement) ExecResult(params ...interface{}) (*ExecResult, error) {
	if s.stmt == nil {
		return nil, &TMJLiteError{Message: "PreparedStatement is closed", Code: ErrUnknown}
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
	return readExec(result), nil
}

func (s *PreparedStatement) Query(params ...interface{}) (*QueryResult, error) {
	if s.stmt == nil {
		return nil, &TMJLiteError{Message: "PreparedStatement is closed", Code: ErrUnknown}
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

// ===================== Cursor: one row at a time (0.3.1+) =====================

// Cursor reads straight from an MVCC snapshot without materializing the table
// or the result set: memory is O(1 row) for a SELECT of N rows.
//
//	cur, _ := db.Cursor("SELECT samples, cpu_hist FROM rs WHERE ns = ?", "ns3")
//	defer cur.Close()
//	for cur.Next() {
//		row := cur.Row()
//		total += row.values[0].(int64)
//	}
//
// Outside a transaction the cursor holds a snapshot until Close (checkpoints
// deferred): close it early.
type Cursor struct {
	c       *C.TmjCursor
	columns []string
	row     Row
	err     error
}

// Cursor opens a streaming SELECT on the shared connection. With params the SQL uses `?`.
func (c *SharedConnection) Cursor(sql string, params ...interface{}) (*Cursor, error) {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))
	var cur *C.TmjCursor
	if len(params) == 0 {
		cur = C.tmjlite_cursor_open_shared(c.db, cSQL)
	} else {
		st := C.tmjlite_prepare_shared(c.db, cSQL)
		if st == nil {
			return nil, errFromShared(c.db)
		}
		defer C.tmjlite_stmt_free(st)
		cParams, err := paramsJSON(params)
		if err != nil {
			return nil, err
		}
		defer C.free(unsafe.Pointer(cParams))
		cur = C.tmjlite_stmt_cursor_shared(c.db, st, cParams)
	}
	if cur == nil {
		return nil, errFromShared(c.db)
	}
	n := int(C.tmjlite_cursor_column_count(cur))
	cols := make([]string, n)
	for i := 0; i < n; i++ {
		if name := C.tmjlite_cursor_column_name(cur, C.int(i)); name != nil {
			cols[i] = C.GoString(name)
		}
	}
	return &Cursor{c: cur, columns: cols}, nil
}

// Columns of the result.
func (cu *Cursor) Columns() []string { return cu.columns }

// Streaming reports whether rows come straight from the snapshot.
func (cu *Cursor) Streaming() bool { return C.tmjlite_cursor_is_streaming(cu.c) == 1 }

// Next advances to the next row. Returns false at the end or on error (see Err).
func (cu *Cursor) Next() bool {
	if cu.c == nil {
		return false
	}
	rc := C.tmjlite_cursor_step(cu.c)
	if rc == 0 {
		return false
	}
	if rc < 0 {
		msg := C.tmjlite_cursor_errmsg(cu.c)
		abi2Symbols()
		code := handleCode(abi2.cursor, unsafe.Pointer(cu.c))
		if msg != nil {
			cu.err = &TMJLiteError{Message: C.GoString(msg), Code: code}
		} else {
			cu.err = &TMJLiteError{Message: "cursor step failed", Code: code}
		}
		return false
	}
	values := make([]interface{}, len(cu.columns))
	for i := range cu.columns {
		if C.tmjlite_cursor_is_null(cu.c, C.int(i)) == 1 {
			values[i] = nil
			continue
		}
		if t := C.tmjlite_cursor_text(cu.c, C.int(i)); t != nil {
			values[i] = typedValue(int(C.tmjlite_cursor_type(cu.c, C.int(i))), C.GoString(t))
		}
	}
	cu.row = Row{columns: cu.columns, values: values}
	return true
}

// Row is the current row after a successful Next.
func (cu *Cursor) Row() Row { return cu.row }

// Err is the error that stopped Next, if any.
func (cu *Cursor) Err() error { return cu.err }

// Close releases the cursor and its snapshot.
func (cu *Cursor) Close() {
	if cu.c != nil {
		C.tmjlite_cursor_close(cu.c)
		cu.c = nil
	}
}
