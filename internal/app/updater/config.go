package updater

import (
	"github.com/alv91/helm-repo-updater/internal/app/git"
)

// HelmUpdaterConfig contains global configuration and required runtime data
type HelmUpdaterConfig struct {
	DryRun         bool
	LogLevel       string
	AppName        string
	UpdateApps     []Change
	File           string
	GitCredentials *git.Credentials
	GitConf        *git.Conf
}

// Change contains the information about the change to be made
type Change struct {
	Key      string
	NewValue string
	OldValue string
}

// ChangeEntry represents values that has been changed by Helm Updater
type ChangeEntry struct {
	OldValue string
	NewValue string
	File     string
	Key      string
}
