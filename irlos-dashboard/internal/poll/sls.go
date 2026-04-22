// SPDX-License-Identifier: GPL-3.0-or-later
package poll

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/gorilla/websocket"
)

// SLSStats queries the Simple Lens Server stats endpoint.
// Returns (recvKbps, sendKbps) as display strings, or "—" on failure.
func SLSStats() (recv, send string) {
	if DevMode {
		return "5120", "4800"
	}
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://localhost:8181/stats")
	if err != nil {
		return "—", "—"
	}
	defer resp.Body.Close()

	var data struct {
		Streams map[string]struct {
			RecvBitrateKbps float64 `json:"recv_bitrate_kbps"`
			SendBitrateKbps float64 `json:"send_bitrate_kbps"`
		} `json:"streams"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "—", "—"
	}
	for _, v := range data.Streams {
		return fmt.Sprintf("%.0f", v.RecvBitrateKbps),
			fmt.Sprintf("%.0f", v.SendBitrateKbps)
	}
	return "—", "—"
}

// OBSScene returns the current OBS program scene name via obs-websocket.
// Returns "n/a" if OBS is not reachable.
func OBSScene() string {
	if DevMode {
		return "Live Scene"
	}

	// Use a short dial timeout via a custom dialer.
	dialer := websocket.Dialer{
		HandshakeTimeout: 500 * time.Millisecond,
	}

	client, err := goobs.New(
		"localhost:4455",
		goobs.WithPassword(""),
		goobs.WithDialer(&dialer),
	)
	if err != nil {
		return "n/a"
	}
	defer client.Disconnect()

	resp, err := client.Scenes.GetCurrentProgramScene()
	if err != nil {
		return "n/a"
	}
	return resp.CurrentProgramSceneName
}
