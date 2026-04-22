// SPDX-License-Identifier: GPL-3.0-or-later
package model

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ethanmanners/irlos-dashboard/internal/config"
	"github.com/ethanmanners/irlos-dashboard/internal/journal"
	"github.com/ethanmanners/irlos-dashboard/internal/poll"
	"github.com/ethanmanners/irlos-dashboard/internal/ui"
)

const (
	minWidth  = 80
	minHeight = 24
	// Ring buffer size for log lines.
	logMaxLines = 2000
	logTrimTo   = 1500
	// How often to poll system state.
	pollInterval = 2 * time.Second
)

// PopupState is the in-model representation of an active popup.
type PopupState struct {
	Kind      ui.PopupKind
	Title     string
	Options   []string
	SelCursor int
	InputBuf  string
	Masked    bool
	Prompt    string
	// Callback info: which tab/field gets the staged value on confirm.
	TabKey   Tab
	FieldKey string
}

// Model is the bubbletea Model for the full dashboard.
type Model struct {
	width  int
	height int

	activeTab Tab
	cursor    int

	state  PolledState
	staged StagedChanges

	logLines  []string
	logOffset int
	logAuto   bool

	popup  *PopupState
	tailer *journal.Tailer

	// lastErr is displayed transiently in the hint bar.
	lastErr string
}

// New creates the initial Model. The tailer is created here so it's
// available in Init() — bubbletea's elm architecture calls Init on the
// value returned by New().
func New() Model {
	t := journal.New()
	return Model{
		width:   80,
		height:  24,
		logAuto: true,
		staged:  NewStagedChanges(),
		tailer:  t,
	}
}

