# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**kleidi-task** is a local-first, single-binary task tracker for developers who use AI assistants. Binary name is `klt`. Written in Go with SQLite storage.

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

## MCP Wiring (how clients connect)

The MCP server is **stdio-only** (`internal/mcp/server.go` — `mcp.StdioTransport`). There is no daemon and no HTTP/SSE MCP endpoint; `klt serve` mounts only the REST API and UI.

Consequences:

- Every MCP client (each Claude Code session, Claude Desktop, Cursor) **spawns its own `klt mcp` child process** and talks to it over stdin/stdout. N sessions = N short-lived processes, each dying with its client. Running `klt` "in the background" shares nothing.
- What is actually shared between sessions is the **SQLite state on disk**: `~/.tasks/registry.db` + each project's `.tasks/tasks.db`.
- **Project resolution:** tools take an optional `project` (slug or `current`). Without it, the server walks up from the MCP process's cwd looking for `.tasks/` (`ProjectService.DetectProject`, same pattern as Git). With a slug, any session can reach any registered project via the registry — cross-project access works from anywhere.
- Requirement for a client: `klt` on `$PATH` (installed at `/usr/local/bin/klt`) and one MCP server entry. Canonical config is the user-scope `~/.claude.json` top-level `mcpServers` under the name `kleidi` → `klt mcp`. Keep exactly one entry; duplicates under other names just spawn redundant processes with duplicate tool sets.

## Task List Output Contract (MANDATORY)

Every surface renders task lists through `internal/render` — one column definition, one placeholder, one order. Canonical columns:

```
# | type | status | pri | category | title
```

Rows keep the service-layer order (`priority DESC, created_at DESC`). Unset priority/category render as `–` (`render.EmptyCell`). The header line is `**<project>** — N open, M done`, and pagination appends `Page X/Y (total: Z)` only when there is more than one page.

Parent/child structure rides **inside the title cell** (`render.Title`), never as a column of its own — a parent shows `Title (done/total)`, a child is prefixed with `render.ChildPrefix`. Splitting a task therefore never widens the table.

**When answering "what tasks do we have":** call `task_list`, then **print the tool's text block verbatim**. Do not rebuild it as bullets, do not add or drop columns, do not re-sort rows, do not replace it with a prose summary. Commentary goes *after* the table, never instead of it. Use the structured `tasks` array for follow-up tool calls; use the text block for anything the user sees.

The same rendering is reachable from the terminal:

```bash
klt list                # aligned table for humans
klt list --format md    # byte-identical to what MCP returns
klt list --format json  # raw tasks for scripts
```

## Spec-First Task Flow

A task description is a spec that accumulates, not a text written once at creation. Four mechanisms enforce that:

1. **Template enforcement on create** — see the Task Workflow section below.
2. **`## Open questions`** — `task` and `feature` templates carry this section. Anything an agent cannot answer from context goes there as a question, never as a "TBD" placeholder buried in another section. Templates also carry `agent_rules`, returned by `template_get` alongside the skeleton and never written into a task description.
3. **`task_review` / `klt review <id>`** — the checkpoint before implementation. Reports missing and empty sections plus the type's review instruction, and performs **no writes**: the author accepts findings, which are then applied with `task_update`.
4. **Phase outputs** — a workflow phase may declare a description section its work must land in (`workflows.phase_outputs`; feature `research → ## Design`, bug `reproducing → ## Root cause`). `task_advance` refuses to move past such a phase while that section is empty, so a design phase cannot leave its conclusion in the chat.

Large work is split rather than described once:

```bash
klt split 42 "Schema and migration" "Service layer" "MCP and CLI wiring"
```

Children inherit the parent's type, priority and category, are ordered by `child_order`, and are limited to **one level** — a child cannot be split further. A parent cannot be completed while any child is open. The MCP equivalent is `task_split`; both call `TaskService.Split`.

Section editing goes through `internal/core/sections.go` (`SectionContent`, `HasSectionContent`, `UpsertSection`, `AppendToSection`) — never rewrite a whole description to change one section.

Changing a column means changing `render.TaskColumns` — then mirroring the order in `internal/ui/templates/project.templ` (`TaskList`/`TaskRow`) and the `grid-template-columns` rules in `internal/ui/static/css/style.css`. Never format a task list ad-hoc in a handler.

