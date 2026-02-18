# ⛽ DB Pump

DB Pump is a powerful, cross-database data generation tool designed to populate your databases with realistic dummy data for testing and development purposes.

It intelligently analyzes your database schema, understands table relationships (Foreign Keys), and generates semantically meaningful data (names, addresses, emails, etc.) while handling complex dependency graphs, including circular references.

---

## 🚀 Features

*   **Multi-Database Support**: Works seamlessly with **MySQL**, **PostgreSQL**, **MSSQL (SQL Server)**, and **Oracle**.
*   **Smart Schema Analysis**: Automatically detects tables, columns, primary keys, and foreign keys.
*   **Dependency Resolution**: Sorts tables based on dependencies to ensure data integrity during insertion. Handles circular dependencies gracefully.
*   **Semantic Data Generation**: Analyzes column names and comments to generate appropriate data (e.g., generating a real city name for a `city` column, not just random strings).
*   **Localized Data**: Supports generating data in **Korean** (names, addresses) based on configuration.
*   **Flexible Filtering**: Target specific tables via configuration or CLI flags.
*   **Performance**: Optimized for bulk insertions with transaction support.

---

## 📦 Installation

To build the project from source:

```bash
# Clone the repository
git clone https://github.com/your-repo/db-pump.git
cd db-pump

# Build
go build -o db-pump.exe main.go
```

---

## ⚙️ Configuration (`db-pump.yaml`)

Configure your database connections and default settings in `db-pump.yaml`.

```yaml
databases:
  - name: "Local MySQL"
    driver: "mysql"
    dsn: "root:root@tcp(127.0.0.1:3306)/sakila?parseTime=true"
    active: true  # Set to true to use this connection

  - name: "Local PostgreSQL"
    driver: "postgres"
    dsn: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
    active: false

settings:
  default_count: 1000       # Default number of rows to generate per table
  language: "ko"            # Data language (e.g., "ko" for Korean)
  tables: []                # List of tables to populate (empty = all tables)
                            # Example: ["users", "orders"]
```

---

## 🛠️ Usage

### 1. Basic Usage (Fill All Tables)

Populate all tables in the active database with the default number of rows (defined in config).

```bash
# Linux / macOS
./db-pump fill

# Windows
db-pump.exe fill
```

### 2. Specify Row Count

Generate a specific number of rows for each table.

```bash
# Linux / macOS
./db-pump fill --count 500

# Windows
db-pump.exe fill --count 500
```

### 3. Filter Specific Tables

Populate only specific tables. This overrides the `tables` setting in `db-pump.yaml`.

```bash
# Process only 'actor' and 'city' tables
# Linux / macOS
./db-pump fill --tables "actor,city"

# Windows
db-pump.exe fill --tables "actor,city"
```

### 4. Clean Before Filling

Truncate tables before inserting new data. **Warning: This deletes existing data.**

```bash
# Linux / macOS
./db-pump fill --clean

# Windows
db-pump.exe fill --clean
```

### 5. Dry Run (Simulation)

Simulate the process without writing any data to the database. Useful for checking the execution order and schema analysis.

```bash
# Linux / macOS
./db-pump fill --dry-run

# Windows
db-pump.exe fill --dry-run
```

### 6. CLI-Only Mode (No Config File)

You can run DB Pump without a `db-pump.yaml` file by providing connection details directly via flags.

```bash
# MySQL
# DSN Format: user:password@tcp(host:port)/dbname
./db-pump fill --dsn "root:password@tcp(localhost:3306)/dbname" --driver mysql

# PostgreSQL
# DSN Format: postgres://user:password@host:port/dbname?sslmode=disable
./db-pump fill --dsn "postgres://user:password@localhost:5432/dbname?sslmode=disable" --driver postgres

# MSSQL (SQL Server)
# DSN Format: sqlserver://user:password@host:port?database=dbname
./db-pump fill --dsn "sqlserver://sa:password@localhost:1433?database=dbname" --driver sqlserver

# Oracle
# DSN Format: oracle://user:password@host:port/service_name
./db-pump fill --dsn "oracle://user:password@localhost:1521/service" --driver oracle
```

---

## 📝 Supported Databases & Drivers

| Database | Driver Name | Notes |
| :--- | :--- | :--- |
| **MySQL** | `mysql` | Supports `INSERT IGNORE` for duplicate handling. |
| **PostgreSQL**| `postgres` | Uses `ON CONFLICT DO NOTHING`. |
| **MSSQL** | `sqlserver` | automatically handles identity inserts and constraints. |
| **Oracle** | `oracle` | Requires Oracle Instant Client or compatible environment. |


## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
