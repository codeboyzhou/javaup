package selfupdate

// ProgressStage identifies a visible phase of the self-update process.
type ProgressStage string

const (
	// ProgressChecking reports that the latest release is being queried.
	ProgressChecking ProgressStage = "checking"
	// ProgressAvailable reports the version transition that will be installed.
	ProgressAvailable ProgressStage = "available"
	// ProgressFetchingChecksums reports that the release checksum list is being fetched.
	ProgressFetchingChecksums ProgressStage = "fetching-checksums"
	// ProgressConnecting reports that the release archive request is being established.
	ProgressConnecting ProgressStage = "connecting"
	// ProgressDownloading reports release archive download progress.
	ProgressDownloading ProgressStage = "downloading"
	// ProgressVerifying reports that the release checksum is being verified.
	ProgressVerifying ProgressStage = "verifying"
	// ProgressExtracting reports that the release executable is being extracted.
	ProgressExtracting ProgressStage = "extracting"
	// ProgressInstalling reports that the downloaded release is being installed.
	ProgressInstalling ProgressStage = "installing"
)

// ProgressEvent describes one observable point in the self-update process.
type ProgressEvent struct {
	Stage      ProgressStage
	Current    string
	Latest     string
	Name       string
	URL        string
	Downloaded int64
	Total      int64
}

// ProgressFunc receives self-update progress events.
type ProgressFunc func(ProgressEvent)

func reportProgress(progress ProgressFunc, event ProgressEvent) {
	if progress != nil {
		progress(event)
	}
}
