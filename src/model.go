package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenMenu screen = iota
	screenInstallForm
	screenInstallLoading
	screenInstallResult
	screenStatusLoading
	screenStatusView
	screenLogsLoading
	screenLogsView
	screenTestConfirm
	screenTestLoading
	screenTestResult
)

type menuItem struct {
	title string
	desc  string
}

func (m menuItem) Title() string       { return m.title }
func (m menuItem) Description() string { return m.desc }
func (m menuItem) FilterValue() string { return m.title }

type statusLoadedMsg struct{ report StatusReport }
type logsLoadedMsg struct{ logs LogView }
type installDoneMsg struct{ result InstallResult }
type testWakeDoneMsg struct{ result TestWakeResult }

type model struct {
	width        int
	height       int
	screen       screen
	menu         list.Model
	spinner      spinner.Model
	viewport     viewport.Model
	pickers      []timePicker
	focusIndex   int
	formError    string
	loadingTitle string
	confirmYes   bool
	status       StatusReport
	logs         LogView
	install      InstallResult
	testResult   TestWakeResult
}

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Bold(true).
			Padding(1, 2)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func newModel() model {
	items := []list.Item{
		menuItem{title: "Install", desc: "Configure shutdown and RTC wake schedule"},
		menuItem{title: "Status", desc: "Inspect installed files, cron, and wakealarm"},
		menuItem{title: "View Logs", desc: "Show the last 50 wakealarm log lines"},
		menuItem{title: "Test Wake", desc: "Shutdown now, auto power-on in 60 s to verify RTC alarm"},
	}

	menu := list.New(items, list.NewDefaultDelegate(), 0, 0)
	menu.Title = "Main Menu"
	menu.SetShowStatusBar(false)
	menu.SetFilteringEnabled(false)
	menu.SetShowHelp(false)
	menu.DisableQuitKeybindings()

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	vp := viewport.New(0, 0)

	return model{
		screen:   screenMenu,
		menu:     menu,
		spinner:  spin,
		viewport: vp,
		pickers:  newTimePickers(),
	}
}

// timeSegment identifies which part of a timePicker is active.
type timeSegment int

const (
	segHour timeSegment = iota
	segMinute
)

type timePicker struct {
	label   string
	hour    int
	minute  int
	seg     timeSegment
	focused bool
}

func newTimePickers() []timePicker {
	return []timePicker{
		{label: "Daily shutdown time", hour: 23, minute: 0, seg: segHour, focused: true},
		{label: "Daily wake-up time", hour: 7, minute: 30, seg: segHour, focused: false},
	}
}

func (tp *timePicker) up() {
	if tp.seg == segHour {
		tp.hour = (tp.hour + 1) % 24
	} else {
		tp.minute = (tp.minute + 1) % 60
	}
}

func (tp *timePicker) down() {
	if tp.seg == segHour {
		tp.hour = (tp.hour + 23) % 24
	} else {
		tp.minute = (tp.minute + 59) % 60
	}
}

func (tp timePicker) Value() string {
	return fmt.Sprintf("%02d:%02d", tp.hour, tp.minute)
}

