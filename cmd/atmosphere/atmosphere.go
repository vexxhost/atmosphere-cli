package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/vexxhost/atmosphere/internal/cli"
)

func main() {
	viper.SetConfigName("atmosphere")
	viper.SetConfigType("toml")

	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/atmosphere/")

	viper.SetEnvPrefix("ATMOSPHERE")
	viper.AutomaticEnv()

	log.SetLevel(log.DebugLevel)
	log.SetReportTimestamp(true)
	log.SetReportCaller(true)

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
