package dialect

// Factory returns the appropriate Dialect implementation based on driver name.
func GetDialect(driver string) Dialect {
	switch driver {
	case "postgres":
		return &PostgresDialect{}
	case "sqlserver", "mssql":
		return &MSSQLDialect{}
	case "oracle":
		return &OracleDialect{}
	default: // mysql
		return &MysqlDialect{}
	}
}

// Ensure interface implementation
var _ Dialect = (*MysqlDialect)(nil)
var _ Dialect = (*PostgresDialect)(nil)
var _ Dialect = (*MSSQLDialect)(nil)
var _ Dialect = (*OracleDialect)(nil)