func (tp timePicker) View() string {
	activeFg := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	dimFg := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeArr := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dimArr := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	colonFg := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	hStr := fmt.Sprintf("%02d", tp.hour)
	mStr := fmt.Sprintf("%02d", tp.minute)

	var hUp, hVal, hDown, mUp, mVal, mDown string

	if tp.focused {
		if tp.seg == segHour {
			hUp, hDown = activeArr.Render("▲"), activeArr.Render("▼")
			hVal = activeFg.Render(hStr)
			mUp, mDown = dimArr.Render("▲"), dimArr.Render("▼")
			mVal = dimFg.Render(mStr)
		} else {
			hUp, hDown = dimArr.Render("▲"), dimArr.Render("▼")
			hVal = dimFg.Render(hStr)
			mUp, mDown = activeArr.Render("▲"), activeArr.Render("▼")
			mVal = activeFg.Render(mStr)
		}
	} else {
		hUp, hDown, mUp, mDown = " ", " ", " ", " "
		hVal = dimFg.Render(hStr)
		mVal = dimFg.Render(mStr)
	}

	colon := colonFg.Render(":")

	// hUp/hDown sit above the first digit of hVal; mUp/mDown above the first digit of mVal.
	topRow := hUp + "    " + mUp
	midRow := hVal + " " + colon + " " + mVal
	botRow := hDown + "    " + mDown

	var borderFg lipgloss.Color
	if tp.focused {
		borderFg = lipgloss.Color("86")
	} else {
		borderFg = lipgloss.Color("238")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderFg).
		Padding(0, 3).
		Render(topRow + "\n" + midRow + "\n" + botRow)

	lbl := lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true).Render(tp.label)
	return lipgloss.JoinVertical(lipgloss.Left, lbl, box)
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeComponents()
		return m, nil
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case installDoneMsg:
		m.install = msg.result
		m.screen = screenInstallResult
		m.setViewportContent(formatInstallResult(msg.result))
		return m, nil
	case statusLoadedMsg:
		m.status = msg.report
		m.screen = screenStatusView
		m.setViewportContent(formatStatusReport(msg.report))
		return m, nil
	case logsLoadedMsg:
		m.logs = msg.logs
		m.screen = screenLogsView
		m.setViewportContent(formatLogsView(msg.logs))
		return m, nil
	case testWakeDoneMsg:
		m.testResult = msg.result
		m.screen = screenTestResult
		m.setViewportContent(formatTestResult(msg.result))
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		// q quits from every screen; Esc/b navigate back on inner screens
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}

	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenInstallForm:
		return m.updateInstallForm(msg)
	case screenTestConfirm:
		return m.updateTestConfirm(msg)
	case screenStatusLoading, screenLogsLoading, screenInstallLoading, screenTestLoading:
		return m.updateLoading(msg)
	case screenStatusView, screenLogsView, screenInstallResult, screenTestResult:
		return m.updateViewport(msg)
	default:
		return m, nil
	}
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Starting..."
	}

	// Header is rendered at the full terminal width so its background colour is
	// always flush edge-to-edge regardless of which screen is active.
	header := renderHeader(m.width, m.screenTitle())

	var body string
	switch m.screen {
	case screenMenu:
		body = m.renderMenu()
	case screenInstallForm:
		body = m.renderInstallForm()
	case screenTestConfirm:
		body = m.renderTestConfirm()
	case screenInstallLoading, screenStatusLoading, screenLogsLoading, screenTestLoading:
		body = m.renderLoading()
	case screenInstallResult, screenStatusView, screenLogsView, screenTestResult:
		body = m.renderViewportScreen()
	}

	// JoinVertical places header and body with no extra blank lines between them.
	// A 2-char side padding on the body gives panels a small inset margin.
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		lipgloss.NewStyle().Padding(0, 2).Render(body),
	)
}

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			selected, ok := m.menu.SelectedItem().(menuItem)
			if !ok {
				return m, nil
			}
			switch selected.title {
			case "Install":
				m.pickers = newTimePickers()
				m.focusIndex = 0
				m.formError = ""
				m.screen = screenInstallForm
				return m, nil
			case "Status":
				m.screen = screenStatusLoading
				m.loadingTitle = "Checking installation status..."
				return m, tea.Batch(m.spinner.Tick, statusCmd())
			case "View Logs":
				m.screen = screenLogsLoading
				m.loadingTitle = "Loading recent logs..."
				return m, tea.Batch(m.spinner.Tick, logsCmd())
			case "Test Wake":
				m.confirmYes = false
				m.screen = screenTestConfirm
				return m, nil
			}
		}
	}

	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m model) updateInstallForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.screen = screenMenu
			m.formError = ""
			return m, nil
		case "up":
			m.pickers[m.focusIndex].up()
			return m, nil
		case "down":
			m.pickers[m.focusIndex].down()
			return m, nil
		case "left", "shift+tab":
			// hour segment: go to minute of previous picker; minute segment: go to hour
			if m.pickers[m.focusIndex].seg == segMinute {
				m.pickers[m.focusIndex].seg = segHour
			} else if m.focusIndex > 0 {
				m.focusIndex--
				m.pickers[m.focusIndex].seg = segMinute
				m.syncPickerFocus()
			}
			return m, nil
		case "right", "tab":
			// hour segment: go to minute; minute segment: go to next picker hour
			if m.pickers[m.focusIndex].seg == segHour {
				m.pickers[m.focusIndex].seg = segMinute
			} else if m.focusIndex < len(m.pickers)-1 {
				m.focusIndex++
				m.pickers[m.focusIndex].seg = segHour
				m.syncPickerFocus()
			}
			return m, nil
		case "enter":
			cfg := ScheduleConfig{
				ShutdownTime: m.pickers[0].Value(),
				WakeTime:     m.pickers[1].Value(),
			}
			m.formError = ""
			m.screen = screenInstallLoading
			m.loadingTitle = "Installing power schedule..."
			return m, tea.Batch(m.spinner.Tick, installCmd(cfg))
		}
	}
	return m, nil
}