// Init is called once on startup by bubbletea.
func (m Model) Init() tea.Cmd {
	t := m.tailer
	go t.Run()
	return tea.Batch(
		tea.SetWindowTitle("Irlos Dashboard"),
		tickCmd(),
		doPollCmd(),
		waitForLog(t),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func doPollCmd() tea.Cmd {
	return func() tea.Msg {
		if poll.DevMode {
			return devPollResult()
		}
		return realPollResult()
	}
}

func realPollResult() PollDoneMsg {
	var s PolledState

	cpu, err := poll.ReadCPU()
	if err == nil {
		s.CPU = cpu
	}

	ramUsed, ramTotal, err := poll.ReadMem()
	if err == nil {
		s.RAMUsed = ramUsed
		s.RAMTotal = ramTotal
	} else {
		s.RAMTotal = 1
	}

	s.GPUUtil, s.GPUTemp = poll.ReadGPU()

	s.StreamLive = poll.IsActive("irlos-session.service")
	s.OBSUp = poll.IsActive("obs.service")
	s.NoalbsUp = poll.IsActive("noalbs.service")
	s.SLSUp = poll.IsActive("sls.service")
	s.VNCUp = poll.IsActive("x11vnc.service") || poll.IsActive("tigervnc@:1.service")
	s.NovncUp = poll.IsActive("novnc.service")
	s.NginxUp = poll.IsActive("nginx.service")

	s.IP = poll.GetIP()
	s.Scene = poll.OBSScene()
	s.RecvBPS, s.SendBPS = poll.SLSStats()
	s.Uptime = poll.StreamUptime("irlos-session.service")

	if cfg, err := config.ReadIrlos(); err == nil {
		s.IrlosCfg = cfg
	}
	if cfg, err := config.ReadOBS(); err == nil {
		s.OBSCfg = cfg
	} else {
		s.OBSCfg = config.OBSDefaults()
	}
	if cfg, err := config.ReadNoalbs(); err == nil {
		s.NoalbsCfg = cfg
	} else {
		s.NoalbsCfg = config.NoalbsConfig{LowThresh: "2500", OfflineThresh: "500"}
	}

	return PollDoneMsg{State: s}
}

func devPollResult() PollDoneMsg {
	return PollDoneMsg{State: PolledState{
		CPU: 42.5, RAMUsed: 4096, RAMTotal: 16384,
		GPUUtil: 67, GPUTemp: 72,
		StreamLive: true, OBSUp: true, NoalbsUp: true,
		SLSUp: true, NginxUp: true,
		IP: "192.168.1.100", Scene: "Live Scene",
		RecvBPS: "5120", SendBPS: "4800", Uptime: "01:23:45",
		IrlosCfg: config.IrlosConfig{
			StreamKey: "live_dev_key_example",
			Platform:  "Kick",
			WifiSSID:  "DevNet",
			SSHPubKey: "ssh-ed25519 AAAA... dev",
			Hostname:  "irlos-dev",
		},
		OBSCfg:    config.OBSConfig{Resolution: "1920x1080", FPS: "60", Bitrate: "6000"},
		NoalbsCfg: config.NoalbsConfig{LowThresh: "2500", OfflineThresh: "500"},
	}}
}

func waitForLog(t *journal.Tailer) tea.Cmd {
	return func() tea.Msg {
		line := <-t.Lines
		return LogLineMsg{Line: line}
	}
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case TickMsg:
		return m, tea.Batch(tickCmd(), doPollCmd())

	case PollDoneMsg:
		if msg.Err != nil {
			log.Printf("poll error: %v", msg.Err)
		}
		m.state = msg.State
		return m, nil

	case LogLineMsg:
		m.logLines = append(m.logLines, msg.Line)
		if len(m.logLines) > logMaxLines {
			m.logLines = m.logLines[len(m.logLines)-logTrimTo:]
		}
		if m.tailer != nil {
			return m, waitForLog(m.tailer)
		}
		return m, nil

	case ServiceCmdMsg:
		if msg.Err != nil {
			m.lastErr = fmt.Sprintf("service error: %v", msg.Err)
		}
		return m, doPollCmd()

	case ConfigWriteMsg:
		if msg.Err != nil {
			m.lastErr = fmt.Sprintf("config write error: %v", msg.Err)
		}
		m.staged.Clear(msg.Tab)
		return m, doPollCmd()

	case ShellReturnMsg:
		// Reinitialise after shell escape.
		return m, tea.ClearScreen

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.popup != nil {
		return m.handlePopupKey(msg)
	}

	switch msg.String() {
	case "left":
		m.activeTab = Tab((int(m.activeTab) - 1 + TabCount) % TabCount)
		m.cursor = 0

	case "right":
		m.activeTab = Tab((int(m.activeTab) + 1) % TabCount)
		m.cursor = 0

	case "up":
		if m.activeTab == TabLogs {
			m.logAuto = false
			if m.logOffset > 0 {
				m.logOffset--
			}
		} else {
			if m.cursor > 0 {
				m.cursor--
			}
		}

	case "down":
		if m.activeTab == TabLogs {
			m.logAuto = false
			max := len(m.logLines) - 1
			if max < 0 {
				max = 0
			}
			if m.logOffset < max {
				m.logOffset++
			}
		} else {
			m.cursor++
		}

	case "end":
		if m.activeTab == TabLogs {
			m.logAuto = true
		}

	case "enter":
		return m.activate()

	case "esc":
		m.popup = nil

	case "q", "Q":
		if m.tailer != nil {
			m.tailer.Stop()
		}
		poll.ShutdownNVML()
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handlePopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.popup
	switch p.Kind {
	case ui.PopupSelect:
		switch msg.String() {
		case "up", "k":
			if p.SelCursor > 0 {
				p.SelCursor--
			}
		case "down", "j":
			if p.SelCursor < len(p.Options)-1 {
				p.SelCursor++
			}
		case "enter":
			val := p.Options[p.SelCursor]
			m.staged.Set(p.TabKey, p.FieldKey, val)
			m.popup = nil
		case "esc":
			m.popup = nil
		}

	case ui.PopupInput:
		switch msg.String() {
		case "esc":
			m.popup = nil
		case "enter":
			m.staged.Set(p.TabKey, p.FieldKey, p.InputBuf)
			m.popup = nil
		case "backspace":
			if len(p.InputBuf) > 0 {
				p.InputBuf = p.InputBuf[:len(p.InputBuf)-1]
			}
		default:
			// Only printable ASCII.
			if len(msg.Runes) == 1 {
				r := msg.Runes[0]
				if r >= 32 && r <= 126 {
					p.InputBuf += string(r)
				}
			}
		}
	}

	m.popup = p
	return m, nil
}

// activate is called on Enter when no popup is open.
func (m Model) activate() (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case TabStream:
		return m.activateStream()
	case TabOBS:
		return m.activateOBS()
	case TabNoalbs:
		return m.activateNoalbs()
	case TabVNC:
		return m.activateVNC()
	case TabConfig:
		return m.activateConfig()
	}
	return m, nil
}

func (m Model) activateStream() (tea.Model, tea.Cmd) {
	live := m.state.StreamLive
	switch m.cursor {
	case ui.StreamRowStartStop:
		unit := "irlos-session.service"
		if live {
			return m, serviceCmd(poll.StopUnit, unit)
		}
		return m, serviceCmd(poll.StartUnit, unit)
	case ui.StreamRowShell:
		return m, shellEscape()
	}
	return m, nil
}

func (m Model) activateOBS() (tea.Model, tea.Cmd) {
	fields := ui.OBSFields(
		m.state.OBSCfg.Resolution, m.state.OBSCfg.FPS,
		m.state.OBSCfg.Bitrate, m.state.IrlosCfg.Platform,
		m.staged.OBS,
	)

	if m.cursor == len(fields) {
		return m, m.applyOBSCmd()
	}
	if m.cursor >= len(fields) {
		return m, nil
	}

	f := fields[m.cursor]
	switch f.Kind {
	case ui.OBSKindDisplay:
		return m, nil
	case ui.OBSKindSelect:
		var opts []string
		switch f.Label {
		case "Resolution":
			opts = []string{"1920x1080", "1280x720", "2560x1440", "3840x2160"}
		case "FPS":
			opts = []string{"60", "30"}
		case "Platform":
			opts = []string{"Kick", "Twitch", "YouTube", "Custom RTMP"}
		}
		curIdx := 0
		for i, o := range opts {
			if o == f.Value {
				curIdx = i
				break
			}
		}
		m.popup = &PopupState{
			Kind:      ui.PopupSelect,
			Title:     f.Label,
			Options:   opts,
			SelCursor: curIdx,
			TabKey:    TabOBS,
			FieldKey:  f.Label,
		}
	case ui.OBSKindInput:
		m.popup = &PopupState{
			Kind:     ui.PopupInput,
			Title:    "OBS › " + f.Label,
			InputBuf: f.Value,
			Prompt:   f.Label + " (kbps):",
			TabKey:   TabOBS,
			FieldKey: f.Label,
		}
	}
	return m, nil
}

func (m Model) activateNoalbs() (tea.Model, tea.Cmd) {
	fields := ui.NoalbsFields(
		m.state.NoalbsCfg.LowThresh,
		m.state.NoalbsCfg.OfflineThresh,
		m.staged.Noalbs,
	)
	if m.cursor == len(fields) {
		return m, m.applyNoalbsCmd()
	}
	if m.cursor >= len(fields) {
		return m, nil
	}
	f := fields[m.cursor]
	m.popup = &PopupState{
		Kind:     ui.PopupInput,
		Title:    "noalbs › " + f.Label,
		InputBuf: f.Value,
		Prompt:   f.Label + " (kbps):",
		TabKey:   TabNoalbs,
		FieldKey: f.Key,
	}
	return m, nil
}

func (m Model) activateVNC() (tea.Model, tea.Cmd) {
	if m.cursor != 0 {
		return m, nil
	}
	if m.state.VNCUp {
		return m, serviceCmd(poll.StopUnit, "x11vnc.service")
	}
	return m, serviceCmd(poll.StartUnit, "x11vnc.service")
}

func (m Model) activateConfig() (tea.Model, tea.Cmd) {
	fields := ui.ConfigFields(
		m.state.IrlosCfg.StreamKey, m.state.IrlosCfg.Platform,
		m.state.IrlosCfg.WifiSSID, m.state.IrlosCfg.SSHPubKey,
		m.state.IrlosCfg.Hostname,
		m.staged.Config,
	)
	if m.cursor == len(fields) {
		return m, m.applyConfigCmd()
	}
	if m.cursor >= len(fields) {
		return m, nil
	}
	f := fields[m.cursor]
	switch f.Kind {
	case ui.ConfigKindSelect:
		curIdx := 0
		for i, o := range f.Opts {
			if o == f.Value {
				curIdx = i
				break
			}
		}
		m.popup = &PopupState{
			Kind:      ui.PopupSelect,
			Title:     f.Label,
			Options:   f.Opts,
			SelCursor: curIdx,
			TabKey:    TabConfig,
			FieldKey:  f.Label,
		}
	case ui.ConfigKindInput:
		m.popup = &PopupState{
			Kind:     ui.PopupInput,
			Title:    "Config › " + f.Label,
			InputBuf: f.Value,
			Masked:   f.Masked,
			Prompt:   f.Label + ":",
			TabKey:   TabConfig,
			FieldKey: f.Label,
		}
	}
	return m, nil
}

// ── Apply commands ────────────────────────────────────────────────────────────

func (m Model) applyOBSCmd() tea.Cmd {
	obs := m.state.OBSCfg
	irlos := m.state.IrlosCfg
	staged := m.staged.OBS

	if v, ok := staged["Resolution"]; ok {
		obs.Resolution = v
	}
	if v, ok := staged["FPS"]; ok {
		obs.FPS = v
	}
	if v, ok := staged["Bitrate"]; ok {
		obs.Bitrate = v
	}
	platformChanged := false
	if v, ok := staged["Platform"]; ok {
		irlos.Platform = v
		platformChanged = true
	}

	return func() tea.Msg {
		if err := config.WriteOBS(obs); err != nil {
			return ConfigWriteMsg{Tab: TabOBS, Err: err}
		}
		if platformChanged {
			if err := config.WriteIrlos(irlos); err != nil {
				return ConfigWriteMsg{Tab: TabOBS, Err: err}
			}
		}
		_ = poll.RestartUnit("irlos-session.service")
		return ConfigWriteMsg{Tab: TabOBS}
	}
}

func (m Model) applyNoalbsCmd() tea.Cmd {
	nc := m.state.NoalbsCfg
	staged := m.staged.Noalbs
	if v, ok := staged["low"]; ok {
		nc.LowThresh = v
	}
	if v, ok := staged["offline"]; ok {
		nc.OfflineThresh = v
	}
	return func() tea.Msg {
		if err := config.WriteNoalbs(nc); err != nil {
			return ConfigWriteMsg{Tab: TabNoalbs, Err: err}
		}
		// Restart noalbs: stop then start.
		_ = poll.StopUnit("noalbs.service")
		_ = poll.StartUnit("noalbs.service")
		return ConfigWriteMsg{Tab: TabNoalbs}
	}
}

func (m Model) applyConfigCmd() tea.Cmd {
	cfg := m.state.IrlosCfg
	staged := m.staged.Config
	hostnameChanged := false

	for ui, ck := range map[string]*string{
		"Stream Key": &cfg.StreamKey,
		"Platform":   &cfg.Platform,
		"WiFi SSID":  &cfg.WifiSSID,
		"SSH PubKey": &cfg.SSHPubKey,
	} {
		if v, ok := staged[ui]; ok {
			*ck = v
		}
	}
	newHostname := cfg.Hostname
	if v, ok := staged["Hostname"]; ok {
		newHostname = v
		cfg.Hostname = v
		hostnameChanged = true
	}

	return func() tea.Msg {
		if err := config.WriteIrlos(cfg); err != nil {
			return ConfigWriteMsg{Tab: TabConfig, Err: err}
		}
		if hostnameChanged {
			cmd := exec.Command("hostnamectl", "set-hostname", newHostname)
			if err := cmd.Run(); err != nil {
				log.Printf("hostnamectl: %v", err)
			}
		}
		return ConfigWriteMsg{Tab: TabConfig}
	}
}

// ── Helper commands ───────────────────────────────────────────────────────────

func serviceCmd(fn func(string) error, unit string) tea.Cmd {
	return func() tea.Msg {
		return ServiceCmdMsg{Unit: unit, Err: fn(unit)}
	}
}

func shellEscape() tea.Cmd {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ShellReturnMsg{Err: err}
	})
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width < minWidth || m.height < minHeight {
		msg := fmt.Sprintf("Terminal too small (%d×%d) — need at least %d×%d",
			m.width, m.height, minWidth, minHeight)
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render(msg))
	}

	var sections []string

	// Row 0: nav bar.
	sections = append(sections, ui.RenderNav(m.width, int(m.activeTab), TabNames))

	// Rows 1-7: wordmark header.
	sections = append(sections, ui.RenderHeader(m.width, m.state.StreamLive))

	// Rows 8-15: system info box (8 lines).
	svcs := []ui.ServiceStatus{
		{Name: "OBS", Active: m.state.OBSUp},
		{Name: "noalbs", Active: m.state.NoalbsUp},
		{Name: "SLS", Active: m.state.SLSUp},
		{Name: "VNC", Active: m.state.VNCUp},
		{Name: "nginx", Active: m.state.NginxUp},
	}
	sections = append(sections, ui.RenderSysinfo(
		m.width,
		m.state.CPU,
		m.state.RAMUsed, m.state.RAMTotal,
		m.state.GPUUtil, m.state.GPUTemp,
		svcs,
	))

	// Tab content box.
	// Layout math mirrors the Python version:
	//   navH=1, headerH=7, sysH=8, gap=1 → cy=17
	//   box height = H - cy - 2  (leave room for hint bar)
	const fixedRows = 1 + 7 + 8 + 1       // nav + header + sys + gap
	boxHeight := m.height - fixedRows - 1 // -1 for hint bar
	if boxHeight < 4 {
		boxHeight = 4
	}
	boxWidth := m.width - 4
	if boxWidth < 20 {
		boxWidth = 20
	}
	innerH := boxHeight - 2
	innerW := boxWidth - 2

	tabContent := m.renderTabContent(innerW, innerH)

	// Wrap in a named box.
	boxed := m.wrapInBox(tabContent, boxWidth, boxHeight, m.activeTab.String())
	sections = append(sections, "", boxed) // gap then box

	// Hint bar.
	sections = append(sections, m.hintBar())

	content := strings.Join(sections, "\n")

	// Overlay popup if active.
	if m.popup != nil {
		params := ui.PopupParams{
			Kind:      m.popup.Kind,
			Title:     m.popup.Title,
			Options:   m.popup.Options,
			SelCursor: m.popup.SelCursor,
			InputBuf:  m.popup.InputBuf,
			Masked:    m.popup.Masked,
			Prompt:    m.popup.Prompt,
		}
		popupStr := ui.RenderPopup(params, m.width)
		content = ui.PlaceOverlay(content, popupStr, m.width, m.height)
	}

	return content
}

