package cmd

import (
	"database/sql"
	"fmt"
	"log"

	"db-pump/internal/dialect"
	"db-pump/internal/schema"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean all data from tables",
	RunE: func(cmd *cobra.Command, args []string) error {
		// New Config Logic
		config, err := GetActiveDBConfig()
		if err != nil {
			return err
		}

		fmt.Printf("🦅 Connected to %s (%s)\n", config.Name, config.Driver)

		db, err := sql.Open(config.Driver, config.DSN)
		if err != nil {
			return fmt.Errorf("failed to open db: %w", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			return fmt.Errorf("failed to connect to db: %w", err)
		}

		// Set Globals
		DriverName = config.Driver
		DB = db
		if config.Driver == "mysql" {
			db.QueryRow("SELECT DATABASE()").Scan(&SchemaName)
		} else if config.Driver == "sqlserver" || config.Driver == "mssql" {
			SchemaName = "dbo"
		} else {
			SchemaName = "public"
		}

		// 0. Get Dialect
		d := dialect.GetDialect(DriverName)
		log.Printf("Using Dialect: %s\n", DriverName)

		// 1. Analyze
		log.Println("Analyzing schema...")
		tables, err := schema.Analyze(DB, d, SchemaName)
		if err != nil {
			return err
		}

		return cleanDatabase(tables, d)
	},
}

func init() {
	RootCmd.AddCommand(cleanCmd)
}

// cleanDatabase truncates tables in reverse order.
func cleanDatabase(tables []*schema.Table, d dialect.Dialect) error {
	log.Println("Disabling Foreign Key Checks...")

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	if err := d.BeforePump(tx); err != nil {
		log.Printf("Warning: Failed to execute BeforePump hook: %v. Continuing...\n", err)
		if _, ok := d.(*dialect.PostgresDialect); ok {
			tx.Rollback()
			tx, _ = DB.Begin()
		}
	}

	count := 0
	total := len(tables)

	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		count++
		// MSSQL: Use DELETE instead of TRUNCATE to avoid FK issues
		var query string
		if _, ok := d.(*dialect.MSSQLDialect); ok {
			query = fmt.Sprintf("DELETE FROM %s", table.Name)
		} else {
			query = d.TruncateQuery(table.Name)
		}
		if _, err := tx.Exec(query); err != nil {
			log.Printf("Warning: Failed to clean %s: %v (continuing...)\n", table.Name, err)
		}

		// MSSQL: Reset IDENTITY seed after DELETE
		if _, ok := d.(*dialect.MSSQLDialect); ok {
			resetQuery := fmt.Sprintf("DBCC CHECKIDENT ('%s', RESEED, 0)", table.Name)
			if _, err := tx.Exec(resetQuery); err != nil {
				log.Printf("Warning: Failed to reset IDENTITY for %s: %v (continuing...)\n", table.Name, err)
			}
		}

		if count%5 == 0 || count == total {
			log.Printf("Cleaned %d/%d tables...", count, total)
		}
	}

	log.Println("Enabling Foreign Key Checks...")
	if err := d.AfterPump(tx); err != nil {
		log.Printf("Warning: Failed to execute AfterPump hook: %v\n", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit cleaning transaction: %w", err)
	}
	tx = nil

	log.Println("Database Cleaned Successfully!")
	return nil
}
