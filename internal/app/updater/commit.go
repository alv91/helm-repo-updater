package updater

import (
	"fmt"
	"github.com/argoproj-labs/argocd-image-updater/ext/git"
	"github.com/argoproj-labs/argocd-image-updater/pkg/log"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

func commitChangesLocked(cfg HelmUpdaterConfig, state *SyncIterationState) error {
	lock := state.GetRepositoryLock(cfg.GitConf.RepoURL)
	lock.Lock()
	defer lock.Unlock()

	return commitChangesGit(cfg, writeOverrides)
}

func writeOverrides(cfg HelmUpdaterConfig, gitC git.Client) (err error, skip bool, apps []Application) {
	targetFile := path.Join(gitC.Root(), cfg.GitConf.File, cfg.File)

	var presented, patched []byte
	var noChange, yamlErr int

	apps = make([]Application, 0)

	_, err = os.Stat(targetFile)
	if err != nil {
		log.Errorf("target file %s doesn't exist.", cfg.File)

		return err, true, nil
	}

	for _, app :=range cfg.UpdateApps {
		// check deployed app
		presented, err = ioutil.ReadFile(targetFile)
		if err != nil {
			return err, true, nil
		}

		// replace helm parameters
		err = yamlReplaceCMD(app, targetFile)
		if err != nil {
			log.Infof("failed to update key %s in %s: %v", app.Key, cfg.AppName, err)

			yamlErr++
		}

		// check patched app
		patched, err = ioutil.ReadFile(targetFile)
		if err != nil {
			return err, true, nil
		}

		// check if there is any change
		if string(patched) == string(presented) {
			log.Infof("target for key %s in %s is the same, skipping", app.Key, cfg.AppName)

			noChange++
		}


		apps = append(apps, app)

	}

	if yamlErr == len(cfg.UpdateApps) {
		return fmt.Errorf("failed during update helm files"), true, nil
	}

	// If the target file already exist in the repository, we will check whether
	// our generated new file is the same as the existing one, and if yes, we
	// don't proceed further for commit.
	if noChange == len(cfg.UpdateApps) {
		log.Debugf("target parameters file and marshaled data for all targets are the same, skipping commit.")

		return nil, true, nil
    }

	err = gitC.Add(targetFile)

	return nil, false, apps
}

func yamlReplaceCMD(app Application, targetFile string) error {
    if !strings.HasPrefix(app.Key, ".") {
		return fmt.Errorf("key %s doesn't start with '.'", app.Key)
	}

	cmd := fmt.Sprintf("yq eval -i '%s=\"%s\"' %s", app.Key, app.Image, targetFile)
	exec := exec.Command("/bin/sh", "-c", cmd)
	exec.Stdout = os.Stdout
	exec.Stderr = os.Stderr

	log.Debugf(exec.String())

	err := exec.Run()
	if err != nil {
		return fmt.Errorf("cmd.Run() failed with %s\n")
	}

	return nil
}

type helmOverride struct {
	Image *Image `json:"image"`
}

type Image struct {
	Tag string `json:"tag"`
}

// marshalParamsOverride marshals the parameter overrides of a given application
// into YAML bytes
func marshalParamsOverride(cfg HelmUpdaterConfig) ([]byte, error) {
	var override []byte
	var err error

	params := helmOverride{
		Image: &Image{
			Tag: cfg.UpdateApps[0].Image,
		},
	}
	override, err = yaml.Marshal(params)
	if err != nil {
		return nil, err
	}

	return override, nil
}

var _ changeWriter = writeOverrides

type changeWriter func(cfg HelmUpdaterConfig, gitC git.Client) (err error, skip bool, apps []Application)

// commitChanges commits any changes required for updating one or more images
// after the UpdateApplication cycle has finished.
func commitChangesGit(cfg HelmUpdaterConfig, write changeWriter) error {
	var apps []Application
	var skip bool

	creds, err := cfg.GitCredentials.NewCreds(cfg.GitConf.RepoURL)
	if err != nil {
		return fmt.Errorf("could not get creds for repo '%s': %v", cfg.AppName, err)
	}
	var gitC git.Client
	tempRoot, err := ioutil.TempDir(os.TempDir(), fmt.Sprintf("git-%s", cfg.AppName))
	if err != nil {
		return err
	}
	defer func() {
		err := os.RemoveAll(tempRoot)
		if err != nil {
			log.Errorf("could not remove temp dir: %v", err)
		}
	}()

	gitC, err = git.NewClientExt(cfg.GitConf.RepoURL, tempRoot, creds, false, false, "")
	if err != nil {
		return err
	}

	err = gitC.Init()
	if err != nil {
		return err
	}
	err = gitC.Fetch("")
	if err != nil {
		return err
	}

	// Set username and e-mail address used to identify the commiter
	if cfg.GitCredentials.Username != "" && cfg.GitCredentials.Email != "" {
		err = gitC.Config(cfg.GitCredentials.Username, cfg.GitCredentials.Email)
		if err != nil {
			return err
		}
	}

	checkOutBranch := cfg.GitConf.Branch

	log.Tracef("targetRevision for update is '%s'", checkOutBranch)
	if checkOutBranch == "" || checkOutBranch == "HEAD" {
		checkOutBranch, err = gitC.SymRefToBranch(checkOutBranch)
		log.Infof("resolved remote default branch to '%s' and using that for operations", checkOutBranch)
		if err != nil {
			return err
		}
	}

	err = gitC.Checkout(checkOutBranch)
	if err != nil {
		return err
	}

	if err, skip, apps = write(cfg, gitC); err != nil {
		return err
	} else if skip {
		return nil
	}

	commitOpts := &git.CommitOptions{}
	if len(apps) > 0 && cfg.GitConf.Message != nil {
		gitCommitMessage = TemplateCommitMessage(cfg.GitConf.Message, cfg.AppName, changeList)
	}

	if gitCommitMessage != "" {
		cm, err := ioutil.TempFile("", "image-updater-commit-msg")
		if err != nil {
			return fmt.Errorf("cold not create temp file: %v", err)
		}
		log.Debugf("Writing commit message to %s", cm.Name())
		err = ioutil.WriteFile(cm.Name(), []byte(gitCommitMessage), 0600)
		if err != nil {
			_ = cm.Close()
			return fmt.Errorf("could not write commit message to %s: %v", cm.Name(), err)
		}
		commitOpts.CommitMessagePath = cm.Name()
		_ = cm.Close()
		defer os.Remove(cm.Name())
	}

	err = gitC.Commit("", commitOpts)
	if err != nil {
		return err
	}
	err = gitC.Push("origin", checkOutBranch, false)
	if err != nil {
		return err
	}

	return nil
}
