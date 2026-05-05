// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HardwareState struct {
	NumaEnabled      bool
	HugepagesEnabled bool
	AffinityEnabled  bool
	PowerLimit       int
	PowerSavingsEnabled bool
	PowerSavingsThreshold int
	ThermalLimit     int
	TurboModeEnabled bool
	Cursor           int
}

type ToggleHardwareMsg string

func (m Model) renderHardwareMenu() string {
	options := []string{
		fmt.Sprintf("[%s] NUMA-aware Allocation", toggleChar(m.hwState.NumaEnabled)),
		fmt.Sprintf("[%s] Hugepages (2MB/1GB)", toggleChar(m.hwState.HugepagesEnabled)),
		fmt.Sprintf("[%s] CPU Core Affinity (Pinning)", toggleChar(m.hwState.AffinityEnabled)),
		fmt.Sprintf("Power Limit: %d%%", m.hwState.PowerLimit),
		fmt.Sprintf("[%s] Power Savings Mode", toggleChar(m.hwState.PowerSavingsEnabled)),
		fmt.Sprintf("Power Savings Threshold: %d%%", m.hwState.PowerSavingsThreshold),
		fmt.Sprintf("Thermal Limit: %d°C", m.hwState.ThermalLimit),
		fmt.Sprintf("[%s] Turbo Mode (Auto-Power)", toggleChar(m.hwState.TurboModeEnabled)),
	}

	var s string
	s += lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Hardware Control Settings") + "\n\n"

	for i, opt := range options {
		cursor := " "
		if m.hwState.Cursor == i {
			cursor = lipgloss.NewStyle().Foreground(accentColor).Render(">")
		}
		s += fmt.Sprintf("%s %s\n", cursor, opt)
	}

	s += "\n " + statLabelStyle.Render("[space] toggle  [m] close menu")
	return boxStyle.Padding(1, 2).Render(s)
}

func toggleChar(b bool) string {
	if b {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("X")
	}
	return " "
}

func (m Model) updateHardware(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.hwState.Cursor > 0 {
				m.hwState.Cursor--
			}
		case "down", "j":
			if m.hwState.Cursor < 7 { // 8 options, 0-7
				m.hwState.Cursor++
			}
		case " ":
			switch m.hwState.Cursor {
			case 0:
				m.hwState.NumaEnabled = !m.hwState.NumaEnabled
				return m, func() tea.Msg { return ToggleHardwareMsg("numa") }
			case 1:
				m.hwState.HugepagesEnabled = !m.hwState.HugepagesEnabled
				return m, func() tea.Msg { return ToggleHardwareMsg("hugepages") }
			case 2:
				m.hwState.AffinityEnabled = !m.hwState.AffinityEnabled
				return m, func() tea.Msg { return ToggleHardwareMsg("affinity") }
			case 4: // Power Savings Mode
				m.hwState.PowerSavingsEnabled = !m.hwState.PowerSavingsEnabled
				return m, func() tea.Msg { return ToggleHardwareMsg("power_savings") }
			case 7: // Turbo Mode
				m.hwState.TurboModeEnabled = !m.hwState.TurboModeEnabled
				return m, func() tea.Msg { return ToggleHardwareMsg("turbo") }
			}
		case "right", "l":
			if m.hwState.Cursor == 3 && m.hwState.PowerLimit < 100 {
				m.hwState.PowerLimit += 5
				return m, func() tea.Msg { return ToggleHardwareMsg("power") }
			}
			if m.hwState.Cursor == 5 && m.hwState.PowerSavingsThreshold < 95 { // Power Savings Threshold
				m.hwState.PowerSavingsThreshold += 5
				return m, func() tea.Msg { return ToggleHardwareMsg("power_savings_threshold") }
			}
			if m.hwState.Cursor == 6 && m.hwState.ThermalLimit < 100 { // Thermal Limit
				m.hwState.ThermalLimit += 1
				return m, func() tea.Msg { return ToggleHardwareMsg("thermal") }
			}
		case "left", "h":
			if m.hwState.Cursor == 3 && m.hwState.PowerLimit > 5 {
				m.hwState.PowerLimit -= 5
				return m, func() tea.Msg { return ToggleHardwareMsg("power") }
			}
			if m.hwState.Cursor == 5 && m.hwState.PowerSavingsThreshold > 5 { // Power Savings Threshold
				m.hwState.PowerSavingsThreshold -= 5
				return m, func() tea.Msg { return ToggleHardwareMsg("power_savings_threshold") }
			}
			if m.hwState.Cursor == 6 && m.hwState.ThermalLimit > 30 { // Thermal Limit
				m.hwState.ThermalLimit -= 1
				return m, func() tea.Msg { return ToggleHardwareMsg("thermal") }
			}
		case "m", "esc":
			m.showHardwareMenu = false
			return m, nil
		}
	}
	return m, nil
}
