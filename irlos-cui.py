#!/usr/bin/env python3
"""
Irlos CUI Dashboard
SSH control room for an IRL streaming server.
"""

import curses
import os
import sys
import json
import time
import threading
import subprocess
import configparser
import urllib.request
from pathlib import Path

# ── File paths ────────────────────────────────────────────────────────────────
IRLOS_CONFIG  = "/etc/irlos/config.json"
OBS_INI       = "/home/irlos/.config/obs-studio/basic/profiles/irlos/basic.ini"
NOALBS_CFG    = "/home/irlos/.config/noalbs/config.json"

# ── Color pair indices ────────────────────────────────────────────────────────
CP_DEFAULT   =  1   # white / black
CP_CYAN      =  2   # cyan  / black
CP_GREEN     =  3   # green / black
CP_RED       =  4   # red   / black
CP_YELLOW    =  5   # yellow / black
CP_TAB_ON    =  6   # black / cyan  (active tab)
CP_TAB_OFF   =  7   # white / black (inactive tab)
CP_SEL       =  8   # black / white (selected row)
CP_STAGED    =  9   # yellow / black
CP_BORDER    = 10   # cyan  / black (box borders)
CP_BAR_OK    = 11   # green  / green  (fill ≤70%)
CP_BAR_WARN  = 12   # yellow / yellow (fill ≤90%)
CP_BAR_CRIT  = 13   # red    / red    (fill >90%)
CP_BAR_EMPTY = 14   # black  / black  (empty track)
CP_DIM       = 15   # dark-gray / black

TABS = ["Stream", "OBS", "noalbs", "VNC", "Logs", "Config"]

# ── ASCII wordmark ─────────────────────────────────────────────────────────────
WORDMARK = [
    " ██╗██████╗ ██╗      ██████╗ ███████╗",
    " ██║██╔══██╗██║     ██╔═══██╗██╔════╝",
    " ██║██████╔╝██║     ██║   ██║███████╗",
    " ██║██╔══██╗██║     ██║   ██║╚════██║",
    " ██║██║  ██║███████╗╚██████╔╝███████║",
    " ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚══════╝",
]
WM_W = max(len(l) for l in WORDMARK)

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────

def sh(cmd, default="", timeout=2):
    try:
        r = subprocess.run(
            cmd, shell=isinstance(cmd, str),
            capture_output=True, text=True, timeout=timeout
        )
        return r.stdout.strip()
    except Exception:
        return default

def svc_active(name):
    return sh(f"systemctl is-active --quiet {name} && echo yes", "") == "yes"

def get_ip():
    out = sh("hostname -I", "")
    return out.split()[0] if out.split() else "0.0.0.0"

_cpu_prev: dict = {}

def read_cpu() -> float:
    try:
        with open("/proc/stat") as f:
            toks = f.readline().split()
        vals  = list(map(int, toks[1:]))
        idle  = vals[3]
        total = sum(vals)
        prev  = _cpu_prev.get("v")
        _cpu_prev["v"] = (idle, total)
        if prev is None:
            return 0.0
        di = idle  - prev[0]
        dt = total - prev[1]
        return 0.0 if dt == 0 else 100.0 * (1.0 - di / dt)
    except Exception:
        return 0.0

def read_ram():
    try:
        info = {}
        with open("/proc/meminfo") as f:
            for line in f:
                k, v = line.split(":", 1)
                info[k.strip()] = int(v.split()[0])
        total = info["MemTotal"]
        avail = info.get("MemAvailable", info.get("MemFree", 0))
        used  = total - avail
        return used // 1024, total // 1024
    except Exception:
        return 0, 1

def read_gpu():
    out = sh(
        "nvidia-smi --query-gpu=utilization.gpu,temperature.gpu"
        " --format=csv,noheader,nounits", "0,0"
    )
    try:
        parts = out.split(",")
        return int(parts[0].strip()), int(parts[1].strip())
    except Exception:
        return 0, 0

def read_irlos_cfg():
    try:
        with open(IRLOS_CONFIG) as f:
            return json.load(f)
    except Exception:
        return {}

def write_irlos_cfg(cfg):
    os.makedirs(os.path.dirname(IRLOS_CONFIG), exist_ok=True)
    with open(IRLOS_CONFIG, "w") as f:
        json.dump(cfg, f, indent=2)

def read_obs_ini():
    cfg = {"resolution": "1920x1080", "fps": "60", "bitrate": "6000"}
    try:
        p = configparser.RawConfigParser(strict=False)
        p.read(OBS_INI)
        w = h = None
        for sec in p.sections():
            for k, v in p.items(sec):
                lk = k.lower()
                if lk == "basecx": w = v
                elif lk == "basecy": h = v
                elif lk == "fpsnumerator": cfg["fps"] = v
                elif lk in ("vbitrate", "bitrate"): cfg["bitrate"] = v
        if w and h:
            cfg["resolution"] = f"{w}x{h}"
    except Exception:
        pass
    return cfg

def write_obs_ini(obs):
    try:
        p = configparser.RawConfigParser(strict=False)
        p.read(OBS_INI)
        res = obs.get("resolution", "1920x1080")
        w, h = res.split("x")
        fps  = obs.get("fps", "60")
        bps  = obs.get("bitrate", "6000")
        for sec in ["Video", "SimpleOutput"]:
            if not p.has_section(sec):
                p.add_section(sec)
        p.set("Video", "BaseCX", w)
        p.set("Video", "BaseCY", h)
        p.set("Video", "OutputCX", w)
        p.set("Video", "OutputCY", h)
        p.set("Video", "FPSNumerator", fps)
        p.set("SimpleOutput", "VBitrate", bps)
        os.makedirs(os.path.dirname(OBS_INI), exist_ok=True)
        with open(OBS_INI, "w") as f:
            p.write(f)
    except Exception:
        pass

