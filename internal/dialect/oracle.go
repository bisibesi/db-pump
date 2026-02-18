package dialect

import (
	"database/sql"
	"fmt"
	"strings"
)

type OracleDialect struct{}

func (d *OracleDialect) GetTablesQuery(schema string) string {
	// Oracle doesn't have a "schema" string concept in quite the same way for current user tables.
	// USER_TABLES lists tables owned by the current user.
	// We include a dummy clause to consume the schema argument if passed by standard callers.
	return `SELECT TABLE_NAME FROM USER_TABLES WHERE :1 IS NOT NULL`
}

func (d *OracleDialect) GetColumnsQuery(schema string) string {
	// Retrieves column information for the current user's tables.
	// We join with USER_CONS_COLUMNS to identify Primary Keys (P) and Unique (U) constraints.
	// We also fetch comments from USER_COL_COMMENTS.
	return `
SELECT
    t.TABLE_NAME,
    t.COLUMN_NAME,
    CASE
        WHEN t.DATA_TYPE = 'NUMBER' AND COALESCE(t.DATA_SCALE, 0) > 0 THEN 'DECIMAL'
        WHEN t.DATA_TYPE = 'NUMBER' THEN 'INTEGER'
        ELSE t.DATA_TYPE
    END,
    t.DATA_TYPE || CASE WHEN t.DATA_LENGTH IS NOT NULL THEN '(' || t.DATA_LENGTH || ')' ELSE '' END,
    COALESCE(t.DATA_PRECISION, t.DATA_LENGTH),
    t.NULLABLE,
    CASE WHEN p.CONSTRAINT_NAME IS NOT NULL THEN 'PRI' ELSE '' END,
    CASE WHEN t.IDENTITY_COLUMN = 'YES' THEN 'auto_increment' ELSE '' END,
    CASE WHEN u.CONSTRAINT_NAME IS NOT NULL THEN 'UNIQUE' ELSE '' END,
    c.COMMENTS
FROM USER_TAB_COLUMNS t
LEFT JOIN (
    SELECT cc.TABLE_NAME, cc.COLUMN_NAME, cc.CONSTRAINT_NAME
    FROM USER_CONS_COLUMNS cc
    JOIN USER_CONSTRAINTS uc ON cc.CONSTRAINT_NAME = uc.CONSTRAINT_NAME
    WHERE uc.CONSTRAINT_TYPE = 'P'
) p ON t.TABLE_NAME = p.TABLE_NAME AND t.COLUMN_NAME = p.COLUMN_NAME
LEFT JOIN (
    SELECT cc.TABLE_NAME, cc.COLUMN_NAME, cc.CONSTRAINT_NAME
    FROM USER_CONS_COLUMNS cc
    JOIN USER_CONSTRAINTS uc ON cc.CONSTRAINT_NAME = uc.CONSTRAINT_NAME
    WHERE uc.CONSTRAINT_TYPE = 'U'
) u ON t.TABLE_NAME = u.TABLE_NAME AND t.COLUMN_NAME = u.COLUMN_NAME
LEFT JOIN USER_COL_COMMENTS c ON t.TABLE_NAME = c.TABLE_NAME AND t.COLUMN_NAME = c.COLUMN_NAME
WHERE :1 IS NOT NULL
ORDER BY t.TABLE_NAME, t.COLUMN_ID`
}

func (d *OracleDialect) GetPrimaryKeysQuery(schema string) string {
	// PK information is handled in GetColumnsQuery via 'PRI' marker.
	// Returning a query that yields empty results matching the expected columns (Table, Column)
	// just in case, or a valid query if the caller relies on it explicitly (though analyzer seems not to).
	return `
SELECT cc.TABLE_NAME, cc.COLUMN_NAME
FROM USER_CONS_COLUMNS cc
JOIN USER_CONSTRAINTS uc ON cc.CONSTRAINT_NAME = uc.CONSTRAINT_NAME
WHERE uc.CONSTRAINT_TYPE = 'P' AND :1 IS NOT NULL`
}

func (d *OracleDialect) GetForeignKeysQuery(schema string) string {
	return `
SELECT
    c.TABLE_NAME,
    c.CONSTRAINT_NAME,
    cc.COLUMN_NAME,
    r.TABLE_NAME AS REF_TABLE,
    rcc.COLUMN_NAME AS REF_COLUMN
FROM USER_CONSTRAINTS c
JOIN USER_CONS_COLUMNS cc
    ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME
    AND c.OWNER = cc.OWNER
JOIN USER_CONSTRAINTS r
    ON c.R_CONSTRAINT_NAME = r.CONSTRAINT_NAME
    AND c.R_OWNER = r.OWNER
JOIN USER_CONS_COLUMNS rcc
    ON r.CONSTRAINT_NAME = rcc.CONSTRAINT_NAME
    AND r.OWNER = rcc.OWNER
    AND cc.POSITION = rcc.POSITION
WHERE c.CONSTRAINT_TYPE = 'R'
AND :1 IS NOT NULL`
}