func (m model) updateTestConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "right", "tab", "shift+tab", "h", "l":
			m.confirmYes = !m.confirmYes
			return m, nil
		case "enter":
			if m.confirmYes {
				m.screen = screenTestLoading
				m.loadingTitle = "Setting RTC alarm — shutting down in seconds..."
				return m, tea.Batch(m.spinner.Tick, testWakeCmd())
			}
			m.screen = screenMenu
			return m, nil
		case "y", "Y":
			m.screen = screenTestLoading
			m.loadingTitle = "Setting RTC alarm — shutting down in seconds..."
			return m, tea.Batch(m.spinner.Tick, testWakeCmd())
		case "n", "N", "esc":
			m.screen = screenMenu
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, nil
		}
	}
	var spinCmd tea.Cmd
	m.spinner, spinCmd = m.spinner.Update(msg)
	return m, spinCmd
}

func (m model) updateViewport(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "b":
			m.screen = screenMenu
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *model) resizeComponents() {
	// hdrH: headerStyle Padding(1,2) → 1 top + "Power-Dawn" + section + 1 bottom = 4 rows.
	// panelV: panelStyle border(1)+pad(1) top, blank+help+pad(1)+border(1) bottom = 6 rows.
	// View() body wrapper uses Padding(0,2): zero vertical rows.
	const hdrH = 4
	const panelV = 6

	// panelStyle.Width(w): width is inclusive of padding but exclusive of borders.
	// panelStyle has Padding(1,2) → 4 chars horizontal overhead inside Width.
	// panelStyle border adds 2 chars outside Width.
	// We target panel outer width = m.width-4 (fits inside the Padding(0,2) body wrapper):
	//   panelStyle.Width(m.width-6) → outer = (m.width-6)+2 = m.width-4. ✓
	//   inner content width = (m.width-6)-4 = m.width-10.
	panelInnerW := max(30, m.width-10)
	viewH := max(5, m.height-hdrH-panelV)

	// Menu panel has 4 extra content rows (label + 2 blanks + help), so shrink list.
	m.menu.SetSize(panelInnerW, max(3, viewH-4))
	m.viewport.Width = panelInnerW
	m.viewport.Height = viewH
}

func (m *model) syncPickerFocus() {
	for i := range m.pickers {
		m.pickers[i].focused = (i == m.focusIndex)
	}
}

func (m *model) setViewportContent(content string) {
	m.viewport.SetContent(content)
	m.viewport.GotoTop()
}

func (m model) renderMenu() string {
	return panelStyle.Width(max(40, m.width-6)).Render(
		labelStyle.Render("Choose an option") + "\n\n" +
			m.menu.View() + "\n\n" +
			helpStyle.Render("Enter: select • q: quit"),
	)
}

func (m model) renderInstallForm() string {
	pickers := lipgloss.JoinHorizontal(lipgloss.Top,
		m.pickers[0].View(),
		lipgloss.NewStyle().Width(6).Render(""),
		m.pickers[1].View(),
	)

	hint := helpStyle.Render("↑/↓: change value  •  ←/→ Tab: next field  •  Enter: install  •  Esc: back")
	var notes string
	if m.formError != "" {
		notes = errorStyle.Render(m.formError) + "\n" + hint
	} else {
		notes = hint
	}

	return panelStyle.Width(max(40, m.width-6)).Render(pickers + "\n\n" + notes)
}

