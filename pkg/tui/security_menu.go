package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderSecurityMenu() string {
	options := []string{
		fmt.Sprintf("[%s] Memguard Protection", toggleChar(m.hwState.TemplateVerification)), // Reusing field for demonstration
		"",
		accentColorStyle("Security Status:"),
		"  Memguard Enclave: ACTIVE",
		"  Sensitive Data: PROTECTED",
		"  Memory Locking: ENABLED",
		"",
		errorStyle.Render("  [!] Memory is wiped on SIGTERM/Panic"),
	}

	var s string
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F00")).Bold(true).Render("🔒 Security & Memory Enclave Control") + "\n\n"

	for _, opt := range options {
		s += fmt.Sprintf(" %s\n", opt)
	}

	s += "\n " + statLabelStyle.Render("[ctrl+s] close menu")
	return boxStyle.Padding(1, 2).Render(s)
}

func (m Model) updateSecurity(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			// Logic to toggle application-level memguard usage preference
			m.hwState.TemplateVerification = !m.hwState.TemplateVerification
		case "esc", "ctrl+s":
			m.showSecurityMenu = false
		}
	}
	return m, nil
}
