package dialect

import (
	"database/sql"
	"fmt"
	"strings"
)

type MysqlDialect struct{}

func (d *MysqlDialect) GetTablesQuery(schema string) string {
	return `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'`
}

func (d *MysqlDialect) GetColumnsQuery(schema string) string {
	return `SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, CHARACTER_MAXIMUM_LENGTH, IS_NULLABLE, COLUMN_KEY, EXTRA, IF(COLUMN_KEY='UNI', 'UNIQUE', NULL) AS IS_UNIQUE, COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME, ORDINAL_POSITION`
}

func (d *MysqlDialect) GetPrimaryKeysQuery(schema string) string {
	// MySQL returns PK info via GetColumnsQuery (COLUMN_KEY = 'PRI').
	// So this can be a no-op or just return empty result query to unify interface.
	return `SELECT TABLE_NAME, COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND COLUMN_KEY = 'PRI'`
}

func (d *MysqlDialect) GetForeignKeysQuery(schema string) string {
	return `SELECT TABLE_NAME, CONSTRAINT_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA = ? AND REFERENCED_TABLE_NAME IS NOT NULL`
}

func (d *MysqlDialect) BeforePump(tx *sql.Tx) error {
	_, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
	return err
}

func (d *MysqlDialect) AfterPump(tx *sql.Tx) error {
	_, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
	return err
}

func (d *MysqlDialect) BeforeTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	_, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
	return err
}

func (d *MysqlDialect) AfterTable(tx *sql.Tx, tableName string, hasIdentity bool) error {
	return nil
}

func (d *MysqlDialect) InsertQuery(table string, cols []string) string {
	vals := GeneratePlaceholders(len(cols), d.Placeholder)
	return fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), vals)
}

func (d *MysqlDialect) TruncateQuery(table string) string {
	return fmt.Sprintf("TRUNCATE TABLE %s", table)
}

func (d *MysqlDialect) Placeholder(index int) string {
	return "?"
}

func (d *MysqlDialect) NormalizeType(sqlType string) string {
	return DefaultNormalizeType(sqlType)
}

func (d *MysqlDialect) GetSchemaName(input string) string {
	return DefaultGetSchemaName(input)
}

func (d *MysqlDialect) GetLimitRowQuery(query string, limit int) string {
	return fmt.Sprintf("%s LIMIT %d", query, limit)
}
