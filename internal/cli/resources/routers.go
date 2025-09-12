package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ovn-org/libovsdb/client"
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
	// Fetch all routers
	routers, err := ovnrouter.List(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to list routers: %w", err)
	}
	
	// Filter by UUID if specified
	if len(names) > 0 {
		filtered := []ovnrouter.Router{}
		uuidSet := make(map[string]bool)
		for _, uuid := range names {
			uuidSet[uuid] = true
		}
		
		for _, router := range routers {
			// Check UUID only
			if uuidSet[router.UUID] {
				filtered = append(filtered, router)
			}
		}
		routers = filtered
	}
	
	// Convert to RouterInfo and fetch external IPs
	routerInfos := make([]RouterInfo, 0, len(routers))
	for _, router := range routers {
		routerInfo := RouterInfo{
			Router: router,
		}
		
		// Fetch external IPs for this router
		if externalIPs, err := router.GetExternalIPs(ctx); err == nil {
			routerInfo.ExternalIPs = externalIPs
		}
		// We ignore errors here to not fail the entire listing
		
		routerInfos = append(routerInfos, routerInfo)
	}
	
	// Sort routers by UUID for consistent output
	sort.Slice(routerInfos, func(i, j int) bool {
		return routerInfos[i].UUID < routerInfos[j].UUID
	})
	
	// Return as a RouterList
	return &RouterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RouterList",
			APIVersion: "atmosphere.vexxhost.com/v1",
		},
		Items: routerInfos,
	}, nil
}

// GetTable converts a runtime.Object list to a table representation
func (r *RouterResource) GetTable(obj runtime.Object) (*metav1.Table, error) {
	routerList, ok := obj.(*RouterList)
	if !ok {
		return nil, fmt.Errorf("expected RouterList, got %T", obj)
	}
	
	routers := routerList.Items
	
	// Define columns
	columns := []metav1.TableColumnDefinition{
		{Name: "UUID", Type: "string", Description: "Router UUID"},
		{Name: "NAME", Type: "string", Description: "Router name from Neutron"},
		{Name: "EXTERNAL-IPS", Type: "string", Description: "External IP addresses (IPv4 and IPv6)"},
		{Name: "ENABLED", Type: "string", Description: "Router enabled status"},
		{Name: "PORTS", Type: "integer", Description: "Number of ports"},
	}
	
	// Build rows
	rows := []metav1.TableRow{}
	for _, routerInfo := range routers {
		// Get router name from ExternalIDs
		routerName := "<none>"
		if routerInfo.ExternalIDs != nil {
			if name, ok := routerInfo.ExternalIDs["neutron:router_name"]; ok {
				routerName = name
			}
		}
		
		// Format external IPs
		externalIPs := "<none>"
		if len(routerInfo.ExternalIPs) > 0 {
			externalIPs = strings.Join(routerInfo.ExternalIPs, ",")
		}
		
		// Get enabled status
		enabled := "true"
		if routerInfo.Enabled != nil && !*routerInfo.Enabled {
			enabled = "false"
		}
		
		row := metav1.TableRow{
			Cells: []interface{}{
				routerInfo.UUID,
				routerName,
				externalIPs,
				enabled,
				len(routerInfo.Ports),
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
