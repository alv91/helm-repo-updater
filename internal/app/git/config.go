package git

import "text/template"

// The default Git commit message's template
const DefaultGitCommitMessage = `🚀 automatic update of {{ .AppName }}

{{ range .AppChanges -}}
updates key {{ .Image }} tag '{{ .OldTag }}' to '{{ .NewTag }}'
{{ end -}}
`

type GitConf struct {
	RepoURL string
	Branch  string
	File    string
	Message *template.Template
}

