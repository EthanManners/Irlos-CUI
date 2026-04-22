// SPDX-License-Identifier: GPL-3.0-or-later
// Package poll provides background system-state polling for the dashboard.
// Each function is safe to call from a tea.Cmd goroutine; none of them import
// the model package.  The model assembles individual results into PolledState.
package poll

import "os"

// DevMode reports whether the process was started with IRLOS_DEV=1.
// When true, callers should return stub values instead of real system data.
var DevMode = os.Getenv("IRLOS_DEV") == "1"
