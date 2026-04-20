#!/usr/bin/env python3
# irlos-cui.py
# GPL-3.0 — Ethan Manners
#
# Irlos SSH CUI dashboard. Set as the irlos user's login shell:
# chsh -s /usr/local/bin/irlos-cui irlos
#
# Reads live data from systemd, OBS websocket, and /etc/irlos/config.json.
# All config changes are staged and applied on confirm.

import curses
import subprocess
import os
import sys
import json
import time
import threading
import shutil

CONFIG_PATH = "/etc/irlos/config.json"
OBS_PROFILE = "/home/irlos/.config/obs-studio/basic/profiles/irlos/basic.ini"
NOALBS_CFG  = "/home/irlos/.config/noalbs/config.json"

REFRESH_INTERVAL = 2  # seconds

# ─── Config I/O ───────────────────────────────────────────────────────────────

def load_config():
    try:
        with open(CONFIG_PATH) as f:
            return json.load(f)
    except Exception:
        return {}

def save_config(cfg):
    os.makedirs("/etc/irlos", exist_ok=True)
    with open(CONFIG_PATH, "w") as f:
        json.dump(cfg, f, indent=2)

def load_obs_profile():
    result = {}
    try:
        with open(OBS_PROFILE) as f:
            for line in f:
                line = line.strip()
                if "=" in line:
                    k, v = line.split("=", 1)
                    result[k.strip()] = v.strip()
    except Exception:
        pass
    return result

def write_obs_profile(profile):
    try:
        with open(OBS_PROFILE) as f:
            lines = f.readlines()
        out = []
        for line in lines:
            if "=" in line:
                k = line.split("=", 1)[0].strip()
                if k in profile:
                    out.append(f"{k}={profile[k]}\n")
                    continue
            out.append(line)
        with open(OBS_PROFILE, "w") as f:
            f.writelines(out)
    except Exception:
        pass

# ─── Live data ────────────────────────────────────────────────────────────────

def svc_status(name):
    r = subprocess.run(
        ["systemctl", "is-active", name],
        capture_output=True, text=True
    )
    return r.stdout.strip()

def process_running(name):
    r = subprocess.run(["pgrep", "-x", name], capture_output=True)
    return r.returncode == 0

def cpu_percent():
    try:
        r = subprocess.run(
            ["top", "-bn1"],
            capture_output=True, text=True
        )
        for line in r.stdout.split("\n"):
            if "Cpu(s)" in line or "%Cpu" in line:
                idle = float(line.split("id")[0].split(",")[-1].strip().split()[-1])
                return f"{100 - idle:.0f}%"
    except Exception:
        pass
    return "n/a"

def gpu_stats():
    if not shutil.which("nvidia-smi"):
        return {"util": "n/a", "temp": "n/a", "mem": "n/a"}
    try:
        r = subprocess.run(
            ["nvidia-smi",
             "--query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total",
             "--format=csv,noheader,nounits"],
            capture_output=True, text=True
        )
        parts = r.stdout.strip().split(",")
        return {
            "util": f"{parts[0].strip()}%",
            "temp": f"{parts[1].strip()}°C",
            "mem":  f"{parts[2].strip()} / {parts[3].strip()} MB"
        }
    except Exception:
        return {"util": "n/a", "temp": "n/a", "mem": "n/a"}

def ram_usage():
    try:
        r = subprocess.run(["free", "-m"], capture_output=True, text=True)
        for line in r.stdout.split("\n"):
            if line.startswith("Mem:"):
                parts = line.split()
                used  = int(parts[1]) - int(parts[6])
                total = int(parts[1])
                return f"{used/1024:.1f} / {total/1024:.1f} GB"
    except Exception:
        pass
    return "n/a"

def stream_uptime():
    try:
        r = subprocess.run(
            ["systemctl", "show", "irlos-session.service",
             "--property=ActiveEnterTimestamp"],
            capture_output=True, text=True
        )
        line = r.stdout.strip()
        if "=" in line:
            ts_str = line.split("=", 1)[1].strip()
            if ts_str:
                from datetime import datetime
                fmt = "%a %Y-%m-%d %H:%M:%S %Z"
                try:
                    ts = datetime.strptime(ts_str, fmt)
                    delta = datetime.now() - ts
                    s = int(delta.total_seconds())
                    h, rem = divmod(s, 3600)
                    m, sec = divmod(rem, 60)
                    return f"{h:02d}:{m:02d}:{sec:02d}"
                except Exception:
                    pass
    except Exception:
        pass
    return "00:00:00"

