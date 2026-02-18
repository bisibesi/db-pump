package dialect

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/denisenkom/go-mssqldb" // SQL Server Driver
)

type MSSQLDialect struct{}

// Helper: MSSQL Driver (go-mssqldb) often prefers @p1, @p2 named parameters over ?
// especially when prepared statements are involved or simple Exec.

func (d *MSSQLDialect) GetTablesQuery(schema string) string {
	// Use @p1 for schema binding
	return `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = @p1 AND TABLE_TYPE = 'BASE TABLE'`
}

func (d *MSSQLDialect) GetColumnsQuery(schema string) string {
	// Include PK, UNIQUE constraints, UNIQUE indexes, Identity info, and MS_Description (Comment)
	return `
		SELECT 
			c.TABLE_NAME, 
			c.COLUMN_NAME, 
			c.DATA_TYPE, 
			c.DATA_TYPE, 
			c.CHARACTER_MAXIMUM_LENGTH, 
			c.IS_NULLABLE, 
			CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 'PRIMARY' ELSE '' END AS COLUMN_KEY,
			CASE 
				WHEN idxc.column_id IS NOT NULL THEN 'identity' 
				WHEN COLUMNPROPERTY(OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME), c.COLUMN_NAME, 'IsIdentity') = 1 THEN 'identity'
				ELSE c.COLUMN_DEFAULT 
			END AS COLUMN_DEFAULT,
			CASE WHEN uq.COLUMN_NAME IS NOT NULL OR ui.COLUMN_NAME IS NOT NULL THEN 'UNIQUE' ELSE '' END AS IS_UNIQUE,
			CAST(ep.value AS NVARCHAR(MAX)) AS COMMENT
		FROM INFORMATION_SCHEMA.COLUMNS c
		LEFT JOIN (
			SELECT kcu.TABLE_NAME, kcu.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu 
				ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY' AND tc.TABLE_SCHEMA = @p1
		) pk ON c.TABLE_NAME = pk.TABLE_NAME AND c.COLUMN_NAME = pk.COLUMN_NAME
		LEFT JOIN (
			SELECT kcu.TABLE_NAME, kcu.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu 
				ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
			WHERE tc.CONSTRAINT_TYPE = 'UNIQUE' AND tc.TABLE_SCHEMA = @p1
		) uq ON c.TABLE_NAME = uq.TABLE_NAME AND c.COLUMN_NAME = uq.COLUMN_NAME
		LEFT JOIN (
			SELECT 
				t.name AS TABLE_NAME,
				col.name AS COLUMN_NAME
			FROM sys.indexes idx
			JOIN sys.index_columns ic ON idx.object_id = ic.object_id AND idx.index_id = ic.index_id
			JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id
			JOIN sys.tables t ON idx.object_id = t.object_id
			JOIN sys.schemas s ON t.schema_id = s.schema_id
			WHERE idx.is_unique = 1 
				AND idx.is_primary_key = 0
				AND s.name = @p1
		) ui ON c.TABLE_NAME = ui.TABLE_NAME AND c.COLUMN_NAME = ui.COLUMN_NAME
		LEFT JOIN sys.identity_columns idxc
			ON idxc.object_id = OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME)
			AND idxc.name = c.COLUMN_NAME
		LEFT JOIN sys.extended_properties ep 
			ON ep.major_id = OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME) 
			AND ep.minor_id = c.ORDINAL_POSITION 
			AND ep.name = 'MS_Description'
		WHERE c.TABLE_SCHEMA = @p1 
		ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION
	`
}

func (d *MSSQLDialect) GetPrimaryKeysQuery(schema string) string {
	return `SELECT T.TABLE_NAME, C.COLUMN_NAME FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS T JOIN INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE C ON T.CONSTRAINT_NAME = C.CONSTRAINT_NAME WHERE T.CONSTRAINT_TYPE = 'PRIMARY KEY' AND T.TABLE_SCHEMA = @p1`
}

