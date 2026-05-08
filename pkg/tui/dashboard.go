// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rnts08/fairchain-miner/pkg/config"
	"github.com/rnts08/fairchain-miner/pkg/memory"
	"github.com/rnts08/fairchain-miner/pkg/metrics"
	"github.com/rnts08/fairchain-miner/pkg/worker"
)

// Styles
var (
	accentColor    = lipgloss.Color("#00FFD1")
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(accentColor).Padding(0, 1)
	boxStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#444444")).Padding(0, 1)
	statLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	statValueStyle = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

func accentColorStyle(s string) string {
	return lipgloss.NewStyle().Foreground(accentColor).Render(s)
}

// Messages
type HashrateMsg struct {
	Rate1m  float64
	Rate15m float64
	Rate24h float64
}

type PriceUpdateMsg float64

type DevFeeActiveMsg struct {
	Active    bool
	Remaining time.Duration
}

type PanicMsg struct{}

type LogType int

const (
	LogMining LogType = iota
	LogStratum
)

type LogEntry struct {
	Timestamp string
	Text      string
	Type      LogType
}

type LogMsg struct {
	Text string
	Type LogType
}

type NetworkMsg struct {
	Target   string
	Diff     float64
	Accepted int64
	Rejected int64
	Stale    int64
}
type WorkerStatsMsg struct {
	Rates   []float64
	Temps   []float64
	Latency time.Duration
}

type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewWorkers
	ViewConfig
	ViewSummary
	ViewAverages // For the new quick stats overlay
)

