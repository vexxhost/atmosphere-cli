package cli

import (
	"fmt"

	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/spf13/cobra"
	"github.com/vexxhost/atmosphere/internal/components"
)

// NewDeployCommand creates and returns the deploy command
func NewDeployCommand() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy Atmosphere components",
		Long: `Deploy various Atmosphere components to your infrastructure.
This command handles the deployment of services, configurations, and resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create workflow
			tf := flow.NewTaskFlow("deploy")
			executor := flow.NewExecutor(10)

			// Add components to deploy
			componentsToInstall := []components.Component{
				components.NewMetricsServer(),
				// Add more components here as needed
			}

			// Create deployment tasks
			for _, component := range componentsToInstall {
				release := component.GetRelease(configFlags)
				componentName := release.ReleaseConfig.Name
				
				tf.NewTask(fmt.Sprintf("deploy-%s", componentName), func() {
					log.Info("Deploying component", "name", componentName)
					
					if err := release.Deploy(); err != nil {
						log.Fatal("Failed to deploy component", "name", componentName, "error", err)
					}
					
					log.Info("Successfully deployed component", "name", componentName)
				})
			}

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
