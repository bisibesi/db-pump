package cmd

import (
	"fmt"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Name   string `mapstructure:"name"`
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
	Active bool   `mapstructure:"active"`
}

// GetActiveDBConfig returns the currently active database configuration.
func GetActiveDBConfig() (*DBConfig, error) {
	var configs []DBConfig

	if err := viper.UnmarshalKey("databases", &configs); err != nil {
		return nil, fmt.Errorf("failed to parse databases config: %w", err)
	}

	var activeConfig *DBConfig
	count := 0

	for i := range configs {
		if configs[i].Active {
			activeConfig = &configs[i]
			count++
		}
	}

	if count == 0 {
		return nil, fmt.Errorf("no active database found in config (set active: true)")
	}
	if count > 1 {
		return nil, fmt.Errorf("multiple active databases found (only one can be active)")
	}

	return activeConfig, nil
}
