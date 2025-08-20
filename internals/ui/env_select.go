package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type envSelectModel struct {
	choices         []string
	cursor          int
	urlPattern      string
	refreshInterval int64
}

func NewEnvSelectModel(envs []string, urlPattern string, interval time.Duration) tea.Model {
	return envSelectModel{
		choices:         envs,
		cursor:          0,
		urlPattern:      urlPattern,
		refreshInterval: int64(interval.Seconds()),
	}
}

func (m envSelectModel) Init() tea.Cmd {
	return nil
}

func (m envSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			selected := m.choices[m.cursor]
			return NewTableModel(m.choices, selected, m.urlPattern, m.refreshInterval), nil
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m envSelectModel) View() string {
	s := "Select an environment:\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += cursor + " " + choice + "\n"
	}
	s += "\n↑/↓ to move • Enter to select • q to quit"
	return s
}
