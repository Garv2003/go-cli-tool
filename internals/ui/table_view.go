package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/garv2003/cli_tool/internals/models"
	"github.com/garv2003/cli_tool/utils"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type tableModel struct {
	envNames        []string
	urlPattern      string
	tableInfos      []TableInfo
	currentIndex    int
	selectedEnv     string
	ticker          *time.Ticker
	refreshInterval time.Duration

	// New states
	filterText  string
	isSearching bool
	textInput   textinput.Model
	sortColumn  string
	sortAsc     bool

	// Detail view
	isDetailView     bool
	selectedInstance models.InstanceItem

	errMsg     string
	retryCount int
	maxRetries int
	retryDelay time.Duration
}

type TableInfo struct {
	Table           table.Model
	ReadyStateCount int
	Build           string
}

func NewTableModel(envNames []string, selectedEnv, urlPattern string, refreshSeconds int64) tableModel {
	ti := textinput.New()
	ti.Placeholder = "Filter by name or state"
	ti.CharLimit = 64
	ti.Width = 30
	ti.CursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := tableModel{
		envNames:        envNames,
		urlPattern:      urlPattern,
		selectedEnv:     selectedEnv,
		sortColumn:      "name",
		sortAsc:         true,
		textInput:       ti,
		refreshInterval: time.Duration(refreshSeconds) * time.Second,
		maxRetries:      3,
		retryDelay:      3 * time.Second,
	}

	m.loadDataAndBuildTables()

	m.ticker = time.NewTicker(m.refreshInterval)

	return m
}

func (m *tableModel) loadDataAndBuildTables() {
	data := fetchSelectedEnvData(m.envNames, m.selectedEnv, m.urlPattern)
	m.tableInfos = buildAllTables(m.envNames, data, m.filterText, m.sortColumn, m.sortAsc)
	// Fix currentIndex to selectedEnv if not "all"
	if m.selectedEnv != "all" {
		for i, env := range m.envNames {
			if env == m.selectedEnv {
				m.currentIndex = i
				break
			}
		}
	}
}

func fetchSelectedEnvData(envs []string, selectedEnv, pattern string) map[string]utils.Instance {
	if selectedEnv == "all" {
		return fetchAllEnvs(envs, pattern)
	}
	inst := utils.FetchEnv(selectedEnv, pattern)
	return map[string]utils.Instance{
		selectedEnv: inst,
	}
}

func fetchAllEnvs(envs []string, pattern string) map[string]utils.Instance {
	allData := utils.FetchMultiple(envs, pattern)

	// Merge into "all"
	var merged []models.InstanceItem
	var build string
	for _, env := range envs {
		if env == "all" {
			continue
		}
		inst := allData[env]
		merged = append(merged, inst.InstanceList...)
		if inst.Build != "" {
			build = inst.Build
		}
	}

	allData["all"] = utils.Instance{
		InstanceList: merged,
		Build:        build,
	}
	return allData
}

func buildAllTables(envs []string, data map[string]utils.Instance, filter, sortColumn string, sortAsc bool) []TableInfo {
	var tables []TableInfo
	for _, env := range envs {
		inst := data[env]
		t := buildFilteredSortedTable(inst, filter, sortColumn, sortAsc)
		tables = append(tables, TableInfo{
			Table:           t,
			ReadyStateCount: countReady(inst.InstanceList),
			Build:           inst.Build,
		})
	}
	return tables
}

func countReady(instances []models.InstanceItem) int {
	count := 0
	for _, i := range instances {
		if strings.ToUpper(i.State) == "READY" {
			count++
		}
	}
	return count
}