## Build & Development Commands

```bash
# Build
task build              # Build the klt binary (templ + go build)
go build ./cmd/klt      # Direct Go build (skip templ)

# Test
go test ./...           # Run all tests
go test ./internal/core # Run tests for a specific package

# SQL code generation (after modifying .sql query files)
sqlc generate

# Database migrations are embedded via go:embed — never execute SQL ad-hoc

# Run
klt init                # Initialize .tasks/ in current directory
klt serve               # Start HTTP server (UI + REST) — no MCP endpoint
klt mcp                 # Start stdio MCP server (one process per MCP client)
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

When working on a kleidi-task task, you MUST follow the task workflow:

1. **Before starting work** — call `task_get` to read the task, its workflow context (current phase, phase instruction, next phase), and template-enriched description
2. **Follow the phase instruction** — each workflow phase has a prompt/instruction. Execute what the phase requires before advancing
3. **Advance when done** — after completing the current phase, call `task_advance` to move to the next phase. Read the suggested skills and next phase instruction
4. **Repeat until complete** — continue through all phases until the workflow is finished
5. **Template-driven descriptions** — when creating a task, ALWAYS call `template_get(type)` first to fetch the template for the task type. Fill in each template section with relevant content based on the task context, then pass the completed template as the `description` to `task_create`

This is enforced at the service layer, not by prompt alone. `TaskService.Create` validates the description against the template's `##` section headings for the task's type. The policy is the per-project `template_enforcement` setting:

| Mode | Behaviour |
|---|---|
| `off` | no validation |
| `warn` | logs missing sections via slog, creates the task (default — an upgrade never starts rejecting) |
| `strict` | rejects `mcp` and `api` creates with `ErrInvalidInput` naming the missing sections |

`cli` and `ui` are interactive sources: a human typing is not skipping the flow, so they are never rejected, and an empty description is filled with the template skeleton instead. An MCP create with an empty description is rejected rather than scaffolded — otherwise the agent could skip `template_get` and still pass. A task type with no template is never constrained.

```bash
klt config get                                 # show current settings
klt config set template_enforcement strict     # this repo runs strict
```

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
- **NO manual DB changes — EVER.** All schema changes go through goose migration files. All data changes go through klt CLI/API/MCP. sqlite3 CLI is read-only for debugging. No ad-hoc SQL execution against production or dev databases.
- **Service layer is the single source of truth** — all 4 entry points call the same services, no duplicated business logic
- **Project detection** follows Git pattern: walk up from cwd looking for `.tasks/` directory
- **All static assets embedded** via `go:embed` — templates, CSS, JS are in the binary
- **ahoylog-css additions** (badge, modal, dropdown, toast, spinner, empty) are PRs to the separate `ahoylog/ahoylog-css` repo, not part of this repo

## Project Structure (key paths)

- `cmd/klt/main.go` — entry point
- `internal/core/` — service layer (business logic, domain types, errors)
- `internal/db/` — DB manager, migrations (`project/` and `registry/`), queries, generated code
- `internal/mcp/` — MCP server, tools, resources (stdio transport only)
- `internal/cli/` — cobra commands
- `internal/api/` — chi REST API with middleware and handlers
- `internal/ui/` — HTMX UI handlers, `.templ` templates, vendored static assets
- `internal/config/` — koanf configuration
- `claude-skill/` — pre-built Claude skill for MCP integration

## Implementation Order

Follow the roadmap: v0.1 (CLI + MCP) → v0.2 (UI + REST) → v0.3 (Polish) → v0.4 (AI-native).

Within v0.1: service layer first → CLI → MCP → binary build.

## Naming

- **kleidi-task** = project name, repo, package, docs
- **klt** = binary name, CLI command, daily usage
- **tasks** = internal directory name (`.tasks/`), URI scheme (`tasks://`)

## Open Decisions

- Timezone handling (UTC storage, local display?)
- Soft delete (MVP: hard delete; v0.4+: soft delete)
- Project rename strategy (slug has UNIQUE constraint, INTEGER PK used)

## Reference

`PROJECT.md` is the authoritative architectural document. `DEV.md` is the authoritative developer guide (build, run, DB inspection, testing). In case of conflict between docs and code, **code is truth** and docs should be updated.
