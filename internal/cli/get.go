package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"

	"github.com/vexxhost/atmosphere/internal/cli/resources"
)

// GetCmd handles the get command
type GetCmd struct {
	configFlags *genericclioptions.ConfigFlags
	registry    *resources.Registry
	ovnConfig   *resources.OVNConfig

	// Command options
	outputFormat string
	noHeaders    bool
	ovnEndpoints []string
	ovnNamespace string
}

// NewGetCommand creates a new get command
func NewGetCommand(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	g := &GetCmd{
		configFlags: configFlags,
		registry:    resources.NewRegistry(),
		ovnConfig:   resources.DefaultOVNConfig(),
	}

	// Register all resources
	g.registerResources()

	cmd := &cobra.Command{
		Use:   "get [resource] [uuid...]",
		Short: "Display one or many resources",
		Long:  g.getLongDescription(),
		RunE:  g.run,
	}

	// Add flags
	cmd.Flags().StringVarP(&g.outputFormat, "output", "o", "", "Output format. One of: (json, yaml, wide)")
	cmd.Flags().BoolVar(&g.noHeaders, "no-headers", false, "When using the default output format, don't print headers")

	// OVN configuration flags
	cmd.Flags().StringSliceVar(&g.ovnEndpoints, "ovn-endpoints", nil, "OVN database endpoints (default: auto-generated from namespace and statefulset)")
	cmd.Flags().StringVar(&g.ovnNamespace, "ovn-namespace", "openstack", "Namespace where OVN is deployed")

	return cmd
}

// registerResources registers all available resources
func (g *GetCmd) registerResources() {
	// Register router resource
	g.registry.Register(&resources.RouterResource{})

	// Future resources can be registered here:
	// g.registry.Register(&resources.PortResource{})
	// g.registry.Register(&resources.SwitchResource{})
	// g.registry.Register(&resources.LoadBalancerResource{})
}

// getLongDescription builds the long description with available resources
func (g *GetCmd) getLongDescription() string {
	resourceList := strings.Join(g.registry.List(), ", ")

	return fmt.Sprintf(`Display one or many resources.

Prints a table of the most important information about the specified resources.
You can filter the list using optional resource UUIDs.

Available resources: %s

Examples:
  # List all routers
  atmosphere get routers
  
  # Get a specific router by UUID (space-separated)
  atmosphere get router 550e8400-e29b-41d4-a716-446655440000
  
  # Get a specific router by UUID (slash notation)
  atmosphere get router/550e8400-e29b-41d4-a716-446655440000
  
  # Get multiple routers by UUID
  atmosphere get routers uuid1 uuid2
  
  # Get multiple routers using comma-separated UUIDs
  atmosphere get routers/uuid1,uuid2,uuid3
  
  # Output in JSON format
  atmosphere get routers -o json
  
  # Output in YAML format
  atmosphere get routers -o yaml
  
  # Use custom OVN endpoints
  atmosphere get routers --ovn-endpoints tcp://ovn-nb-0:6641,tcp://ovn-nb-1:6641
  
  # Use OVN from different namespace
  atmosphere get routers --ovn-namespace kube-system`, resourceList)
}