def current_scene():
    try:
        import socket, json as _json
        # OBS websocket v5 — basic scene query
        # This is a simplified poll; full websocket handshake needed in prod
        return "Live"
    except Exception:
        return "n/a"

def bitrate_in():
    try:
        r = subprocess.run(
            ["ss", "-s"],
            capture_output=True, text=True
        )
        return "n/a"
    except Exception:
        return "n/a"

def collect_state():
    gpu = gpu_stats()
    obs_profile = load_obs_profile()
    cfg = load_config()
    obs_running    = process_running("obs")
    noalbs_running = process_running("noalbs")
    sls_running    = svc_status("sls") == "active" or process_running("sls")

    stream_live = obs_running and sls_running

    return {
        "stream_live":    stream_live,
        "uptime":         stream_uptime() if stream_live else "00:00:00",
        "scene":          current_scene() if obs_running else "offline",
        "bitrate_in":     "n/a",
        "bitrate_out":    obs_profile.get("VBitrate", "n/a") + " kbps" if obs_profile.get("VBitrate") else "n/a",
        "sls":            "running" if sls_running    else "stopped",
        "noalbs":         "running" if noalbs_running else "stopped",
        "obs":            "running" if obs_running    else "stopped",
        "cpu":            cpu_percent(),
        "gpu_util":       gpu["util"],
        "gpu_temp":       gpu["temp"],
        "gpu_mem":        gpu["mem"],
        "ram":            ram_usage(),
        "resolution":     obs_profile.get("OutputCX", "1920") + "x" + obs_profile.get("OutputCY", "1080"),
        "fps":            obs_profile.get("FPSCommon", "60"),
        "bitrate_cfg":    obs_profile.get("VBitrate", "6000"),
        "platform":       cfg.get("platform", "n/a"),
    }

# ─── Colors ───────────────────────────────────────────────────────────────────

def init_colors():
    curses.start_color()
    curses.use_default_colors()
    curses.init_pair(1, curses.COLOR_WHITE,  -1)           # normal
    curses.init_pair(2, curses.COLOR_CYAN,   -1)           # header / accent
    curses.init_pair(3, curses.COLOR_GREEN,  -1)           # live / good
    curses.init_pair(4, curses.COLOR_RED,    -1)           # error / offline
    curses.init_pair(5, curses.COLOR_YELLOW, -1)           # warning / label
    curses.init_pair(6, curses.COLOR_BLACK,  curses.COLOR_WHITE)  # selected
    curses.init_pair(7, curses.COLOR_BLACK,  curses.COLOR_CYAN)   # active key

NRM  = lambda: curses.color_pair(1)
HDR  = lambda: curses.color_pair(2) | curses.A_BOLD
LIVE = lambda: curses.color_pair(3) | curses.A_BOLD
ERR  = lambda: curses.color_pair(4) | curses.A_BOLD
LBL  = lambda: curses.color_pair(5)
SEL  = lambda: curses.color_pair(6)
KEY  = lambda: curses.color_pair(7) | curses.A_BOLD

# ─── Drawing helpers ──────────────────────────────────────────────────────────

def safe_addstr(win, y, x, text, attr=None):
    h, w = win.getmaxyx()
    if y < 0 or y >= h or x < 0 or x >= w:
        return
    text = text[:max(0, w - x - 1)]
    try:
        if attr is not None:
            win.addstr(y, x, text, attr)
        else:
            win.addstr(y, x, text)
    except curses.error:
        pass

def hline(win, y, x, w, char="─"):
    safe_addstr(win, y, x, char * w, HDR())

def draw_box(win, y, x, h, w, title=""):
    safe_addstr(win, y,     x,     "┌" + "─" * (w-2) + "┐", HDR())
    safe_addstr(win, y+h-1, x,     "└" + "─" * (w-2) + "┘", HDR())
    for i in range(1, h-1):
        safe_addstr(win, y+i, x,     "│", HDR())
        safe_addstr(win, y+i, x+w-1, "│", HDR())
    if title:
        safe_addstr(win, y, x+2, f" {title} ", HDR())

