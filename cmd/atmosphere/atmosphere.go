package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/vexxhost/atmosphere/internal/cli"
	"github.com/vexxhost/atmosphere/internal/config"
)

var k = koanf.New(".")

func main() {
	log.SetLevel(log.DebugLevel)

	// Load configuration
	k.Load(structs.Provider(config.Config{}, "conf"), nil)

	// TODO: Add configuration file loading when needed
	// if err := k.Load(file.Provider("atmosphere.toml"), toml.Parser()); err != nil {
	// 	log.Fatalf("error loading config: %v", err)
	// }

	// Create and execute the root command
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
