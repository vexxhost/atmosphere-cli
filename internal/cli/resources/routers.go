package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ovn-org/libovsdb/client"
	apiv1alpha1 "github.com/vexxhost/atmosphere/apis/v1alpha1"
	"github.com/vexxhost/atmosphere/internal/ovnrouter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RouterResource handles router resources
type RouterResource struct{}

// Name returns the resource name
func (r *RouterResource) Name() string {
	return "routers"
}

// Aliases returns alternative names for the resource
func (r *RouterResource) Aliases() []string {
	return []string{"router"}
}

// List fetches routers and returns them as a runtime.Object
func (r *RouterResource) List(ctx context.Context, client client.Client, names []string) (runtime.Object, error) {
	// Fetch all routers (they already have external IPs populated)
	routerList, err := ovnrouter.List(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to list routers: %w", err)
	}
	
	// Filter by UUID if specified
	if len(names) > 0 {
		filtered := []apiv1alpha1.Router{}
		uuidSet := make(map[string]bool)
		for _, uuid := range names {
			uuidSet[uuid] = true
		}
		
		for _, router := range routerList.Items {
			// Check UUID only
			if uuidSet[string(router.UID)] {
				filtered = append(filtered, router)
			}
		}
		routerList.Items = filtered
	}
	
	// Sort routers by UUID for consistent output
	sort.Slice(routerList.Items, func(i, j int) bool {
		return string(routerList.Items[i].UID) < string(routerList.Items[j].UID)
	})
	
	return routerList, nil
}

// GetTable converts a runtime.Object list to a table representation (standard view)
func (r *RouterResource) GetTable(obj runtime.Object) (*metav1.Table, error) {
	routerList, ok := obj.(*apiv1alpha1.RouterList)
	if !ok {
		return nil, fmt.Errorf("expected RouterList, got %T", obj)
	}
	
	routers := routerList.Items
	
	// Define columns for standard view
	columns := []metav1.TableColumnDefinition{
		{Name: "UUID", Type: "string", Description: "Router UUID"},
		{Name: "NAME", Type: "string", Description: "Router name from Neutron"},
		{Name: "AGENT", Type: "string", Description: "Current hosting agent"},
		{Name: "EXTERNAL-IPS", Type: "string", Description: "External IP addresses (IPv4 and IPv6)"},
	}
	
	// Build rows
	rows := []metav1.TableRow{}
	for _, router := range routers {
		// Use the Name from ObjectMeta
		routerName := router.Name
		
		// Get hosting agent
		agent := router.Status.Agent
		if agent == "" {
			agent = "<none>"
		}
		
		// Format external IPs (already populated in the Router object)
		externalIPs := "<none>"
		if len(router.Status.ExternalIPs) > 0 {
			externalIPs = strings.Join(router.Status.ExternalIPs, ",")
		}
		
		row := metav1.TableRow{
			Cells: []interface{}{
				string(router.UID),
				routerName,
				agent,
				externalIPs,
			},
		}
		rows = append(rows, row)
	}
	
	// Create and return table
	return &metav1.Table{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Table",
			APIVersion: "meta.k8s.io/v1",
		},
		ColumnDefinitions: columns,
		Rows:              rows,
	}, nil
}

// GetWideTable converts a runtime.Object list to a wide table representation
func (r *RouterResource) GetWideTable(obj runtime.Object) (*metav1.Table, error) {
	routerList, ok := obj.(*apiv1alpha1.RouterList)
	if !ok {
		return nil, fmt.Errorf("expected RouterList, got %T", obj)
	}
	
	routers := routerList.Items
	
	// Define columns for wide view (includes all fields)
	columns := []metav1.TableColumnDefinition{
		{Name: "UUID", Type: "string", Description: "Router UUID"},
		{Name: "NAME", Type: "string", Description: "Router name from Neutron"},
		{Name: "AGENT", Type: "string", Description: "Current hosting agent"},
		{Name: "EXTERNAL-IPS", Type: "string", Description: "External IP addresses (IPv4 and IPv6)"},
		{Name: "ENABLED", Type: "string", Description: "Router enabled status"},
		{Name: "PORTS", Type: "integer", Description: "Number of ports"},
	}
	
	// Build rows
	rows := []metav1.TableRow{}
	for _, router := range routers {
		// Use the Name from ObjectMeta
		routerName := router.Name
		
		// Get hosting agent
		agent := router.Status.Agent
		if agent == "" {
			agent = "<none>"
		}
		
		// Format external IPs
		externalIPs := "<none>"
		if len(router.Status.ExternalIPs) > 0 {
			externalIPs = strings.Join(router.Status.ExternalIPs, ",")
		}
		
		// For wide view, we'll simplify for now - no enabled/ports info
		enabled := "N/A"
		ports := 0
		
		row := metav1.TableRow{
			Cells: []interface{}{
				string(router.UID),
				routerName,
				agent,
				externalIPs,
				enabled,
				ports,
			},
		}
		rows = append(rows, row)
	}
	
	// Create and return table
	return &metav1.Table{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Table",
			APIVersion: "meta.k8s.io/v1",
		},
		ColumnDefinitions: columns,
		Rows:              rows,
	}, nil
}