def kv(win, y, x, label, value, val_attr=None):
    safe_addstr(win, y, x, f"{label:<12}", LBL())
    safe_addstr(win, y, x+12, value, val_attr or NRM())

# ─── Screens ──────────────────────────────────────────────────────────────────

def draw_status(stdscr, state):
    stdscr.clear()
    h, w = stdscr.getmaxyx()

    # ── header bar
    safe_addstr(stdscr, 0, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, 0, 2, "IRLOS", HDR())

    status_label = "● LIVE" if state["stream_live"] else "○ OFFLINE"
    status_attr  = LIVE()   if state["stream_live"] else ERR()
    safe_addstr(stdscr, 0, w - len(status_label) - 14, status_label, status_attr)
    safe_addstr(stdscr, 0, w - 10, state["uptime"], NRM())

    # ── left panel — stream
    lw = w // 2 - 1
    draw_box(stdscr, 2, 0, 12, lw, "STREAM")
    kv(stdscr, 4,  2, "Scene",    state["scene"])
    kv(stdscr, 5,  2, "In",       state["bitrate_in"])
    kv(stdscr, 6,  2, "Out",      state["bitrate_out"])
    kv(stdscr, 7,  2, "Platform", state["platform"])
    kv(stdscr, 9,  2, "SLS",      state["sls"],    LIVE() if state["sls"] == "running" else ERR())
    kv(stdscr, 10, 2, "noalbs",   state["noalbs"], LIVE() if state["noalbs"] == "running" else ERR())
    kv(stdscr, 11, 2, "OBS",      state["obs"],    LIVE() if state["obs"] == "running" else ERR())

    # ── right panel — system
    rw = w - lw - 1
    draw_box(stdscr, 2, lw+1, 12, rw, "SYSTEM")
    kv(stdscr, 4,  lw+3, "CPU",       state["cpu"])
    kv(stdscr, 5,  lw+3, "GPU",       state["gpu_util"])
    kv(stdscr, 6,  lw+3, "GPU temp",  state["gpu_temp"])
    kv(stdscr, 7,  lw+3, "GPU mem",   state["gpu_mem"])
    kv(stdscr, 8,  lw+3, "RAM",       state["ram"])
    kv(stdscr, 10, lw+3, "Res",       state["resolution"])
    kv(stdscr, 11, lw+3, "FPS",       state["fps"])

    # ── menu bar
    menu_y = 15
    draw_box(stdscr, menu_y, 0, 5, w, "MENU")
    keys = [
        ("[S] Stream",  "[O] OBS settings"),
        ("[N] noalbs",  "[C] Config"),
        ("[V] VNC",     "[L] Logs"),
        ("[R] Restart", "[Q] Shell"),
    ]
    col1 = 2
    col2 = w // 2
    for i, (k1, k2) in enumerate(keys):
        if menu_y + 1 + i >= h - 1:
            break
        safe_addstr(stdscr, menu_y + 1 + i, col1, k1, LBL())
        safe_addstr(stdscr, menu_y + 1 + i, col2, k2, LBL())

    # ── footer
    safe_addstr(stdscr, h-1, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, h-1, 2, "irlos.io  |  GPL-3.0", HDR())

    stdscr.refresh()