func (m model) renderTestConfirm() string {
	dangerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	orangeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	banner := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("203")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("203")).
		Padding(0, 2).
		Width(max(36, m.width-16)).
		Render("⚡  WARNING — IMMEDIATE SHUTDOWN")

	warnings := strings.Join([]string{
		labelStyle.Render("What will happen:"),
		orangeStyle.Render("  1. The RTC wake alarm will be set to now + 60 seconds"),
		orangeStyle.Render("  2. This computer will shut down immediately"),
		orangeStyle.Render("  3. It should power back on automatically in ~60 seconds"),
		"",
		dangerStyle.Render("  ✗  Unsaved work will be lost."),
		dangerStyle.Render("  ✗  Only works on bare-metal hardware with RTC alarm support."),
		dangerStyle.Render("  ✗  Virtual machines (VirtualBox, VMware, etc.) will NOT wake."),
	}, "\n")

	// --- buttons ---
	btnBase := lipgloss.NewStyle().
		Padding(0, 4).
		Border(lipgloss.RoundedBorder())

	var noBtn, yesBtn string
	if m.confirmYes {
		// Yes is focused
		yesBtn = btnBase.
			Background(lipgloss.Color("203")).
			Foreground(lipgloss.Color("230")).
			BorderForeground(lipgloss.Color("203")).
			Bold(true).
			Render("⚡ Yes, shut down")
		noBtn = btnBase.
			Foreground(lipgloss.Color("245")).
			BorderForeground(lipgloss.Color("238")).
			Render("  Cancel")
	} else {
		// No is focused (safe default)
		noBtn = btnBase.
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("230")).
			BorderForeground(lipgloss.Color("63")).
			Bold(true).
			Render("← Cancel")
		yesBtn = btnBase.
			Foreground(lipgloss.Color("245")).
			BorderForeground(lipgloss.Color("238")).
			Render("⚡ Yes, shut down")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		noBtn,
		lipgloss.NewStyle().Width(4).Render(""),
		yesBtn,
	)

	navHint := helpStyle.Render("←/→ or Tab: switch  •  Enter: confirm  •  y: shutdown  •  n/Esc: cancel")

	return panelStyle.Width(max(40, m.width-6)).
		Render(banner + "\n\n" + warnings + "\n\n" + buttons + "\n\n" + navHint)
}

func (m model) renderLoading() string {
	return panelStyle.Width(max(40, m.width-6)).Render(
		fmt.Sprintf("%s %s\n\n%s",
			m.spinner.View(),
			labelStyle.Render(m.loadingTitle),
			mutedStyle.Render("This may prompt for sudo during installation and verification."),
		),
	)
}

func (m model) renderViewportScreen() string {
	return panelStyle.Width(max(40, m.width-6)).Render(
		m.viewport.View() + "\n\n" +
			helpStyle.Render("↑/↓/PgUp/PgDn: scroll • b/Esc: back"),
	)
}

func (m model) screenTitle() string {
	switch m.screen {
	case screenInstallForm, screenInstallLoading, screenInstallResult:
		return "Install"
	case screenStatusLoading, screenStatusView:
		return "Status"
	case screenLogsLoading, screenLogsView:
		return "Logs"
	case screenTestConfirm, screenTestLoading, screenTestResult:
		return "Test Wake"
	default:
		return "Main Menu"
	}
}

func renderHeader(width int, section string) string {
	appNameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Bold(true)
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("195"))
	content := appNameStyle.Render("Power-Dawn") + "\n" + sectionStyle.Render(section)
	// Width(width): headerStyle has no border, so the full block is exactly `width`
	// characters wide — the purple background fills edge-to-edge every frame.
	return headerStyle.Width(max(40, width)).Render(content)
}

func statusCmd() tea.Cmd {
	return func() tea.Msg {
		return statusLoadedMsg{report: LoadStatus()}
	}
}

func logsCmd() tea.Cmd {
	return func() tea.Msg {
		return logsLoadedMsg{logs: LoadLogs(50)}
	}
}