type Model struct {
	version string
	algo    string
	workers int
	uptime  time.Time

	target     string
	difficulty float64
	accepted   int64
	rejected   int64
	stale      int64

	logViewport      viewport.Model
	allLogs          []LogEntry
	width            int
	height           int
	viewMode         ViewMode
	showHardwareMenu bool
	showPanicConfirm bool
	secureEntry      bool
	logFilter        int            // 0: All, 1: Mining, 2: Stratum
	initialConfig    *config.Config // Store initial config for form reset/save
	configStore      *config.Store
	hwState          HardwareState
	devFeeActive     bool
	showSecurityMenu bool
	panicMode        bool
	devFeeRemaining  time.Duration

	rate1m              float64
	rate15m             float64
	rate24h             float64
	history             []float64 // For sparkline, uses 1m rate
	maxHashrate         float64
	latencyHistory      []float64 // For latency sparkline
	totalAcceptedShares int64
	totalRejectedShares int64
	totalStaleShares    int64
	avgSolveTime        time.Duration
	simulatedReward     float64

	workerRates []float64
	workerTemps []float64

	// Config inputs
	inputs     []textinput.Model
	focusIndex int

	// Power Savings
	powerSavingsActive bool
	originalPowerLimit int
	lastBatteryCheck   time.Time
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If the menu is open, delegate input handling to the hardware logic
	if m.showHardwareMenu {
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "m" {
			m.showHardwareMenu = false
			return m, m.saveConfig()
		}
		var cmd tea.Cmd
		m, cmd = m.updateHardware(msg)
		return m, tea.Batch(cmd, m.saveConfig())
	}

	if m.showSecurityMenu {
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "ctrl+s" {
			m.showSecurityMenu = false
			return m, nil
		}
		var cmd tea.Cmd
		m, cmd = m.updateSecurity(msg)
		return m, cmd
	}

	if m.showPanicConfirm {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch strings.ToLower(km.String()) {
			case "y":
				return m, func() tea.Msg { return PanicMsg{} }
			case "n", "esc":
				m.showPanicConfirm = false
				return m, nil
			}
		}
		return m, nil
	}

	if m.viewMode == ViewConfig {
		return m.updateConfig(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c": // Save and Quit
			return m, tea.Sequence(m.saveConfig(), tea.Quit)
		case "ctrl+s": // Hidden Security Menu
			m.showSecurityMenu = true
			return m, nil
		case "P": // Panic Mode (Shift+P)
			m.showPanicConfirm = true
			return m, nil
		case "m": // Hardware Menu
			m.showHardwareMenu = true
			return m, nil
		case "w": // Worker View
			if m.viewMode == ViewWorkers {
				m.viewMode = ViewDashboard
			} else {
				m.viewMode = ViewWorkers
			}
		case "c": // Config Form
			// Initialize config form with current values
			m.viewMode = ViewConfig
			m.inputs[0].SetValue(m.target)
			m.inputs[1].SetValue(m.initialConfig.StratumUser)
			m.inputs[2].SetValue(fmt.Sprintf("%d", m.workers))
			m.inputs[3].SetValue(fmt.Sprintf("%d", m.hwState.PowerLimit))
			m.inputs[4].SetValue(fmt.Sprintf("%.2f", m.initialConfig.ElectricityCost))
			m.inputs[5].SetValue(fmt.Sprintf("%d", m.initialConfig.HardwareTDP))
			m.inputs[6].SetValue(fmt.Sprintf("%.2f", m.initialConfig.CoinPrice))
			m.inputs[7].SetValue(m.initialConfig.PriceOracleAPI)
			m.focusIndex = 0
			m.inputs[0].Focus()
		case "s": // Summary View
			if m.viewMode == ViewSummary {
				m.viewMode = ViewDashboard
			} else {
				m.viewMode = ViewSummary
			}
			return m, nil
		case "a": // Quick Stats Overlay
			if m.viewMode == ViewAverages {
				m.viewMode = ViewDashboard
			} else {
				m.viewMode = ViewAverages
			}
			return m, nil
		case "f": // Log Filter
			m.logFilter = (m.logFilter + 1) % 3
			m.refreshLogs()
			return m, nil
		case "ctrl+l": // Clear logs
			m.allLogs = nil
			m.logViewport.SetContent("")
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.logViewport.Width = msg.Width - 4
		m.logViewport.Height = 6

	case HashrateMsg:
		m.rate1m = msg.Rate1m
		m.rate15m = msg.Rate15m
		m.rate24h = msg.Rate24h
		if m.rate1m > m.maxHashrate {
			m.maxHashrate = m.rate1m
		}
		m.history = append(m.history, m.rate1m)
		if len(m.history) > 50 {
			m.history = m.history[1:]
		}

	case WorkerStatsMsg:
		m.workerRates = msg.Rates
		m.workerTemps = msg.Temps
		latMs := float64(msg.Latency.Milliseconds())
		m.latencyHistory = append(m.latencyHistory, latMs)
		if len(m.latencyHistory) > 50 {
			m.latencyHistory = m.latencyHistory[1:]
		}

		// Turbo Mode Logic
		if m.hwState.TurboModeEnabled && len(m.workerTemps) > 0 {
			var maxT float64
			for _, t := range m.workerTemps {
				if t > maxT {
					maxT = t
				}
			}

			// Target range is [Limit-15, Limit-5]
			if maxT < float64(m.hwState.ThermalLimit-15) {
				if m.hwState.PowerLimit < 100 {
					m.hwState.PowerLimit++
					worker.SetGlobalPowerLimit(m.hwState.PowerLimit)
				}
			} else if maxT > float64(m.hwState.ThermalLimit-5) {
				if m.hwState.PowerLimit > 5 { // Don't go below 5%
					m.hwState.PowerLimit--
					worker.SetGlobalPowerLimit(m.hwState.PowerLimit)
				}
			}
		}

		// Thermal Protection: automatically lower power limit if too hot
		if m.hwState.ThermalLimit > 0 && len(m.workerTemps) > 0 {
			var maxT float64
			for _, t := range m.workerTemps {
				if t > maxT {
					maxT = t
				}
			}

			if maxT > float64(m.hwState.ThermalLimit) {
				if m.hwState.PowerLimit > 10 {
					m.hwState.PowerLimit -= 5
					worker.SetGlobalPowerLimit(m.hwState.PowerLimit)
					// Log protection event
					m.Logf(LogMining, "Thermal Protection: Core at %.1f°C! Throttling power to %d%%", maxT, m.hwState.PowerLimit)
				}
			}
		}

	case NetworkMsg:
		m.target = msg.Target
		m.difficulty = msg.Diff
		m.accepted = msg.Accepted
		m.rejected = msg.Rejected
		m.stale = msg.Stale

	case LogMsg:
		entry := LogEntry{
			Timestamp: time.Now().Format("15:04:05"),
			Text:      msg.Text,
			Type:      msg.Type,
		}
		m.allLogs = append(m.allLogs, entry)
		if len(m.allLogs) > 200 { // Increased log buffer
			m.allLogs = m.allLogs[1:]
		}
		m.refreshLogs()
		return m, nil

	case PriceUpdateMsg:
		m.initialConfig.CoinPrice = float64(msg)

	case DevFeeActiveMsg:
		m.devFeeActive = msg.Active
		m.devFeeRemaining = msg.Remaining

	case PanicMsg:
		m.panicMode = true
		m.target = "DISCONNECTED"
		if m.initialConfig != nil {
			m.initialConfig.StratumUser = "[CLEARED]"
			m.initialConfig.StratumAddr = "[CLEARED]"
		}
		for i := range m.inputs {
			m.inputs[i].SetValue("")
		}
		m.Log("!!! PANIC MODE ACTIVATED !!! Disconnecting and clearing memory.", LogMining)
		return m, tea.Quit

	case ToggleHardwareMsg:
		// Trigger backend hooks based on TUI actions
		switch string(msg) {
		case "numa":
			memory.SetNumaEnabled(m.hwState.NumaEnabled)
			m.Logf(LogMining, "Hardware: NUMA-aware allocation set to %v", m.hwState.NumaEnabled)
		case "hugepages":
			memory.SetHugepagesEnabled(m.hwState.HugepagesEnabled)
			m.Logf(LogMining, "Hardware: Hugepages set to %v", m.hwState.HugepagesEnabled)
		case "affinity":
			worker.SetAffinityEnabled(m.hwState.AffinityEnabled)
			m.Logf(LogMining, "Hardware: CPU Affinity set to %v", m.hwState.AffinityEnabled)
		case "power":
			worker.SetGlobalPowerLimit(m.hwState.PowerLimit)
			m.Logf(LogMining, "Hardware: Power Limit set to %d%%", m.hwState.PowerLimit)
		case "power_savings":
			m.Logf(LogMining, "Hardware: Power Savings mode set to %v", m.hwState.PowerSavingsEnabled)
		case "power_savings_threshold":
			m.Logf(LogMining, "Hardware: Power Savings Threshold set to %d%%", m.hwState.PowerSavingsThreshold)
		case "thermal":
			m.Logf(LogMining, "Hardware: Thermal Limit set to %d°C", m.hwState.ThermalLimit)
		case "turbo":
			m.Logf(LogMining, "Hardware: Turbo Mode set to %v", m.hwState.TurboModeEnabled)
			// Limit adjustment handled in model state
		}

	case SummaryMsg: // New message type for summary data
		m.totalAcceptedShares = msg.TotalAcceptedShares
		m.totalRejectedShares = msg.TotalRejectedShares
		m.totalStaleShares = msg.TotalStaleShares
		m.avgSolveTime = msg.AvgSolveTime
		m.simulatedReward = msg.SimulatedReward
	}

	return m, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showHardwareMenu {
		menu := m.renderHardwareMenu()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, menu)
	}

	if m.showSecurityMenu {
		menu := m.renderSecurityMenu()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, menu)
	}

	if m.showPanicConfirm {
		prompt := m.renderPanicConfirm()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, prompt)
	}

	switch m.viewMode {
	case ViewWorkers:
		return m.renderWorkerDetailedView()
	case ViewConfig:
		return m.renderConfigView()
	case ViewSummary:
		return m.renderSummaryView()
	case ViewAverages:
		stats := m.renderQuickStats()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, stats)
	default:
		return m.renderDashboard()
	}
}

