package workflows

import (
	"fmt"

	"github.com/charmbracelet/log"
	flow "github.com/noneback/go-taskflow"
	"github.com/vexxhost/atmosphere/internal/helm"
)

func CheckHelmReleaseExistsCondition(sf *flow.Subflow, release *helm.Release) *flow.Task {
	conditionName := fmt.Sprintf("check-release-exists-%s", release.Name)

	return sf.NewCondition(conditionName, func() uint {
		exists, err := release.Exists()
		if err != nil {
			log.Fatal("Failed to check if release exists", "error", err)
		}

		if !exists {
			return 0 // install
		}

		return 1 // upgrade
	})
}

func InstallHelmReleaseTask(sf *flow.Subflow, release *helm.Release) *flow.Task {
	return sf.NewTask(fmt.Sprintf("install-release-%s", release.Name), func() {
		log.Info("Installing release", "name", release.Name)

		release, err := release.Install()
		if err != nil {
			log.Fatal("Install failed", "error", err)
		}

		log.Info("Successfully installed release", "name", release.Name, "version", release.Version)
	})
}

func UpgradeHelmReleaseTask(sf *flow.Subflow, release *helm.Release) *flow.Task {
	return sf.NewTask(fmt.Sprintf("upgrade-release-%s", release.Name), func() {
		log.Info("Upgrading release", "name", release.Name)

		release, err := release.Upgrade()
		if err != nil {
			log.Fatal("Upgrade failed", "error", err)
		}

		log.Info("Successfully upgraded release", "name", release.Name, "version", release.Version)
	})
}