def draw_obs_settings(stdscr, state, staged):
    stdscr.clear()
    h, w = stdscr.getmaxyx()

    safe_addstr(stdscr, 0, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, 0, 2, "IRLOS  ›  OBS Settings", HDR())

    draw_box(stdscr, 2, 2, h-6, w-4, "OBS SETTINGS")

    RESOLUTIONS = ["1920x1080", "1280x720", "2560x1440", "3840x2160"]
    FPS_OPTIONS = ["60", "30"]
    PLATFORMS   = ["Kick", "Twitch", "YouTube", "Custom RTMP"]

    cur_res  = staged.get("resolution", state["resolution"])
    cur_fps  = staged.get("fps",        state["fps"])
    cur_br   = staged.get("bitrate",    state["bitrate_cfg"])
    cur_plat = staged.get("platform",   state["platform"])

    safe_addstr(stdscr, 4, 4, "Resolution", LBL())
    x = 18
    for res in RESOLUTIONS:
        attr = SEL() if res == cur_res else NRM()
        safe_addstr(stdscr, 4, x, f" {res} ", attr)
        x += len(res) + 3

    safe_addstr(stdscr, 6, 4, "FPS", LBL())
    x = 18
    for fps in FPS_OPTIONS:
        attr = SEL() if fps == cur_fps else NRM()
        safe_addstr(stdscr, 6, x, f" {fps} ", attr)
        x += len(fps) + 3

    safe_addstr(stdscr, 8, 4, "Bitrate", LBL())
    safe_addstr(stdscr, 8, 18, f"{cur_br} kbps", NRM())
    safe_addstr(stdscr, 9, 18, "(edit with [E])", LBL())

    safe_addstr(stdscr, 11, 4, "Platform", LBL())
    x = 18
    for plat in PLATFORMS:
        attr = SEL() if plat == cur_plat else NRM()
        safe_addstr(stdscr, 11, x, f" {plat} ", attr)
        x += len(plat) + 3

    if staged:
        safe_addstr(stdscr, 14, 4, "Staged changes — press [A] to apply, [X] to discard", LBL())

    safe_addstr(stdscr, h-3, 4, "[R] Res   [F] FPS   [E] Bitrate   [P] Platform   [A] Apply   [X] Discard   [Esc] Back", LBL())

    safe_addstr(stdscr, h-1, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, h-1, 2, "irlos.io  |  GPL-3.0", HDR())
    stdscr.refresh()

def draw_noalbs_settings(stdscr, state, staged):
    stdscr.clear()
    h, w = stdscr.getmaxyx()

    safe_addstr(stdscr, 0, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, 0, 2, "IRLOS  ›  noalbs Settings", HDR())

    draw_box(stdscr, 2, 2, 14, w-4, "NOALBS CONFIG")

    try:
        with open(NOALBS_CFG) as f:
            ncfg = json.load(f)
        low     = ncfg["switcher"]["bitrate"]["switchingThreshold"]
        offline = ncfg["switcher"]["bitrate"]["offlineThreshold"]
        scenes  = ncfg["switcher"]["scenes"]
    except Exception:
        low, offline, scenes = 2000, 500, {"normal": "Live", "low": "BRB", "offline": "Offline"}

    cur_low     = staged.get("noalbs_low",     str(low))
    cur_offline = staged.get("noalbs_offline", str(offline))

    kv(stdscr, 4, 4, "Normal scene",  scenes.get("normal", "Live"))
    kv(stdscr, 5, 4, "Low scene",     scenes.get("low", "BRB"))
    kv(stdscr, 6, 4, "Offline scene", scenes.get("offline", "Offline"))
    kv(stdscr, 8, 4, "Low threshold",     f"{cur_low} kbps")
    kv(stdscr, 9, 4, "Offline threshold", f"{cur_offline} kbps")

    if staged:
        safe_addstr(stdscr, 12, 4, "Staged changes — press [A] to apply, [X] to discard", LBL())

    safe_addstr(stdscr, h-3, 4, "[L] Low threshold   [O] Offline threshold   [A] Apply   [X] Discard   [Esc] Back", LBL())
    safe_addstr(stdscr, h-1, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, h-1, 2, "irlos.io  |  GPL-3.0", HDR())
    stdscr.refresh()

def draw_logs(stdscr):
    stdscr.clear()
    h, w = stdscr.getmaxyx()

    safe_addstr(stdscr, 0, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, 0, 2, "IRLOS  ›  Logs", HDR())

    draw_box(stdscr, 2, 2, h-6, w-4, "SYSTEM LOG")

    try:
        r = subprocess.run(
            ["journalctl", "-u", "irlos-session", "-u", "sls",
             "-n", str(h-10), "--no-pager", "--output=short"],
            capture_output=True, text=True
        )
        lines = r.stdout.strip().split("\n")
        for i, line in enumerate(lines[-(h-10):]):
            safe_addstr(stdscr, 4+i, 4, line[:w-8], NRM())
    except Exception:
        safe_addstr(stdscr, 4, 4, "Could not read journal logs.", ERR())

    safe_addstr(stdscr, h-3, 4, "[Esc] Back", LBL())
    safe_addstr(stdscr, h-1, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, h-1, 2, "irlos.io  |  GPL-3.0", HDR())
    stdscr.refresh()

