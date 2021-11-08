package updater

import (
	"fmt"
	"github.com/argoproj-labs/argocd-image-updater/pkg/log"
)

// Stores some statistics about the results of a run
type Result struct {
	NumApplicationsProcessed int
	NumImagesFound           int
	NumImagesUpdated         int
	NumImagesConsidered      int
	NumSkipped               int
	NumErrors                int
}

// ChangeEntry represents an image that has been changed by Image Updater
type ChangeEntry struct {
	OldTag string
	NewTag string
	File   string
	Key    string
}

func needsUpdate(updateTag string, deployedTag string) bool {
	// If the latest tag does not match image's current tag or the kustomize image is different, it means we have an update candidate.
	return updateTag == deployedTag
}

// UpdateApplication update all images of a single application. Will run in a goroutine.
func UpdateApplication(cfg HelmUpdaterConfig, state *SyncIterationState) Result {
	var needUpdate = false
	var gitCommitMessage string
	result := Result{}

	// check the deployed tag
	presentedValue := "test"
	changeList := make([]ChangeEntry, 0)

	fmt.Println(len(cfg.UpdateApps))

	for _, app := range cfg.UpdateApps {

		if !needsUpdate(app.Image, presentedValue) {

			log.Infof("Setting new image to %s, %s", cfg.AppName, app.Image)
			needUpdate = true

			changeList = append(changeList, ChangeEntry{
				presentedValue,
				app.Image,
				cfg.File,
				app.Key,
			})

		}
	}

	if needUpdate {
		log.Debugf("Using commit message: %s", gitCommitMessage)

		err := commitChangesLocked(cfg, state)
		if err != nil {
			log.Errorf("Could not update application spec: %v", err)
		} else {
			log.Infof("Successfully updated the live application spec")
		}
	}

	return result
}
