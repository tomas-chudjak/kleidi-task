# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**kvik-tasks** is a local-first, single-binary task tracker for developers who use AI assistants. Binary name is `kvt`. Written in Go with SQLite storage.

Key differentiator: MCP-first design — the MCP server is a primary interface, not a wrapper around the REST API. All interfaces (MCP/CLI/REST/UI) share a single service layer.

## Architecture

```
External clients (Claude Desktop, Cursor, Browser)
    ↓ MCP(stdio) / HTTP
Single Go binary
    ↓
4 entry points: MCP Server | CLI (cobra) | REST API (chi) | UI Server (HTMX)
    ↓
Service Layer (TaskService, ProjectService, SearchService)
    ↓
Repository Layer (sqlc generated)
    ↓
DB Manager (multi-DB routing)
    ↓
Registry SQLite (~/.tasks/registry.db) + Per-project SQLite (.tasks/tasks.db)
```

**Per-project DB with global registry:** Each project has its own SQLite DB in `.tasks/tasks.db`. The global registry at `~/.tasks/registry.db` only maps `slug → path`. Cross-project queries require aggregation.

**Work item types:** Single `tasks` table with `type` column: `task`, `bug`, `feature`, `hotfix`. Title prefix auto-detection (e.g., "BUG: title" → bug, "FEAT: title" → feature).

## Build & Development Commands

```bash
# Build
task build              # Build the kvt binary (templ + go build)
go build ./cmd/kvt      # Direct Go build (skip templ)

# Test
go test ./...           # Run all tests
go test ./internal/core # Run tests for a specific package

# SQL code generation (after modifying .sql query files)
sqlc generate

# Database migrations are embedded via go:embed — never execute SQL ad-hoc

# Run
kvt init                # Initialize .tasks/ in current directory
kvt serve               # Start HTTP server (UI + REST + MCP HTTP)
kvt mcp                 # Start stdio MCP server
```

## Tech Stack

- **Go 1.22+** — single binary, no CGO (uses `modernc.org/sqlite`)
- **sqlc** — type-safe Go from SQL (queries in `internal/db/queries/`, output in `internal/db/generated/`)
- **goose** — embedded migrations
- **chi v5** — HTTP router
- **cobra** — CLI framework
- **modelcontextprotocol/go-sdk** — MCP server implementation
- **koanf v2** — configuration
- **slog** — logging (stdlib)
- **templ** — type-safe Go templates for UI
- **HTMX + ahoylog-css** — server-rendered UI with `@ahoylog/ahoylog-css` design system
- **SortableJS** — kanban drag & drop

## Task Workflow (MANDATORY)

When working on a kvik-tasks task, you MUST follow the task workflow:

1. **Before starting work** — call `task_get` to read the task, its workflow context (current phase, phase instruction, next phase), and template-enriched description
2. **Follow the phase instruction** — each workflow phase has a prompt/instruction. Execute what the phase requires before advancing
3. **Advance when done** — after completing the current phase, call `task_advance` to move to the next phase. Read the suggested skills and next phase instruction
4. **Repeat until complete** — continue through all phases until the workflow is finished
5. **Template-driven descriptions** — when creating a task (`task_create`), if a template exists for the task type, fill in each template section with relevant content and call `task_update` to save it

Example flow for a feature task:
```
task_get #66 → read workflow (phase: "todo", instruction: "...")
  → do the work for this phase
task_advance #66 → moves to next phase, returns new instruction
  → do the work for next phase
task_advance #66 → ... until complete
```

Never skip phases. Never ignore phase instructions. The workflow is the source of truth for how work progresses on a task.

## Key Conventions

- **sqlc workflow:** modify SQL in `internal/db/queries/` → run `sqlc generate` → use generated code in `internal/db/generated/`
- **NO manual DB changes — EVER.** All schema changes go through goose migration files. All data changes go through kvt CLI/API/MCP. sqlite3 CLI is read-only for debugging. No ad-hoc SQL execution against production or dev databases.
- **Service layer is the single source of truth** — all 4 entry points call the same services, no duplicated business logic
- **Project detection** follows Git pattern: walk up from cwd looking for `.tasks/` directory
- **All static assets embedded** via `go:embed` — templates, CSS, JS are in the binary
- **ahoylog-css additions** (badge, modal, dropdown, toast, spinner, empty) are PRs to the separate `ahoylog/ahoylog-css` repo, not part of this repo

## Project Structure (key paths)

- `cmd/kvt/main.go` — entry point
- `internal/core/` — service layer (business logic, domain types, errors)
- `internal/db/` — DB manager, migrations (`project/` and `registry/`), queries, generated code
- `internal/mcp/` — MCP server, tools, resources, transports
- `internal/cli/` — cobra commands
- `internal/api/` — chi REST API with middleware and handlers
- `internal/ui/` — HTMX UI handlers, `.templ` templates, vendored static assets
- `internal/config/` — koanf configuration
- `claude-skill/` — pre-built Claude skill for MCP integration

## Implementation Order

Follow the roadmap: v0.1 (CLI + MCP) → v0.2 (UI + REST) → v0.3 (Polish) → v0.4 (AI-native).

Within v0.1: service layer first → CLI → MCP → binary build.

## Naming

- **kvik-tasks** = project name, repo, package, docs
- **kvt** = binary name, CLI command, daily usage
- **tasks** = internal directory name (`.tasks/`), URI scheme (`tasks://`)

## Open Decisions

- Timezone handling (UTC storage, local display?)
- Soft delete (MVP: hard delete; v0.4+: soft delete)
- Project rename strategy (slug has UNIQUE constraint, INTEGER PK used)

## Reference

`PROJECT.md` is the authoritative architectural document. `DEV.md` is the authoritative developer guide (build, run, DB inspection, testing). In case of conflict between docs and code, **code is truth** and docs should be updated.
