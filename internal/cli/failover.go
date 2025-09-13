package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"

	apiv1alpha1 "github.com/vexxhost/atmosphere/apis/v1alpha1"
	"github.com/vexxhost/atmosphere/internal/cli/resources"
	"github.com/vexxhost/atmosphere/internal/ovnrouter"
)

// FailoverCmd handles the failover command
type FailoverCmd struct {
	configFlags *genericclioptions.ConfigFlags
	ovnConfig   *resources.OVNConfig

	// Command options
	ovnEndpoints []string
	ovnNamespace string
	timeout      time.Duration
	all          bool
}

// NewFailoverCommand creates a new failover command
func NewFailoverCommand(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	f := &FailoverCmd{
		configFlags: configFlags,
		ovnConfig:   resources.DefaultOVNConfig(),
		timeout:     30 * time.Second,
	}

	cmd := &cobra.Command{
		Use:   "failover [router-uuid...]",
		Short: "Trigger failover for one or more routers",
		Long: `Trigger failover for one or more routers.

This command will move routers from their current hosting gateway chassis 
to the next available one by swapping priorities between the highest and lowest.

Examples:
  # Failover a single router
  atmosphere failover 550e8400-e29b-41d4-a716-446655440000
  
  # Failover multiple routers
  atmosphere failover uuid1 uuid2 uuid3
  
  # Failover multiple routers using comma-separated UUIDs
  atmosphere failover uuid1,uuid2,uuid3
  
  # Failover all routers
  atmosphere failover --all
  
  # Failover with custom timeout
  atmosphere failover uuid1 --timeout=60s
  
  # Use custom OVN endpoints
  atmosphere failover uuid1 --ovn-endpoints tcp://ovn-nb-0:6641,tcp://ovn-nb-1:6641`,
		RunE: f.run,
	}

	// Add flags
	cmd.Flags().BoolVar(&f.all, "all", false, "Failover all routers")
	cmd.Flags().DurationVar(&f.timeout, "timeout", 30*time.Second, "Timeout for each router failover")

	// OVN configuration flags
	cmd.Flags().StringSliceVar(&f.ovnEndpoints, "ovn-endpoints", nil, "OVN database endpoints (default: auto-generated from namespace and statefulset)")
	cmd.Flags().StringVar(&f.ovnNamespace, "ovn-namespace", "openstack", "Namespace where OVN is deployed")

	return cmd
}

// run executes the failover command
func (f *FailoverCmd) run(cmd *cobra.Command, args []string) error {
	// Check arguments
	if !f.all && len(args) == 0 {
		return fmt.Errorf("you must specify router UUIDs or use --all flag")
	}

	if f.all && len(args) > 0 {
		return fmt.Errorf("cannot specify router UUIDs when using --all flag")
	}

	// Parse router UUIDs from arguments
	var routerUUIDs []string
	if !f.all {
		for _, arg := range args {
			// Support comma-separated UUIDs using K8s helper
			uuids := resource.SplitResourceArgument(arg)
			routerUUIDs = append(routerUUIDs, uuids...)
		}
	}

	// Update OVN config with command line options
	if len(f.ovnEndpoints) > 0 {
		f.ovnConfig.Endpoints = f.ovnEndpoints
	}
	if f.ovnNamespace != "" {
		f.ovnConfig.Namespace = f.ovnNamespace
	}

	// Connect to OVN
	ctx := context.Background()
	ovnClient, err := f.connectToOVN(ctx)
	if err != nil {
		return err
	}
	defer ovnClient.Close()

	// Get routers to failover
	var routers []apiv1alpha1.Router

	if f.all {
		// Get all routers
		routerList, err := ovnrouter.List(ctx, ovnClient)
		if err != nil {
			return fmt.Errorf("failed to list routers: %w", err)
		}
		routers = routerList.Items
		fmt.Printf("Found %d routers to failover\n", len(routers))
	} else {
		// Get specific routers by UUID
		allRouters, err := ovnrouter.List(ctx, ovnClient)
		if err != nil {
			return fmt.Errorf("failed to list routers: %w", err)
		}

		// Create a map for quick lookup
		routerMap := make(map[string]apiv1alpha1.Router)
		for _, r := range allRouters.Items {
			routerMap[string(r.UID)] = r
		}

		// Find requested routers
		for _, uuid := range routerUUIDs {
			if router, ok := routerMap[uuid]; ok {
				routers = append(routers, router)
			} else {
				return fmt.Errorf("router with UUID %q not found", uuid)
			}
		}
	}

	if len(routers) == 0 {
		fmt.Println("No routers to failover")
		return nil
	}

	// Perform failover for each router
	successCount := 0
	failureCount := 0

	for _, router := range routers {
		// Get router name for display
		routerName := router.Name
		if router.Name != string(router.UID) {
			routerName = fmt.Sprintf("%s (%s)", router.Name, router.UID)
		}

		fmt.Printf("Triggering failover for router %s... ", routerName)

		// Create a context with timeout for this specific failover
		failoverCtx, cancel := context.WithTimeout(ctx, f.timeout)
		err := ovnrouter.Failover(failoverCtx, ovnClient, &router)
		cancel()

		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
			failureCount++
			// Continue with other routers even if one fails
		} else {
			fmt.Printf("SUCCESS\n")
			successCount++
		}
	}

	// Print summary
	fmt.Printf("\nFailover complete: %d succeeded, %d failed\n", successCount, failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d router(s) failed to failover", failureCount)
	}

	return nil
}

// connectToOVN establishes connection to OVN database
func (f *FailoverCmd) connectToOVN(ctx context.Context) (client.Client, error) {
	// Get database model
	dbModel, err := nbdb.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("failed to get database model: %w", err)
	}

	// Get endpoints
	endpoints := f.ovnConfig.GetNBEndpoints()

	// Create client
	ovnClient, err := client.NewOVSDBClient(
		dbModel,
		client.WithEndpoint(strings.Join(endpoints, ",")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OVN client: %w", err)
	}

	// Connect
	if err := ovnClient.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to OVN: %w", err)
	}

	// Monitor the database
	if _, err := ovnClient.MonitorAll(ctx); err != nil {
		return nil, fmt.Errorf("failed to monitor OVN database: %w", err)
	}

	return ovnClient, nil
}
