package git

import (
	"fmt"
	"github.com/argoproj-labs/argocd-image-updater/ext/git"
)

type GitCredentials struct {
	Username   string
	Password   string
	Email      string
	SSHPrivKey string
}


func (g GitCredentials) NewCreds(repoURL string) (git.Creds, error) {
	if ok, _ := git.IsSSHURL(repoURL); ok {
		if g.SSHPrivKey != "" {

			return git.NewSSHCreds(g.SSHPrivKey, "", true), nil
		}
	} else if git.IsHTTPSURL(repoURL) {
		if g.Username != "" && g.Password != "" {

			return git.NewHTTPSCreds(g.Username, g.Password, "", "", true, ""), nil
		}
	}

	return nil, fmt.Errorf("unknown repository type")
}