func (m Model) renderDashboard() string {
	filterLabel := "All"
	if m.logFilter == 1 {
		filterLabel = "Mining"
	}
	if m.logFilter == 2 {
		filterLabel = "Stratum"
	}

	footer := statLabelStyle.Render(fmt.Sprintf(
		" [q] Quit  [P] PANIC  [m] Hardware  [w] Workers  [c] Config  [s] Summary  [a] Averages  [f] Filter: %s  [FEE: 1.0%%]",
		filterLabel))
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), m.statsRow(), m.logView(), footer)
}

func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.viewMode = ViewDashboard
			return m, nil
		case "ctrl+v": // Toggle Secure Entry (Masking)
			m.secureEntry = !m.secureEntry
			if m.secureEntry {
				m.inputs[1].EchoMode = textinput.EchoPassword
			} else {
				m.inputs[1].EchoMode = textinput.EchoNormal
			}
			return m, nil
		case "tab", "shift+tab", "up", "down":
			if msg.String() == "shift+tab" || msg.String() == "up" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			for i := range m.inputs {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)

		case "enter":
			if m.focusIndex == len(m.inputs) {
				m.viewMode = ViewDashboard

				// Only update Stratum config if input actually changed and is not a solo RPC address
				newAddr := m.inputs[0].Value()
				newUser := m.inputs[1].Value()

				// Do NOT save solo RPC addresses into Stratum config field
				if !strings.HasPrefix(strings.ToLower(newAddr), "http://") && !strings.HasPrefix(strings.ToLower(newAddr), "https://") {
					m.initialConfig.StratumAddr = []byte(newAddr)
					m.initialConfig.StratumUser = []byte(newUser)
				}

				m.initialConfig.ElectricityCost, _ = fmt.Sscanf(m.inputs[4].Value(), "%f", &m.initialConfig.ElectricityCost)
				m.initialConfig.HardwareTDP, _ = fmt.Sscanf(m.inputs[5].Value(), "%d", &m.initialConfig.HardwareTDP)
				m.initialConfig.CoinPrice, _ = fmt.Sscanf(m.inputs[6].Value(), "%f", &m.initialConfig.CoinPrice)
				m.initialConfig.PriceOracleAPI = m.inputs[7].Value()

				if err := m.configStore.Save(m.initialConfig); err != nil {
					m.Logf(LogMining, "Error saving config: %v", err)
				} else {
					m.Logf(LogMining, "Config saved: Address=%s, User=%s", m.initialConfig.StratumAddr, m.initialConfig.StratumUser)
				}

				return m, nil
			}
			m.focusIndex++
			for i := range m.inputs {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) renderConfigView() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n")
	b.WriteString(accentColorStyle(" Miner Configuration ") + "\n\n")

	for i := range m.inputs {
		b.WriteString(fmt.Sprintf("%s %s\n", statLabelStyle.Render(m.inputs[i].Placeholder+":"), m.inputs[i].View()))
	}
	b.WriteString("\n " + statLabelStyle.Render("[ctrl+v] toggle credential masking"))

	cursor := " "
	if m.focusIndex == len(m.inputs) {
		cursor = lipgloss.NewStyle().Foreground(accentColor).Render(">")
	}
	b.WriteString(fmt.Sprintf("\n%s [ Save & Exit ]\n", cursor))

	b.WriteString("\n " + statLabelStyle.Render("[tab] switch  [enter] select  [esc] cancel"))
	return boxStyle.Width(m.width - 2).Height(m.height - 4).Render(b.String())
}

