# DEV.md — Developer Guide

> Authoritative source for build, run, database inspection, and testing.
> Update this document whenever the build pipeline or tooling changes.

---

## 1. Prerequisites

```bash
# Go 1.22+
go version    # must be >= 1.22

# sqlc — type-safe Go code generator from SQL
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc version

# templ — type-safe Go templates for UI
go install github.com/a-h/templ/cmd/templ@latest
templ version

# goose — migrations (we use embedded, but CLI is useful for debugging)
go install github.com/pressly/goose/v3/cmd/goose@latest
goose --version

# sqlite3 CLI — database inspection (read-only!)
# Ubuntu/Debian:
sudo apt install sqlite3
# macOS:
brew install sqlite3
# Verify:
sqlite3 --version
```

---

## 2. Build

```bash
# Via Taskfile (preferred)
task build

# Direct Go build
go build -o kvt ./cmd/kvt

# Verify
./kvt version
```

### Build pipeline (what happens)

```
1. templ generate          → compiles .templ → .go files
2. go build ./cmd/kvt      → compiles binary with embedded assets
```

> **Note:** `task build` runs both steps. If you're only changing Go code (not .templ), `go build` alone is sufficient.

---

## 3. Initialization and running

### First run (new project)

```bash
# Navigate to your project directory
cd ~/projects/my-app

# Initialize kvik-tasks in this project
kvt init --name "My App"

# Result:
#   .tasks/
#   └── tasks.db          ← per-project SQLite database
#
#   ~/.tasks/
#   └── registry.db       ← global registry (created automatically on first init)
```

### CLI usage

```bash
kvt add "Implement login"                  # add a task
kvt add "CSS bug on mobile" --bug          # add a bug
kvt list                                    # list tasks for current project
kvt list --status todo                      # filter by status
kvt done 1                                  # mark task #1 as done
kvt show 1                                  # task detail
```

### HTTP server (UI + REST API)

```bash
kvt serve                    # default port 7842
kvt serve --port 8080        # custom port

# Endpoints:
#   http://localhost:7842/           → Web UI (HTMX)
#   http://localhost:7842/api/v1/    → REST API
```

### MCP server (for Claude Desktop / Cursor)

```bash
kvt mcp                      # stdio transport (for AI clients)
```

Claude Desktop configuration (`~/.claude/claude_desktop_config.json`):
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

---

## 4. Database

### File locations

| Database | Path | Purpose |
|---|---|---|
| **Per-project DB** | `<project>/.tasks/tasks.db` | Tasks and bugs for this project |
| **Global registry** | `~/.tasks/registry.db` | Slug → path mapping for all projects |

### Connecting and inspection

```bash
# Per-project database (from project root)
sqlite3 .tasks/tasks.db

# Global registry
sqlite3 ~/.tasks/registry.db
```

> **STRICTLY FORBIDDEN:** Any manual database modifications (INSERT, UPDATE, DELETE, ALTER, DROP) via sqlite3 CLI or any other tool. All schema changes go through goose migration files. All data changes go through `kvt` CLI / API / MCP. The sqlite3 CLI is for **reading and debugging only**.

### Useful SQL queries

#### Tasks

```sql
-- All active tasks (todo + doing), ordered by priority
SELECT id, type, title, status, priority, source, created_at
FROM tasks
WHERE status != 'done'
ORDER BY priority DESC, created_at DESC;

-- Open bugs
SELECT id, title, status, priority FROM tasks WHERE type = 'bug' AND status != 'done';

-- Statistics
SELECT status, COUNT(*) as count FROM tasks GROUP BY status;

-- Last 10 completed
SELECT id, title, completed_at FROM tasks
WHERE status = 'done'
ORDER BY completed_at DESC
LIMIT 10;

-- Tasks by source (where they were created from)
SELECT source, COUNT(*) FROM tasks GROUP BY source;
```

#### Registry

```sql
-- All registered projects
SELECT id, slug, name, path, cached_todo_count, cached_doing_count FROM projects;

-- Check paths
SELECT slug, path FROM projects ORDER BY last_seen_at DESC;
```

### SQLite tips

```bash
# Formatted output
sqlite3 -header -column .tasks/tasks.db "SELECT * FROM tasks;"

# Table schema
sqlite3 .tasks/tasks.db ".schema tasks"

# Export to CSV
sqlite3 -header -csv .tasks/tasks.db "SELECT * FROM tasks;" > export.csv

# Check WAL mode — kvt can run concurrently with sqlite3 (safe reads)
sqlite3 .tasks/tasks.db "PRAGMA journal_mode;"
```

> **Note:** When `kvt serve` is running, sqlite3 CLI can safely read data (WAL mode).

---

## 5. Development — sqlc workflow

After changing SQL queries:

```bash
# 1. Edit SQL in internal/db/queries/*.sql
# 2. Generate Go code
sqlc generate
# 3. Output goes to internal/db/generated/
```

### sqlc configuration

```bash
# Verify sqlc.yaml is valid
sqlc verify

# Full regeneration
sqlc generate
```

---

## 6. Testing

### Run all tests

```bash
go test ./...
```

### Tests by layer

```bash
# Service layer (business logic)
go test ./internal/core/...

# Database (queries, migrations, manager)
go test ./internal/db/...

# CLI
go test ./internal/cli/...

# MCP server
go test ./internal/mcp/...

# REST API
go test ./internal/api/...

# UI handlers
go test ./internal/ui/...

# Config
go test ./internal/config/...
```

### Verbose and coverage

```bash
# Verbose — see each test individually
go test -v ./internal/core/...

# With coverage report
go test -cover ./...

# Coverage HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
# Open coverage.html in browser
```

### Running a specific test

```bash
# By name (regex)
go test -v -run "TestTaskCreate" ./internal/core/...

# All tests containing "Bug"
go test -v -run "Bug" ./...
```

### Post-feature checklist

```bash
# 1. All tests pass
go test ./...

# 2. Build works
task build

# 3. Manual smoke test
./kvt init --name "test-project"
./kvt add "Test task"
./kvt add "Test bug" --bug
./kvt list
./kvt done 1
./kvt list --status done

# 4. Database looks correct
sqlite3 .tasks/tasks.db "SELECT * FROM tasks;"

# 5. Cleanup test project
rm -rf .tasks/
```

---

## 7. Taskfile overview

We use [Task](https://taskfile.dev) (Go-native task runner) instead of Make.

```bash
task              # show available tasks
task build        # templ generate + go build
task test         # go test ./...
task test:verbose # tests with verbose output
task test:cover   # tests with coverage HTML report
task sqlc         # sqlc generate
task clean        # clean build artifacts
task lint         # go vet
task check        # lint + test
task dev          # build + run kvt serve with hot-reload (air)
```

---

## 8. Troubleshooting

### "kvt init" reports registry.db error
```bash
# Check if ~/.tasks/ exists
ls -la ~/.tasks/
# If not, kvt init creates it automatically. If permissions are the issue:
mkdir -p ~/.tasks && chmod 755 ~/.tasks
```

### sqlite3 reports "database is locked"
```bash
# Stop kvt serve, then access the database
# Or use read-only mode:
sqlite3 -readonly .tasks/tasks.db "SELECT * FROM tasks;"
```

### sqlc generate fails
```bash
# Check sqlc.yaml
sqlc verify
# Check that SQL syntax is valid
sqlite3 :memory: < internal/db/queries/tasks.sql
```
