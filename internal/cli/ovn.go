package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// OVN command options
type ovnCmdOptions struct {
	namespace     string
	endpoints     []string
	nbStatefulSet string
	sbStatefulSet string
	nbPort        string
	sbPort        string
}

// defaultOVNOptions returns default OVN options
func defaultOVNOptions() *ovnCmdOptions {
	return &ovnCmdOptions{
		namespace:     "openstack",
		nbStatefulSet: "ovn-ovsdb-nb",
		sbStatefulSet: "ovn-ovsdb-sb",
		nbPort:        "6641",
		sbPort:        "6642",
	}
}

// newOVNNbctlCmd creates the ovn-nbctl subcommand
func newOVNNbctlCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	opts := defaultOVNOptions()
	
	cmd := &cobra.Command{
		Use:                "ovn-nbctl [args...]",
		Short:              "Execute ovn-nbctl commands on the OVN northbound database",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOVNCommand(configFlags, "nb", args, opts)
		},
	}
	
	// Add flags for OVN configuration
	cmd.Flags().StringVar(&opts.namespace, "ovn-namespace", opts.namespace, "Namespace where OVN is deployed")
	cmd.Flags().StringSliceVar(&opts.endpoints, "ovn-endpoints", nil, "OVN database endpoints (default: auto-generated)")
	
	return cmd
}

// newOVNSbctlCmd creates the ovn-sbctl subcommand
func newOVNSbctlCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	opts := defaultOVNOptions()
	
	cmd := &cobra.Command{
		Use:                "ovn-sbctl [args...]",
		Short:              "Execute ovn-sbctl commands on the OVN southbound database",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOVNCommand(configFlags, "sb", args, opts)
		},
	}
	
	// Add flags for OVN configuration
	cmd.Flags().StringVar(&opts.namespace, "ovn-namespace", opts.namespace, "Namespace where OVN is deployed")
	cmd.Flags().StringSliceVar(&opts.endpoints, "ovn-endpoints", nil, "OVN database endpoints (default: auto-generated)")
	
	return cmd
}

// runOVNCommand executes the OVN command via Kubernetes API
func runOVNCommand(configFlags *genericclioptions.ConfigFlags, dbType string, args []string, opts *ovnCmdOptions) error {
	var stsName, cmdName, dbPort string

	switch dbType {
	case "nb":
		stsName = opts.nbStatefulSet
		cmdName = "ovn-nbctl"
		dbPort = opts.nbPort
	case "sb":
		stsName = opts.sbStatefulSet
		cmdName = "ovn-sbctl"
		dbPort = opts.sbPort
	default:
		return fmt.Errorf("invalid database type: %s", dbType)
	}

	// Build the database connection string
	var dbConnections []string
	if len(opts.endpoints) > 0 {
		dbConnections = opts.endpoints
	} else {
		// Generate default endpoints
		dbConnections = []string{
			fmt.Sprintf("tcp:%s-0.%s.%s.svc.cluster.local:%s", stsName, stsName, opts.namespace, dbPort),
			fmt.Sprintf("tcp:%s-1.%s.%s.svc.cluster.local:%s", stsName, stsName, opts.namespace, dbPort),
			fmt.Sprintf("tcp:%s-2.%s.%s.svc.cluster.local:%s", stsName, stsName, opts.namespace, dbPort),
		}
	}
	dbString := strings.Join(dbConnections, ",")

	// Get Kubernetes client
	restConfig, err := configFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Get the first pod of the StatefulSet
	podName := fmt.Sprintf("%s-0", stsName)

	// Build command
	command := []string{
		cmdName,
		fmt.Sprintf("--db=%s", dbString),
	}
	command = append(command, args...)

	// Create exec request
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(opts.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)

	// Execute the command
	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Stream the command
	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})

	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
