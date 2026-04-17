//go:build tui
// +build tui

package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func runTUI(simulate bool, rpcAddr string, workers int, power int) {
	p := tea.NewProgram(NewTUIModel(rpcAddr, workers, power), tea.WithAltScreen())
	
	go func() {
		// Simulate mining activity for TUI
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		for range ticker.C {
			hashrate := 12000 + float64(workers) * (750 + float64(rand.Int63n(1000)))
			p.Send(HashrateUpdateMsg(hashrate))
			
			if time.Now().Unix()%12 == 0 {
				p.Send(ShareMsg{Valid: true, Difficulty: 1.0})
			}
		}
	}()
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
	}
}

type TUIModel struct {
	tabs        tabModel
	spinner     spinner.Model
	activeTab   int
	running     bool
	connection  string
	workers     int
	power       int
	hashrate    uint64
	hashHistory []float64
	sharesValid uint64
	sharesTotal uint64
	log         []string
	quitConfirm bool
	width       int
	height      int
	focusIndex   int // 0: Status Tab, 1: Log Tab, 2-6: Buttons
	logOffset    int
	shareHistory []bool
}

const (
	maxHashHistory = 60
)

func NewTUIModel(conn string, workers, power int) TUIModel {
	t := tabModel{
		items: []string{"Status", "Log"},
		active: 0,
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return TUIModel{
		tabs:        t,
		spinner:     s,
		activeTab:   0,
		running:     true,
		connection:  conn,
		workers:     workers,
		power:       power,
		hashHistory: make([]float64, 0, maxHashHistory),
		log:         make([]string, 0, 100),
		width:       80,
		height:      24,
		shareHistory: make([]bool, 0, maxHashHistory),
	}
}

func (m TUIModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.quitConfirm {
			switch msg.String() {
			case "y", "Y":
				return m, tea.Quit
			case "n", "N", "q", "esc":
				m.quitConfirm = false
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "q":
			m.quitConfirm = true
			return m, nil
		case "tab":
			m.focusIndex = (m.focusIndex + 1) % 7 // 0: Status Tab, 1: Log Tab, 2-6: Buttons
			if m.focusIndex < 2 {
				m.activeTab = m.focusIndex
				m.tabs.active = m.activeTab
			} else if m.activeTab == 1 {
				m.focusIndex = 0
				m.activeTab = 0
				m.tabs.active = 0
			}
			return m, nil
		case "shift+tab":
			m.focusIndex = (m.focusIndex - 1 + 7) % 7
			if m.focusIndex < 2 {
				m.activeTab = m.focusIndex
				m.tabs.active = m.activeTab
			}
			return m, nil
		case "up":
			if m.activeTab == 1 {
				m.logOffset++
			} else if m.activeTab == 0 {
				switch m.focusIndex {
				case 4: m.focusIndex = 2
				case 5: m.focusIndex = 3
				case 6: m.focusIndex = 3
				}
			}
			return m, nil
		case "down":
			if m.activeTab == 1 && m.logOffset > 0 {
				m.logOffset--
			} else if m.activeTab == 0 {
				switch m.focusIndex {
				case 2: m.focusIndex = 4
				case 3: m.focusIndex = 5
				case 0, 1: m.focusIndex = 2
				}
			}
			return m, nil
		case "left":
			if m.activeTab == 0 {
				switch m.focusIndex {
				case 1: m.focusIndex = 0
				case 3: m.focusIndex = 2
				case 5: m.focusIndex = 4
				case 6: m.focusIndex = 3
				}
			}
			return m, nil
		case "right":
			if m.activeTab == 0 {
				switch m.focusIndex {
				case 0: m.focusIndex = 1
				case 2: m.focusIndex = 3
				case 4: m.focusIndex = 5
				case 3, 5: m.focusIndex = 6
				}
			}
			return m, nil
		case "enter", " ":
			if m.activeTab == 0 && m.focusIndex >= 2 {
				switch m.focusIndex {
				case 2: // Workers +
					if m.workers < 128 { m.workers++; m.addLog(fmt.Sprintf("Workers -> %d", m.workers)) }
				case 3: // Workers -
					if m.workers > 1 { m.workers--; m.addLog(fmt.Sprintf("Workers -> %d", m.workers)) }
				case 4: // Power +
					if m.power < 100 { m.power += 5; m.addLog(fmt.Sprintf("Power -> %d%%", m.power)) }
				case 5: // Power -
					if m.power > 5 { m.power -= 5; m.addLog(fmt.Sprintf("Power -> %d%%", m.power)) }
				case 6: // Pause/Resume
					m.running = !m.running
					m.addLog(fmt.Sprintf("Mining %s", map[bool]string{true: "resumed", false: "paused"}[m.running]))
				}
			}
			return m, nil
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case HashrateUpdateMsg:
		m.hashrate = uint64(msg)
		if len(m.hashHistory) >= m.width-15 {
			m.hashHistory = m.hashHistory[1:]
			m.shareHistory = m.shareHistory[1:]
		}
		m.hashHistory = append(m.hashHistory, float64(msg))
		m.shareHistory = append(m.shareHistory, false)
		return m, nil

	case ShareMsg:
		m.sharesTotal++
		if msg.Valid {
			m.sharesValid++
			if len(m.shareHistory) > 0 {
				m.shareHistory[len(m.shareHistory)-1] = true
			}
			m.addLog(fmt.Sprintf("[OK] Share accepted (diff: %.2f)", msg.Difficulty))
		} else {
			m.addLog(fmt.Sprintf("[ERROR] Share rejected: %s", msg.Reason))
		}
		return m, nil

	case LogMsg:
		m.addLog(string(msg))
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

func (m TUIModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	if m.quitConfirm {
		modal := quitStyle.Render("\n  Quit miner? [Y/n]  \n")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
	}

	// Header
	tabView := m.tabs.View()
	metrics := fmt.Sprintf("  Hashrate: %.1f H/s | Shares: %d/%d | Power: %d%% ", 
		float64(m.hashrate), m.sharesValid, m.sharesTotal, m.power)
	header := lipgloss.JoinHorizontal(lipgloss.Center, tabView, hashrateStyle.UnsetPadding().Render(metrics))
	header = lipgloss.NewStyle().Width(m.width).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color("240")).Render(header)

	var content string
	switch m.activeTab {
	case 0:
		content = m.statusView()
	case 1:
		content = m.logView()
	}

	help := helpStyle.Width(m.width).Render(" [Tab] Focus  [Enter] Select  [Arrow ↑/↓] Scroll Logs  [q] Quit")

	// Assemble everything with sticky footer
	mainContent := lipgloss.NewStyle().Height(m.height - 5).Width(m.width).Render(content)
	
	combined := lipgloss.JoinVertical(lipgloss.Left,
		header,
		mainContent,
		help,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, combined)
}

func (m TUIModel) statusView() string {
	status := "RUNNING"
	statusColor := lipgloss.Color("42")
	if !m.running {
		status = "PAUSED"
		statusColor = lipgloss.Color("214")
	}

	statusLine := fmt.Sprintf("%s  %s", m.spinner.View(), statusStyle.Foreground(statusColor).Render(status))

	statsBox := boxStyle.Width(30).Render(lipgloss.JoinVertical(lipgloss.Left,
		labelStyle.Render("Connection: ")+valueStyle.Render(m.connection),
		labelStyle.Render("Workers:    ")+valueStyle.Render(fmt.Sprintf("%d", m.workers)),
		labelStyle.Render("Power:      ")+valueStyle.Render(fmt.Sprintf("%d%%", m.power)),
	))

	// Buttons logic
	btnStyle := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	focusStyle := btnStyle.Copy().BorderForeground(lipgloss.Color("205")).Foreground(lipgloss.Color("205"))

	getBtn := func(idx int, label string) string {
		style := btnStyle
		if m.focusIndex == idx {
			style = focusStyle
		}
		return style.Render(label)
	}

	workersBtn := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, labelStyle.Render("Workers"), lipgloss.JoinHorizontal(lipgloss.Top, getBtn(2, " + "), getBtn(3, " - "))),
	)
	powerBtn := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, labelStyle.Render("Power"), lipgloss.JoinHorizontal(lipgloss.Top, getBtn(4, " ^ "), getBtn(5, " v "))),
	)

	pauseText := " PAUSE "
	if !m.running {
		pauseText = " RESUME "
	}
	pauseBtn := lipgloss.NewStyle().MarginTop(1).Render(getBtn(6, pauseText))

	controls := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.NewStyle().MarginLeft(2).Render(workersBtn),
		lipgloss.NewStyle().MarginLeft(2).Render(powerBtn),
		lipgloss.NewStyle().MarginLeft(4).Render(pauseBtn),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Center, statsBox, controls)

	graph := m.hashGraph()

	return lipgloss.JoinVertical(lipgloss.Left,
		statusLine,
		topRow,
		graph,
	)
}

