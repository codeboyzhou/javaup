package project

import (
	"context"
)

const uninitializationSteps = 2

type configRemover interface {
	Delete(projectRoot string) (path string, removed bool, err error)
}

// Uninitializer removes a project's persisted local configuration.
type Uninitializer struct {
	store configRemover
}

// NewDefaultUninitializer wires the platform-specific project configuration store.
func NewDefaultUninitializer() (*Uninitializer, error) {
	store, err := NewDefaultConfigStore()
	if err != nil {
		return nil, err
	}
	return NewUninitializer(store), nil
}

// NewUninitializer creates an uninitializer from a replaceable configuration store.
func NewUninitializer(store configRemover) *Uninitializer {
	return &Uninitializer{store: store}
}

// Uninitialize removes root's saved project configuration if it exists.
func (u *Uninitializer) Uninitialize(
	ctx context.Context,
	root string,
	progress ProgressFunc,
) (path string, removed bool, err error) {
	reportUninitProgress(progress, 1, projectStepName, ProgressStarted, "Resolving current project directory")
	canonicalRoot, err := canonicalProjectRoot(root)
	if err != nil {
		reportUninitFailure(progress, 1, projectStepName, err)
		return "", false, err
	}
	reportUninitProgress(progress, 1, projectStepName, ProgressSucceeded, canonicalRoot)

	reportUninitProgress(progress, 2, configStepName, ProgressStarted, "Removing local project configuration")
	if err := ctx.Err(); err != nil {
		reportUninitFailure(progress, 2, configStepName, err)
		return "", false, err
	}
	path, removed, err = u.store.Delete(canonicalRoot)
	if err != nil {
		reportUninitFailure(progress, 2, configStepName, err)
		return path, false, err
	}
	message := path
	if !removed {
		message = "No saved configuration found"
	}
	reportUninitProgress(progress, 2, configStepName, ProgressSucceeded, message)
	return path, removed, nil
}

func reportUninitProgress(progress ProgressFunc, step int, name string, state ProgressState, message string) {
	reportProgress(progress, ProgressEvent{
		Step: step, Total: uninitializationSteps, Name: name, Message: message, State: state,
	})
}

func reportUninitFailure(progress ProgressFunc, step int, name string, err error) {
	reportUninitProgress(progress, step, name, ProgressFailed, err.Error())
}