func buildFilteredSortedTable(inst utils.Instance, filter, sortColumn string, sortAsc bool) table.Model {
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "HOSTNAME", Width: 30},
		{Title: "STATE", Width: 20},
	}

	// Filter
	filter = strings.ToLower(filter)
	var filteredRows []table.Row
	for _, v := range inst.InstanceList {
		if filter == "" ||
			strings.Contains(strings.ToLower(v.Name), filter) ||
			strings.Contains(strings.ToLower(v.State), filter) {
			filteredRows = append(filteredRows, table.Row{
				v.Name,
				v.HostName,
				v.State,
			})
		}
	}

	// Sort
	sortFunc := func(i, j int) bool {
		var less bool
		switch sortColumn {
		case "name":
			less = filteredRows[i][0] < filteredRows[j][0]
		case "state":
			less = filteredRows[i][2] < filteredRows[j][2]
		default:
			less = filteredRows[i][0] < filteredRows[j][0]
		}
		if !sortAsc {
			return !less
		}
		return less
	}

	// Simple bubble sort (small data)
	for i := 0; i < len(filteredRows); i++ {
		for j := 0; j < len(filteredRows)-1-i; j++ {
			if !sortFunc(j, j+1) {
				filteredRows[j], filteredRows[j+1] = filteredRows[j+1], filteredRows[j]
			}
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(filteredRows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Style
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

func (m tableModel) Init() tea.Cmd {
	return nil
}

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isDetailView {
			if msg.String() == "b" {
				m.isDetailView = false
				return m, nil
			}
			return m, nil
		}

		if m.isSearching {
			switch msg.String() {
			case "enter":
				m.filterText = m.textInput.Value()
				m.isSearching = false
				m.textInput.Blur()
				m.loadDataAndBuildTables()
				return m, nil

			case "esc":
				m.isSearching = false
				m.textInput.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "/":
			m.isSearching = true
			m.textInput.SetValue("")
			m.textInput.Focus()
			return m, nil

		case "c":
			if !m.isSearching && m.filterText != "" {
				m.filterText = ""
				m.loadDataAndBuildTables()
			}
			return m, nil

		case "left", "right":
			if m.selectedEnv == "all" {
				if msg.String() == "left" {
					m.currentIndex = (m.currentIndex - 1 + len(m.envNames)) % len(m.envNames)
				} else {
					m.currentIndex = (m.currentIndex + 1) % len(m.envNames)
				}
			}
			return m, nil

		case "up", "down":
			tbl := m.tableInfos[m.currentIndex].Table
			tbl, cmd := tbl.Update(msg)
			m.tableInfos[m.currentIndex].Table = tbl
			return m, cmd

		case "s":
			if !m.isSearching {
				if m.sortColumn == "name" {
					m.sortColumn = "state"
				} else {
					m.sortColumn = "name"
				}
				m.sortAsc = !m.sortAsc
				m.loadDataAndBuildTables()
			}
			return m, nil

		case "enter":
			tbl := m.tableInfos[m.currentIndex].Table
			row := tbl.SelectedRow()
			if len(row) >= 3 {
				m.selectedInstance = models.InstanceItem{
					Name:     row[0],
					HostName: row[1],
					State:    row[2],
					Build:    m.tableInfos[m.currentIndex].Build, // add build from current env info
				}
				m.isDetailView = true
			}
			return m, nil

		case "r":
			m.loadDataAndBuildTables()
			return m, nil

		case "esc":
			// ESC blur table focus if focused or quit if not focused
			tbl := m.tableInfos[m.currentIndex].Table
			if tbl.Focused() {
				tbl.Blur()
				m.tableInfos[m.currentIndex].Table = tbl
				return m, nil
			}
			return m, tea.Quit

		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tickMsg:
		m.loadDataAndBuildTables()
		return m, tea.Tick(m.refreshInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}

	return m, nil
}

func (m tableModel) View() string {
	if m.isDetailView {
		return renderDetailView(m.selectedInstance)
	}

	env := m.envNames[m.currentIndex]
	readyCount := m.tableInfos[m.currentIndex].ReadyStateCount
	build := m.tableInfos[m.currentIndex].Build

	envHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		PaddingBottom(1).
		PaddingLeft(1).
		PaddingRight(1).
		Render(fmt.Sprintf("Environment: %s READY %d BUILD %q", strings.ToUpper(env), readyCount, build))

	var instructionText string
	if m.isSearching {
		instructionText = "Type filter and press Enter to apply, Esc to cancel."
	} else {
		base := "←/→: Switch envs (only on 'all')  ↑/↓: Move  /: Filter  s: Sort  r: Refresh  esc: Focus/Blur  enter: Details  q: Quit"
		instructionText = base
	}

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingBottom(1).
		PaddingLeft(1).
		PaddingRight(1).
		Render(instructionText + "  c: Clear filter")

	filterView := ""
	if m.isSearching {
		filterView = m.textInput.View() + "\n"
	}

	tableView := lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1).
		Render(m.tableInfos[m.currentIndex].Table.View())

	finalView := lipgloss.JoinVertical(lipgloss.Left,
		envHeader,
		instructions,
		filterView,
		tableView,
	)

	return baseStyle.Render(finalView)
}

func renderDetailView(inst models.InstanceItem) string {
	detailStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Width(50).
		Align(lipgloss.Center)

	content := fmt.Sprintf(
		"Instance Details\n\nName: %s\nHostName: %s\nState: %s\nPress b to return.",
		inst.Name, inst.HostName, inst.State,
	)

	return detailStyle.Render(content)
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
