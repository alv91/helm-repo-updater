package updater

import (
	"bytes"
	"github.com/argoproj-labs/argocd-image-updater/pkg/log"
	"text/template"
)

// templateCommitMessage renders a commit message template and returns it as
// as a string. If the template could not be rendered, returns a default
// message.
func TemplateCommitMessage(tpl *template.Template, appName string, changeList []ChangeEntry) string {
	var cmBuf bytes.Buffer

	type commitMessageChange struct {
		Image  string
		OldTag string
		NewTag string
	}

	type commitMessageTemplate struct {
		AppName    string
		AppChanges []commitMessageChange
	}

	// We need to transform the change list into something more viable for the
	// writer of a template.
	changes := make([]commitMessageChange, 0)
	for _, c := range changeList {
		changes = append(changes, commitMessageChange{c.Key, c.OldTag, c.NewTag})
	}

	tplData := commitMessageTemplate{
		AppName:    appName,
		AppChanges: changes,
	}

	err := tpl.Execute(&cmBuf, tplData)
	if err != nil {
		log.Errorf("could not execute template for Git commit message: %v", err)
		return "build: update of application " + appName
	}

	return cmBuf.String()
}
