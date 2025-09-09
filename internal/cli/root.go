package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	configFlags = genericclioptions.NewConfigFlags(true)
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "atmosphere",
		Short: "Atmosphere CLI",
		Long:  `Atmosphere is a tool for managing cloud infrastructure deployments.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "Error showing help: %v\n", err)
			}
		},
	}

	configFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(newOVNNbctlCmd(configFlags))
	rootCmd.AddCommand(newOVNSbctlCmd(configFlags))

	return rootCmd
}