func installCmd(cfg ScheduleConfig) tea.Cmd {
	return func() tea.Msg {
		return installDoneMsg{result: PerformInstall(cfg)}
	}
}

func testWakeCmd() tea.Cmd {
	return func() tea.Msg {
		return testWakeDoneMsg{result: PerformTestWake()}
	}
}

func wakeStatusBanner(willWake bool, reason string) string {
	if willWake {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("42")).
			Padding(0, 1).
			Render("✓  " + reason)
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("203")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("203")).
		Padding(0, 1).
		Render("✗  " + reason)
}

func formatInstallResult(result InstallResult) string {
	// Determine RTC failure early so we can build the banner.
	rtcFailed := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "wakealarm") || strings.Contains(w, "rtcwake") {
			rtcFailed = true
			break
		}
	}

	var banner string
	switch {
	case result.Success && !rtcFailed:
		banner = wakeStatusBanner(true, "Will wake up at "+result.WakeTime)
	case rtcFailed:
		banner = wakeStatusBanner(false, "Will NOT wake up — RTC alarms not supported on this hardware or VM")
	default:
		banner = wakeStatusBanner(false, "Will NOT wake up — installation did not complete")
	}

	var lines []string
	lines = append(lines, banner, "")
	if result.Success {
		lines = append(lines, successStyle.Render("Installation completed."))
	} else {
		lines = append(lines, errorStyle.Render("Installation did not complete cleanly."))
	}
	if result.FatalError != "" {
		lines = append(lines, errorStyle.Render(result.FatalError))
	}

	lines = append(lines,
		"",
		labelStyle.Render("Selected schedule"),
		fmt.Sprintf("  Shutdown: %s", result.ShutdownTime),
		fmt.Sprintf("  Wake:     %s", result.WakeTime),
		"",
		labelStyle.Render("Installer checks"),
		fmt.Sprintf("  Script created:  %s", yesNo(result.ScriptCreated)),
		fmt.Sprintf("  Log prepared:    %s", yesNo(result.LogPrepared)),
		fmt.Sprintf("  Cron installed:  %s", yesNo(result.CronInstalled)),
	)

	if result.CurrentWakealarm != "" {
		lines = append(lines,
			"",
			labelStyle.Render("Current RTC wakealarm"),
			"  "+result.CurrentWakealarm,
		)
	}

	if result.CronPreview != "" {
		lines = append(lines,
			"",
			labelStyle.Render("Managed cron block"),
			result.CronPreview,
		)
	}

	if result.RecentLog != "" {
		lines = append(lines,
			"",
			labelStyle.Render("Recent log output"),
			result.RecentLog,
		)
	}

	var otherWarnings []string
	for _, w := range result.Warnings {
		if !strings.Contains(w, "wakealarm") && !strings.Contains(w, "rtcwake") {
			otherWarnings = append(otherWarnings, w)
		}
	}

	if rtcFailed {
		lines = append(lines,
			"",
			warningStyle.Render("⚠  RTC wakealarm could not be set"),
			mutedStyle.Render("  This hardware or VM does not support RTC alarm writes."),
			mutedStyle.Render("  Shutdown scheduling (cron) is installed and will work."),
			mutedStyle.Render("  Automatic power-on requires bare-metal with ACPI/RTC support."),
		)
	}

	if len(otherWarnings) > 0 {
		lines = append(lines, "", warningStyle.Render("Warnings"))
		for _, warning := range otherWarnings {
			lines = append(lines, "- "+warning)
		}
	}

	return strings.Join(lines, "\n")
}