func (m TUIModel) logView() string {
	var lines []string
	contentHeight := m.height - 7 // Adjusted for new header/footer
	
	// Determine slicing window based on offset
	start := max(0, len(m.log)-contentHeight-m.logOffset)
	end := start + contentHeight 
	if end > len(m.log) { end = len(m.log) }
	
	for i := start; i < end; i++ {
		lines = append(lines, logStyle.Width(m.width-4).Render(m.log[i]))
	}

	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	
	return logBoxStyle.Width(m.width - 2).Height(contentHeight).Render(strings.Join(lines, "\n"))
}

func (m TUIModel) hashGraph() string {
	if len(m.hashHistory) == 0 {
		return graphStyle.Render("\n  Waiting for hashrate data...\n")
	}

	maxVal := 1.0
	for _, v := range m.hashHistory {
		if v > maxVal { maxVal = v }
	}

	graphHeight := m.height - 12
	if graphHeight < 4 { graphHeight = 4 }
	
	var graph strings.Builder
	
	// Y-axis labels and bars
	for h := graphHeight; h >= 0; h-- {
		val := float64(h) * maxVal / float64(graphHeight)
		label := fmt.Sprintf("%6.0f |", val)
		if h == graphHeight { label = "   MAX |" }
		if h == 0 { label = "     0 |" }
		
		graph.WriteString(labelStyle.Render(label))
		
		threshold := val
		for _, v := range m.hashHistory {
			if v >= threshold && threshold > 0 {
				color := "38" // blueish
				if v > maxVal*0.8 {
					color = "196"
				} else if v > maxVal*0.5 {
					color = "208"
				}
				graph.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("#"))
			} else {
				graph.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("235")).Render("."))
			}
		}
		graph.WriteString("\n")
	}

	// Timeline
	graph.WriteString("       +")
	for i := range m.hashHistory {
		if m.shareHistory[i] {
			graph.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("S"))
		} else {
			graph.WriteString("-")
		}
	}
	graph.WriteString(labelStyle.Render(" TIME ->\n"))
	graph.WriteString(labelStyle.Render(fmt.Sprintf("        Markers: S = Share Found | Graph represents Hashrate over time")))

	return graphStyle.Width(m.width).Render(graph.String())
}

