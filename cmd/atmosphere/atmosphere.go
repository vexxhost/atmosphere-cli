package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/vexxhost/atmosphere/internal/cli"
)

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetReportTimestamp(true)
	log.SetReportCaller(true)

	// Create and execute the root command
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
