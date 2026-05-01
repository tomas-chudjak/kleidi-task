# kvik-tasks

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-embedded-003B57?logo=sqlite&logoColor=white)](https://www.sqlite.org)

Local-first, single-binary task tracker for developers who use AI assistants. Built with MCP-first design — the AI integration is a primary interface, not an afterthought.

## Why kvik-tasks?

Existing task managers are designed for humans clicking buttons. kvik-tasks is designed for developers who work alongside AI assistants like Claude, Cursor, and Copilot. Tasks live with your project (per-project SQLite), sync across AI sessions via MCP, and everything runs locally — no cloud, no vendor lock-in.

## Features

**Core**
- Single Go binary, no external dependencies
- Per-project SQLite databases with global registry
- CLI, REST API, Web UI, and MCP server — all sharing one service layer

**Task Management**
- Task types: task, bug, feature, hotfix, plus custom types
- Full-text search (FTS5)
- Categories, priorities, bulk operations
- Task templates for quick creation
- Export/import as JSON or Markdown

**AI Integration**
- MCP server (stdio) for Claude Desktop, Cursor, VS Code
- Task workflows with phase-based AI prompts
- Source code scanning for TODO/FIXME/HACK comments
- Claude skill with auto-parse title prefixes

**Web UI**
- Dashboard with project overview and stats
- Kanban board with drag-and-drop
- Vim-like keyboard shortcuts
- Markdown editor for descriptions
- Workflow editor with trigger configuration

**Infrastructure**
- Docker deployment with compose
- HTTP Basic Auth for teams
- Script hooks on task lifecycle events
- Git commit to task linking
- VS Code extension with task sidebar

## Installation

### Go install

```bash
go install github.com/tomas-chudjak/kvik-tasks/cmd/kvt@latest
```

### Build from source

```bash
git clone https://github.com/tomas-chudjak/kvik-tasks.git
cd kvik-tasks
task setup    # installs templ, sqlc, goose, air
task build    # builds the kvt binary
task install  # symlinks to /usr/local/bin
```

Requires Go 1.22+.

### Docker

```bash
git clone https://github.com/tomas-chudjak/kvik-tasks.git
cd kvik-tasks
docker compose up -d
```

Web UI available at http://localhost:7842. See [docs/docker.md](docs/docker.md) for details.

## Quick start

```bash
# Initialize a project
cd my-project
kvt init

# Add tasks
kvt add "Implement user authentication"
kvt add "BUG: Login fails on Firefox"        # auto-detected as bug
kvt add --feature "Dark mode support"

# View tasks
kvt list
kvt list --status todo --type bug

# Work on tasks
kvt done 1

# Start web UI
kvt serve
# Open http://localhost:7842
```

## MCP setup

Add kvik-tasks as an MCP server in your AI tool:

### Claude Desktop

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "kvik-tasks": {
      "command": "kvt",
      "args": ["mcp"]
    }
  }
}
```

### Claude Code

```bash
claude mcp add --scope project --transport stdio kvik-tasks -- kvt mcp
```

### Cursor

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "kvik-tasks": {
      "command": "kvt",
      "args": ["mcp"]
    }
  }
}
```

Once connected, try: "task: implement search feature" or "show my tasks".

See [docs/mcp-usage.md](docs/mcp-usage.md) for the full MCP tool reference.

## CLI reference

| Command | Description |
|---------|-------------|
| `kvt init` | Initialize `.tasks/` in current directory |
| `kvt add <title>` | Create a task (`--bug`, `--feature`, `--hotfix`, or prefix detection) |
| `kvt list` | List tasks (`--status`, `--type` filters) |
| `kvt show <id>` | Show task detail |
| `kvt done <id>` | Mark task as complete |
| `kvt update <id>` | Update task fields |
| `kvt delete <id>` | Delete a task |
| `kvt advance <id>` | Advance task to next workflow phase |
| `kvt archive <id>` | Archive a completed task |
| `kvt suggest` | Scan source code for TODO/FIXME comments |
| `kvt export` | Export tasks as JSON or Markdown |
| `kvt import <file>` | Import tasks from file |
| `kvt serve` | Start HTTP server (UI + API) |
| `kvt mcp` | Start MCP stdio server |
| `kvt user add <name>` | Add a user (enables Basic Auth) |
| `kvt backup` | Backup project database |
| `kvt version` | Show version |

See [docs/cli.md](docs/cli.md) for all flags and examples.

## How it works

```
You (CLI / Browser / Claude)
    |
  kvt binary
    |
  Service Layer (TaskService, ProjectService, WorkflowService)
    |
  SQLite
    |-- ~/.tasks/registry.db      (global project registry + users)
    '-- <project>/.tasks/tasks.db (per-project tasks)
```

Each project gets its own SQLite database in `.tasks/`. A global registry at `~/.tasks/registry.db` maps project slugs to paths.

## Documentation

| Document | Description |
|----------|-------------|
| [Installation](docs/installation.md) | Install and build guide |
| [CLI Reference](docs/cli.md) | All commands, flags, and examples |
| [REST API](docs/rest-api.md) | API endpoints and curl examples |
| [MCP Integration](docs/mcp-usage.md) | MCP tools and AI assistant setup |
| [Configuration](docs/configuration.md) | Global and project config options |
| [Workflows](docs/workflows.md) | Phase-based task workflows |
| [Authentication](docs/authentication.md) | HTTP Basic Auth for teams |
| [Docker](docs/docker.md) | Container deployment |
| [Script Hooks](docs/hooks.md) | Task lifecycle event hooks |
| [Git Integration](docs/git-integration.md) | Commit to task linking |
| [VS Code Extension](docs/vscode-extension.md) | Task sidebar for VS Code/Cursor |

## Tech stack

Go, SQLite (modernc.org/sqlite), sqlc, chi, cobra, HTMX, templ, ahoylog-css, MCP Go SDK, SortableJS.

## License

MIT
