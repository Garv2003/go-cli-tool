package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/garv2003/cli_tool/internals/config"
	"github.com/garv2003/cli_tool/internals/models"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type TableInfo struct {
	table           table.Model
	readyStateCount int
	build           string
}

type model struct {
	tableInfos   []TableInfo
	currentIndex int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.tableInfos[m.currentIndex].table.Focused() {
				m.tableInfos[m.currentIndex].table.Blur()
			} else {
				m.tableInfos[m.currentIndex].table.Focus()
			}
		case "left":
			m.currentIndex = (m.currentIndex - 1 + len(envNames)) % len(envNames)
		case "right":
			m.currentIndex = (m.currentIndex + 1) % len(envNames)
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.tableInfos[m.currentIndex].table.SelectedRow()[1]),
			)
		}
	}
	m.tableInfos[m.currentIndex].table, cmd = m.tableInfos[m.currentIndex].table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	envHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		PaddingBottom(1).
		PaddingLeft(1).
		PaddingRight(1).
		Render(fmt.Sprintf("Environment: %s READY %d BUILD %q", strings.ToUpper(envNames[m.currentIndex]), 55, m.tableInfos[m.currentIndex].build))

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingBottom(1).
		PaddingLeft(1).
		PaddingRight(1).
		Render("←/→: Switch envs  ↑/↓: Move  esc: Focus/Blur  enter: Select  q: Quit")

	topSection := lipgloss.JoinVertical(lipgloss.Left, envHeader, instructions)

	tableView := lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1).
		Render(m.tableInfos[m.currentIndex].table.View())

	finalView := lipgloss.JoinVertical(lipgloss.Left,
		topSection,
		tableView,
	)

	return baseStyle.Render(finalView)
}

func BuildTable(instance Instance) TableInfo {
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "HostName", Width: 30},
		{Title: "STATE", Width: 20},
	}

	var rows []table.Row
	var readyStateCount int

	for _, v := range instance.InstanceList {
		if v.State == "READY" {
			readyStateCount++
		}

		rows = append(rows, table.Row{
			v.Name,
			v.HostName,
			v.State,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

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

	return TableInfo{
		table:           t,
		readyStateCount: readyStateCount,
		build:           instance.build,
	}
}

func GetEnvStatus(envName string) {
	defer wg.Done()

	finalUrl := fmt.Sprintf(urlPattern, envName)

	res, err := http.Get(finalUrl)

	if err != nil {
		panic(err)
	}

	var parseResponse map[string][]models.ResponseInstanceItem

	err = json.NewDecoder(res.Body).Decode(&parseResponse)

	if err != nil {
		panic(err)
	}

	var finalData []models.InstanceItem
	var build string

	for key, item := range parseResponse {
		for _, subItem := range item {
			build = subItem.Build
			tmp := models.InstanceItem{
				Name:     key,
				HostName: subItem.HostName,
				State:    subItem.State,
			}
			finalData = append(finalData, tmp)
		}
	}

	data[envName] = Instance{
		InstanceList: finalData,
		build:        build,
	}
}

type Instance struct {
	InstanceList []models.InstanceItem
	build        string
}

var envNames []string
var urlPattern string
var wg sync.WaitGroup
var data = make(map[string]Instance)
var tableInfos []TableInfo

func main() {

	configPath := "config.json"

	cfg := &config.Config{}
	if err := cfg.ReloadFromFile(configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	envNames = cfg.EnvNames
	urlPattern = cfg.URLPattern

	runtime.GOMAXPROCS(runtime.NumCPU())

	for _, v := range envNames {
		wg.Add(1)
		go GetEnvStatus(v)
	}

	wg.Wait()

	for _, v := range envNames {
		tableInfos = append(tableInfos, BuildTable(data[v]))
	}

	m := model{tableInfos: tableInfos, currentIndex: 0}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
