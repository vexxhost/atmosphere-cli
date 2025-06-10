package cli

import (
	"fmt"

	flow "github.com/noneback/go-taskflow"
	"github.com/spf13/cobra"
	"github.com/vexxhost/atmosphere/internal/workflows"
)

// NewDeployCommand creates and returns the deploy command
func NewDeployCommand() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy Atmosphere components",
		Long: `Deploy various Atmosphere components to your infrastructure.
This command handles the deployment of services, configurations, and resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Now we can use the global configFlags directly
			// Example of how to get a Kubernetes client:
			// config, err := configFlags.ToRESTConfig()
			// if err != nil {
			//     return err
			// }
			// clientset, err := kubernetes.NewForConfig(config)
			// if err != nil {
			//     return err
			// }

			// Create the executor
			executor := flow.NewExecutor(10) // max concurrent tasks

			// Create the deployment workflow
			tf := workflows.CreateDeployWorkflow(configFlags)

			// Execute the workflow
			executor.Run(tf).Wait()

			fmt.Println("Deployment completed successfully!")
			return nil
		},
	}

	// Add command-specific flags here as needed
	// Example: deployCmd.Flags().StringP("config", "c", "", "Path to configuration file")

	return deployCmd
}