def read_noalbs():
    cfg = {"low": "2500", "offline": "500"}
    try:
        with open(NOALBS_CFG) as f:
            data = json.load(f)
        thr = data.get("switcher", {}).get("thresholds", {})
        cfg["low"]     = str(thr.get("low", 2500))
        cfg["offline"] = str(thr.get("offline", 500))
    except Exception:
        pass
    return cfg

def write_noalbs(cfg):
    try:
        try:
            with open(NOALBS_CFG) as f:
                data = json.load(f)
        except Exception:
            data = {}
        data.setdefault("switcher", {}).setdefault("thresholds", {})
        data["switcher"]["thresholds"]["low"]     = int(cfg.get("low", 2500))
        data["switcher"]["thresholds"]["offline"] = int(cfg.get("offline", 500))
        with open(NOALBS_CFG, "w") as f:
            json.dump(data, f, indent=2)
    except Exception:
        pass

def sls_stats():
    try:
        with urllib.request.urlopen("http://localhost:8181/stats", timeout=1) as r:
            data = json.loads(r.read())
        for k, v in data.get("streams", {}).items():
            return (str(v.get("recv_bitrate_kbps", "n/a")),
                    str(v.get("send_bitrate_kbps", "n/a")))
    except Exception:
        pass
    return "n/a", "n/a"

def stream_uptime():
    raw = sh(
        "systemctl show irlos-session.service "
        "--property=ActiveEnterTimestamp --value", ""
    )
    if raw and raw != "n/a":
        try:
            from datetime import datetime, timezone
            parts = raw.split()
            if len(parts) >= 3:
                dt = datetime.strptime(f"{parts[1]} {parts[2]}", "%Y-%m-%d %H:%M:%S")
                dt = dt.replace(tzinfo=timezone.utc)
                delta = int(
                    (datetime.now(timezone.utc) - dt).total_seconds()
                )
                hh, rem = divmod(delta, 3600)
                mm, ss  = divmod(rem, 60)
                return f"{hh:02d}:{mm:02d}:{ss:02d}"
        except Exception:
            pass
    return "--:--:--"

def obs_scene():
    out = sh("obs-cli scene current 2>/dev/null", "")
    return out if out else "n/a"

# ─────────────────────────────────────────────────────────────────────────────
# Main Application
# ─────────────────────────────────────────────────────────────────────────────

