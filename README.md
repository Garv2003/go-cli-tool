# go-cli-tool

A terminal UI for monitoring the health of backend service instances across multiple
environments. Built in Go with the [Bubble Tea](https://github.com/charmbracelet/bubbletea)
framework, the tool reads a list of environments from a JSON config file, polls a
status endpoint for each one over HTTP, and renders the returned instances in an
interactive table that you can filter, sort, and drill into. It is a read-only
dashboard driven entirely by `config.json` — there are no subcommands or flags.

## Features

- **Environment picker** — an initial list screen to select a single environment, or
  `all` to view every configured environment merged together.
- **Live status tables** — each instance is shown with its `NAME`, `HOSTNAME`, and
  `STATE`, plus a per-environment header showing the count of instances in the
  `READY` state and the current build string.
- **Auto-refresh** — data is re-fetched on a ticker at the interval defined by
  `refresh_interval` in the config; a manual refresh is also bound to `r`.
- **Filtering** — press `/` to type a substring; rows are matched case-insensitively
  against both the instance name and its state. Press `c` to clear the active filter.
- **Sorting** — press `s` to toggle the sort column between `name` and `state` and
  flip the ascending/descending direction.
- **Detail view** — press `Enter` on a selected row to open a bordered panel showing
  that instance's name, hostname, and state; press `b` to return.
- **Environment switching in `all` mode** — when `all` is selected, `←`/`→` cycle
  through the individual environment tables.
- **Concurrent fetching** — in `all` mode every environment is polled in parallel
  using goroutines and a `sync.WaitGroup`, then merged into a combined table.

## How it works / Architecture

```
config.json ──► config.ReloadFromFile ──► main
                                            │
                                            ▼
                          ui.NewEnvSelectModel (Bubble Tea program)
                                            │  Enter
                                            ▼
                              ui.NewTableModel (per-env tables)
                                            │
                            utils.FetchEnv / utils.FetchMultiple
                                            │  HTTP GET urlPattern % env
                                            ▼
                          JSON ──► models.ResponseInstanceItem
                                    └► models.InstanceItem (table rows)
```

1. **Startup** — `cmd/main.go` loads `config.json` into a `config.Config`
   (`urlPattern`, `envNames`, `refresh_interval`), appends a synthetic `all`
   environment, sets `GOMAXPROCS` to the number of CPUs, and starts a Bubble Tea
   program with the environment-select model.
2. **Environment selection** — `internals/ui/env_select.go` renders the list and
   handles `↑`/`↓`/`Enter`/`q`. On `Enter` it transitions to the table model for the
   chosen environment.
3. **Data fetch** — `utils/fetch.go` builds each URL with
   `fmt.Sprintf(urlPattern, env)`, issues an `http.Get`, and decodes the response
   (a `map[string][]ResponseInstanceItem`) into flat `InstanceItem` rows. Any error
   yields an empty `Instance`. `FetchMultiple` runs one goroutine per environment for
   the `all` view.
4. **Table model** — `internals/ui/table_view.go` holds all UI state (current env
   index, filter text, sort column/direction, detail-view flag, ticker). It builds a
   `bubbles/table` per environment, applies the filter and an in-place sort, counts
   `READY` instances, and styles everything with Lip Gloss. Key handling covers
   search input, sorting, refresh, env switching, and the detail view.

**Key packages**

| Path | Responsibility |
|---|---|
| `cmd/main.go` | Entry point; config load and Bubble Tea bootstrap |
| `internals/config` | JSON config struct and file loader |
| `internals/models` | Wire (`ResponseInstanceItem`) and view (`InstanceItem`) types |
| `internals/ui/env_select.go` | Environment selection screen |
| `internals/ui/table_view.go` | Table screen: fetch orchestration, filter/sort, detail view |
| `utils/fetch.go` | HTTP fetching (single and concurrent) and JSON decoding |

## Tech stack

- **Go 1.23** (`go.mod` module `github.com/garv2003/cli_tool`)
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** `v1.3.6` — TUI runtime / Elm-style model
- **[Bubbles](https://github.com/charmbracelet/bubbles)** `v0.21.0` — `table` and `textinput` components
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** `v1.1.0` — styling and layout
- Standard library `net/http`, `encoding/json`, and `sync` for fetching

## Getting started

The tool expects a `config.json` in the working directory. It is not committed to the
repo, so create one first:

```json
{
  "urlPattern": "http://status.example.com/%s/instances",
  "envNames": ["dev", "staging", "prod"],
  "refresh_interval": 10
}
```

- `urlPattern` — a format string; `%s` is replaced with each environment name.
- `envNames` — the environments to poll (an `all` entry is added automatically).
- `refresh_interval` — auto-refresh period in seconds.

The status endpoint must return JSON shaped as
`{ "<serviceName>": [ { "build": "...", "hostName": "...", "id": "...", "state": "..." } ] }`.

Build and run:

```bash
go mod download
go run ./cmd            # run directly
# or
go build -o env-monitor ./cmd && ./env-monitor
```

## Usage

There are no command-line arguments; all configuration comes from `config.json`.
Once running, the tool is driven by keyboard.

**Environment select screen**

| Key | Action |
|---|---|
| `↑` / `↓` | Move selection |
| `Enter` | Open the table for the selected environment |
| `q` / `Ctrl+C` | Quit |

**Table screen**

| Key | Action |
|---|---|
| `↑` / `↓` | Move the table cursor |
| `←` / `→` | Switch environments (only in `all` mode) |
| `/` | Start filtering (type text, `Enter` to apply, `Esc` to cancel) |
| `c` | Clear the active filter |
| `s` | Toggle sort column (name/state) and direction |
| `r` | Refresh now |
| `Enter` | Open the detail view for the selected instance |
| `b` | Return from the detail view |
| `Esc` | Blur the table if focused, otherwise quit |
| `q` / `Ctrl+C` | Quit |

## Project structure

```
go-cli-tool/
├── cmd/
│   └── main.go              # Entry point: load config, start Bubble Tea program
├── internals/
│   ├── config/
│   │   └── config.go        # Config struct + ReloadFromFile (JSON)
│   ├── models/
│   │   └── models.go        # ResponseInstanceItem (wire) and InstanceItem (view)
│   └── ui/
│       ├── env_select.go    # Environment-selection model
│       └── table_view.go    # Table model: fetch, filter, sort, detail view
├── utils/
│   └── fetch.go             # HTTP fetch (single + concurrent) and JSON decode
├── go.mod
└── go.sum
```
