package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"db-pump/internal/dialect" // Import
	"db-pump/internal/engine"
	"db-pump/internal/schema"

	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	count  int
	clean  bool
	dryRun bool
	tables []string
)

var fillCmd = &cobra.Command{
	Use:   "fill",
	Short: "Fill the database with random data",
	RunE: func(cmd *cobra.Command, args []string) error {
		// New Config Logic
		var config DBConfig
		activeConfig, err := GetActiveDBConfig()
		if err == nil {
			config = *activeConfig
		} else {
			// Fallback to CLI flags if config file is missing or invalid
			// If DriverName and DB are already set by RootCmd.PersistentPreRunE,
			// we can reconstruct a partial config for logging purposes or just rely on globals.
			// RootCmd already sets DriverName and DB global.
			// We just need a dummy config struct for the print statement below.

			// If DB connection failed in RootCmd, we shouldn't be here (it returns error)
			// EXCEPT initialized global SchemaName might differ.

			if DriverName == "" {
				return fmt.Errorf("could not determine driver: ensure config file exists or use --dsn and --driver flags")
			}
			config = DBConfig{
				Name:   "CLI Wrapper",
				Driver: DriverName,
				DSN:    dsn,
				Active: true,
			}
		}

		fmt.Printf("🦅 Connected via %s (%s)\n", config.Driver, config.DSN)

		// Note: DB is already initialized in RootCmd.PersistentPreRunE
		// If we re-open here, we might duplicate.
		// BUT RootCmd logic assumes it should "return nil" and let us handle?
		// No, RootCmd.PersistentPreRunE does sql.Open and sets global DB.
		// So we can skip re-opening if DB is already valid.

		if DB == nil {
			// This path should ideally not be reached if RootCmd does its job.
			// Re-open just in case
			db, err := sql.Open(config.Driver, config.DSN)
			if err != nil {
				return fmt.Errorf("failed to open db: %w", err)
			}
			DB = db
		}

		db := DB // Local alias for compatibility
		// defer db.Close() // RootCmd might manage lifecycle or we defer here?
		// Cobra doesn't auto-close. We should defer close if we opened it.
		// Since RootCmd opened it, maybe RootCmd PostRun should close it?
		// For now, let's just use it.

		if err := db.Ping(); err != nil {
			return fmt.Errorf("failed to connect to db: %w", err)
		}

		// Set Globals for compatibility if needed (DB, DriverName, SchemaName)
		DriverName = config.Driver
		DB = db
		// SchemaName logic (already done in RootCmd potentially, but verify)
		if SchemaName == "" {
			if config.Driver == "mysql" {
				db.QueryRow("SELECT DATABASE()").Scan(&SchemaName)
			} else if config.Driver == "sqlserver" || config.Driver == "mssql" {
				SchemaName = "dbo"
			} else {
				SchemaName = "public"
			}
		}

		// Fetch count from Viper (Flag > Config > Default)
		targetCount := viper.GetInt("settings.default_count")
		if count > 0 { // Flag override
			targetCount = count
		}

		// 0. Get Dialect
		d := dialect.GetDialect(DriverName)
		log.Printf("Using Dialect: %s\n", DriverName)

		// 1. Analyze
		log.Println("Analyzing schema...")
		allTables, err := schema.Analyze(DB, d, SchemaName)
		if err != nil {
			return err
		}

		// Filter tables strategy:
		// 1. Check CLI flag --tables
		// 2. If empty, check config settings.tables
		// 3. If both empty, process all tables.
		var targetTableNames []string

		// 1. Flag
		if len(tables) > 0 {
			targetTableNames = tables
		} else {
			// 2. Config
			configTables := viper.GetStringSlice("settings.tables")
			if len(configTables) > 0 {
				targetTableNames = configTables
			}
		}

		var targetTables []*schema.Table
		if len(targetTableNames) > 0 {
			// Create a map for requested tables for O(1) lookup
			reqTables := make(map[string]bool)
			for _, t := range targetTableNames {
				reqTables[strings.ToLower(t)] = true // Use strings
			}

			// Filter
			for _, t := range allTables {
				if reqTables[strings.ToLower(t.Name)] {
					targetTables = append(targetTables, t)
				}
			}

			if len(targetTables) == 0 {
				return fmt.Errorf("no matching tables found for inputs: %v", targetTableNames)
			}
		} else {
			targetTables = allTables
		}

		// Clean if requested
		if clean {
			if err := cleanDatabase(targetTables, d); err != nil {
				return err
			}
		}

		// Dry Run
		if dryRun {
			log.Println("[SIMULATION] Dry-Run Mode Active: No data will be written.")
			fmt.Printf("🔍 Analysis Results:\n")
			for i, t := range targetTables {
				fmt.Printf("[%02d] %s (Dependencies: %v)\n", i+1, t.Name, t.Dependencies)
			}
			return nil
		}

		log.Printf("Starting pump with count=%d per table...", targetCount)
		start := time.Now()

		// 2. Setup Progress Bar
		uiprogress.Start()
		bar := uiprogress.AddBar(100).AppendCompleted().PrependElapsed()
		bar.PrependFunc(func(b *uiprogress.Bar) string {
			return "Processing: "
		})

		// 3. Pump
		results, err := engine.Pump(DB, d, targetTables, targetCount, func() {
			bar.Incr()
		})

		uiprogress.Stop()

		if err != nil {
			return err
		}

		// 4. Verification Step
		verifiedResults := engine.VerifyInjection(DB, results)

		elapsed := time.Since(start)

		// 5. Final Report
		fmt.Println("\n📊 Summary Report (Dependency Order):")
		total := 0
		for i, r := range verifiedResults {
			icon := "✓"
			if r.Status != "VERIFIED_OK" {
				icon = "!"
			}
			// Status color/format
			statusDisplay := r.Status
			if statusDisplay == "VERIFIED_OK" {
				statusDisplay = "OK (Verified)"
			}

			fmt.Printf("[%s] [%02d/%02d] %-20s : %d rows (Target: %d) - %s\n",
				icon, i+1, len(results), r.TableName, r.Actual, r.Target, statusDisplay)
			if r.ErrorMsg != "" {
				fmt.Printf("    └ Error: %s\n", r.ErrorMsg)
			}
			total += r.Actual
		}
		fmt.Println("--------------------------------------------------")
		fmt.Printf("Total Operations: %d\n", total)
		log.Printf("Pump Done! Time Elapsed: %s", elapsed)

		return nil
	},
}

func init() {
	RootCmd.AddCommand(fillCmd)

	// CLI Flags
	fillCmd.Flags().IntVar(&count, "count", 0, "Number of records to generate per table (overrides config)")
	fillCmd.Flags().BoolVar(&clean, "clean", false, "Clean tables before filling")
	fillCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate the process without writing to DB")
	fillCmd.Flags().StringSliceVarP(&tables, "tables", "t", []string{}, "Specific tables to fill (comma-separated)")

	viper.BindPFlag("settings.default_count", fillCmd.Flags().Lookup("count"))
	viper.SetDefault("settings.default_count", 100)
	// Bind tables flag? No, typically slice flags are tricky to bind bidirectionally with Viper simply.
	// We handle explicit precedence in Code: Flag > Config > All.
}