func (m Model) renderWorkerDetailedView() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n")
	b.WriteString(accentColorStyle(" Individual Worker Performance ") + "\n")
	b.WriteString(statLabelStyle.Render(strings.Repeat("─", m.width-6)) + "\n\n")

	b.WriteString(fmt.Sprintf(" %-12s | %-18s | %-10s\n", "Worker ID", "Hashrate", "Temperature"))
	b.WriteString(fmt.Sprintf(" %-12s | %-18s | %-10s\n", "────────────", "──────────────────", "──────────"))

	for i := 0; i < m.workers; i++ {
		rate := 0.0
		if i < len(m.workerRates) {
			rate = m.workerRates[i]
		}
		temp := 0.0
		if i < len(m.workerTemps) {
			temp = m.workerTemps[i]
		}

		tempStr := "N/A"
		if temp > 0 {
			tempStr = fmt.Sprintf("%.1f°C", temp)
		}

		b.WriteString(fmt.Sprintf(" Worker #%-3d  | %-18s | %s\n",
			i, metrics.FormatHashrate(rate), tempStr))
	}

	b.WriteString("\n " + statLabelStyle.Render("[w] Back to Dashboard"))
	return boxStyle.Width(m.width - 2).Height(m.height - 4).Render(b.String())
}

// SummaryMsg is a new message type for summary data
type SummaryMsg struct {
	TotalAcceptedShares int64
	TotalRejectedShares int64
	TotalStaleShares    int64
	AvgSolveTime        time.Duration
	SimulatedReward     float64
}