func formatStatusReport(report StatusReport) string {
	var banner string
	switch {
	case report.ScriptExists && report.CronInstalled && report.RTCWritable && report.WakeTime != "":
		banner = wakeStatusBanner(true, "Will wake up at "+report.WakeTime)
	case !report.ScriptExists || !report.CronInstalled || report.WakeTime == "":
		banner = wakeStatusBanner(false, "Will NOT wake up — schedule not installed")
	case !report.RTCAvailable:
		banner = wakeStatusBanner(false, "Will NOT wake up — no RTC device detected")
	default:
		banner = wakeStatusBanner(false, "Will NOT wake up — RTC alarms not supported on this hardware or VM")
	}

	lines := []string{
		banner, "",
		labelStyle.Render("Files"),
		fmt.Sprintf("  %s exists: %s", scriptPath, yesNo(report.ScriptExists)),
		fmt.Sprintf("  %s exists: %s", logPath, yesNo(report.LogExists)),
		fmt.Sprintf("  %s exists: %s", rtcPath, yesNo(report.RTCPathExists)),
		"",
		labelStyle.Render("RTC support"),
		fmt.Sprintf("  Path detected:   %s", yesNo(report.RTCPathExists)),
		fmt.Sprintf("  Alarm writable:  %s", yesNo(report.RTCWritable)),
		fmt.Sprintf("  Status:          %s", fallback(report.RTCStatus, "unknown")),
		fmt.Sprintf("  Current value:   %s", fallback(report.WakealarmValue, "—")),
	}

	if report.RTCHardwareNote != "" {
		lines = append(lines, "", warningStyle.Render("⚠  "+report.RTCHardwareNote))
	}

	lines = append(lines,
		"",
		labelStyle.Render("Schedule"),
		fmt.Sprintf("  Shutdown time: %s", fallback(report.ShutdownTime, "not detected")),
		fmt.Sprintf("  Wake time:     %s", fallback(report.WakeTime, "not detected")),
		fmt.Sprintf("  Cron block:    %s", yesNo(report.CronInstalled)),
		fmt.Sprintf("  Fully ready:   %s", yesNo(report.Installed)),
	)

	if report.PermissionNotice != "" {
		lines = append(lines, "", warningStyle.Render(report.PermissionNotice))
	}
	if report.CronPreview != "" {
		lines = append(lines, "", labelStyle.Render("Managed cron block"), report.CronPreview)
	}
	if len(report.Errors) > 0 {
		lines = append(lines, "", errorStyle.Render("Errors"))
		for _, err := range report.Errors {
			lines = append(lines, "- "+err)
		}
	}

	return strings.Join(lines, "\n")
}

func formatLogsView(view LogView) string {
	var lines []string

	// --- App log (power-dawn.log) ---
	lines = append(lines, labelStyle.Render("App log"), "  "+view.AppLogPath)
	if view.AppLogMsg != "" {
		lines = append(lines, "", mutedStyle.Render(view.AppLogMsg))
	}
	if view.AppLog != "" {
		lines = append(lines, "", view.AppLog)
	}

	lines = append(lines, "", strings.Repeat("─", 40), "")

	// --- Cron script log (wakealarm-ensure.log) ---
	lines = append(lines, labelStyle.Render("Cron script log"), "  "+view.Path)
	if view.Message != "" {
		lines = append(lines, "", mutedStyle.Render(view.Message))
	}
	if view.Content != "" {
		lines = append(lines, "", view.Content)
	}
	if len(view.Warnings) > 0 {
		lines = append(lines, "", warningStyle.Render("Warnings"))
		for _, warning := range view.Warnings {
			lines = append(lines, "  "+warning)
		}
	}
	return strings.Join(lines, "\n")
}

func formatTestResult(result TestWakeResult) string {
	var lines []string
	if result.Error != "" {
		lines = append(lines,
			wakeStatusBanner(false, "Test failed — RTC alarm could not be set or shutdown blocked"),
			"",
			errorStyle.Render(result.Error),
		)
	} else {
		lines = append(lines,
			wakeStatusBanner(true, "Shutdown initiated — computer will wake at "+result.WakeTime),
			"",
			fmt.Sprintf("  RTC alarm epoch:  %d", result.WakeEpoch),
			fmt.Sprintf("  RTC alarm set:    %s", yesNo(result.RTCSet)),
		)
	}
	if len(result.Warnings) > 0 {
		lines = append(lines, "", warningStyle.Render("Warnings"))
		for _, w := range result.Warnings {
			lines = append(lines, "  "+w)
		}
	}
	return strings.Join(lines, "\n")
}

func fallback(value, fallbackValue string) string {
	if strings.TrimSpace(value) == "" {
		return fallbackValue
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
