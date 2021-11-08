package updater

import (
	"github.com/alv91/helm-repo-updater/internal/app/git"
)

// HelmUpdaterConfig contains global configuration and required runtime data
type HelmUpdaterConfig struct {
	DryRun           bool
	LogLevel         string
	AppName          string
	UpdateApps       []Application
	File			 string
	GitCredentials   *git.GitCredentials
	GitConf          *git.GitConf
}

type Application struct {
	Key   string
	Image string
}