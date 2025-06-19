package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/spf13/cobra"
	"github.com/vexxhost/atmosphere/internal/atmosphere"
	"github.com/vexxhost/atmosphere/internal/components"
	"github.com/vexxhost/atmosphere/internal/config"
)

// NewDeployCommand creates and returns the deploy command
func NewDeployCommand() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy Atmosphere components",
		Long: `Deploy various Atmosphere components to your infrastructure.
This command handles the deployment of services, configurations, and resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			configFile, _ := cmd.Flags().GetString("config")
			if err := config.Load(configFile); err != nil {
				log.Warn("failed to load config file", "error", err)
			}

			// Create context with atmosphere configuration
			ctx := atmosphere.New(context.Background(), configFlags)

			// Create workflow
			tf := flow.NewTaskFlow("deploy")
			executor := flow.NewExecutor(10)

			// Get overrides
			metricsServerOverrides, _ := config.GetHelmComponent("metrics-server")

			vaultOperatorOverrides, _ := config.GetHelmComponent("vault-operator")
			vaultOperatorCRDsOverrides, _ := config.GetManifestComponent("vault-operator-crds")
			vaultOperatorRBACOverrides, _ := config.GetManifestComponent("vault-operator-rbac")

			// Create component tasks
			metricsServer := components.NewMetricsServer(metricsServerOverrides)
			_ = metricsServer.GetTask(ctx, tf)

			vaultOperatorCRDs := components.NewVaultOperatorCRDs(vaultOperatorCRDsOverrides)
			vaultOperatorCRDsTask := vaultOperatorCRDs.GetTask(ctx, tf)
			vaultOperatorRBAC := components.NewVaultOperatorRBAC(vaultOperatorRBACOverrides)
			vaultOperatorRBACTask := vaultOperatorRBAC.GetTask(ctx, tf)
			vaultOperator := components.NewVaultOperator(vaultOperatorOverrides)
			vaultOperatorTask := vaultOperator.GetTask(ctx, tf)
			vaultOperatorTask.Succeed(vaultOperatorCRDsTask, vaultOperatorRBACTask) // Vault operator depends on CRDs and RBAC

			// Example: Add more components with dependencies
			// certManager := components.NewCertManager()
			// certManagerTask := certManager.GetTask(ctx, tf)

			// monitoring := components.NewMonitoring()
			// monitoringTask := monitoring.GetTask(ctx, tf)
			// monitoringTask.Succeed(metricsServerTask) // Monitoring depends on metrics-server

			// ingress := components.NewIngress()
			// ingressTask := ingress.GetTask(ctx, tf)
			// ingressTask.Succeed(certManagerTask) // Ingress depends on cert-manager for TLS

			// Execute the workflow
			executor.Run(tf).Wait()

			fmt.Println("Deployment completed successfully!")
			return nil
		},
	}

	// Add command-specific flags
	deployCmd.Flags().StringP("config", "c", "", "Path to configuration file")

	return deployCmd
}