class Dashboard:

    def __init__(self, scr):
        self.scr       = scr
        self.active    = 0          # active tab index
        self.cur       = 0          # cursor row within tab
        self.popup     = None       # active popup dict or None
        self.staged    = {}         # {tab: {field: value}}
        self.log_lines = []
        self.log_off   = 0
        self.log_auto  = True
        self.running   = True
        self.state     = {}

        curses.curs_set(0)
        scr.timeout(2000)
        self._setup_colors()
        self._poll()

        t = threading.Thread(target=self._log_tail, daemon=True)
        t.start()

    # ── Setup ─────────────────────────────────────────────────────────────────

    def _setup_colors(self):
        curses.start_color()
        try:
            curses.use_default_colors()
            bg = -1
        except Exception:
            bg = curses.COLOR_BLACK

        def P(idx, fg, b=bg):
            curses.init_pair(idx, fg, b)

        P(CP_DEFAULT, curses.COLOR_WHITE)
        P(CP_CYAN,    curses.COLOR_CYAN)
        P(CP_GREEN,   curses.COLOR_GREEN)
        P(CP_RED,     curses.COLOR_RED)
        P(CP_YELLOW,  curses.COLOR_YELLOW)
        curses.init_pair(CP_TAB_ON,  curses.COLOR_BLACK, curses.COLOR_CYAN)
        curses.init_pair(CP_TAB_OFF, curses.COLOR_WHITE, curses.COLOR_BLACK)
        curses.init_pair(CP_SEL,     curses.COLOR_BLACK, curses.COLOR_WHITE)
        P(CP_STAGED,    curses.COLOR_YELLOW)
        P(CP_BORDER,    curses.COLOR_CYAN)
        curses.init_pair(CP_BAR_OK,   curses.COLOR_GREEN,  curses.COLOR_GREEN)
        curses.init_pair(CP_BAR_WARN, curses.COLOR_YELLOW, curses.COLOR_YELLOW)
        curses.init_pair(CP_BAR_CRIT, curses.COLOR_RED,    curses.COLOR_RED)
        curses.init_pair(CP_BAR_EMPTY,curses.COLOR_BLACK,  curses.COLOR_BLACK)
        P(CP_DIM, curses.COLOR_BLACK)

    def _poll(self):
        s = {}
        s["cpu"]            = read_cpu()
        s["ram"]            = read_ram()
        s["gpu"]            = read_gpu()
        s["stream_live"]    = svc_active("irlos-session")
        s["obs_up"]         = svc_active("obs")
        s["noalbs_up"]      = svc_active("noalbs") or bool(sh("pgrep -x noalbs",""))
        s["sls_up"]         = svc_active("sls")
        s["vnc_up"]         = (svc_active("x11vnc") or svc_active("tigervnc@:1")
                                or bool(sh("pgrep -x x11vnc","")))
        s["novnc_up"]       = svc_active("novnc") or bool(sh("pgrep -x novnc",""))
        s["nginx_up"]       = svc_active("nginx")
        s["ip"]             = get_ip()
        s["scene"]          = obs_scene()
        s["recv_bps"], s["send_bps"] = sls_stats()
        s["uptime"]         = stream_uptime()
        s["irlos_cfg"]      = read_irlos_cfg()
        s["obs_cfg"]        = read_obs_ini()
        s["noalbs_cfg"]     = read_noalbs()
        self.state = s

    def _log_tail(self):
        try:
            proc = subprocess.Popen(
                ["journalctl", "-f", "-n", "300",
                 "-u", "irlos-session.service",
                 "-u", "sls.service",
                 "--no-pager", "--output=short"],
                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True
            )
            for line in proc.stdout:
                self.log_lines.append(line.rstrip())
                if len(self.log_lines) > 2000:
                    self.log_lines = self.log_lines[-1500:]
        except Exception:
            self.log_lines.append("  [journal unavailable — run as root or with systemd access]")

    # ── Drawing primitives ────────────────────────────────────────────────────

    def put(self, y, x, text, attr=0):
        H, W = self.scr.getmaxyx()
        if y < 0 or y >= H or x < 0 or x >= W:
            return
        avail = W - x
        if avail <= 0:
            return
        try:
            self.scr.addstr(y, x, text[:avail], attr)
        except curses.error:
            pass

    def box(self, y, x, h, w, title="", cp=CP_BORDER):
        a = curses.color_pair(cp)
        top = "┌" + "─" * (w - 2) + "┐"
        bot = "└" + "─" * (w - 2) + "┘"
        self.put(y, x, top, a)
        self.put(y + h - 1, x, bot, a)
        for i in range(1, h - 1):
            self.put(y + i, x,         "│", a)
            self.put(y + i, x + w - 1, "│", a)
        if title:
            ts = f" {title} "
            tx = x + max(1, (w - len(ts)) // 2)
            self.put(y, tx, ts, a | curses.A_BOLD)

    def bar(self, y, x, pct, width=20, label="", right_text=""):
        """Filled-block bar with colour-coded fill."""
        pct = max(0.0, min(1.0, pct))
        filled = int(pct * width)
        empty  = width - filled

        if pct >= 0.90:   fill_cp = CP_BAR_CRIT
        elif pct >= 0.70: fill_cp = CP_BAR_WARN
        else:             fill_cp = CP_BAR_OK

        lx = x
        if label:
            self.put(y, lx, f"{label:<12}", curses.color_pair(CP_YELLOW))
            lx += 12

        for i in range(filled):
            self.put(y, lx + i, " ", curses.color_pair(fill_cp) | curses.A_REVERSE)
        for i in range(empty):
            self.put(y, lx + filled + i, "░",
                     curses.color_pair(CP_DIM) | curses.A_DIM)

        pct_str = f" {pct*100:4.1f}%"
        self.put(y, lx + width, pct_str, curses.color_pair(CP_DEFAULT))

        if right_text:
            self.put(y, lx + width + 8, right_text, curses.color_pair(CP_CYAN))

    def dot(self, y, x, active, label=""):
        d_cp = CP_GREEN if active else CP_RED
        self.put(y, x, "●", curses.color_pair(d_cp) | curses.A_BOLD)
        if label:
            self.put(y, x + 2, label, curses.color_pair(CP_DEFAULT))

    # ── Layout sections ───────────────────────────────────────────────────────

    def draw_nav(self, y):
        H, W = self.scr.getmaxyx()
        self.put(y, 0, " " * W, curses.color_pair(CP_TAB_OFF))
        x = 2
        for i, tab in enumerate(TABS):
            label = f"  {tab}  "
            cp   = CP_TAB_ON if i == self.active else CP_TAB_OFF
            attr = curses.color_pair(cp) | (curses.A_BOLD if i == self.active else 0)
            self.put(y, x, label, attr)
            x += len(label) + 1
        ts = time.strftime("%H:%M:%S")
        self.put(y, W - len(ts) - 2, ts, curses.color_pair(CP_CYAN))

    def draw_header(self, y0):
        H, W = self.scr.getmaxyx()
        wx = max(0, (W - WM_W) // 2)
        for i, line in enumerate(WORDMARK):
            self.put(y0 + i, wx, line, curses.color_pair(CP_CYAN) | curses.A_BOLD)

        is_live = self.state.get("stream_live", False)
        if is_live:
            badge = "● LIVE"
            bcp   = curses.color_pair(CP_GREEN) | curses.A_BOLD
        else:
            badge = "○ OFFLINE"
            bcp   = curses.color_pair(CP_RED) | curses.A_BOLD

        bx = wx + WM_W + 3
        if bx + len(badge) < W:
            self.put(y0 + 2, bx, badge, bcp)
        else:
            self.put(y0 + len(WORDMARK) + 1,
                     (W - len(badge)) // 2, badge, bcp)
        return y0 + len(WORDMARK) + 1

    def draw_sysinfo(self, y0):
        H, W = self.scr.getmaxyx()
        pw = min(W - 4, 68)
        px = (W - pw) // 2
        ph = 8
        self.box(y0, px, ph, pw, title="System", cp=CP_BORDER)

        bw  = min(24, pw - 36)
        bx  = px + 2
        cpu = self.state.get("cpu", 0.0)
        ram_used, ram_total = self.state.get("ram", (0, 1))
        gpu_util, gpu_temp  = self.state.get("gpu", (0, 0))

        self.bar(y0+1, bx, cpu/100,                    bw, "CPU",
                 f"{cpu:5.1f}%")
        self.bar(y0+2, bx, ram_used/max(ram_total,1),  bw, "RAM",
                 f"{ram_used}MB / {ram_total}MB")
        self.bar(y0+3, bx, gpu_util/100,               bw, "GPU Util",
                 f"{gpu_util:3d}%")
        self.bar(y0+4, bx, gpu_temp/100,               bw, "GPU Temp",
                 f"{gpu_temp:3d}°C")

        svcs = [
            ("OBS",    self.state.get("obs_up",    False)),
            ("noalbs", self.state.get("noalbs_up", False)),
            ("SLS",    self.state.get("sls_up",    False)),
            ("VNC",    self.state.get("vnc_up",    False)),
            ("nginx",  self.state.get("nginx_up",  False)),
        ]
        sx = bx
        for name, up in svcs:
            if sx + len(name) + 5 >= px + pw - 1:
                break
            self.dot(y0 + 6, sx, up)
            self.put(y0 + 6, sx + 2, name, curses.color_pair(CP_DEFAULT))
            sx += len(name) + 4

        return y0 + ph

    # ── Tab bodies ────────────────────────────────────────────────────────────

    def _tab_stream(self, y, x, h, w):
        s    = self.state
        live = s.get("stream_live", False)

        if live:
            big_lbl = "◉  L I V E"
            big_cp  = curses.color_pair(CP_GREEN) | curses.A_BOLD
        else:
            big_lbl = "◎  O F F L I N E"
            big_cp  = curses.color_pair(CP_RED) | curses.A_BOLD
        self.put(y+1, x + (w - len(big_lbl)) // 2, big_lbl, big_cp)

        uptime = s.get("uptime", "--:--:--")
        up_lbl = f"Uptime  {uptime}"
        self.put(y+2, x + (w - len(up_lbl)) // 2, up_lbl,
                 curses.color_pair(CP_CYAN))

        info = [
            ("Scene",        s.get("scene", "n/a")),
            ("Incoming SRT", f"{s.get('recv_bps','n/a')} kbps"),
            ("Outgoing",     f"{s.get('send_bps','n/a')} kbps"),
            ("Platform",     s.get("irlos_cfg",{}).get("platform", "n/a")),
        ]
        iy = y + 4
        for label, value in info:
            self.put(iy, x+3, f"{label:<18}", curses.color_pair(CP_YELLOW))
            self.put(iy, x+21, value,          curses.color_pair(CP_DEFAULT))
            iy += 1

        n_info   = len(info)
        actions  = [
            "Stop Stream" if live else "Start Stream",
            "[Shell]",
        ]
        act_y = y + h - len(actions) - 2
        self.put(act_y - 1, x+1, "├" + "─"*(w-2) + "┤",
                 curses.color_pair(CP_BORDER))

        for i, lbl in enumerate(actions):
            row_i = n_info + i
            sel   = (self.cur == row_i)
            attr  = curses.color_pair(CP_SEL) if sel else curses.color_pair(CP_CYAN)|curses.A_BOLD
            pad   = f"  {lbl:<{w-6}}  "
            self.put(act_y + i, x+1, pad[:w-2], attr)

        return n_info + len(actions)

    def _tab_obs(self, y, x, h, w):
        s    = self.state
        obs  = s.get("obs_cfg", {})
        icfg = s.get("irlos_cfg", {})
        st   = self.staged.get("OBS", {})

        fields = [
            ("Resolution", st.get("Resolution", obs.get("resolution","1920x1080")),
             ["1920x1080","1280x720","2560x1440","3840x2160"], "sel"),
            ("FPS",        st.get("FPS",         obs.get("fps","60")),
             ["60","30"],                                       "sel"),
            ("Bitrate",    st.get("Bitrate",      obs.get("bitrate","6000")),
             None, "inp"),
            ("Encoder",    "NVENC H.264",
             None, "disp"),
            ("Platform",   st.get("Platform",    icfg.get("platform","Kick")),
             ["Kick","Twitch","YouTube","Custom RTMP"], "sel"),
        ]

        for i, (lbl, val, _, kind) in enumerate(fields):
            sel  = (self.cur == i and self.popup is None)
            stgd = lbl in st
            is_dim = kind == "disp"

            if is_dim:
                lbl_a = curses.color_pair(CP_DIM) | curses.A_DIM
                val_a = curses.color_pair(CP_DIM) | curses.A_DIM
            elif sel:
                lbl_a = val_a = curses.color_pair(CP_SEL)
            else:
                lbl_a = curses.color_pair(CP_YELLOW)
                val_a = curses.color_pair(CP_DEFAULT)

            ry = y + 2 + i
            self.put(ry, x+3, f"{lbl:<18}", lbl_a)
            self.put(ry, x+21, f"{val:<22}", val_a)
            if stgd:
                self.put(ry, x+44, "[staged]",
                         curses.color_pair(CP_STAGED)|curses.A_BOLD)

        sep_y = y + 2 + len(fields) + 1
        self.put(sep_y-1, x+1, "├" + "─"*(w-2) + "┤",
                 curses.color_pair(CP_BORDER))
        conf_sel = (self.cur == len(fields))
        conf_a   = curses.color_pair(CP_SEL) if conf_sel else curses.color_pair(CP_GREEN)|curses.A_BOLD
        self.put(sep_y, x+3, "  ► Apply Changes & Restart OBS  ", conf_a)

        return len(fields) + 1

    def _tab_noalbs(self, y, x, h, w):
        s  = self.state
        nc = s.get("noalbs_cfg", {})
        st = self.staged.get("noalbs", {})

        fields = [
            ("low",     st.get("low",     nc.get("low","2500")),    "Low Bitrate Thresh"),
            ("offline", st.get("offline", nc.get("offline","500")), "Offline Bitrate Thresh"),
        ]

        for i, (key, val, lbl) in enumerate(fields):
            sel  = (self.cur == i and self.popup is None)
            stgd = key in st
            lbl_a = curses.color_pair(CP_SEL) if sel else curses.color_pair(CP_YELLOW)
            val_a = curses.color_pair(CP_SEL) if sel else curses.color_pair(CP_DEFAULT)
            ry = y + 2 + i
            self.put(ry, x+3, f"{lbl:<26}", lbl_a)
            self.put(ry, x+29, f"{val} kbps", val_a)
            if stgd:
                self.put(ry, x+40, "[staged]", curses.color_pair(CP_STAGED)|curses.A_BOLD)

        my = y + 2 + len(fields) + 1
        self.put(my, x+3, "Scene Mappings", curses.color_pair(CP_CYAN)|curses.A_BOLD)
        mappings = [("Normal","Live",CP_GREEN),("Low","BRB",CP_YELLOW),("Offline","Offline",CP_RED)]
        for j, (state_n, scene_n, col) in enumerate(mappings):
            self.put(my+1+j, x+5, f"{state_n:<10}", curses.color_pair(col))
            self.put(my+1+j, x+15, f"→  {scene_n}", curses.color_pair(CP_DEFAULT))

        sep_y = my + 1 + len(mappings) + 1
        self.put(sep_y, x+1, "├" + "─"*(w-2) + "┤", curses.color_pair(CP_BORDER))
        conf_sel = (self.cur == len(fields))
        conf_a   = curses.color_pair(CP_SEL) if conf_sel else curses.color_pair(CP_GREEN)|curses.A_BOLD
        self.put(sep_y+1, x+3, "  ► Apply Changes & Restart noalbs  ", conf_a)

        return len(fields) + 1

    def _tab_vnc(self, y, x, h, w):
        s        = self.state
        vnc_up   = s.get("vnc_up",   False)
        novnc_up = s.get("novnc_up", False) or s.get("nginx_up", False)
        ip       = s.get("ip", "unknown")

        self.put(y+2, x+3, "VNC Service      ", curses.color_pair(CP_YELLOW))
        self.dot(y+2, x+20, vnc_up,
                 "Running" if vnc_up else "Stopped")

        self.put(y+4, x+3, "SSH Tunnel Command",
                 curses.color_pair(CP_YELLOW)|curses.A_BOLD)
        cmd = f"ssh -L 5900:{ip}:5900 irlos@{ip}"
        self.put(y+5, x+5, cmd, curses.color_pair(CP_CYAN)|curses.A_BOLD)

        self.put(y+7, x+3, "noVNC Web Panel  ", curses.color_pair(CP_YELLOW))
        self.dot(y+7, x+20, novnc_up)
        if novnc_up:
            url = f"http://{ip}:6080/vnc.html"
            self.put(y+8, x+5, url, curses.color_pair(CP_CYAN)|curses.A_BOLD)
        else:
            self.put(y+8, x+5, "(nginx/noVNC not running)",
                     curses.color_pair(CP_RED))

        self.put(y+h-4, x+1, "├" + "─"*(w-2) + "┤",
                 curses.color_pair(CP_BORDER))
        sel   = (self.cur == 0)
        lbl   = "  Stop VNC  " if vnc_up else "  Start VNC  "
        tog_a = curses.color_pair(CP_SEL) if sel else curses.color_pair(CP_CYAN)|curses.A_BOLD
        self.put(y+h-3, x+3, lbl, tog_a)

        return 1

    def _tab_logs(self, y, x, h, w):
        inner_h = h - 3
        lines   = self.log_lines

        if self.log_auto:
            self.log_off = max(0, len(lines) - inner_h)

        if self.log_auto:
            banner   = " [Auto-scroll  ↑/↓ pause  End resume] "
            banner_a = curses.color_pair(CP_GREEN)
        else:
            banner   = " [Paused  End=resume  ↑/↓ scroll] "
            banner_a = curses.color_pair(CP_YELLOW)|curses.A_BOLD
        self.put(y+1, x+2, banner, banner_a)

        for i in range(inner_h):
            li = self.log_off + i
            ry = y + 2 + i
            if li < len(lines):
                ln    = lines[li]
                upper = ln.upper()
                if "ERROR" in upper or "FAILED" in upper or "CRIT" in upper:
                    cp = CP_RED
                elif "WARN" in upper:
                    cp = CP_YELLOW
                elif "INFO" in upper or "STARTED" in upper or "ACTIVE" in upper:
                    cp = CP_GREEN
                else:
                    cp = CP_DEFAULT
                self.put(ry, x+1, ln[:w-2], curses.color_pair(cp))

        return 0

    def _tab_config(self, y, x, h, w):
        s   = self.state
        cfg = s.get("irlos_cfg", {})
        st  = self.staged.get("Config", {})

        fields = [
            ("Stream Key", st.get("Stream Key",  cfg.get("stream_key",  "")),
             None,                                                             True),
            ("Platform",   st.get("Platform",    cfg.get("platform",    "Kick")),
             ["Kick","Twitch","YouTube","Custom RTMP"],                        False),
            ("WiFi SSID",  st.get("WiFi SSID",   cfg.get("wifi_ssid",   "")),
             None,                                                             False),
            ("SSH PubKey", st.get("SSH PubKey",  cfg.get("ssh_pubkey",  "")),
             None,                                                             False),
            ("Hostname",   st.get("Hostname",    cfg.get("hostname",    "irlos")),
             None,                                                             False),
        ]

        for i, (lbl, val, opts, masked) in enumerate(fields):
            sel  = (self.cur == i and self.popup is None)
            stgd = lbl in st
            lbl_a = curses.color_pair(CP_SEL) if sel else curses.color_pair(CP_YELLOW)
            val_a = curses.color_pair(CP_SEL) if sel else curses.color_pair(CP_DEFAULT)

            dval = val
            if masked and val:
                dval = "●" * min(len(val), 20) + f"  ({len(val)} chars)"

            ry = y + 2 + i
            self.put(ry, x+3, f"{lbl:<16}", lbl_a)
            self.put(ry, x+19, f"{dval:<36}", val_a)
            if stgd:
                self.put(ry, x+56, "[staged]", curses.color_pair(CP_STAGED)|curses.A_BOLD)

        sep_y = y + 2 + len(fields) + 1
        self.put(sep_y-1, x+1, "├" + "─"*(w-2) + "┤", curses.color_pair(CP_BORDER))
        conf_sel = (self.cur == len(fields))
        conf_a   = curses.color_pair(CP_SEL) if conf_sel else curses.color_pair(CP_GREEN)|curses.A_BOLD
        self.put(sep_y, x+3, "  ► Apply Config Changes  ", conf_a)

        return len(fields) + 1

    # ── Popup drawing ─────────────────────────────────────────────────────────

    def draw_popup(self):
        if not self.popup:
            return
        H, W = self.scr.getmaxyx()
        p = self.popup

        if p["kind"] == "sel":
            opts = p["opts"]
            ph   = len(opts) + 5
            pw   = max(44, max(len(o) for o in opts) + 10)
            pw   = min(pw, W - 4)
            py   = (H - ph) // 2
            px   = (W - pw) // 2

            for i in range(ph):
                self.put(py+i, px, " " * pw, curses.color_pair(CP_DEFAULT))
            self.box(py, px, ph, pw, title=p["title"], cp=CP_BORDER)

            for i, opt in enumerate(opts):
                sel    = (i == p["cur"])
                marker = "►" if sel else " "
                attr   = curses.color_pair(CP_SEL) if sel else curses.color_pair(CP_DEFAULT)
                self.put(py+2+i, px+2, f"{marker} {opt:<{pw-6}}", attr)

            hint = "↑↓ select   Enter confirm   Esc cancel"
            self.put(py+ph-2, px + (pw-len(hint))//2, hint,
                     curses.color_pair(CP_YELLOW))

        elif p["kind"] == "inp":
            ph  = 8
            pw  = min(64, W - 4)
            py  = (H - ph) // 2
            px  = (W - pw) // 2
            buf = p["buf"]
            mask = p.get("masked", False)
            disp = "●" * len(buf) if mask else buf

            for i in range(ph):
                self.put(py+i, px, " " * pw, curses.color_pair(CP_DEFAULT))
            self.box(py, px, ph, pw, title=p["title"], cp=CP_BORDER)

            prompt = p.get("prompt", "Enter value:")
            self.put(py+2, px+3, prompt, curses.color_pair(CP_YELLOW))

            fw          = pw - 6
            disp_trunc  = disp[-fw:] if len(disp) > fw else disp
            self.put(py+4, px+2, "┌" + "─"*fw + "┐", curses.color_pair(CP_BORDER))
            self.put(py+5, px+2, "│",                  curses.color_pair(CP_BORDER))
            self.put(py+5, px+3, f"{disp_trunc:<{fw}}",
                     curses.color_pair(CP_DEFAULT)|curses.A_BOLD)
            self.put(py+5, px+3+fw, "│",               curses.color_pair(CP_BORDER))

            # blinking cursor
            cur_x = px + 3 + min(len(disp_trunc), fw - 1)
            if (int(time.time() * 2) % 2) == 0:
                self.put(py+5, cur_x, "▌", curses.color_pair(CP_CYAN)|curses.A_BOLD)

            hint = "Enter confirm   Esc cancel   Backspace delete"
            self.put(py+ph-2, px + max(0, (pw-len(hint))//2), hint,
                     curses.color_pair(CP_YELLOW))

    # ── Main draw ─────────────────────────────────────────────────────────────

    def draw(self):
        H, W = self.scr.getmaxyx()
        if H < 24 or W < 80:
            self.scr.erase()
            msg = f"Terminal too small ({W}×{H}) — need at least 80×24"
            self.put(H//2, max(0,(W-len(msg))//2), msg,
                     curses.color_pair(CP_RED)|curses.A_BOLD)
            self.scr.refresh()
            return

        self.scr.erase()
        try:
            self.scr.bkgd(' ', curses.color_pair(CP_DEFAULT))
        except curses.error:
            pass

        self.draw_nav(0)
        hdr_end = self.draw_header(1)
        sys_end = self.draw_sysinfo(hdr_end + 1)

        cy = sys_end + 1
        ch = H - cy - 2
        cw = W - 4
        cx = 2

        tab = TABS[self.active]
        self.box(cy, cx, ch, cw, title=f" {tab} ", cp=CP_BORDER)

        if ch >= 5 and cw >= 20:
            if   tab == "Stream": max_r = self._tab_stream(cy, cx, ch, cw)
            elif tab == "OBS":    max_r = self._tab_obs(   cy, cx, ch, cw)
            elif tab == "noalbs": max_r = self._tab_noalbs(cy, cx, ch, cw)
            elif tab == "VNC":    max_r = self._tab_vnc(   cy, cx, ch, cw)
            elif tab == "Logs":   max_r = self._tab_logs(  cy, cx, ch, cw)
            elif tab == "Config": max_r = self._tab_config(cy, cx, ch, cw)
            else:                 max_r = 0

            if max_r > 0:
                self.cur = min(self.cur, max_r - 1)

        self.draw_popup()

        hint = (" ←/→ tabs  ↑/↓ navigate  Enter select"
                "  Esc close  End resume log  q quit ")
        self.put(H-1, 0, hint[:W].ljust(W), curses.color_pair(CP_CYAN))

        self.scr.refresh()

    # ── Input handling ────────────────────────────────────────────────────────

    def handle_popup_key(self, k):
        p = self.popup
        if p["kind"] == "sel":
            if k in (curses.KEY_UP, ord('k')):
                p["cur"] = max(0, p["cur"] - 1)
            elif k in (curses.KEY_DOWN, ord('j')):
                p["cur"] = min(len(p["opts"]) - 1, p["cur"] + 1)
            elif k in (curses.KEY_ENTER, 10, 13):
                val = p["opts"][p["cur"]]
                cb  = p["cb"]
                self.popup = None
                cb(val)
            elif k == 27:
                self.popup = None

        elif p["kind"] == "inp":
            if k == 27:
                self.popup = None
            elif k in (curses.KEY_ENTER, 10, 13):
                cb  = p["cb"]
                val = p["buf"]
                self.popup = None
                cb(val)
            elif k in (curses.KEY_BACKSPACE, 127, 8):
                p["buf"] = p["buf"][:-1]
            elif 32 <= k <= 126:
                p["buf"] += chr(k)

    def _open_sel(self, title, opts, cur_val, cb):
        idx = 0
        for i, o in enumerate(opts):
            if o == cur_val: idx = i; break
        self.popup = {"kind":"sel","title":title,"opts":opts,"cur":idx,"cb":cb}

    def _open_inp(self, title, cur_val, cb, masked=False, prompt="Enter value:"):
        self.popup = {"kind":"inp","title":title,"buf":cur_val,
                      "cb":cb,"masked":masked,"prompt":prompt}

    def _stage(self, tab, key, val):
        self.staged.setdefault(tab, {})[key] = val

    # ── Apply actions ─────────────────────────────────────────────────────────

    def _apply_obs(self):
        s   = self.state
        obs = dict(s.get("obs_cfg", {}))
        st  = self.staged.pop("OBS", {})
        for ui, ck in [("Resolution","resolution"),("FPS","fps"),("Bitrate","bitrate")]:
            if ui in st: obs[ck] = st[ui]
        write_obs_ini(obs)
        if "Platform" in st:
            cfg = dict(s.get("irlos_cfg", {}))
            cfg["platform"] = st["Platform"]
            write_irlos_cfg(cfg)
        sh("systemctl restart irlos-session.service &")

    def _apply_noalbs(self):
        s  = self.state
        nc = dict(s.get("noalbs_cfg", {}))
        st = self.staged.pop("noalbs", {})
        for k in ("low", "offline"):
            if k in st: nc[k] = st[k]
        write_noalbs(nc)
        sh("pkill -x noalbs 2>/dev/null; sleep 0.5; noalbs &")

    def _apply_config(self):
        s   = self.state
        cfg = dict(s.get("irlos_cfg", {}))
        st  = self.staged.pop("Config", {})
        for ui, ck in [("Stream Key","stream_key"),("Platform","platform"),
                       ("WiFi SSID","wifi_ssid"),("SSH PubKey","ssh_pubkey"),
                       ("Hostname","hostname")]:
            if ui in st: cfg[ck] = st[ui]
        write_irlos_cfg(cfg)
        if "Hostname" in st:
            sh(f"hostnamectl set-hostname {st['Hostname']}")

    # ── Row activation ────────────────────────────────────────────────────────

    def _activate(self):
        s   = self.state
        tab = TABS[self.active]

        if tab == "Stream":
            live = s.get("stream_live", False)
            if self.cur == 4:
                if live: sh("systemctl stop irlos-session.service &")
                else:    sh("systemctl start irlos-session.service &")
            elif self.cur == 5:
                self._shell_escape()

        elif tab == "OBS":
            obs  = s.get("obs_cfg", {})
            icfg = s.get("irlos_cfg", {})
            st   = self.staged.get("OBS", {})
            fields = [
                ("Resolution", st.get("Resolution",obs.get("resolution","1920x1080")),
                 ["1920x1080","1280x720","2560x1440","3840x2160"], "sel"),
                ("FPS",        st.get("FPS",        obs.get("fps","60")),
                 ["60","30"],                                        "sel"),
                ("Bitrate",    st.get("Bitrate",    obs.get("bitrate","6000")),
                 None, "inp"),
                ("Encoder",    "NVENC H.264", None, "disp"),
                ("Platform",   st.get("Platform",   icfg.get("platform","Kick")),
                 ["Kick","Twitch","YouTube","Custom RTMP"], "sel"),
            ]
            if self.cur < len(fields):
                lbl, val, opts, kind = fields[self.cur]
                if kind == "disp": return
                if kind == "sel":
                    def cb(v, l=lbl): self._stage("OBS", l, v)
                    self._open_sel(lbl, opts, val, cb)
                else:
                    def cb(v, l=lbl): self._stage("OBS", l, v)
                    self._open_inp(f"OBS › {lbl}", val, cb,
                                   prompt=f"{lbl} (kbps):")
            elif self.cur == len(fields):
                self._apply_obs()

        elif tab == "noalbs":
            nc  = s.get("noalbs_cfg", {})
            st  = self.staged.get("noalbs", {})
            fld = [
                ("low",     st.get("low",     nc.get("low","2500")),    "Low Bitrate Thresh"),
                ("offline", st.get("offline", nc.get("offline","500")), "Offline Bitrate Thresh"),
            ]
            if self.cur < len(fld):
                key, val, lbl = fld[self.cur]
                def cb(v, k=key): self._stage("noalbs", k, v)
                self._open_inp(f"noalbs › {lbl}", val, cb,
                               prompt=f"{lbl} (kbps):")
            elif self.cur == len(fld):
                self._apply_noalbs()

        elif tab == "VNC":
            if self.cur == 0:
                if s.get("vnc_up", False):
                    sh("systemctl stop x11vnc 2>/dev/null; pkill x11vnc 2>/dev/null &")
                else:
                    sh("systemctl start x11vnc 2>/dev/null || "
                       "x11vnc -display :0 -forever -nopw -bg 2>/dev/null")

        elif tab == "Config":
            cfg = s.get("irlos_cfg", {})
            st  = self.staged.get("Config", {})
            fld = [
                ("Stream Key", st.get("Stream Key",  cfg.get("stream_key",  "")),
                 None, True),
                ("Platform",   st.get("Platform",    cfg.get("platform","Kick")),
                 ["Kick","Twitch","YouTube","Custom RTMP"], False),
                ("WiFi SSID",  st.get("WiFi SSID",   cfg.get("wifi_ssid",  "")),
                 None, False),
                ("SSH PubKey", st.get("SSH PubKey",  cfg.get("ssh_pubkey", "")),
                 None, False),
                ("Hostname",   st.get("Hostname",    cfg.get("hostname","irlos")),
                 None, False),
            ]
            if self.cur < len(fld):
                lbl, val, opts, masked = fld[self.cur]
                if opts:
                    def cb(v, l=lbl): self._stage("Config", l, v)
                    self._open_sel(lbl, opts, val, cb)
                else:
                    def cb(v, l=lbl): self._stage("Config", l, v)
                    self._open_inp(f"Config › {lbl}", val, cb,
                                   masked=masked, prompt=f"{lbl}:")
            elif self.cur == len(fld):
                self._apply_config()

    # ── Shell escape ──────────────────────────────────────────────────────────

    def _shell_escape(self):
        curses.endwin()
        print()
        print("\033[1;36m" + "━"*62)
        print("  Irlos Shell  ·  Type  exit  to return to the dashboard")
        print("━"*62 + "\033[0m")
        print()
        shell = os.environ.get("SHELL", "/bin/bash")
        os.system(shell)
        self.scr = curses.initscr()
        curses.start_color()
        curses.noecho()
        curses.cbreak()
        curses.curs_set(0)
        self.scr.keypad(True)
        self.scr.timeout(2000)
        self._setup_colors()
        self.scr.clear()

    # ── Key dispatch ──────────────────────────────────────────────────────────

    def handle_key(self, k):
        tab = TABS[self.active]

        if self.popup:
            self.handle_popup_key(k)
            return

        if k == curses.KEY_LEFT:
            self.active = (self.active - 1) % len(TABS)
            self.cur = 0
        elif k == curses.KEY_RIGHT:
            self.active = (self.active + 1) % len(TABS)
            self.cur = 0
        elif k == curses.KEY_UP:
            if tab == "Logs":
                self.log_auto = False
                self.log_off  = max(0, self.log_off - 1)
            else:
                self.cur = max(0, self.cur - 1)
        elif k == curses.KEY_DOWN:
            if tab == "Logs":
                self.log_auto = False
                self.log_off  = min(max(0, len(self.log_lines)-1),
                                    self.log_off + 1)
            else:
                self.cur += 1
        elif k == curses.KEY_END and tab == "Logs":
            self.log_auto = True
        elif k in (curses.KEY_ENTER, 10, 13):
            self._activate()
        elif k == 27:
            self.popup = None
        elif k in (ord('q'), ord('Q')):
            self.running = False

    # ── Main loop ─────────────────────────────────────────────────────────────

    def loop(self):
        last_poll = 0.0
        while self.running:
            now = time.monotonic()
            if now - last_poll >= 2.0:
                self._poll()
                last_poll = now
            self.draw()
            k = self.scr.getch()
            if k != -1:
                self.handle_key(k)


# ─────────────────────────────────────────────────────────────────────────────
# Entry point
# ─────────────────────────────────────────────────────────────────────────────

def _main(scr):
    Dashboard(scr).loop()


if __name__ == "__main__" or (
    len(sys.argv) > 0 and sys.argv[0].startswith("-")
):
    try:
        curses.wrapper(_main)
    except KeyboardInterrupt:
        pass
    finally:
        try:
            curses.endwin()
        except Exception:
            pass
