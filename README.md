# kvik-tasks

Local-first, single-binary task tracker for developers who use AI assistants.

**Binary name:** `kvt`

## What is this?

kvik-tasks is a task tracker designed with AI integration as a first-class feature. It runs entirely on your machine — no cloud, no vendor lock-in. The built-in MCP server lets Claude Desktop, Cursor, and other AI assistants create, list, and manage your tasks natively.

## Features

- **Single Go binary** — download and run, no dependencies
- **SQLite per-project** — tasks live with your project, optionally in Git
- **MCP-first** — native Model Context Protocol server for AI assistants
- **CLI** — fast terminal interface for daily use
- **REST API** — for custom integrations (v0.2)
- **Web UI** — HTMX-based dashboard (v0.2)

## Quick start

### Install

```bash
# From source
go install github.com/ahoylog/kvik-tasks/cmd/kvt@latest

# Or download binary from releases
# https://github.com/ahoylog/kvik-tasks/releases
```

### Initialize a project

```bash
cd ~/projects/my-app
kvt init --name "My App"
```

### Use the CLI

```bash
kvt add "Implement user auth"           # add a task
kvt add "CSS broken on mobile" --bug    # add a bug
kvt add --feature "Dark mode support"    # add a feature
kvt add --hotfix "Fix crash on start"    # add a hotfix
kvt add "BUG: login broken"             # auto-detected from prefix
kvt add "FEAT: dark mode"               # auto-detected (also: FEATURE:, HOTFIX:, TASK:)
kvt list                                 # list all tasks
kvt list --status todo --type bug        # filter
kvt done 1                               # mark as done
kvt show 1                               # task details
kvt update 1 --status doing --priority 5 # update fields
kvt delete 2                             # delete permanently
kvt project list                         # all projects
kvt project stats                        # todo/doing/done counts
```

### Connect to Claude Desktop

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

Now Claude can create tasks, list bugs, and track your work across sessions.

## How it works

```
You (CLI / Browser / Claude)
    ↓
  kvt binary
    ↓
  Service Layer
    ↓
  SQLite
    ├── ~/.tasks/registry.db      (global project registry)
    └── <project>/.tasks/tasks.db (per-project tasks)
```

Each project gets its own SQLite database in `.tasks/`. A global registry at `~/.tasks/registry.db` maps project slugs to paths.

## MCP Tools

| Tool | Description |
|---|---|
| `task_create` | Create a new task or bug |
| `task_list` | List tasks with filters (status, type, project) |
| `task_get` | Get task details |
| `task_update` | Update task fields |
| `task_complete` | Mark task as done |
| `task_delete` | Delete a task |
| `project_list` | List all projects |
| `project_current` | Detect current project from cwd |
| `project_stats` | Get task statistics |

## Development

See [DEV.md](DEV.md) for build instructions, database inspection, and testing guide.

```bash
task build     # build the binary
task test      # run tests
task sqlc      # regenerate SQL code
```

## Architecture

See [PROJECT.md](PROJECT.md) for the full architectural document.

## License

MIT
