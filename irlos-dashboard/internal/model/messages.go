// SPDX-License-Identifier: GPL-3.0-or-later
package model

import "time"

// TickMsg is emitted every 2 seconds to drive polling.
type TickMsg time.Time

// PollDoneMsg carries the results of a completed system poll.
type PollDoneMsg struct {
	State PolledState
	Err   error // non-nil means a poll partially failed; State still populated
}

// LogLineMsg carries a single new journal line.
type LogLineMsg struct{ Line string }

// ServiceCmdMsg reports the result of a systemd Start/Stop request.
type ServiceCmdMsg struct {
	Unit string
	Err  error
}

// ConfigWriteMsg reports the result of a config write operation.
type ConfigWriteMsg struct {
	Tab Tab
	Err error
}

// ShellReturnMsg is sent after the user exits the shell escape.
type ShellReturnMsg struct{ Err error }