func (d *OracleDialect) BeforePump(tx *sql.Tx) error {
	// 1. Set NLS Formats to match Go's time format (standardizing on ISO-8601-like)
	// Go's GenerateValue returns "2006-01-02 15:04:05" for dates.
	if _, err := tx.Exec("ALTER SESSION SET NLS_DATE_FORMAT = 'YYYY-MM-DD HH24:MI:SS'"); err != nil {
		return fmt.Errorf("failed to set NLS_DATE_FORMAT: %w", err)
	}
	if _, err := tx.Exec("ALTER SESSION SET NLS_TIMESTAMP_FORMAT = 'YYYY-MM-DD HH24:MI:SS'"); err != nil {
		return fmt.Errorf("failed to set NLS_TIMESTAMP_FORMAT: %w", err)
	}

	// 2. Disable all FK constraints for the current user to allow bulk insertion without ordering issues.
	// Note: In Oracle, DDL (ALTER) implicitly commits the transaction.
	rows, err := tx.Query("SELECT TABLE_NAME, CONSTRAINT_NAME FROM USER_CONSTRAINTS WHERE CONSTRAINT_TYPE = 'R' AND STATUS = 'ENABLED'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var constraints []struct {
		Table string
		Name  string
	}

	for rows.Next() {
		var t, n string
		if err := rows.Scan(&t, &n); err != nil {
			return err
		}
		constraints = append(constraints, struct{ Table, Name string }{t, n})
	}
	rows.Close()

	for _, c := range constraints {
		// Oracle names are case sensitive if quoted, but typically stored upper case.
		// We use standard identifiers.
		query := fmt.Sprintf("ALTER TABLE %s DISABLE CONSTRAINT %s", c.Table, c.Name)
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to disable constraint %s on %s: %w", c.Name, c.Table, err)
		}
	}
	return nil
}

func (d *OracleDialect) AfterPump(tx *sql.Tx) error {
	// Re-enable constraints.
	rows, err := tx.Query("SELECT TABLE_NAME, CONSTRAINT_NAME FROM USER_CONSTRAINTS WHERE CONSTRAINT_TYPE = 'R' AND STATUS = 'DISABLED'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var constraints []struct {
		Table string
		Name  string
	}

	for rows.Next() {
		var t, n string
		if err := rows.Scan(&t, &n); err != nil {
			return err
		}
		constraints = append(constraints, struct{ Table, Name string }{t, n})
	}
	rows.Close()

	for _, c := range constraints {
		query := fmt.Sprintf("ALTER TABLE %s ENABLE CONSTRAINT %s", c.Table, c.Name)
		if _, err := tx.Exec(query); err != nil {
			// Don't fail hard if re-enabling fails (e.g. data violation), but valid pump shouldn't violate.
			// Just log it or return error? Let's return error but user might ignore.
			return fmt.Errorf("failed to enable constraint %s on %s: %w", c.Name, c.Table, err)
		}
	}
	return nil
}

func (d *OracleDialect) BeforeTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	return nil
}

func (d *OracleDialect) AfterTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	return nil
}

func (d *OracleDialect) InsertQuery(table string, cols []string) string {
	vals := GeneratePlaceholders(len(cols), d.Placeholder)
	// Debugging: Print problematic SQL
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(cols, ", "),
		vals)
	// fmt.Printf("[DEBUG SQL] %s\n", sql)
	return sql
}

func (d *OracleDialect) TruncateQuery(table string) string {
	return fmt.Sprintf("TRUNCATE TABLE %s", table)
}

func (d *OracleDialect) Placeholder(index int) string {
	// Oracle uses :1, :2, etc. (1-based index)
	return fmt.Sprintf(":%d", index+1)
}

func (d *OracleDialect) NormalizeType(sqlType string) string {
	s := strings.ToLower(sqlType)
	if strings.Contains(s, "char") || strings.Contains(s, "clob") {
		return "string"
	}
	if strings.Contains(s, "int") || strings.Contains(s, "number") || strings.Contains(s, "float") {
		return "integer"
	}
	if strings.Contains(s, "date") || strings.Contains(s, "time") || strings.Contains(s, "year") {
		return "datetime"
	}
	return s
}

func (d *OracleDialect) GetSchemaName(input string) string {
	return input
}

func (d *OracleDialect) GetLimitRowQuery(query string, limit int) string {
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", query, limit)
}
