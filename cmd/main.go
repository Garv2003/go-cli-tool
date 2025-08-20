package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/garv2003/cli_tool/internals/config"
	"github.com/garv2003/cli_tool/internals/ui"
)

var (
	envNames        []string
	urlPattern      string
	refreshInterval time.Duration
)

func main() {
	cfg := &config.Config{}
	if err := cfg.ReloadFromFile("config.json"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	envNames = append(cfg.EnvNames, "all")
	urlPattern = cfg.URLPattern
	refreshInterval = time.Duration(cfg.RefreshInterval) * time.Second
	runtime.GOMAXPROCS(runtime.NumCPU())

	initialModel := ui.NewEnvSelectModel(envNames, urlPattern, refreshInterval)
	if _, err := tea.NewProgram(initialModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
