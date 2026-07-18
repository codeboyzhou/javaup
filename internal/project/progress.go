package project

// ProgressState identifies the lifecycle state of an initialization step.
type ProgressState string

const (
	// ProgressStarted marks the beginning of a step.
	ProgressStarted ProgressState = "started"
	// ProgressInfo adds useful detail to the active step.
	ProgressInfo ProgressState = "info"
	// ProgressSucceeded marks a completed step.
	ProgressSucceeded ProgressState = "succeeded"
	// ProgressFailed marks a failed step.
	ProgressFailed ProgressState = "failed"
)

// ProgressEvent reports one meaningful initialization operation.
type ProgressEvent struct {
	Step    int
	Total   int
	Name    string
	Message string
	State   ProgressState
}

// ProgressFunc receives initialization progress events.
type ProgressFunc func(event ProgressEvent)

func reportProgress(progress ProgressFunc, event ProgressEvent) {
	if progress != nil {
		progress(event)
	}
}
