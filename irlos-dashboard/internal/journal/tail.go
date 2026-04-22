// SPDX-License-Identifier: GPL-3.0-or-later
// Package journal tails the systemd journal for selected units and sends
// new lines into a channel consumed by the bubbletea Update loop.
package journal

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
)

// Units that the Logs tab follows.
var units = []string{
	"irlos-session.service",
	"sls.service",
}

// Tailer follows the journal and sends each new line to Lines.
// Start it once with go t.Run() and consume t.Lines from the model.
type Tailer struct {
	Lines chan string
	done  chan struct{}
}

// New creates a Tailer. Call Run in a goroutine.
func New() *Tailer {
	return &Tailer{
		Lines: make(chan string, 256),
		done:  make(chan struct{}),
	}
}

// Stop signals the tailer goroutine to exit.
func (t *Tailer) Stop() {
	close(t.done)
}

// Run blocks, tailing the journal until Stop is called.
// Errors are logged to stderr; the function never panics.
func (t *Tailer) Run() {
	if os.Getenv("IRLOS_DEV") == "1" {
		t.runDev()
		return
	}

	j, err := sdjournal.NewJournal()
	if err != nil {
		log.Printf("journal: open failed: %v", err)
		t.Lines <- fmt.Sprintf("[journal unavailable: %v]", err)
		return
	}
	defer j.Close()

	for _, unit := range units {
		if err := j.AddMatch("_SYSTEMD_UNIT=" + unit); err != nil {
			log.Printf("journal: AddMatch(%s): %v", unit, err)
		}
		if err := j.AddDisjunction(); err != nil {
			log.Printf("journal: AddDisjunction: %v", err)
		}
	}

	// Seek to the last 300 entries, then follow.
	if err := j.SeekTail(); err != nil {
		log.Printf("journal: SeekTail: %v", err)
	}
	// Step back 300 lines so we get recent history on startup.
	for i := 0; i < 300; i++ {
		if n, err := j.Previous(); err != nil || n == 0 {
			break
		}
	}

	for {
		select {
		case <-t.done:
			return
		default:
		}

		n, err := j.Next()
		if err != nil {
			log.Printf("journal: Next: %v", err)
			return
		}
		if n == 0 {
			// No new entries — wait.
			status := j.Wait(500 * time.Millisecond)
			if status == sdjournal.SD_JOURNAL_INVALIDATE {
				log.Printf("journal: invalidated, rotating")
			}
			continue
		}

		entry, err := j.GetEntry()
		if err != nil {
			continue
		}

		line := formatEntry(entry)
		select {
		case t.Lines <- line:
		case <-t.done:
			return
		}
	}
}

func formatEntry(e *sdjournal.JournalEntry) string {
	ts := time.Unix(0, int64(e.RealtimeTimestamp)*int64(time.Microsecond))
	hostname := e.Fields["_HOSTNAME"]
	unit := e.Fields["_SYSTEMD_UNIT"]
	msg := e.Fields["MESSAGE"]
	priority := e.Fields["PRIORITY"]

	level := priorityLabel(priority)
	return fmt.Sprintf("%s %s %s[%s]: %s",
		ts.Format("Jan 02 15:04:05"),
		hostname,
		unit,
		level,
		msg,
	)
}

func priorityLabel(p string) string {
	switch p {
	case "0", "1", "2":
		return "CRIT"
	case "3":
		return "ERROR"
	case "4":
		return "WARN"
	case "5":
		return "NOTICE"
	case "6":
		return "INFO"
	case "7":
		return "DEBUG"
	default:
		return "INFO"
	}
}

// runDev generates synthetic log lines for IRLOS_DEV=1 mode.
func (t *Tailer) runDev() {
	fakeLines := []string{
		"Apr 21 10:00:00 irlos-dev irlos-session.service[INFO]: Stream started",
		"Apr 21 10:00:01 irlos-dev sls.service[INFO]: SRT listener ready on :9999",
		"Apr 21 10:00:02 irlos-dev irlos-session.service[INFO]: OBS scene: Live Scene",
		"Apr 21 10:00:03 irlos-dev sls.service[INFO]: Client connected from 10.0.0.5",
		"Apr 21 10:00:05 irlos-dev irlos-session.service[WARN]: Bitrate fluctuation detected",
		"Apr 21 10:00:10 irlos-dev sls.service[INFO]: recv 5120 kbps send 4800 kbps",
		"Apr 21 10:00:15 irlos-dev irlos-session.service[ERROR]: Dropped frames: 3",
	}

	idx := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	// Send historical lines immediately.
	for _, line := range fakeLines {
		select {
		case t.Lines <- line:
		case <-t.done:
			return
		}
	}
	for {
		select {
		case <-ticker.C:
			line := fmt.Sprintf("%s irlos-dev sls.service[INFO]: heartbeat #%d",
				time.Now().Format("Jan 02 15:04:05"), idx)
			idx++
			select {
			case t.Lines <- line:
			case <-t.done:
				return
			}
		case <-t.done:
			return
		}
	}
}

// WaitForLine is not used externally; the model constructs the tea.Cmd
// directly using the Lines channel.  This function is kept for documentation.
// See model.waitForLog().
