package dialect

import "database/sql"

// Dialect abstracts database-specific operations.
type Dialect interface {
	// Metadata Queries (Schema Introspection)
	GetTablesQuery(schema string) string
	GetColumnsQuery(schema string) string
	GetPrimaryKeysQuery(schema string) string
	GetForeignKeysQuery(schema string) string

	// Execution Hooks (Global Level)
	BeforePump(tx *sql.Tx) error
	AfterPump(tx *sql.Tx) error

	// Execution Hooks (Table Level) - For IDENTITY_INSERT etc.
	BeforeTable(tx *sql.Tx, tableName string, hasIdentity bool) error
	AfterTable(tx *sql.Tx, tableName string, hasIdentity bool) error

	// Query Generation
	InsertQuery(table string, cols []string) string
	TruncateQuery(table string) string
	Placeholder(index int) string // Returns ?, $1, @p1, etc.

	// Helpers
	NormalizeType(sqlType string) string
	GetSchemaName(input string) string
	GetLimitRowQuery(query string, limit int) string
}