func (m Model) renderTabContent(width, height int) string {
	switch m.activeTab {
	case TabStream:
		// Clamp cursor to valid range.
		maxCur := ui.StreamRowShell
		if m.cursor > maxCur {
			m.cursor = maxCur
		}
		return ui.RenderStream(
			width, height, m.cursor,
			m.state.StreamLive,
			m.state.Uptime, m.state.Scene,
			m.state.RecvBPS, m.state.SendBPS,
			m.state.IrlosCfg.Platform,
			m.staged.OBS,
		)

	case TabOBS:
		fields := ui.OBSFields(
			m.state.OBSCfg.Resolution, m.state.OBSCfg.FPS,
			m.state.OBSCfg.Bitrate, m.state.IrlosCfg.Platform,
			m.staged.OBS,
		)
		if m.cursor > len(fields) {
			m.cursor = len(fields)
		}
		return ui.RenderOBS(width, height, m.cursor, fields, m.staged.OBS)

	case TabNoalbs:
		fields := ui.NoalbsFields(
			m.state.NoalbsCfg.LowThresh,
			m.state.NoalbsCfg.OfflineThresh,
			m.staged.Noalbs,
		)
		if m.cursor > len(fields) {
			m.cursor = len(fields)
		}
		return ui.RenderNoalbs(width, height, m.cursor, fields, m.staged.Noalbs)

	case TabVNC:
		if m.cursor > 0 {
			m.cursor = 0
		}
		novncUp := m.state.NovncUp || m.state.NginxUp
		return ui.RenderVNC(width, height, m.cursor, m.state.VNCUp, novncUp, m.state.IP)

	case TabLogs:
		if m.logAuto {
			m.logOffset = len(m.logLines) - height
			if m.logOffset < 0 {
				m.logOffset = 0
			}
		}
		return ui.RenderLogs(width, height, m.logLines, m.logOffset, m.logAuto)

	case TabConfig:
		fields := ui.ConfigFields(
			m.state.IrlosCfg.StreamKey, m.state.IrlosCfg.Platform,
			m.state.IrlosCfg.WifiSSID, m.state.IrlosCfg.SSHPubKey,
			m.state.IrlosCfg.Hostname,
			m.staged.Config,
		)
		if m.cursor > len(fields) {
			m.cursor = len(fields)
		}
		return ui.RenderConfig(width, height, m.cursor, fields, m.staged.Config)
	}
	return ""
}

