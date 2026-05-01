---
title: CLI Reference
weight: 2
---

The `kvt` command-line interface provides full task management from the terminal.

## Project commands

### kvt init

Initialize a new project in the current directory. Creates `.tasks/` with a SQLite database.

```bash
kvt init
kvt init --name "My Project"
```

### kvt project list

List all registered projects.

```bash
kvt project list
```

### kvt project stats

Show task statistics for the current project.

```bash
kvt project stats
kvt project stats my-project    # specific project by slug
```

## Task commands

### kvt add

Create a new task. Type is detected from flags or title prefix.

```bash
kvt add "Implement auth"                    # task (default)
kvt add --bug "Login broken"                # bug
kvt add --feature "Dark mode"               # feature
kvt add --hotfix "Fix crash on start"       # hotfix

# Auto-detection from title prefix (prefix is stripped)
kvt add "BUG: Login broken"                 # bug
kvt add "FEAT: Dark mode"                   # feature
kvt add "FEATURE: Dark mode"                # feature
kvt add "HOTFIX: Fix crash"                 # hotfix
kvt add "TODO: Refactor auth module"        # task
```

Flags:

| Flag | Description |
|------|-------------|
| `--bug` | Create as bug |
| `--feature` | Create as feature |
| `--hotfix` | Create as hotfix |
| `--description`, `-d` | Task description |
| `--priority`, `-p` | Priority (0 = default, higher = more important) |

### kvt list

List tasks with optional filters.

```bash
kvt list                          # all tasks
kvt list --status todo            # only todo
kvt list --status doing           # only in progress
kvt list --type bug               # only bugs
kvt list --status todo --type bug # combined filters
```

Flags:

| Flag | Description |
|------|-------------|
| `--status` | Filter by status: `todo`, `doing`, `done` |
| `--type` | Filter by type: `task`, `bug`, `feature`, `hotfix` |
| `--all` | Include all statuses |

### kvt show

Show detailed information about a task.

```bash
kvt show 42
```

### kvt done

Mark a task as completed.

```bash
kvt done 42
```

### kvt update

Update task fields.

```bash
kvt update 42 --title "New title"
kvt update 42 --status doing
kvt update 42 --priority 5
kvt update 42 --description "Updated description"
```

Flags:

| Flag | Description |
|------|-------------|
| `--title` | New title |
| `--description`, `-d` | New description |
| `--status` | New status: `todo`, `doing`, `done` |
| `--priority`, `-p` | New priority |

### kvt delete

Permanently delete a task.

```bash
kvt delete 42
```

### kvt advance

Advance a task to the next phase in its workflow.

```bash
kvt advance 42
```

See [workflows.md](workflows.md) for details on phase-based workflows.

### kvt archive / unarchive

Archive a completed task (removes from active views) or restore it.

```bash
kvt archive 42
kvt unarchive 42
```

## Search and suggestions

### kvt suggest

Scan project source code for TODO, FIXME, HACK, and XXX comments. Suggests new tasks and checks for duplicates against existing tasks.

```bash
kvt suggest
```

Respects `.gitignore` — only scans tracked and untracked non-ignored files.

## Import / Export

### kvt export

Export tasks from the current project.

```bash
kvt export                    # JSON to stdout
kvt export --format markdown  # Markdown to stdout
kvt export > tasks.json       # save to file
```

### kvt import

Import tasks from a file.

```bash
kvt import tasks.json
kvt import tasks.md --format markdown
```

## Server commands

### kvt serve

Start the HTTP server with Web UI and REST API.

```bash
kvt serve                        # localhost:7842
kvt serve --port 8080            # custom port
kvt serve --host 0.0.0.0        # bind to all interfaces (for Docker)
```

### kvt mcp

Start the MCP server (stdio transport) for AI assistant integration.

```bash
kvt mcp
```

This is typically not run directly — it's configured as an MCP server in Claude Desktop, Cursor, or VS Code. See [mcp-usage.md](mcp-usage.md).

## User management

### kvt user add

Create a new user. Enables HTTP Basic Auth on the server.

```bash
kvt user add tomas
# Password for tomas: ****
```

### kvt user list

List registered users.

```bash
kvt user list
```

See [authentication.md](authentication.md) for details.

## Utility

### kvt backup

Backup the current project's database.

```bash
kvt backup
```

### kvt version

Show the kvt version.

```bash
kvt version
```
