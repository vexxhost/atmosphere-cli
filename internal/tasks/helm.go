package tasks

import (
	"fmt"

	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/helm"
)

func DeployHelmChartTask(tf *flow.TaskFlow, release *helm.Release) *flow.Task {
	return tf.NewSubflow(fmt.Sprintf("deploy-%s", release.ReleaseConfig.Name), func(sf *flow.Subflow) {
		cond := sf.NewCondition(fmt.Sprintf("check-release-exists-%s", release.ReleaseConfig.Name), func() uint {
			exists, err := release.Exists()
			if err != nil {
				log.Fatal("Failed to check if release exists", "error", err)
			}

			if !exists {
				return 0 // install
			}

			return 1 // upgrade
		})

		cond.Precede(
			sf.NewTask(fmt.Sprintf("install-release-%s", release.ReleaseConfig.Name), func() {
				log.Info("Installing release", "name", release.ReleaseConfig.Name)

				release, err := release.Install()
				if err != nil {
					log.Fatal("Install failed", "error", err)
				}

				log.Info("Successfully installed release", "name", release.ReleaseConfig.Name, "revision", release.Revision)
			}),
			sf.NewTask(fmt.Sprintf("upgrade-release-%s", release.ReleaseConfig.Name), func() {
				log.Info("Checking for changes in release", "name", release.ReleaseConfig.Name)

				release, err := release.Upgrade()
				if err != nil {
					log.Fatal("Upgrade failed", "error", err)
				}

				log.Info("Upgrade process completed", "name", release.ReleaseConfig.Name, "revision", release.Revision)
			}),
		)
	})
}
