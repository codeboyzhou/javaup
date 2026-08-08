package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/codeboyzhou/javaup/internal/selfupdate"
	"github.com/fatih/color"
	"golang.org/x/term"
)

const defaultUpdateProgressWidth = 80

type updateProgressRenderer struct {
	writer          io.Writer
	interactive     bool
	accent          *color.Color
	success         *color.Color
	downloading     bool
	printedDownload bool
	terminalWidth   func() int
	err             error
}

func newUpdateProgressRenderer(writer io.Writer) *updateProgressRenderer {
	interactive := false
	terminalWidth := func() int { return defaultUpdateProgressWidth }
	if output, ok := writer.(*os.File); ok {
		interactive = isTerminalDescriptor(output.Fd())
		if interactive {
			descriptor := int(output.Fd())
			terminalWidth = func() int {
				width, _, err := term.GetSize(descriptor)
				if err != nil || width <= 0 {
					return defaultUpdateProgressWidth
				}
				return width
			}
		}
	}
	return &updateProgressRenderer{
		writer:        writer,
		interactive:   interactive,
		accent:        newOutputStyle(writer, color.FgCyan),
		success:       newOutputStyle(writer, color.FgGreen),
		terminalWidth: terminalWidth,
	}
}

func (r *updateProgressRenderer) Report(event selfupdate.ProgressEvent) {
	if r.err != nil {
		return
	}
	if event.Stage != selfupdate.ProgressDownloading {
		r.endDownloadLine()
	}

	switch event.Stage {
	case selfupdate.ProgressChecking:
		r.writeLine("Checking for updates...")
	case selfupdate.ProgressAvailable:
		r.writeLine(fmt.Sprintf("Updating jup (%s -> %s)", event.Current, event.Latest))
	case selfupdate.ProgressFetchingChecksums:
		r.writeLine("Fetching checksums from " + event.URL)
	case selfupdate.ProgressConnecting:
		r.writeLine("Downloading new version from " + event.URL)
	case selfupdate.ProgressDownloading:
		r.reportDownload(event)
	case selfupdate.ProgressVerifying:
		r.writeLine("Checking hash... " + r.success.Sprint("ok"))
	case selfupdate.ProgressExtracting:
		r.writeLine("Extracting archive...")
	case selfupdate.ProgressInstalling:
		r.writeLine("Installing update...")
	}
}

func (r *updateProgressRenderer) reportDownload(event selfupdate.ProgressEvent) {
	r.downloading = true
	if !r.interactive {
		if !r.printedDownload {
			size := ""
			if event.Total > 0 {
				size = " (" + formatByteSize(event.Total) + ")"
			}
			r.writeLine(r.accent.Sprint(event.Name) + size + "...")
			r.printedDownload = true
		}
		return
	}

	line := formatDownloadProgress(event.Name, event.Downloaded, event.Total, r.terminalWidth())
	line = strings.Replace(line, event.Name, r.accent.Sprint(event.Name), 1)
	_, r.err = fmt.Fprintf(r.writer, "\r%s", line)
}

func (r *updateProgressRenderer) endDownloadLine() {
	if !r.downloading {
		return
	}
	if r.interactive && r.err == nil {
		_, r.err = fmt.Fprintln(r.writer)
	}
	r.downloading = false
}

func (r *updateProgressRenderer) writeLine(line string) {
	if r.err == nil {
		_, r.err = fmt.Fprintln(r.writer, line)
	}
}

func (r *updateProgressRenderer) Finish() error {
	r.endDownloadLine()
	return r.err
}

func (r *updateProgressRenderer) Success(message string) string {
	return r.success.Sprint(message)
}

func formatDownloadProgress(name string, downloaded, total int64, terminalWidth int) string {
	if total <= 0 {
		return fmt.Sprintf("%s (%s downloaded)", name, formatByteSize(downloaded))
	}
	percentage := min(downloaded*100/total, int64(100))
	prefix := fmt.Sprintf("%s (%s) [", name, formatByteSize(total))
	suffix := fmt.Sprintf("] %3d%%", percentage)
	barWidth := max(terminalWidth-len(prefix)-len(suffix), 1)
	completed := int(percentage) * barWidth / 100
	bar := strings.Repeat("=", completed)
	if completed < barWidth {
		bar += ">" + strings.Repeat(" ", barWidth-completed-1)
	}
	return prefix + bar + suffix
}

func formatByteSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	units := []string{"KB", "MB", "GB", "TB"}
	for _, unit := range units {
		value /= 1024
		if value < 1024 || unit == units[len(units)-1] {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%d B", bytes)
}
