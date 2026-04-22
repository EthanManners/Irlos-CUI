// SPDX-License-Identifier: GPL-3.0-or-later
package poll

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

// dbusConn is a persistent D-Bus connection. nil means we are using the
// systemctl fallback.
var dbusConn *dbus.Conn

// dbusConnTried avoids repeated connection attempts when D-Bus is absent.
var dbusConnTried bool

func getDBusConn() *dbus.Conn {
	if dbusConnTried {
		return dbusConn
	}
	dbusConnTried = true
	conn, err := dbus.New()
	if err != nil {
		log.Printf("systemd D-Bus unavailable, falling back to systemctl: %v", err)
		return nil
	}
	dbusConn = conn
	return conn
}

// IsActive returns true when the named systemd unit is in the active state.
func IsActive(unit string) bool {
	conn := getDBusConn()
	if conn != nil {
		return isActiveDBus(conn, unit)
	}
	return isActiveSystemctl(unit)
}

func isActiveDBus(conn *dbus.Conn, unit string) bool {
	props, err := conn.GetUnitPropertiesContext(context.Background(), unit)
	if err != nil {
		return false
	}
	state, _ := props["ActiveState"].(string)
	return state == "active"
}

func isActiveSystemctl(unit string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", unit)
	return cmd.Run() == nil
}

// StartUnit starts a systemd unit. It returns immediately; the result is best-effort.
func StartUnit(unit string) error {
	conn := getDBusConn()
	if conn != nil {
		ch := make(chan string, 1)
		_, err := conn.StartUnitContext(context.Background(), unit, "replace", ch)
		return err
	}
	return runSystemctl("start", unit)
}

// StopUnit stops a systemd unit.
func StopUnit(unit string) error {
	conn := getDBusConn()
	if conn != nil {
		ch := make(chan string, 1)
		_, err := conn.StopUnitContext(context.Background(), unit, "replace", ch)
		return err
	}
	return runSystemctl("stop", unit)
}

// RestartUnit restarts a systemd unit.
func RestartUnit(unit string) error {
	conn := getDBusConn()
	if conn != nil {
		ch := make(chan string, 1)
		_, err := conn.RestartUnitContext(context.Background(), unit, "replace", ch)
		return err
	}
	return runSystemctl("restart", unit)
}

func runSystemctl(action, unit string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "systemctl", action, unit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s %s: %w: %s", action, unit, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// StreamUptime returns the active-since duration of irlos-session.service
// formatted as hh:mm:ss, or "--:--:--" on failure.
func StreamUptime(unit string) string {
	conn := getDBusConn()
	if conn != nil {
		return streamUptimeDBus(conn, unit)
	}
	return "--:--:--"
}

func streamUptimeDBus(conn *dbus.Conn, unit string) string {
	props, err := conn.GetUnitPropertiesContext(context.Background(), unit)
	if err != nil {
		return "--:--:--"
	}
	// ActiveEnterTimestamp is microseconds since epoch.
	ts, _ := props["ActiveEnterTimestamp"].(uint64)
	if ts == 0 {
		return "--:--:--"
	}
	startSec := int64(ts / 1_000_000)
	now := time.Now().Unix()
	delta := now - startSec
	if delta < 0 {
		delta = 0
	}
	hh := delta / 3600
	mm := (delta % 3600) / 60
	ss := delta % 60
	return fmt.Sprintf("%02d:%02d:%02d", hh, mm, ss)
}

// GetIP returns the first non-loopback IP address from hostname -I.
func GetIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "hostname", "-I").Output()
	if err != nil {
		return "0.0.0.0"
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "0.0.0.0"
	}
	return fields[0]
}