func (d *MSSQLDialect) GetForeignKeysQuery(schema string) string {
	return `SELECT KCU1.TABLE_NAME, KCU1.CONSTRAINT_NAME, KCU1.COLUMN_NAME, KCU2.TABLE_NAME AS REF_TABLE, KCU2.COLUMN_NAME AS REF_COLUMN FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS RC JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE KCU1 ON RC.CONSTRAINT_NAME = KCU1.CONSTRAINT_NAME JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE KCU2 ON RC.UNIQUE_CONSTRAINT_NAME = KCU2.CONSTRAINT_NAME WHERE KCU1.TABLE_SCHEMA = @p1`
}

func (d *MSSQLDialect) BeforePump(tx *sql.Tx) error {
	// Disable all constraints on all tables to allow bulk operations and avoid FK loops
	// Using sp_msforeachtable is efficient but undocumented. Let's use standard loop.
	rows, err := tx.Query("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA = 'dbo'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return err
		}
		tables = append(tables, t)
	}
	rows.Close()

	for _, t := range tables {
		if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s NOCHECK CONSTRAINT all", t)); err != nil {
			return fmt.Errorf("failed to disable constraints on %s: %w", t, err)
		}
	}
	return nil
}

func (d *MSSQLDialect) AfterPump(tx *sql.Tx) error {
	// Re-enable constraints
	rows, err := tx.Query("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA = 'dbo'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return err
		}
		tables = append(tables, t)
	}
	rows.Close()

	for _, t := range tables {
		// WITH CHECK CHECK ensures existing data is validated, which is safer,
		// but if we inserted bad data we might want just CHECK CONSTRAINT all (without checking existing).
		// However, for consistency we usually want to validate.
		// If verification fails, it might error here.
		// Let's use simple CHECK CONSTRAINT all which enables it for future.
		// Or "WITH CHECK CHECK CONSTRAINT all" to validate.
		// Given we are generating random data that *should* be valid, let's try to validate.
		if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s WITH CHECK CHECK CONSTRAINT all", t)); err != nil {
			// If validation fails, we might warn but still succeed?
			// Or return error.
			// Let's log it as error.
			return fmt.Errorf("failed to enable constraints on %s: %w", t, err)
		}
	}
	return nil
}

func (d *MSSQLDialect) BeforeTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	// Disable all constraints on this table to allow circular dependencies
	_, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s NOCHECK CONSTRAINT all", tableName))
	return err
}

func (d *MSSQLDialect) AfterTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	// Do not re-enable constraints here. We enable them globally in AfterPump.
	// This supports circular dependencies (e.g. store <-> staff).
	return nil
}

func (d *MSSQLDialect) InsertQuery(table string, cols []string) string {
	vals := GeneratePlaceholders(len(cols), d.Placeholder)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), vals)
}

func (d *MSSQLDialect) TruncateQuery(table string) string {
	return fmt.Sprintf("TRUNCATE TABLE %s", table)
}

func (d *MSSQLDialect) Placeholder(index int) string {
	return fmt.Sprintf("@p%d", index+1)
}

func (d *MSSQLDialect) NormalizeType(sqlType string) string {
	t := strings.ToLower(sqlType)
	switch t {
	case "nvarchar", "nchar", "text", "ntext":
		return "varchar"
	case "bit":
		return "boolean"
	case "tinyint":
		return "tinyint" // 0-255
	case "smallint":
		return "smallint"
	case "int":
		return "int"
	case "bigint":
		return "bigint"
	case "decimal", "numeric", "money", "smallmoney":
		return "decimal"
	case "float", "real":
		return "float"
	case "datetime", "datetime2", "smalldatetime", "date":
		return "datetime"
	case "image", "binary", "varbinary":
		return "blob"
	default:
		return t
	}
}

func (d *MSSQLDialect) GetSchemaName(input string) string {
	if input == "" {
		return "dbo"
	}
	return input
}

func (d *MSSQLDialect) GetLimitRowQuery(query string, limit int) string {
	// Simple T-SQL TOP injection
	trimmed := strings.TrimSpace(query)
	if strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") {
		// Use Replace once. Note: this replaces the first occurrence.
		// If query is "SELECT ...", it becomes "SELECT TOP N ...".
		// Case insensitive replacement would be better but "SELECT" is standard.
		// We assume standard generated queries.
		return strings.Replace(query, "SELECT", fmt.Sprintf("SELECT TOP %d", limit), 1)
	}
	return query
}