func (m Model) renderSummaryView() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n")
	b.WriteString(accentColorStyle(" Session Summary ") + "\n")
	b.WriteString(statLabelStyle.Render(strings.Repeat("─", m.width-6)) + "\n\n")

	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Total Accepted Shares:", statValueStyle.Render(fmt.Sprint(m.totalAcceptedShares))))
	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Total Rejected Shares:", errorStyle.Render(fmt.Sprint(m.totalRejectedShares))))
	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Total Stale Shares:", statValueStyle.Foreground(lipgloss.Color("#FFCC00")).Render(fmt.Sprint(m.totalStaleShares))))
	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Average Solve Time:", statValueStyle.Render(m.avgSolveTime.Round(time.Second).String())))

	// Calculate 24h Estimate: (Hashrate * BlockReward * 86400) / (Diff * 2^32)
	// Using m.rate1m for a real-time estimate.
	est24h := 0.0
	if m.difficulty > 0 {
		est24h = (m.rate1m * 50.0 * 86400) / (m.difficulty * 4295032833.0)
	}

	dailyRevFiat := est24h * m.initialConfig.CoinPrice
	dailyCostFiat := (float64(m.initialConfig.HardwareTDP) / 1000.0) * 24.0 * m.initialConfig.ElectricityCost
	dailyProfit := dailyRevFiat - dailyCostFiat

	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Est. 24h Revenue:", successStyle.Render(fmt.Sprintf("%.8f Coins ($%.2f)", est24h, dailyRevFiat))))
	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Est. 24h Elec. Cost:", errorStyle.Render(fmt.Sprintf("$%.2f", dailyCostFiat))))

	profitStyle := successStyle
	if dailyProfit < 0 {
		profitStyle = errorStyle
	}

	breakEven := 0.0
	if est24h > 0 {
		breakEven = dailyCostFiat / est24h
	}

	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Est. Daily Profit:", profitStyle.Bold(true).Render(fmt.Sprintf("$%.2f", dailyProfit))))
	b.WriteString(fmt.Sprintf(" %-25s %s\n", "Break-even Price:", statLabelStyle.Render(fmt.Sprintf("$%.2f", breakEven))))

	b.WriteString("\n " + statLabelStyle.Render("[s] Back to Dashboard"))
	return boxStyle.Width(m.width - 2).Height(m.height - 4).Render(b.String())
}

func (m Model) renderQuickStats() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		accentColorStyle(" Hashrate Averages "),
		"",
		fmt.Sprintf("%s %s", statLabelStyle.Render(" 1m :"), statValueStyle.Render(metrics.FormatHashrate(m.rate1m))),
		fmt.Sprintf("%s %s", statLabelStyle.Render(" 15m:"), statValueStyle.Render(metrics.FormatHashrate(m.rate15m))),
		fmt.Sprintf("%s %s", statLabelStyle.Render(" 24h:"), statValueStyle.Render(metrics.FormatHashrate(m.rate24h))),
	)
	return boxStyle.Padding(1, 2).Render(content)
}

func (m Model) Log(msg string, cat LogType) {
	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Text:      msg,
		Type:      cat,
	}
	m.allLogs = append(m.allLogs, entry)
	if len(m.allLogs) > 200 {
		m.allLogs = m.allLogs[1:]
	}
	m.refreshLogs()
}

func (m Model) Logf(cat LogType, format string, args ...interface{}) {
	m.Log(fmt.Sprintf(format, args...), cat)
}

func (m Model) LogMining(msg string) {
	m.Log(msg, LogMining)
}

func (m Model) LogStratum(msg string) {
	m.Log(msg, LogStratum)
}

func (m Model) refreshLogs() {
	var filtered []LogEntry
	for _, entry := range m.allLogs {
		if m.logFilter == 0 || int(entry.Type) == m.logFilter-1 {
			filtered = append(filtered, entry)
		}
	}

	var content strings.Builder
	for _, entry := range filtered {
		typeStr := ""
		switch entry.Type {
		case LogMining:
			typeStr = statLabelStyle.Render("[M]")
		case LogStratum:
			typeStr = statLabelStyle.Render("[S]")
		}
		content.WriteString(fmt.Sprintf("%s %s %s\n", entry.Timestamp, typeStr, entry.Text))
	}
	m.logViewport.SetContent(content.String())
}

