package dialect

import (
	"database/sql"
	"fmt"
	"strings"
)

type PostgresDialect struct{}

// Helper to fix schema name if needed (usually public)
func (d *PostgresDialect) getSchema(schema string) string {
	if schema == "" {
		return "public"
	}
	return schema
}

func (d *PostgresDialect) GetTablesQuery(schema string) string {
	// use $1 placeholder
	return `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = $1 AND TABLE_TYPE = 'BASE TABLE'`
}

func (d *PostgresDialect) GetColumnsQuery(schema string) string {
	// Returns generic columns matching interface structure.
	// Postgres specific: UDT_NAME is often better than DATA_TYPE.
	// We select UDT_NAME as DATA_TYPE here for better processing later if needed, or stick to standard.
	// We also select COLUMN_DEFAULT as the last column (EXTRA in MySQL).
	// Subqueries used to fetch PRIMARY KEY and UNIQUE constraints correctly.
	return `SELECT 
    c.table_name, 
    c.column_name, 
    c.data_type, 
    c.udt_name, 
    c.character_maximum_length, 
    c.is_nullable, 
    (SELECT 'PRI' FROM information_schema.table_constraints tc 
     JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name 
     WHERE tc.constraint_type = 'PRIMARY KEY' 
     AND kcu.table_schema = c.table_schema AND kcu.table_name = c.table_name AND kcu.column_name = c.column_name LIMIT 1) AS COLUMN_KEY,
    c.column_default, 
    (SELECT 'UNIQUE' FROM information_schema.table_constraints tc 
     JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name 
     WHERE tc.constraint_type = 'UNIQUE' 
     AND kcu.table_schema = c.table_schema AND kcu.table_name = c.table_name AND kcu.column_name = c.column_name LIMIT 1) AS IS_UNIQUE,
    NULL AS COMMENT
FROM information_schema.columns c
WHERE c.table_schema = $1 
ORDER BY c.table_name, c.ordinal_position`
}

func (d *PostgresDialect) GetPrimaryKeysQuery(schema string) string {
	return `SELECT kcu.table_name, kcu.column_name FROM information_schema.key_column_usage kcu JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name WHERE kcu.table_schema = $1 AND tc.constraint_type = 'PRIMARY KEY'`
}

func (d *PostgresDialect) GetForeignKeysQuery(schema string) string {
	return `SELECT kcu.table_name, kcu.constraint_name, kcu.column_name, ccu.table_name AS referenced_table_name, ccu.column_name AS referenced_column_name FROM information_schema.key_column_usage kcu JOIN information_schema.constraint_column_usage ccu ON kcu.constraint_name = ccu.constraint_name JOIN information_schema.table_constraints tc ON kcu.constraint_name = tc.constraint_name WHERE kcu.table_schema = $1 AND tc.constraint_type = 'FOREIGN KEY'`
}

func (d *PostgresDialect) BeforePump(tx *sql.Tx) error {
	// Use DEFERRED constraints for circular dependencies.
	// This works for foreign keys defined as DEFERRABLE.
	// If keys are NOT DEFERRABLE, this statement might not help immediately,
	// but session_replication_role often requires superuser.
	// Let's try to mix or fallback. For now, strict 'SET CONSTRAINTS ALL DEFERRED' is safer for general users.
	_, err := tx.Exec("SET CONSTRAINTS ALL DEFERRED")
	return err
}

func (d *PostgresDialect) AfterPump(tx *sql.Tx) error {
	// Constraints are checked at commit time automatically when deferred.
	// Nothing explicit needed unless we want to force check immediately.
	_, err := tx.Exec("SET CONSTRAINTS ALL IMMEDIATE")
	return err
}

func (d *PostgresDialect) BeforeTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	// Try session_replication_role for circular dependencies (superuser required)
	if _, err := tx.Exec("SET session_replication_role = 'replica'"); err != nil {
		// Fallback to deferred constraints if permission denied (though might not work for non-deferrable FKs)
		_, err2 := tx.Exec("SET CONSTRAINTS ALL DEFERRED")
		return fmt.Errorf("replication_role failed: %v, deferred failed: %v", err, err2)
	}
	return nil
}

func (d *PostgresDialect) AfterTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	_, err := tx.Exec("SET session_replication_role = 'origin'")
	return err
}

func (d *PostgresDialect) InsertQuery(table string, cols []string) string {
	// Generate placeholders ($1, $2, ...)
	vals := GeneratePlaceholders(len(cols), d.Placeholder)

	// RETURNING clause logic is handled in Pumper currently via string concat hack.
	// We just return base INSERT ... ON CONFLICT DO NOTHING.
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING", table, strings.Join(cols, ", "), vals)
}

func (d *PostgresDialect) TruncateQuery(table string) string {
	return fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
}

func (d *PostgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index+1)
}

func (d *PostgresDialect) NormalizeType(sqlType string) string {
	t := strings.ToLower(sqlType)
	switch t {
	case "int4", "int2":
		return "int"
	case "int8":
		return "bigint"
	case "float4":
		return "float"
	case "float8":
		return "double"
	case "bpchar":
		return "char"
	case "varchar":
		return "varchar"
	default:
		return t
	}
}

func (d *PostgresDialect) GetSchemaName(input string) string {
	if input == "" {
		return "public"
	}
	return input
}

func (d *PostgresDialect) GetLimitRowQuery(query string, limit int) string {
	return fmt.Sprintf("%s LIMIT %d", query, limit)
}
