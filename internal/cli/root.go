package cli

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var configFlags = genericclioptions.NewConfigFlags(true)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "atmosphere",
		Short: "Atmosphere CLI",
		Long:  `Atmosphere is a tool for managing cloud infrastructure deployments.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Show help if no subcommand is provided
			cmd.Help()
		},
	}

	configFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(NewDeployCommand())

	return rootCmd
}
