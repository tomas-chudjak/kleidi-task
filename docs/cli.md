---
title: CLI Reference
weight: 2
---

The `klt` command-line interface provides full task management from the terminal.

## Project commands

### klt init

Initialize a new project in the current directory. Creates `.tasks/` with a SQLite database.

```bash
klt init
klt init --name "My Project"
```

### klt project list

List all registered projects.

```bash
klt project list
```

### klt project stats

Show task statistics for the current project.

```bash
klt project stats
klt project stats my-project    # specific project by slug
```

## Task commands

### klt add

Create a new task. Type is detected from flags or title prefix.

```bash
klt add "Implement auth"                    # task (default)
klt add --bug "Login broken"                # bug
klt add --feature "Dark mode"               # feature
klt add --hotfix "Fix crash on start"       # hotfix

# Auto-detection from title prefix (prefix is stripped)
klt add "BUG: Login broken"                 # bug
klt add "FEAT: Dark mode"                   # feature
klt add "FEATURE: Dark mode"                # feature
klt add "HOTFIX: Fix crash"                 # hotfix
klt add "TODO: Refactor auth module"        # task
```

Flags:

| Flag | Description |
|------|-------------|
| `--bug` | Create as bug |
| `--feature` | Create as feature |
| `--hotfix` | Create as hotfix |
| `--description`, `-d` | Task description |
| `--priority`, `-p` | Priority (0 = default, higher = more important) |

### klt list

List tasks with optional filters.

```bash
klt list                          # all tasks
klt list --status todo            # only todo
klt list --status doing           # only in progress
klt list --type bug               # only bugs
klt list --status todo --type bug # combined filters
```

Flags:

| Flag | Description |
|------|-------------|
| `--status` | Filter by status: `todo`, `doing`, `done` |
| `--type` | Filter by type: `task`, `bug`, `feature`, `hotfix` |
| `--all` | Include all statuses |

### klt show

Show detailed information about a task.

```bash
klt show 42
```

### klt done

Mark a task as completed.

```bash
klt done 42
```

### klt update

Update task fields.

```bash
klt update 42 --title "New title"
klt update 42 --status doing
klt update 42 --priority 5
klt update 42 --description "Updated description"
```

Flags:

| Flag | Description |
|------|-------------|
| `--title` | New title |
| `--description`, `-d` | New description |
| `--status` | New status: `todo`, `doing`, `done` |
| `--priority`, `-p` | New priority |

### klt delete

Permanently delete a task.

```bash
klt delete 42
```

### klt advance

Advance a task to the next phase in its workflow.

```bash
klt advance 42
```

See [workflows.md](workflows.md) for details on phase-based workflows.

### klt archive / unarchive

Archive a completed task (removes from active views) or restore it.

```bash
klt archive 42
klt unarchive 42
```

## Search and suggestions

### klt suggest

Scan project source code for TODO, FIXME, HACK, and XXX comments. Suggests new tasks and checks for duplicates against existing tasks.

```bash
klt suggest
```

Respects `.gitignore` — only scans tracked and untracked non-ignored files.

## Import / Export

### klt export

Export tasks from the current project.

```bash
klt export                    # JSON to stdout
klt export --format markdown  # Markdown to stdout
klt export > tasks.json       # save to file
```

### klt import

Import tasks from a file.

```bash
klt import tasks.json
klt import tasks.md --format markdown
```

## Server commands

### klt serve

Start the HTTP server with Web UI and REST API.

```bash
klt serve                        # localhost:7842
klt serve --port 8080            # custom port
klt serve --host 0.0.0.0        # bind to all interfaces (for Docker)
```

### klt mcp

Start the MCP server (stdio transport) for AI assistant integration.

```bash
klt mcp
```

This is typically not run directly — it's configured as an MCP server in Claude Desktop, Cursor, or VS Code. See [mcp-usage.md](mcp-usage.md).

## User management

### klt user add

Create a new user. Enables HTTP Basic Auth on the server.

```bash
klt user add tomas
# Password for tomas: ****
```

### klt user list

List registered users.

```bash
klt user list
```

See [authentication.md](authentication.md) for details.

## Utility

### klt backup

Backup the current project's database.

```bash
klt backup
```

### klt version

Show the klt version.

```bash
klt version
```
