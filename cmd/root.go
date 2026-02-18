package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
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
		// Use Viper to get DSN (Flag > Config > Default)
		connStr := viper.GetString("database.dsn")
		if connStr == "" {
			return fmt.Errorf("database.dsn is required (via flag or config)")
		}

		// Detect Driver (Allow override via config if needed, but auto-detect is usually fine)
		// Check config first for explicit driver
		configDriver := viper.GetString("database.driver")
		if configDriver != "" {
			DriverName = configDriver
		} else {
			if strings.Contains(connStr, "postgres") || strings.Contains(connStr, "sslmode") {
				DriverName = "postgres"
			} else {
				DriverName = "mysql"
			}
		}

		var err error
		DB, err = sql.Open(DriverName, connStr)
		if err != nil {
			return fmt.Errorf("failed to open db: %w", err)
		}
		if err := DB.Ping(); err != nil {
			return fmt.Errorf("failed to connect to db: %w", err)
		}

		// Fetch current database/schema name for Analyzer
		if DriverName == "mysql" {
			if err := DB.QueryRow("SELECT DATABASE()").Scan(&SchemaName); err != nil {
				return fmt.Errorf("failed to get database name: %w", err)
			}
		} else {
			SchemaName = "public"
		}

		if SchemaName == "" && DriverName == "mysql" {
			return fmt.Errorf("no database selected in DSN")
		}

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
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./db-pump.yaml)")
	RootCmd.PersistentFlags().StringVar(&dsn, "dsn", "", "Database Source Name (DSN)")

	// Bind dsn flag to viper
	viper.BindPFlag("database.dsn", RootCmd.PersistentFlags().Lookup("dsn"))

	// Set default for Viper (fallback if no config/flag)
	viper.SetDefault("database.dsn", "root:root@tcp(127.0.0.1:3306)/sakila?parseTime=true")
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
	}
}