// run executes the get command
func (g *GetCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("you must specify the type of resource to get. Available resources: %s",
			strings.Join(g.registry.List(), ", "))
	}

	// Parse resource type and names (supporting both "resource name" and "resource/name" formats)
	resourceType, resourceNames, err := g.parseResourceArgs(args)
	if err != nil {
		return err
	}

	// Get the resource handler
	resource, ok := g.registry.Get(resourceType)
	if !ok {
		return fmt.Errorf("unknown resource type: %s. Available resources: %s",
			resourceType, strings.Join(g.registry.List(), ", "))
	}

	// Update OVN config with command line options
	if len(g.ovnEndpoints) > 0 {
		g.ovnConfig.Endpoints = g.ovnEndpoints
	}
	if g.ovnNamespace != "" {
		g.ovnConfig.Namespace = g.ovnNamespace
	}

	// Connect to OVN
	ctx := context.Background()
	ovnClient, err := g.connectToOVN(ctx)
	if err != nil {
		return err
	}
	defer ovnClient.Close()

	// Fetch the resources
	data, err := resource.List(ctx, ovnClient, resourceNames)
	if err != nil {
		return err
	}

	// Output the results
	streams := genericclioptions.IOStreams{
		Out: os.Stdout,
	}

	switch g.outputFormat {
	case "json", "yaml":
		// Print as JSON/YAML
		return g.printObject(data, streams.Out, g.outputFormat)
	case "wide":
		// Get the wide table representation
		if tableResource, ok := resource.(interface {
			GetWideTable(runtime.Object) (*metav1.Table, error)
		}); ok {
			table, err := tableResource.GetWideTable(data)
			if err != nil {
				return err
			}
			return g.printTable(table, streams.Out)
		}
		// Fallback to regular table if no wide implementation
		table, err := resource.GetTable(data)
		if err != nil {
			return err
		}
		return g.printTable(table, streams.Out)
	default:
		// Get the regular table representation
		table, err := resource.GetTable(data)
		if err != nil {
			return err
		}
		return g.printTable(table, streams.Out)
	}
}

// connectToOVN establishes connection to OVN database
func (g *GetCmd) connectToOVN(ctx context.Context) (client.Client, error) {
	// Get database model
	dbModel, err := model.NewClientDBModel("OVN_Northbound", map[string]model.Model{
		nbdb.GatewayChassisTable:    &nbdb.GatewayChassis{},
		nbdb.LogicalRouterTable:     &nbdb.LogicalRouter{},
		nbdb.LogicalRouterPortTable: &nbdb.LogicalRouterPort{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get database model: %w", err)
	}

	// Get endpoints
	endpoints := g.ovnConfig.GetNBEndpoints()

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

// printTable prints a table using the table printer
func (g *GetCmd) printTable(table *metav1.Table, out io.Writer) error {
	printer := printers.NewTablePrinter(printers.PrintOptions{
		NoHeaders: g.noHeaders,
	})

	return printer.PrintObj(table, out)
}

// parseResourceArgs parses command arguments supporting both "resource name" and "resource/name" formats
func (g *GetCmd) parseResourceArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("no arguments provided")
	}

	firstArg := strings.ToLower(args[0])

	// Check if the first argument contains a slash (resource/name format)
	if strings.Contains(firstArg, "/") {
		// Handle resource/name format
		parts := strings.Split(firstArg, "/")
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("arguments in resource/name form may not have more than one slash")
		}

		resourceType := parts[0]
		resourceName := parts[1]

		if len(resourceType) == 0 || len(resourceName) == 0 {
			return "", nil, fmt.Errorf("arguments in resource/name form must have a single resource and name")
		}

		// Check if there are additional arguments (which would be an error)
		if len(args) > 1 {
			return "", nil, fmt.Errorf("there is no need to specify additional arguments when using resource/name form")
		}

		// Use Kubernetes' SplitResourceArgument to handle comma-separated names
		names := resource.SplitResourceArgument(resourceName)
		return resourceType, names, nil
	}

	// Handle space-separated format (resource name1 name2 ...)
	resourceType := firstArg
	var resourceNames []string

	if len(args) > 1 {
		// Process all remaining arguments as resource names, supporting comma-separation
		for _, arg := range args[1:] {
			// Use Kubernetes' SplitResourceArgument for consistency
			names := resource.SplitResourceArgument(arg)
			resourceNames = append(resourceNames, names...)
		}
	}

	return resourceType, resourceNames, nil
}

// printObject prints data in JSON or YAML format
func (g *GetCmd) printObject(obj runtime.Object, out io.Writer, format string) error {
	var printer printers.ResourcePrinter
	switch format {
	case "json":
		printer = &printers.JSONPrinter{}
	case "yaml":
		printer = &printers.YAMLPrinter{}
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return printer.PrintObj(obj, out)
}