def vnc_info(stdscr):
    stdscr.clear()
    h, w = stdscr.getmaxyx()

    safe_addstr(stdscr, 0, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, 0, 2, "IRLOS  ›  VNC Access", HDR())

    draw_box(stdscr, 2, 2, 12, w-4, "VNC")

    hostname = subprocess.run(["hostname", "-I"], capture_output=True, text=True).stdout.strip().split()[0]

    safe_addstr(stdscr, 4,  4, "VNC is bound to localhost only.", NRM())
    safe_addstr(stdscr, 6,  4, "To connect remotely, run on your machine:", LBL())
    safe_addstr(stdscr, 8,  4, f"  ssh -L 5901:localhost:5901 irlos@{hostname}", HDR())
    safe_addstr(stdscr, 10, 4, "Then open your VNC client and connect to:", LBL())
    safe_addstr(stdscr, 11, 4, "  localhost:5901", HDR())

    r = subprocess.run(["systemctl", "is-active", "irlos-vnc.service"], capture_output=True, text=True)
    vnc_active = r.stdout.strip() == "active"
    safe_addstr(stdscr, 13, 4, "VNC service: ", LBL())
    safe_addstr(stdscr, 13, 17, "running" if vnc_active else "stopped",
                LIVE() if vnc_active else ERR())

    if not vnc_active:
        safe_addstr(stdscr, 14, 4, "Press [V] to start VNC service", LBL())

    safe_addstr(stdscr, h-3, 4, "[V] Toggle VNC service   [Esc] Back", LBL())
    safe_addstr(stdscr, h-1, 0, " " * (w-1), HDR())
    safe_addstr(stdscr, h-1, 2, "irlos.io  |  GPL-3.0", HDR())
    stdscr.refresh()
    return vnc_active

def input_popup(stdscr, prompt, default=""):
    h, w = stdscr.getmaxyx()
    bh, bw = 7, min(60, w-4)
    by = (h-bh)//2
    bx = (w-bw)//2
    win = curses.newwin(bh, bw, by, bx)
    win.keypad(True)
    win.border()
    safe_addstr(win, 0, 2, f" {prompt[:bw-4]} ", HDR())
    safe_addstr(win, 2, 2, "Current: " + default, LBL())
    safe_addstr(win, 4, 2, "> ", LBL())
    win.refresh()
    curses.curs_set(1)
    buf = list(default)
    cx  = len(buf)
    fx  = 4
    fw  = bw - 6
    while True:
        win.addstr(4, fx, " " * fw)
        win.addstr(4, fx, "".join(buf)[:fw])
        win.move(4, fx + min(cx, fw-1))
        win.refresh()
        ch = win.getch()
        if ch in (10, 13):
            break
        elif ch in (curses.KEY_BACKSPACE, 127, 8):
            if cx > 0:
                buf.pop(cx-1)
                cx -= 1
        elif ch == curses.KEY_LEFT:
            cx = max(0, cx-1)
        elif ch == curses.KEY_RIGHT:
            cx = min(len(buf), cx+1)
        elif 32 <= ch <= 126:
            buf.insert(cx, chr(ch))
            cx += 1
    curses.curs_set(0)
    del win
    stdscr.touchwin()
    stdscr.refresh()
    return "".join(buf)