func (m *TUIModel) addLog(line string) {
	ts := time.Now().Format("15:04:05")
	line = fmt.Sprintf("[%s] %s", ts, line)
	if len(m.log) >= 100 {
		m.log = m.log[1:]
	}
	m.log = append(m.log, line)
}

type HashrateUpdateMsg float64
type ShareMsg struct {
	Valid      bool
	Difficulty float64
	Reason     string
}
type LogMsg string

var (
	statusStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Margin(0, 0)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	hashrateStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true).
		Padding(0, 2)
	sharesStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true).
		Padding(0, 2)
	graphStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("38")).
		Margin(1, 0)
	logBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Margin(0, 0)
	logStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
	quitStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 4).
		Align(lipgloss.Center)
)

type tabModel struct {
	items  []string
	active int
}

func (m tabModel) Update(msg tea.Msg) (tabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.active = (m.active + 1) % len(m.items)
		case "shift+tab":
			m.active = (m.active - 1 + len(m.items)) % len(m.items)
		}
	}
	return m, nil
}

func (m tabModel) ActiveIndex() int {
	return m.active
}

func (m tabModel) View() string {
	var s strings.Builder
	for i, item := range m.items {
		style := helpStyle.Copy().Padding(0, 1).UnsetWidth()
		if i == m.active {
			style = statusStyle.Copy().
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("235")).
				Padding(0, 1)
		}
		s.WriteString(style.Render(item))
		if i < len(m.items)-1 {
			s.WriteString(" | ")
		}
	}
	return s.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}