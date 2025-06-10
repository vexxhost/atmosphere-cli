package workflows

import (
	"fmt"

	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/helm"
)

// TaskFlow wraps go-taskflow's TaskFlow with custom methods
type TaskFlow struct {
	*flow.TaskFlow
}

// NewTaskFlow creates a new custom TaskFlow
func NewTaskFlow() *TaskFlow {
	return &TaskFlow{
		TaskFlow: flow.NewTaskFlow("deploy"),
	}
}

// NewDeployHelmChartFlow creates a subflow for deploying a helm chart
// chartRef can be:
// - Local path: "./charts/myapp" or "/absolute/path/to/chart"
// - OCI registry: "oci://registry.example.com/charts/myapp"
// - Repository chart: "bitnami/postgresql"
func (tf *TaskFlow) NewDeployHelmChartFlow(release *helm.Release) *flow.Task {
	return tf.NewSubflow(fmt.Sprintf("deploy-chart-%s", release.Name), func(sf *flow.Subflow) {
		cond := CheckHelmReleaseExistsCondition(sf, release)
		cond.Precede(
			InstallHelmReleaseTask(sf, release),
			UpgradeHelmReleaseTask(sf, release),
		)
	})
}