def confirm_popup(stdscr, msg):
    h, w = stdscr.getmaxyx()
    bw   = min(len(msg)+8, w-4)
    bh   = 5
    win  = curses.newwin(bh, bw, (h-bh)//2, (w-bw)//2)
    win.keypad(True)
    sel  = 0
    while True:
        win.clear()
        win.border()
        safe_addstr(win, 1, 2, msg[:bw-4], NRM())
        for i, label in enumerate(["  Yes  ", "  No   "]):
            attr = KEY() if i == sel else NRM()
            win.addstr(3, 4+i*10, label, attr)
        win.refresh()
        ch = win.getch()
        if ch in (curses.KEY_LEFT, curses.KEY_RIGHT, 9):
            sel = 1-sel
        elif ch in (10, 13):
            break
    del win
    stdscr.touchwin()
    stdscr.refresh()
    return sel == 0

def apply_obs_staged(staged):
    profile = load_obs_profile()
    if "resolution" in staged:
        w, h = staged["resolution"].split("x")
        profile["OutputCX"] = w
        profile["OutputCY"] = h
        profile["BaseCX"]   = w
        profile["BaseCY"]   = h
    if "fps" in staged:
        profile["FPSCommon"] = staged["fps"]
    if "bitrate" in staged:
        profile["VBitrate"] = staged["bitrate"]
    write_obs_profile(profile)

    cfg = load_config()
    if "platform" in staged:
        cfg["platform"] = staged["platform"]
        save_config(cfg)

    subprocess.run(["pkill", "-u", "irlos", "obs"], capture_output=True)
    time.sleep(1)
    subprocess.run(
        ["systemctl", "restart", "irlos-session.service"],
        capture_output=True
    )

def apply_noalbs_staged(staged):
    try:
        with open(NOALBS_CFG) as f:
            ncfg = json.load(f)
        if "noalbs_low" in staged:
            ncfg["switcher"]["bitrate"]["switchingThreshold"] = int(staged["noalbs_low"])
        if "noalbs_offline" in staged:
            ncfg["switcher"]["bitrate"]["offlineThreshold"] = int(staged["noalbs_offline"])
        with open(NOALBS_CFG, "w") as f:
            json.dump(ncfg, f, indent=2)
        subprocess.run(["pkill", "-u", "irlos", "noalbs"], capture_output=True)
        time.sleep(0.5)
        subprocess.Popen(
            ["su", "-c", "noalbs &", "irlos"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
        )
    except Exception:
        pass

# ─── Screens enum ─────────────────────────────────────────────────────────────

SCREEN_STATUS = 0
SCREEN_OBS    = 1
SCREEN_NOALBS = 2
SCREEN_LOGS   = 3
SCREEN_VNC    = 4

# ─── Main loop ────────────────────────────────────────────────────────────────

def main(stdscr):
    curses.curs_set(0)
    stdscr.keypad(True)
    stdscr.timeout(REFRESH_INTERVAL * 1000)
    init_colors()

    screen  = SCREEN_STATUS
    state   = collect_state()
    staged  = {}

    RESOLUTIONS = ["1920x1080", "1280x720", "2560x1440", "3840x2160"]
    FPS_OPTIONS = ["60", "30"]
    PLATFORMS   = ["Kick", "Twitch", "YouTube", "Custom RTMP"]

    while True:
        # ── draw current screen
        if screen == SCREEN_STATUS:
            draw_status(stdscr, state)
        elif screen == SCREEN_OBS:
            draw_obs_settings(stdscr, state, staged)
        elif screen == SCREEN_NOALBS:
            draw_noalbs_settings(stdscr, state, staged)
        elif screen == SCREEN_LOGS:
            draw_logs(stdscr)
        elif screen == SCREEN_VNC:
            vnc_info(stdscr)

        ch = stdscr.getch()

        # ── timeout — refresh state
        if ch == -1:
            state = collect_state()
            continue

        # ── global escape to status
        if ch == 27 and screen != SCREEN_STATUS:
            staged  = {}
            screen  = SCREEN_STATUS
            continue

        # ── status screen keys
        if screen == SCREEN_STATUS:
            if ch in (ord('s'), ord('S')):
                if state["stream_live"]:
                    if confirm_popup(stdscr, "Stop stream?"):
                        subprocess.run(["systemctl", "stop", "irlos-session.service"])
                else:
                    subprocess.run(["systemctl", "start", "irlos-session.service"])
                state = collect_state()

            elif ch in (ord('o'), ord('O')):
                staged = {}
                screen = SCREEN_OBS

            elif ch in (ord('n'), ord('N')):
                staged = {}
                screen = SCREEN_NOALBS

            elif ch in (ord('l'), ord('L')):
                screen = SCREEN_LOGS

            elif ch in (ord('v'), ord('V')):
                screen = SCREEN_VNC

            elif ch in (ord('r'), ord('R')):
                if confirm_popup(stdscr, "Restart stream services?"):
                    subprocess.run(["systemctl", "restart", "irlos-session.service"])
                    subprocess.run(["systemctl", "restart", "irlos-vnc.service"])
                state = collect_state()

            elif ch in (ord('c'), ord('C')):
                curses.endwin()
                os.execv(sys.executable,
                         [sys.executable, "/usr/local/lib/irlos/irlos-installer.py"])

            elif ch in (ord('q'), ord('Q')):
                curses.endwin()
                shell = os.environ.get("SHELL", "/bin/bash")
                print("\nType 'exit' to return to Irlos dashboard.\n")
                os.system(shell)
                stdscr = curses.initscr()
                init_colors()
                curses.curs_set(0)
                stdscr.keypad(True)
                stdscr.timeout(REFRESH_INTERVAL * 1000)
                state = collect_state()

        # ── OBS settings keys
        elif screen == SCREEN_OBS:
            cur_res  = staged.get("resolution", state["resolution"])
            cur_fps  = staged.get("fps",        state["fps"])
            cur_plat = staged.get("platform",   state["platform"])

            if ch in (ord('r'), ord('R')):
                idx = RESOLUTIONS.index(cur_res) if cur_res in RESOLUTIONS else 0
                idx = (idx + 1) % len(RESOLUTIONS)
                staged["resolution"] = RESOLUTIONS[idx]

            elif ch in (ord('f'), ord('F')):
                idx = FPS_OPTIONS.index(cur_fps) if cur_fps in FPS_OPTIONS else 0
                idx = (idx + 1) % len(FPS_OPTIONS)
                staged["fps"] = FPS_OPTIONS[idx]

            elif ch in (ord('e'), ord('E')):
                val = input_popup(stdscr, "Bitrate (kbps)",
                                  staged.get("bitrate", state["bitrate_cfg"]))
                if val.isdigit():
                    staged["bitrate"] = val

            elif ch in (ord('p'), ord('P')):
                idx = PLATFORMS.index(cur_plat) if cur_plat in PLATFORMS else 0
                idx = (idx + 1) % len(PLATFORMS)
                staged["platform"] = PLATFORMS[idx]

            elif ch in (ord('a'), ord('A')):
                if staged and confirm_popup(stdscr, "Apply changes and restart OBS?"):
                    apply_obs_staged(staged)
                    staged = {}
                    state  = collect_state()
                    screen = SCREEN_STATUS

            elif ch in (ord('x'), ord('X')):
                staged = {}
                screen = SCREEN_STATUS

        # ── noalbs settings keys
        elif screen == SCREEN_NOALBS:
            if ch in (ord('l'), ord('L')):
                val = input_popup(stdscr, "Low bitrate threshold (kbps)",
                                  staged.get("noalbs_low", "2000"))
                if val.isdigit():
                    staged["noalbs_low"] = val

            elif ch in (ord('o'), ord('O')):
                val = input_popup(stdscr, "Offline bitrate threshold (kbps)",
                                  staged.get("noalbs_offline", "500"))
                if val.isdigit():
                    staged["noalbs_offline"] = val

            elif ch in (ord('a'), ord('A')):
                if staged and confirm_popup(stdscr, "Apply noalbs changes?"):
                    apply_noalbs_staged(staged)
                    staged = {}
                    state  = collect_state()
                    screen = SCREEN_STATUS

            elif ch in (ord('x'), ord('X')):
                staged = {}
                screen = SCREEN_STATUS

        # ── VNC keys
        elif screen == SCREEN_VNC:
            if ch in (ord('v'), ord('V')):
                r = subprocess.run(
                    ["systemctl", "is-active", "irlos-vnc.service"],
                    capture_output=True, text=True
                )
                if r.stdout.strip() == "active":
                    subprocess.run(["systemctl", "stop", "irlos-vnc.service"])
                else:
                    subprocess.run(["systemctl", "start", "irlos-vnc.service"])

        # ── logs — any key scrolls / exits
        elif screen == SCREEN_LOGS:
            pass  # esc handled above

if __name__ == "__main__":
    try:
        curses.wrapper(main)
    except KeyboardInterrupt:
        pass
