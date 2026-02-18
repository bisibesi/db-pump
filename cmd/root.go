package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dsn        string
	DB         *sql.DB
	SchemaName string // Only relevant for MySQL mostly, or passed to Analyzer
	cfgFile    string
	DriverName string // "mysql" or "postgres"
)

var RootCmd = &cobra.Command{
	Use:   "db-pump",
	Short: "A database population tool",
	Long: `
  ____  ____    ____  _   _ __  __ ____  
 |  _ \|  _ \  |  _ \| | | |  \/  |  _ \ 
 | | | | |_) | | |_) | | | | |\/| | |_) |
 | |_| |  _ <  |  __/| |_| | |  | |  __/ 
 |____/|_| \_\ |_|    \___/|_|  |_|_|    
                                         
DB PUMP 🦅 - Database Data Generator & Pumper
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 1. DSN Strategy: Flag > Config > Env
		// If DSN flag is set, it overrides everything.
		// If not, we try to load from config "database.dsn" (but the structure is complex usually)
		// Actually, our config structure is `databases: [{...}]`.
		// But for CLI-only mode, we allow a simple --dsn flag.

		connStr := cmd.Flag("dsn").Value.String()
		driver := cmd.Flag("driver").Value.String()

		// If no flag, fallback to Active Config from file
		if connStr == "" {
			activeConfig, err := GetActiveDBConfig()
			if err == nil {
				connStr = activeConfig.DSN
				driver = activeConfig.Driver
			}
		}

		if connStr == "" {
			return fmt.Errorf("DSN is required. Provide it via --dsn flag or a config file")
		}

		// Driver Detection
		if driver == "" {
			if strings.Contains(connStr, "postgres") {
				driver = "postgres"
			} else if strings.Contains(connStr, "mysql") {
				driver = "mysql"
			} else if strings.Contains(connStr, "sqlserver") {
				driver = "sqlserver"
			} else if strings.Contains(connStr, "oracle") {
				driver = "oracle"
			} else {
				return fmt.Errorf("could not detect driver from DSN, please specify --driver")
			}
		}

		DriverName = driver
		dsn = connStr // Store it for subcommands

		// Note: Actual DB connection is done in fill/clean commands now to support dynamic config?
		// No, root command sets up global state usually, but `fill` command was refactored
		// to call GetActiveDBConfig() itself.
		// We need to sync them.
		// If we are in CLI mode, GetActiveDBConfig might fail if no config file.
		// We should let subcommands handle connection OR standardise here.
		// Given `fill.go` logic:
		/*
			config, err := GetActiveDBConfig()
			if err != nil { return err }
		*/
		// We need to update `fill.go` and `clean.go` to support CLI flags too.
		// For now, let's just validations here.
		return nil
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Define flags
	// Define flags
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./db-pump.yaml)")
	RootCmd.PersistentFlags().StringVar(&dsn, "dsn", "", "Database Source Name (DSN) for CLI-only mode")
	RootCmd.PersistentFlags().StringVar(&DriverName, "driver", "", "Database Driver (mysql, postgres, sqlserver, oracle) for CLI-only mode")

	// Bind dsn flag to viper
	viper.BindPFlag("cli.dsn", RootCmd.PersistentFlags().Lookup("dsn"))
	viper.BindPFlag("cli.driver", RootCmd.PersistentFlags().Lookup("driver"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// 1. Executable Directory (Priority 1)
		ex, err := os.Executable()
		if err == nil {
			exePath := filepath.Dir(ex)
			viper.AddConfigPath(exePath)
		}

		// 2. Current Directory (Priority 2)
		viper.AddConfigPath(".")

		viper.SetConfigName("db-pump")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		// Just log if it's explicitly set but failing.
		// If implicit, we don't care, we might be in CLI mode.
		if cfgFile != "" {
			fmt.Println("Error reading config file:", err)
		}
	}
}
