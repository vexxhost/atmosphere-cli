package cli

import (
	"github.com/spf13/cobra"
	"github.com/vexxhost/atmosphere/internal/config"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	configFlags = genericclioptions.NewConfigFlags(true)
	configFile  string
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "atmosphere",
		Short: "Atmosphere CLI",
		Long:  `Atmosphere is a tool for managing cloud infrastructure deployments.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return config.Load(configFile)
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Show help if no subcommand is provided
			cmd.Help()
		},
	}

	// Add config file flag
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.atmosphere/atmosphere.yaml)")

	configFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(NewDeployCommand())
	rootCmd.AddCommand(newOVNNbctlCmd(configFlags))
	rootCmd.AddCommand(newOVNSbctlCmd(configFlags))

	return rootCmd
}