func (m Model) renderPanicConfirm() string {
	s := errorStyle.Copy().Bold(true).Render("!!! CONFIRM PANIC MODE !!!") + "\n\n"
	s += "This will clear all sensitive credentials from memory\n"
	s += "and immediately terminate the miner.\n\n"
	s += lipgloss.NewStyle().Foreground(accentColor).Render("[Y] Confirm") + "    " + statLabelStyle.Render("[N] Cancel")
	return boxStyle.Padding(1, 2).Render(s)
}

func CheckPowerSavings(app *App, workers int) {
	// Placeholder for power savings check
	// Could check battery status or other conditions
}

func GetCPUTemps(count int) []float64 {
	temps := make([]float64, count)
	// Basic Linux thermal check
	for i := 0; i < count; i++ {
		data, err := os.ReadFile(fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", i))
		if err == nil {
			var t int
			fmt.Sscanf(string(data), "%d", &t)
			temps[i] = float64(t) / 1000.0
		}
	}
	return temps
}
func (m Model) header() string {
	title := titleStyle.Render(fmt.Sprintf(" FAIRCHAIN MINER %s ", m.version))

	var devFeeIndicator string
	if m.devFeeActive {
		devFeeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("201")). // Bright Pink/Purple
			Bold(true).
			Padding(0, 1).
			Render(fmt.Sprintf("⚡ DEV-FEE ACTIVE (%s)", m.devFeeRemaining.Round(time.Second)))
	} else {
		devFeeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1).
			Render(fmt.Sprintf("Dev-Fee in: %s", m.devFeeRemaining.Round(time.Second)))
	}

	var modeLabel string
	if strings.HasPrefix(strings.ToLower(m.target), "http://") || strings.HasPrefix(strings.ToLower(m.target), "https://") {
		modeLabel = fmt.Sprintf(" 🟢 SOLO RPC | %s ", m.target)
	} else if strings.HasPrefix(strings.ToLower(m.target), "stratum+tcp://") {
		modeLabel = fmt.Sprintf(" 🟢 STRATUM | %s ", m.target)
	} else if m.target != "" {
		modeLabel = fmt.Sprintf(" | %s ", m.target)
	}

	info := fmt.Sprintf("  Algo: %s | Workers: %d | Uptime: %s%s",
		m.algo, m.workers, time.Since(m.uptime).Round(time.Second), modeLabel)
	return title + info + " " + devFeeIndicator + "\n"
}

func (m Model) statsRow() string {
	wStats := m.width / 6
	wLatency := m.width / 6
	wGraph := m.width - wStats - wLatency - 6

	stats := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("%s %s", statLabelStyle.Render("Rate:"), statValueStyle.Render(metrics.FormatHashrate(m.rate1m))),
		fmt.Sprintf("%s %s", statLabelStyle.Render("Acc :"), successStyle.Render(fmt.Sprint(m.accepted))),
		fmt.Sprintf("%s %s", statLabelStyle.Render("Rej :"), errorStyle.Render(fmt.Sprint(m.rejected))),
	)
	statsBox := boxStyle.Width(wStats).Render(stats)

	graph := m.renderSparkline(m.history, wGraph, 14, accentColor, m.maxHashrate)
	graphBox := boxStyle.Width(wGraph + 4).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			statLabelStyle.Render("Hashrate History"),
			graph,
		),
	)

	latencyGraph := m.renderSparkline(m.latencyHistory, wLatency, 5, lipgloss.Color("#FFA500"), 0)
	latencyBox := boxStyle.Width(wLatency + 4).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			statLabelStyle.Render("Latency (ms)"),
			latencyGraph,
		),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, statsBox, graphBox, latencyBox)
}

func (m Model) logView() string {
	return boxStyle.Width(m.width - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			statLabelStyle.Render("System Logs"),
			m.logViewport.View(),
		),
	)
}