func (m Model) wrapInBox(inner string, width, height int, title string) string {
	titleStr := " " + title + " "
	topPad := (width - 2 - len(titleStr)) / 2
	if topPad < 0 {
		topPad = 0
	}
	topLeft := strings.Repeat("─", topPad)
	topRight := strings.Repeat("─", width-2-topPad-len(titleStr))
	top := ui.StyleBorder.Render("┌"+topLeft) +
		ui.StyleBorder.Bold(true).Render(titleStr) +
		ui.StyleBorder.Render(topRight+"┐")

	bot := ui.StyleBorder.Render("└" + strings.Repeat("─", width-2) + "┘")

	innerLines := strings.Split(inner, "\n")
	var rows []string
	rows = append(rows, top)
	for _, line := range innerLines {
		rows = append(rows, ui.StyleBorder.Render("│")+line+ui.StyleBorder.Render("│"))
	}
	rows = append(rows, bot)

	// Pad to exact height.
	for len(rows) < height {
		rows = append(rows, ui.StyleBorder.Render("│")+strings.Repeat(" ", width-2)+ui.StyleBorder.Render("│"))
	}
	return strings.Join(rows[:height], "\n")
}

func (m Model) hintBar() string {
	hint := " ←/→ tabs  ↑/↓ navigate  Enter select  Esc close  End resume log  q quit "
	if m.lastErr != "" {
		hint = " ERROR: " + m.lastErr + " "
	}
	line := ui.StyleCyan.Render(hint)
	lw := ui.VisibleWidth(line)
	if lw < m.width {
		line += strings.Repeat(" ", m.width-lw)
	}
	return line
}
