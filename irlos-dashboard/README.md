# irlosd — Irlos Dashboard

A single-binary SSH-accessible TUI control room for the [Irlos](https://github.com/ethanmanners/irlos) IRL streaming OS.  
Written in Go using [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Features

- **Stream tab** — start/stop stream, live uptime counter, SRT bitrate, OBS scene, shell escape
- **OBS tab** — configure resolution, FPS, bitrate, platform; apply restarts the session
- **noalbs tab** — adjust low/offline bitrate thresholds for automatic scene switching
- **VNC tab** — start/stop x11vnc, SSH tunnel command, noVNC URL
- **Logs tab** — live journal tail from `irlos-session.service` and `sls.service`; auto-scroll, pause, resume
- **Config tab** — edit stream key (masked), platform, WiFi SSID, SSH pubkey, hostname

### Improvements over the Python prototype

| Area | Python (prototype) | Go (this) |
|------|-------------------|-----------|
| OBS scene | shells out to `obs-cli` | direct obs-websocket via `goobs` |
| Systemd control | shells out to `systemctl` | D-Bus via `go-systemd` |
| GPU metrics | shells out to `nvidia-smi` | NVML via `go-nvml` |
| Journal | `subprocess.Popen(journalctl -f)` | `sdjournal` follow iterator |
| Config writes | direct file.write | temp-file + fsync + rename |
| Log race condition | background thread mutating shared slice | channel → bubbletea `LogLineMsg` |

## Requirements

- Go 1.22+
- Linux (systemd)
- NVIDIA GPU + driver for GPU metrics (graceful N/A if absent)
- OBS 28+ with obs-websocket enabled for scene reading (graceful N/A if absent)

## Build

```sh
git clone https://github.com/ethanmanners/irlos-dashboard
cd irlos-dashboard
go mod tidy         # downloads all dependencies
make build          # produces ./bin/irlosd
```

### Install to system path

```sh
make install        # installs to /usr/local/bin/irlosd (requires sudo)
```

## Usage

```sh
irlosd              # run directly
```

To use as the login shell for the `irlos` user:

```sh
sudo chsh -s /usr/local/bin/irlosd irlos
```

### Keybindings

| Key | Action |
|-----|--------|
| `←` / `→` | Switch tabs |
| `↑` / `↓` | Navigate rows (Logs tab: scroll) |
| `Enter` | Open popup / activate action |
| `Esc` | Close popup |
| `End` | Resume log auto-scroll |
| `q` / `Q` | Quit |

## Development (no real Irlos system)

Set `IRLOS_DEV=1` to stub out all system calls:

```sh
IRLOS_DEV=1 ./bin/irlosd
```

In dev mode:
- CPU, RAM, GPU return synthetic values
- All services show predetermined states
- Journal streams synthetic heartbeat lines every 2 seconds
- Config files are not read or written

## Testing

```sh
make test           # go test ./...
```

The `internal/config/` package has roundtrip and atomicity tests for all three
config files. They use `t.TempDir()` and require no real system files.

## File paths (production)

| File | Purpose |
|------|---------|
| `/etc/irlos/config.json` | Stream key, platform, WiFi, SSH pubkey, hostname |
| `/home/irlos/.config/obs-studio/basic/profiles/irlos/basic.ini` | OBS profile |
| `/home/irlos/.config/noalbs/config.json` | noalbs bitrate thresholds |

## Known limitations / future work

- OBS WebSocket password is currently hardcoded to empty. If you set a password in OBS, update `poll/sls.go`.  
  See [issue #1](https://github.com/ethanmanners/irlos-dashboard/issues/1).
- VNC tab only controls `x11vnc.service`. TigerVNC support is planned.  
  See [issue #2](https://github.com/ethanmanners/irlos-dashboard/issues/2).
- WiFi SSID change does not automatically reconnect; it only writes the config for the next boot.  
  See [issue #3](https://github.com/ethanmanners/irlos-dashboard/issues/3).

## License

GPL-3.0-or-later. See the SPDX headers in each source file.