func (m Model) renderSparkline(data []float64, width, height int, color lipgloss.Color, scaleMax float64) string {
	if len(data) == 0 {
		return strings.Repeat(" ", width)
	}
	runes := []rune(" ▂▃▄▅▆▇█")
	var b strings.Builder

	plotData := data
	if len(plotData) < width {
		prefix := make([]float64, width-len(plotData))
		plotData = append(prefix, plotData...)
	} else if len(plotData) > width {
		plotData = plotData[len(plotData)-width:]
	}

	maxVal := scaleMax
	if maxVal <= 0 {
		for _, v := range plotData {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal <= 0 {
		maxVal = 1
	}

	for _, val := range plotData {
		if val <= 0 {
			b.WriteRune(' ')
			continue
		}
		idx := int((val / maxVal) * float64(len(runes)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(runes) {
			idx = len(runes) - 1
		}
		b.WriteRune(runes[idx])
	}
	return lipgloss.NewStyle().Foreground(color).Render(b.String())
}

// App Handler
type App struct {
	prog          *tea.Program
	initialConfig *config.Config
}

// NewApp creates a new TUI application.
func NewApp(workers int, algo string, initialCfg *config.Config, store *config.Store) *App {
	m := NewModel("v0.1.0", algo, workers, initialCfg, store)

	a := &App{
		// Initialize the program once
		prog:          tea.NewProgram(m, tea.WithAltScreen()),
		initialConfig: initialCfg,
	}

	// Start a goroutine to periodically check battery status for power savings
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
		defer ticker.Stop()
		for range ticker.C {
			// Send a message to the TUI model to update battery status
			CheckPowerSavings(a, workers)
		}
	}()

	// Start Price Oracle ticker
	go func() {
		// Initial fetch
		if p, err := fetchPrice(a.initialConfig.PriceOracleAPI); err == nil {
			a.prog.Send(PriceUpdateMsg(p))
		}

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if p, err := fetchPrice(a.initialConfig.PriceOracleAPI); err == nil {
				a.prog.Send(PriceUpdateMsg(p))
			}
		}
	}()

	return a
}

func (a *App) Run() error {
	finalModel, err := a.prog.Run()
	if err != nil {
		return err
	}
	finalModel.(Model).saveConfig()
	return nil
}

func (a *App) Log(msg string, cat LogType) { a.prog.Send(LogMsg{Text: msg, Type: cat}) }
func (a *App) Logf(cat LogType, format string, args ...interface{}) {
	a.Log(fmt.Sprintf(format, args...), cat)
}
func (a *App) LogMining(msg string)  { a.Log(msg, LogMining) }
func (a *App) LogStratum(msg string) { a.Log(msg, LogStratum) }
func (a *App) UpdateHashrate(r1, r15, r24 float64) {
	a.prog.Send(HashrateMsg{Rate1m: r1, Rate15m: r15, Rate24h: r24})
}
func (a *App) UpdateWorkers(rates, temps []float64, latency time.Duration) {
	a.prog.Send(WorkerStatsMsg{Rates: rates, Temps: temps, Latency: latency})
}
func (a *App) UpdateNetwork(target string, diff float64, acc, rej, stale int64) {
	a.prog.Send(NetworkMsg{Target: target, Diff: diff, Accepted: acc, Rejected: rej, Stale: stale})
}
func (a *App) UpdateSummary(acc, rej, stale int64, avg time.Duration, reward float64) {
	a.prog.Send(SummaryMsg{
		TotalAcceptedShares: acc,
		TotalRejectedShares: rej,
		TotalStaleShares:    stale,
		AvgSolveTime:        avg,
		SimulatedReward:     reward,
	})
}

func (a *App) SetDevFeeActive(active bool, remaining time.Duration) {
	a.prog.Send(DevFeeActiveMsg{Active: active, Remaining: remaining})
}

// NewModel helper updated to initialize HardwareState
func NewModel(version, algo string, workers int, initialCfg *config.Config, store *config.Store) Model {
	addr := textinput.New()
	addr.Placeholder = "Pool / Node Address (stratum+tcp:// or http://)"
	user := textinput.New()
	user.Placeholder = "Wallet Address / Worker Name"
	workerCount := textinput.New()
	workerCount.Placeholder = "Worker Threads"
	powerLimit := textinput.New()
	powerLimit.Placeholder = "Power Limit (%)"
	elecCost := textinput.New()
	elecCost.Placeholder = "Elec. Cost ($/kWh)"
	hwTdp := textinput.New()
	hwTdp.Placeholder = "Hardware TDP (Watts)"
	coinPrice := textinput.New()
	coinPrice.Placeholder = "Coin Price ($)"
	priceOracleAPI := textinput.New()
	priceOracleAPI.Placeholder = "Price Oracle API URL"

	// Initialize text inputs with current config values
	addr.SetValue(string(initialCfg.StratumAddr))
	user.SetValue(string(initialCfg.StratumUser))
	workerCount.SetValue(fmt.Sprintf("%d", workers))
	powerLimit.SetValue(fmt.Sprintf("%d", initialCfg.PowerLimit))
	elecCost.SetValue(fmt.Sprintf("%.2f", initialCfg.ElectricityCost))
	hwTdp.SetValue(fmt.Sprintf("%d", initialCfg.HardwareTDP))
	coinPrice.SetValue(fmt.Sprintf("%.2f", initialCfg.CoinPrice))
	priceOracleAPI.SetValue(initialCfg.PriceOracleAPI)

	return Model{
		version:       version,
		algo:          algo,
		workers:       workers,
		uptime:        time.Now(),
		logViewport:   viewport.New(0, 0),
		configStore:   store,
		initialConfig: initialCfg,
		inputs:        []textinput.Model{addr, user, workerCount, powerLimit, elecCost, hwTdp, coinPrice, priceOracleAPI},
		hwState: HardwareState{
			NumaEnabled:           memory.IsNumaEnabled(),
			HugepagesEnabled:      memory.IsHugepagesEnabled(),
			AffinityEnabled:       worker.IsAffinityEnabled(),
			PowerLimit:            worker.GetGlobalPowerLimit(),
			ThermalLimit:          initialCfg.ThermalLimit,
			PowerSavingsEnabled:   initialCfg.PowerSavingsEnabled,
			PowerSavingsThreshold: initialCfg.PowerSavingsThreshold,
			TurboModeEnabled:      initialCfg.TurboModeEnabled,
		},
		// Initialize summary fields
		totalAcceptedShares: initialCfg.TotalAcceptedShares,
		totalRejectedShares: initialCfg.TotalRejectedShares,
		totalStaleShares:    initialCfg.TotalStaleShares,
		avgSolveTime:        0, // Will be updated by reporter
		simulatedReward:     0, // Will be updated by reporter
	}
}

func (m Model) saveConfig() tea.Cmd {
	// Never save HTTP RPC addresses into Stratum config field (fixes #1 protocol error bug)
	stratumAddr := []byte(m.inputs[0].Value())
	if strings.HasPrefix(strings.ToLower(string(stratumAddr)), "http://") || strings.HasPrefix(strings.ToLower(string(stratumAddr)), "https://") {
		// If it's an RPC address, leave existing Stratum config untouched
		stratumAddr = m.initialConfig.StratumAddr
	}

	cfg := &config.Config{
		StratumAddr:           stratumAddr,
		StratumUser:           []byte(m.inputs[1].Value()),
		NumaEnabled:           m.hwState.NumaEnabled,
		HugepagesEnabled:      m.hwState.HugepagesEnabled,
		AffinityEnabled:       m.hwState.AffinityEnabled,
		PowerLimit:            m.hwState.PowerLimit,
		ThermalLimit:          m.hwState.ThermalLimit,
		PowerSavingsEnabled:   m.hwState.PowerSavingsEnabled,
		PowerSavingsThreshold: m.hwState.PowerSavingsThreshold,
		TurboModeEnabled:      m.hwState.TurboModeEnabled,
		// Save current summary stats as well
		TotalAcceptedShares: m.totalAcceptedShares,
		TotalRejectedShares: m.totalRejectedShares,
		TotalStaleShares:    m.totalStaleShares,
		ElectricityCost:     m.initialConfig.ElectricityCost,
		PriceOracleAPI:      m.initialConfig.PriceOracleAPI,
		HardwareTDP:         m.initialConfig.HardwareTDP,
		CoinPrice:           m.initialConfig.CoinPrice,
	}
	if err := m.configStore.Save(cfg); err != nil {
		m.Logf(LogMining, "Error saving config: %v", err)
	} else {
		m.LogMining("Configuration saved.")
	}
	return nil
}
