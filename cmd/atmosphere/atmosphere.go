package main

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/cli"
	"github.com/vexxhost/atmosphere/internal/tomlconfig"
)

func main() {
	// Set up logging first
	logLevelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	logLevel := log.DebugLevel // Default to debug
	if logLevelStr == "INFO" {
		logLevel = log.InfoLevel
	} else if logLevelStr == "ERROR" {
		logLevel = log.ErrorLevel
	}

	log.SetLevel(logLevel)
	log.SetReportTimestamp(true)
	log.SetReportCaller(true)

	// Load configuration with our BurntSushi/toml parser to preserve case
	_, err := tomlconfig.LoadConfig()
	if err != nil {
		log.Warn("Error reading config file with case preservation", "error", err)
	}

	// Keep viper setup for environment variables and CLI integration
	viper.SetConfigName("atmosphere")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/atmosphere/")
	viper.SetEnvPrefix("ATMOSPHERE")
	viper.AutomaticEnv()

	// Load viper config for CLI command integration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debug("No config file found, using defaults")
		} else {
			log.Warn("Error reading config file", "error", err)
		}
	} else {
		log.Debug("Using config file", "file", viper.ConfigFileUsed())
	}

	// Create and execute the root command
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
