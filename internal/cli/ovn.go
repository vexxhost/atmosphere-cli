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

const (
	ovnNamespace = "openstack"
	ovnNBSts     = "ovn-ovsdb-nb"
	ovnSBSts     = "ovn-ovsdb-sb"
)

// newOVNNbctlCmd creates the ovn-nbctl subcommand
func newOVNNbctlCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "ovn-nbctl [args...]",
		Short:              "Execute ovn-nbctl commands on the OVN northbound database",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOVNCommand(configFlags, "nb", args)
		},
	}
	return cmd
}

// newOVNSbctlCmd creates the ovn-sbctl subcommand
func newOVNSbctlCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "ovn-sbctl [args...]",
		Short:              "Execute ovn-sbctl commands on the OVN southbound database",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOVNCommand(configFlags, "sb", args)
		},
	}
	return cmd
}

// runOVNCommand executes the OVN command via Kubernetes API
func runOVNCommand(configFlags *genericclioptions.ConfigFlags, dbType string, args []string) error {
	var stsName, cmdName, dbPort string
	
	switch dbType {
	case "nb":
		stsName = ovnNBSts
		cmdName = "ovn-nbctl"
		dbPort = "6641"
	case "sb":
		stsName = ovnSBSts
		cmdName = "ovn-sbctl"
		dbPort = "6642"
	default:
		return fmt.Errorf("invalid database type: %s", dbType)
	}

	// Build the database connection string
	dbConnections := []string{
		fmt.Sprintf("tcp:%s-0.%s.%s.svc.cluster.local:%s", stsName, stsName, ovnNamespace, dbPort),
		fmt.Sprintf("tcp:%s-1.%s.%s.svc.cluster.local:%s", stsName, stsName, ovnNamespace, dbPort),
		fmt.Sprintf("tcp:%s-2.%s.%s.svc.cluster.local:%s", stsName, stsName, ovnNamespace, dbPort),
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
		Namespace(ovnNamespace).
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