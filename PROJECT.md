# kleidi-task — Strategic Architecture Document

> **Version:** 0.1 (draft)
> **Status:** Draft for implementation
> **Repository:** `tomas-chudjak/kleidi-task` *(provisional — final GitHub org to be confirmed before repo creation)*
> **Binary name:** `klt`
> **License:** MIT
> **Goal:** Local task tracker with first-class Claude integration via MCP

---

## Naming conventions

This project has **two names** serving different purposes:

| Name | Usage | Examples |
|---|---|---|
| **`kleidi-task`** | Project name, repo, package name, documentation | `github.com/tomas-chudjak/kleidi-task`, `@ahoylog/kleidi-task` (npm), README header, web `kleidi-task.dev` |
| **`klt`** | Binary name, CLI command, daily usage | `klt add "..."`, `klt list`, `cmd/klt/main.go`, Homebrew formula installs as `klt` |

**Rationale:** The project needs a **searchable, distinctive name** (`kleidi-task`) for discovery and brand. Users search for it on Google, npm registry, GitHub. But they type a **short command** (`klt`) daily — dozens to hundreds of times. This dual-name pattern is common (`kubernetes` repo → `kubectl` binary, `terraform` project → `tf` alias).

In this document:
- "kleidi-task" means the project as a whole (architecture, repo, documentation)
- "klt" means the CLI binary (commands, paths in `cmd/`)

---

## 1. Executive Summary

**kleidi-task** is a local, single-binary task tracker for developers who work with multiple projects and use AI assistants (Claude, Cursor, etc.) as part of their workflow.

It solves a specific problem: AI assistants today lack a good way to track tasks and bugs across sessions and projects. Markdown files are a first step but quickly outgrow their usefulness. Existing task managers (Linear, Jira, Vikunja) are designed for humans, not for AI integration.

**Key features:**
- Single Go binary, no external runtime dependency
- SQLite per-project DB (tasks live with the project)
- MCP-first integration — native support for Claude Desktop, Cursor, Continue
- CLI for quick terminal usage
- REST API for custom integrations
- HTMX-based web UI built on **ahoylog-css** (Ahoylog design system)
- Local-first — everything runs on your machine, no cloud

**Target audience:** Developers managing multiple personal/work projects who want an AI-friendly task tracking tool without committing to a SaaS platform.

---

## 2. Problem and Motivation

### 2.1 Problem

Developers using AI assistants currently have these options for task management:

1. **Markdown files** (`Tasks.md`, `Bugs.md`) — simple but don't scale. AI can't efficiently read/edit tables, hard to filter, no cross-project view.
2. **SaaS task managers** (Linear, Jira, Asana) — powerful but closed source, vendor lock-in, poor AI integration, online-only.
3. **Self-hosted tools** (Vikunja, Plane, Focalboard) — open source but designed for human workflows. No native MCP support, harder AI integrations.
4. **CLI tools** (Taskwarrior) — local, fast, but lack project structure and modern AI integration.

### 2.2 Hypothesis

There is room for an **AI-native, local-first task tracker** that:
- Runs locally (no cloud, no vendor lock-in)
- Has an MCP server built in from the start (not as an add-on)
- Keeps tasks with the project (per-project DB, optionally in Git)
- Is simple enough for one person to use across dozens of projects
- Is modern enough to become a pleasant alternative for teams

### 2.3 Non-goals

To stay focused:
- **Not trying to replace Jira/Linear for large teams.** Multi-tenant SaaS with advanced permissions is not the goal.
- **Not trying to be a knowledge management tool.** Notion, Logseq, Obsidian solve a different problem.
- **Not trying to be a general workflow engine.** Temporal, n8n, Argo solve different problems.

---

## 3. Architecture (high-level)

```
┌─────────────────────────────────────────────────────────────────┐
│                      EXTERNAL CLIENTS                            │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Claude     │  │   Cursor /   │  │   Browser    │          │
│  │   Desktop    │  │   Continue   │  │   (UI)       │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │ MCP             │ MCP              │ HTTP             │
│         │ (stdio)         │ (stdio)          │                  │
└─────────┼─────────────────┼──────────────────┼──────────────────┘
          │                 │                  │
          │                 │                  │
┌─────────▼─────────────────▼──────────────────▼──────────────────┐
│                                                                  │
│              kleidi-task BINARY (single Go binary)               │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │   MCP    │  │   CLI    │  │   REST   │  │    UI    │        │
│  │  Server  │  │   Cmds   │  │   API    │  │  Server  │        │
│  │ (stdio)  │  │ (cobra)  │  │  (chi)   │  │ (HTMX)   │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
│       └─────────────┴─────────────┴─────────────┘               │
│                          │                                       │
│              ┌───────────▼───────────┐                           │
│              │   SERVICE LAYER       │                           │
│              │   (business logic)    │                           │
│              │                       │                           │
│              │ - TaskService         │                           │
│              │ - ProjectService      │                           │
│              │ - SearchService       │                           │
│              └───────────┬───────────┘                           │
│                          │                                       │
│              ┌───────────▼───────────┐                           │
│              │   REPOSITORY LAYER    │                           │
│              │   (sqlc generated)    │                           │
│              └───────────┬───────────┘                           │
│                          │                                       │
│              ┌───────────▼───────────┐                           │
│              │   DB MANAGER          │                           │
│              │   (multi-DB routing)  │                           │
│              └───────┬─────────┬─────┘                           │
└──────────────────────┼─────────┼─────────────────────────────────┘
                       │         │
              ┌────────▼───┐  ┌──▼──────────────┐
              │  Registry  │  │  Per-project    │
              │  SQLite    │  │  SQLite DBs     │
              │ (~/.tasks) │  │ (.tasks/ each)  │
              └────────────┘  └─────────────────┘
```

### 3.1 Core architectural principles

1. **Service layer as single source of truth.** All 4 entry points (MCP/CLI/REST/UI) call the same service layer. No duplicated business logic.

2. **Per-project DB with global registry.** Tasks live with the project. The global registry only maintains the `slug → path` mapping.

3. **Embedded UI and static assets.** Everything (templates, CSS, JS) is in the Go binary via `go:embed`. Distribution is a single file.

4. **MCP-first design.** MCP tools are designed as the primary interface, not as a wrapper around the REST API.

5. **Progressive enhancement.** The application works without JavaScript (server-rendered HTML). HTMX adds interactivity.

6. **Multi-user ready, single-user simple.** Schema contains `user_id` fields from the start, but the auth layer is optional (default: single-user mode).

---

## 4. Tech Stack

### 4.1 Backend

| Component | Technology | Version | Rationale |
|---|---|---|---|
| **Language** | Go | 1.22+ | Single binary, excellent concurrency, mature ecosystem |
| **DB driver** | `modernc.org/sqlite` | latest | Pure Go, no CGO, simpler cross-compilation |
| **DB queries** | `sqlc` | v1.27+ | Type-safe Go code generated from SQL |
| **Migrations** | `goose` | v3+ | Embedded migrations via `go:embed` |
| **HTTP router** | `chi` | v5 | Lightweight, idiomatic Go |
| **CLI framework** | `cobra` | latest | De facto standard (kubectl, gh, hugo) |
| **MCP server** | `modelcontextprotocol/go-sdk` | latest | Official Go MCP SDK |
| **Config** | `koanf` | v2 | Modern alternative to Viper |
| **Logging** | `slog` (stdlib) | - | In Go stdlib since 1.21 |
| **Validation** | `go-playground/validator` | v10 | Standard for struct validation |

### 4.2 Frontend

| Component | Technology | Rationale |
|---|---|---|
| **Templating** | `templ` | Type-safe Go templating, compiled |
| **CSS framework** | `@ahoylog/ahoylog-css` | Ahoylog design system |
| **Interactivity** | HTMX | Server-rendered HTML fragments, no SPA build |
| **Minor JS** | Alpine.js (optional) | For things HTMX doesn't handle well (modal toggling) |
| **Drag & drop** | SortableJS | For kanban board (~5KB, zero dependencies) |

### 4.3 Build & Distribution

| Component | Technology | Rationale |
|---|---|---|
| **Build orchestration** | `Taskfile` (go-task) | Modern, readable YAML |
| **Multi-platform builds** | `goreleaser` | Standard for Go OSS projects |
| **CI/CD** | GitHub Actions | Free, good Go support |
| **Test framework** | stdlib `testing` + `testify` | Idiomatic Go |
| **DB testing** | `testcontainers-go` (optional) | For integration tests |

### 4.4 Rejected alternatives

For transparency, we record what was considered and why it didn't make the cut:

- **GORM/Ent (ORM):** Too much magic, harder to control SQL. `sqlc` is more explicit.
- **Templ + Alpine without HTMX:** Possible, but HTMX is more natural for this use case (server speaks HTML, not JSON).
- **React/Vue/Svelte SPA:** Build step and complexity unnecessary for an application of this scope.
- **Tailwind/DaisyUI:** We have our own ahoylog-css, which is a showcase value.
- **Postgres:** SQLite is perfect for a local single-user tool. Postgres adds operational overhead without benefit.
- **gRPC:** Unnecessarily heavy for a local tool. REST + MCP are sufficient.

---

## 5. Data Model

### 5.1 Per-project database (`<project>/.tasks/tasks.db`)

```sql
-- Meta table for versioning and project metadata
CREATE TABLE meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- Initial records:
-- ('schema_version', '1')
-- ('project_slug', 'webapp')
-- ('project_name', 'My Web App')
-- ('created_at', '2026-04-27T10:00:00Z')

-- Main table for tasks and bugs
CREATE TABLE tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    type         TEXT NOT NULL DEFAULT 'task'
                 CHECK(type IN ('task', 'bug', 'feature', 'hotfix')),
    title        TEXT NOT NULL,
    description  TEXT,
    status       TEXT NOT NULL DEFAULT 'todo'
                 CHECK(status IN ('todo', 'doing', 'done')),
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,

    -- Multi-user readiness (default = 1 = local user)
    created_by   INTEGER NOT NULL DEFAULT 1,
    assigned_to  INTEGER,

    -- Priority (higher number = higher priority, default 0 = normal)
    priority     INTEGER NOT NULL DEFAULT 0,

    -- Audit trail — set automatically by entry point, not by user
    source       TEXT NOT NULL DEFAULT 'cli'
                 CHECK(source IN ('cli', 'mcp', 'ui', 'api')),

    -- Future extensions (stored as JSON)
    metadata     TEXT
);

-- Indexes
CREATE INDEX idx_tasks_status ON tasks(status) WHERE status != 'done';
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_assigned ON tasks(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority) WHERE status != 'done';

-- Single trigger: updated_at always, completed_at on transition to 'done'
CREATE TRIGGER tasks_after_update AFTER UPDATE ON tasks
BEGIN
    UPDATE tasks SET
        updated_at = CURRENT_TIMESTAMP,
        completed_at = CASE
            WHEN NEW.status = 'done' AND OLD.status != 'done' THEN CURRENT_TIMESTAMP
            WHEN NEW.status != 'done' THEN NULL
            ELSE completed_at
        END
    WHERE id = NEW.id;
END;

-- Prepared for v0.2: Full-text search
-- CREATE VIRTUAL TABLE tasks_fts USING fts5(title, description, content=tasks);
```

### 5.2 Global registry (`~/.tasks/registry.db`)

```sql
-- List of projects and their locations
CREATE TABLE projects (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    slug         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    path         TEXT NOT NULL UNIQUE,        -- absolute path to .tasks/
    last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Cached stats (updated on each access)
    cached_todo_count   INTEGER DEFAULT 0,
    cached_doing_count  INTEGER DEFAULT 0,
    cached_total_count  INTEGER DEFAULT 0,
    stats_updated_at    DATETIME
);

-- Users (multi-user readiness, MVP = only 'local')
CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO users (id, username) VALUES (1, 'local');

-- API tokens for REST/MCP HTTP authentication
CREATE TABLE api_tokens (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    name        TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,    -- SHA-256 hash, NOT plain token
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME,
    expires_at  DATETIME
);

CREATE INDEX idx_tokens_hash ON api_tokens(token_hash);
```

### 5.3 Design decisions

**Why per-project DB:**
- Tasks travel with the project (zip, git clone, move)
- Optional Git tracking — `.tasks/tasks.db` in `.gitignore` or not
- Isolation (corruption in one project doesn't affect others)
- Backing up a project = backing up tasks
- Mentally clean

**Why bug = task with `type='bug'`:**
- No schema duplication
- Filtering by `type` is trivial
- Converting a bug to a task (or vice versa) is just an UPDATE

**Why `completed_at` as a separate column:**
- Stats ("completed this week") don't need a full table scan
- If a user un-marks a task as done and back, we get a new timestamp

**Why `metadata` JSON column:**
- Future-proof for things like labels, due dates, parent_id without schema migration
- Not used in MVP, but available

---

## 6. MCP Interface (most important part)

MCP is the primary interface through which AI assistants access kleidi-task.

### 6.1 Tools (actions)

```typescript
// Task management
task_create(
    project: string,           // project slug (or "current")
    title: string,
    type?: "task" | "bug" | "feature" | "hotfix" | "feature" | "hotfix",  // default: "task", auto-detected from title prefix
    description?: string,
    priority?: number           // default: 0, higher = more important
): Task
// source is set automatically based on entry point (mcp/cli/ui/api)

task_list(
    project?: string,           // filter by project
    status?: "todo" | "doing" | "done",
    type?: "task" | "bug" | "feature" | "hotfix",
    limit?: number              // default: 50
): Task[]

task_get(id: number): TaskDetail

task_update(
    id: number,
    title?: string,
    description?: string,
    status?: "todo" | "doing" | "done",
    type?: "task" | "bug" | "feature" | "hotfix",
    priority?: number
): Task

task_complete(id: number): Task    // shortcut for status='done'

task_delete(id: number): { success: boolean }

// Projects
project_list(): Project[]

project_current(): Project | null   // detect based on cwd

project_stats(slug?: string): {
    todo: number,
    doing: number,
    done: number,
    bugs_open: number
}
```

### 6.2 Resources (context that Claude can "read")

```
tasks://projects                  → JSON list of all projects
tasks://project/{slug}            → project overview (stats + recent tasks)
tasks://project/{slug}/tasks      → all tasks for a project
tasks://task/{id}                 → task detail
```

Resources allow Claude to see context **without explicitly calling a tool**. Claude Desktop can expose them as "files" in the UI.

### 6.3 MCP API design principles

1. **Tools are actions, resources are context.** Don't mix them.
2. **Return values are consistent structures.** No strings like "OK", always JSON.
3. **Project is optional where it makes sense.** If Claude knows where the user is (project_current), no need to specify it every time.
4. **No destructive defaults.** `task_delete` doesn't exist as "destructive cleanup". Requires explicit action.
5. **Errors are readable for AI.** Error messages are designed so Claude can advise the user (e.g., "Project 'foo' not found. Available projects: webapp, ahoylog-css").

### 6.4 Transport

- **stdio** — primary mode for Claude Desktop, Cursor, Continue. User adds `klt mcp` as MCP server in client config.
- **HTTP** — secondary mode for remote or multi-client scenarios. Binds to `localhost:7842/mcp` (port configurable).

---

## 7. CLI Interface

CLI is the primary human-facing interface. Based on the principle of "Git-style ergonomics".

### 7.1 Commands

```bash
# Initialization
klt init [--name <name>]    # Create .tasks/ in current dir
klt doctor                   # Verify setup, list projects

# Tasks (most common)
klt add "Title" [--bug|--feature|--hotfix] [--description "..."] [--project <slug>]
klt add "BUG: title"          # auto-detected from prefix (also: FEATURE:, FEAT:, HOTFIX:, TASK:)
klt list [--status todo] [--type bug] [--all] [--project <slug>]
klt done <id>
klt show <id>
klt update <id> [--title "..."] [--status doing] [--description "..."]
klt delete <id>

# Aliases for ergonomics
klt t add "..."     # alias for task add
klt b "..."         # alias for add --bug
klt ls              # alias for list

# Projects
klt project list
klt project show [<slug>]
klt project stats [<slug>]

# Servers
klt serve           # HTTP server (UI + REST API + MCP HTTP)
klt serve --port 7842
klt mcp             # stdio MCP server (for Claude Desktop)

# Utility
klt export [--format json|markdown] [--project <slug>]
klt import <file> [--format json|markdown]
klt version
```

### 7.2 Project detection (key for UX)

When you run `klt add` without `--project`:

```
1. Are you in a directory with .tasks/? → use this DB
2. Are you in a subdirectory of a project with .tasks/? → walk up, use the found one
3. No .tasks/ found? → error: "Run `klt init` first"
```

(Inspired by the Git pattern with `.git/`.)

### 7.3 Output formats

CLI default is human-friendly text. For scripting:

```bash
klt list --output json     # JSON output
klt list --output csv      # CSV output
klt list --output ids      # IDs only, one per line
```

---

## 8. REST API

### 8.1 Endpoints

```
# Projects
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/{slug}
GET    /api/v1/projects/{slug}/tasks
GET    /api/v1/projects/{slug}/stats

# Tasks
GET    /api/v1/tasks                    # ?project=&status=&type=
POST   /api/v1/tasks
GET    /api/v1/tasks/{id}
PATCH  /api/v1/tasks/{id}
DELETE /api/v1/tasks/{id}
POST   /api/v1/tasks/{id}/complete

# Search (v0.2+)
GET    /api/v1/search?q=...&project=...

# System
GET    /api/v1/health
GET    /api/v1/version
```

### 8.2 Authentication

- **Single-user mode:** Optional API token. Without token = `localhost`-only access.
- **Multi-user mode (future):** Bearer token in `Authorization` header.

### 8.3 OpenAPI spec

`docs/openapi.yaml` — generated via `oapi-codegen`. This file is the single source of truth for API schemas and handler code is generated from it.

---

## 9. Web UI (HTMX + ahoylog-css)

### 9.1 Pages (MVP)

```
/                      Dashboard — all projects + summary
/p/{slug}              Project view — task list with filters
/p/{slug}/board        Kanban board (todo/doing/done columns)
/t/{id}                Task detail view
/settings              API tokens, config
```

### 9.2 Existing components from ahoylog-css

Already available:
- `k-container` — outer wrapper
- `k-stack` — vertical layout for task list
- `k-cluster` — horizontal layout for filter bar and button rows
- `k-grid` — kanban columns (3 columns: todo/doing/done)
- `k-card` — task card in list
- `k-button` — all actions (including `--danger` variant)
- `k-field` + `k-label` + `k-input` + `k-textarea` + `k-select` — forms
- `k-checkbox` — bulk actions

Spacing/text/visibility utilities — used for layout fine-tuning.

### 9.3 Components that need to be added

These will be added **directly to ahoylog-css** as PRs, enriching the Ahoylog design system:

1. **`k-badge`** — status badges (todo/doing/done) and type badges (task/bug)
2. **`k-modal`** — task create/edit dialog (`<dialog>` element + styling)
3. **`k-dropdown`** — filter selectors, action menus (native `<details>` element)
4. **`k-toast`** — feedback after HTMX actions
5. **`k-spinner`** — loading indicator for HTMX hx-indicator
6. **`k-empty`** — empty state component ("no tasks")

Estimate: ~500-800 lines of SCSS, 1-2 days of work.

### 9.4 HTMX patterns

**Example: changing task status without page reload:**
```html
<select
    hx-patch="/api/v1/tasks/42"
    hx-vals='{"status": "doing"}'
    hx-trigger="change"
    hx-swap="outerHTML"
    hx-target="closest .k-card"
    class="k-select"
>
    <option>todo</option>
    <option selected>doing</option>
    <option>done</option>
</select>
```

**Example: adding a task with OOB toast notification:**
```html
<form
    hx-post="/api/v1/tasks"
    hx-target="#task-list"
    hx-swap="afterbegin"
>
    <input class="k-input" name="title" required>
    <button class="k-button">Add</button>
</form>
```

Server returns an HTML fragment of the new task + OOB swap toast.

### 9.5 Drag & drop for kanban

SortableJS (5KB) + HTMX endpoint:

```html
<div class="k-stack" data-status="todo" hx-trigger="kanban-drop"
     hx-post="/api/v1/tasks/reorder"
     hx-include="closest .kanban-column">
    <!-- task cards -->
</div>
```

JavaScript glue (~30 lines) to connect SortableJS with HTMX trigger.

---

## 10. Project Structure

```
kleidi-task/
├── cmd/
│   └── klt/
│       └── main.go                    # Entry point (binary name: klt)
├── internal/
│   ├── core/                          # Service layer (business logic)
│   │   ├── task.go                    # TaskService
│   │   ├── project.go                 # ProjectService
│   │   ├── search.go                  # SearchService (v0.2+)
│   │   ├── errors.go                  # Domain errors
│   │   └── types.go                   # Domain types
│   ├── db/
│   │   ├── manager.go                 # DB connection management (multi-DB)
│   │   ├── migrations/
│   │   │   ├── project/               # Migrations for per-project DB
│   │   │   │   └── 001_initial.sql
│   │   │   └── registry/              # Migrations for global DB
│   │   │       └── 001_initial.sql
│   │   ├── queries/
│   │   │   ├── tasks.sql              # sqlc queries
│   │   │   ├── projects.sql
│   │   │   └── tokens.sql
│   │   └── generated/                 # sqlc output (committed to repo)
│   │       ├── tasks.sql.go
│   │       ├── projects.sql.go
│   │       └── models.go
│   ├── mcp/
│   │   ├── server.go                  # MCP server
│   │   ├── tools.go                   # Tool implementations
│   │   ├── resources.go               # Resource implementations
│   │   └── transport.go               # stdio + HTTP transports
│   ├── cli/
│   │   ├── root.go                    # Cobra root cmd
│   │   ├── init.go
│   │   ├── add.go
│   │   ├── list.go
│   │   ├── done.go
│   │   ├── show.go
│   │   ├── update.go
│   │   ├── delete.go
│   │   ├── project.go
│   │   ├── serve.go
│   │   └── mcp.go
│   ├── api/
│   │   ├── server.go                  # chi router setup
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── logging.go
│   │   │   └── cors.go
│   │   ├── handlers/
│   │   │   ├── tasks.go
│   │   │   ├── projects.go
│   │   │   └── system.go
│   │   └── openapi.yaml               # API spec
│   ├── ui/
│   │   ├── server.go                  # HTMX route handlers
│   │   ├── handlers/
│   │   │   ├── dashboard.go
│   │   │   ├── project.go
│   │   │   ├── task.go
│   │   │   └── settings.go
│   │   ├── templates/                 # .templ files
│   │   │   ├── layout.templ
│   │   │   ├── dashboard.templ
│   │   │   ├── project.templ
│   │   │   ├── task.templ
│   │   │   └── components/
│   │   │       ├── task_card.templ
│   │   │       ├── badge.templ
│   │   │       └── toast.templ
│   │   └── static/
│   │       ├── css/
│   │       │   └── ahoylog.min.css     # vendored from @ahoylog/ahoylog-css
│   │       ├── js/
│   │       │   ├── htmx.min.js
│   │       │   ├── alpine.min.js
│   │       │   └── sortable.min.js
│   │       └── icons/
│   └── config/
│       ├── config.go                  # Loader
│       └── defaults.go
├── pkg/                               # Public API (if needed)
│   └── client/                        # Go client for REST API
├── scripts/
│   ├── update-ahoylog-css.sh         # Vendoring ahoylog-css
│   └── release.sh
├── docs/
│   ├── ARCHITECTURE.md                # This document
│   ├── INSTALL.md
│   ├── CLI.md
│   ├── MCP.md
│   ├── API.md
│   ├── CLAUDE_SKILL.md                # Claude integration guide
│   └── CONTRIBUTING.md
├── claude-skill/                      # Pre-built Claude skill
│   ├── SKILL.md
│   └── README.md
├── examples/
│   ├── claude-desktop-config.json
│   └── cursor-config.json
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── go.mod
├── go.sum
├── sqlc.yaml                          # sqlc config
├── .goreleaser.yaml
├── Makefile
├── LICENSE
├── README.md
└── CHANGELOG.md
```

---

## 11. Roadmap

### v0.1 — MVP ✅ Complete

**Scope:** Functional CLI + MCP server with SQLite backend.

- [x] Schema + migrations (per-project + registry)
- [x] DB Manager with multi-DB routing
- [x] sqlc setup, generated queries
- [x] Service layer: TaskService, ProjectService
- [x] CLI: `init`, `add`, `list`, `done`, `show`, `update`, `delete`
- [x] MCP server (stdio) with 9 tools (task CRUD + project management)
- [x] Project detection (walk-up logic)
- [x] Single binary build (Taskfile + symlink install)
- [x] Pre-built Claude skill in `claude-skill/` with auto-parse title/description
- [x] `todo:` prefix support for task creation
- [x] Dev setup task (`task setup` installs all tools)

### v0.2 — UI ✅ Complete

**Scope:** Web UI + REST API.

- [x] REST API (chi router) with full task CRUD + project endpoints
- [x] HTMX UI with dashboard, project view, task detail
- [x] Templ templates for all pages
- [x] Velocity-inspired design with Plus Jakarta Sans, Lucide icons
- [x] ahoylog-css design system integration (k-grid, badges, cards)
- [x] Inline task editing on detail page with auto-save
- [x] OOB stats updates on task actions
- [ ] Light/dark mode toggle
- [ ] Settings page (API tokens management)

### v0.3 — Polish (in progress)

**Scope:** Quality of life features.

- [x] Kanban board with drag & drop (SortableJS)
- [x] Filtering UI (multi-select status and type, toggle logic)
- [x] Pagination with page count and navigation
- [x] Priority filtering (min_priority)
- [x] Date range filtering (created_after/created_before)
- [x] Collapsible advanced filters panel
- [x] Sidebar task creation form with expandable description
- [x] View toggle (List/Board) with history-aware back navigation
- [x] Type filter on kanban board
- [x] Unified dynamic SQL query (single query with sqlc.narg replaces 4 separate queries)
- [ ] FTS5 full-text search
- [ ] Bulk operations (select multiple → mark done)
- [ ] Keyboard shortcuts (vim-like: j/k navigate, x complete)
- [ ] Export/import (JSON, Markdown)
- [ ] Stats dashboard (per-project, global)
- [ ] Markdown editor for task description
- [ ] Archive/cleanup strategy for completed tasks

### v0.4 — AI-native features

**Scope:** Advanced AI integration.

- [ ] Extended MCP toolset (search, bulk operations, smart filters)
- [ ] Conversation → task linking (save context from which task was created)
- [ ] Git integration (commit message → task reference, branch → project)
- [ ] AI suggested tasks (based on git diff, file patterns)
- [ ] Task templates
- [ ] Task workflows (phase-based with auto skill triggers)

### v1.0 — Production ready

**Scope:** Stabilization, multi-user.

- [ ] Multi-user support (auth, permissions)
- [ ] WebDAV/CalDAV export for compatibility with mobile apps
- [ ] Hosted variant (Docker compose for teams)
- [ ] Plugin system (script hooks)
- [ ] Comprehensive docs
- [ ] First-class Cursor extension

---

## 12. Claude Skill integration

The repository includes `claude-skill/SKILL.md`, which can be installed into Claude.

### Example SKILL.md

```markdown
---
name: kleidi-task
description: Use when user mentions tasks, bugs, todos, or wants to track work across projects. Connects to local kleidi-task instance via MCP.
---

# kleidi-task Integration

## When to use
- User says "task: ...", "bug: ...", "todo: ..."
- User asks "what am I working on", "show my tasks", "list bugs"
- User wants to mark something as done
- User mentions tracking work in current project

## Prerequisites
The user must have kleidi-task installed and either:
- MCP server configured in Claude Desktop config, OR
- `klt serve` running locally

## How to use
Use the MCP tools `task_create`, `task_list`, `task_complete`, etc.

**Always check `task_list` before creating** to avoid duplicates.

**Project detection:**
1. If user mentions specific project, use that slug
2. If unclear, call `project_current` (uses cwd) or `project_list` and ask

## Example workflows

### "Bug: login fails on Firefox"
1. Call `project_current` to get current project
2. Call `task_create(project=current, title="login fails on Firefox", type="bug")`
3. Confirm: "Created bug #42 in project webapp"

### "What am I working on?"
1. Call `task_list(status="doing")`
2. Format as readable list with project, type, title
```

---

## 13. Key Architectural Decisions (ADRs)

### ADR-001: Per-project SQLite DB with global registry

**Context:** Tasks must be accessible cross-project, but project-isolated.

**Decision:** Each project has its own SQLite DB in `.tasks/tasks.db`. The global registry in `~/.tasks/registry.db` only maintains the `slug → path` mapping.

**Consequences:**
- (+) Tasks travel with the project
- (+) Optional Git tracking
- (+) Isolation
- (−) Cross-project queries require aggregation logic
- (−) DB connection management is more complex

### ADR-002: Single table with `type` column for all work item types

**Context:** We need to track tasks, bugs, features, and hotfixes.

**Decision:** Single `tasks` table with `type` column: `task`, `bug`, `feature`, `hotfix`. Title prefix detection auto-classifies (e.g., "BUG: title" → bug).

**Consequences:**
- (+) No schema duplication
- (+) Conversion between types is just an UPDATE
- (+) New types can be added with a migration
- (−) UI must filter by type

### ADR-003: HTMX instead of SPA framework

**Context:** We need a web UI with reasonable interactivity.

**Decision:** Server-rendered HTML with HTMX for dynamic interactions.

**Consequences:**
- (+) Zero build step for frontend
- (+) Single binary distribution
- (+) Progressive enhancement out of the box
- (−) Some complex UI patterns are harder
- (−) Not a "modern stack" for some developers

### ADR-004: ahoylog-css instead of Tailwind/DaisyUI

**Context:** We need a styling framework.

**Decision:** Use our own ahoylog-css, add missing components.

**Consequences:**
- (+) Showcase for ahoylog-css
- (+) No Node.js / Tailwind build step
- (+) Strategic consistency with Ahoylog ecosystem
- (−) Must add modal, dropdown, toast, badge, spinner
- (−) Smaller community / fewer ready-made patterns

### ADR-005: MCP-first design

**Context:** AI integration is the primary use case.

**Decision:** MCP server is a first-class citizen, not a wrapper around the REST API.

**Consequences:**
- (+) Better AI UX
- (+) Tools designed for AI workflow
- (−) Potential logic duplication between MCP and REST (mitigated via service layer)

### ADR-006: Single binary distribution

**Context:** Users want simple installation.

**Decision:** Everything (templates, CSS, JS) embedded via `go:embed`.

**Consequences:**
- (+) `go install` or download binary = done
- (+) No external assets
- (−) Binary is larger (~30MB with embedded assets)
- (−) Update ahoylog-css = rebuild kleidi-task

### ADR-007: Source field automatically set by entry point

**Context:** We need to know where a task originated (CLI, MCP, UI, API) for audit and debugging.

**Decision:** `source` is a NOT NULL column with CHECK constraint. Each entry point (CLI/MCP/UI/API) sets it automatically when calling a service layer method. Users don't set it manually.

**Implementation note:** In the Go service layer, `source` is a required parameter (no default). The DB `DEFAULT 'cli'` is only a safety net for direct SQL access. If an entry point doesn't set source, a compile error is better than a silent fallback.

**Consequences:**
- (+) Consistent audit trail
- (+) No possibility of forgetting to set source
- (−) Service layer methods need a `source` parameter

### ADR-008: Priority as INTEGER (higher = more important)

**Context:** Task tracker needs prioritization.

**Decision:** `priority INTEGER NOT NULL DEFAULT 0`. Higher number = higher priority. No enum, free scale.

**Consequences:**
- (+) Flexible — user defines their own scale
- (+) Simple sorting (ORDER BY priority DESC)
- (+) Default 0 = normal task, no need to set priority
- (−) UI must offer reasonable defaults (e.g., 0/1/2/3)

---

## 14. Security

### 14.1 Threats

| Threat | Probability | Impact | Mitigation |
|---|---|---|---|
| Unauthorized API access | Medium | Medium | API tokens hashed (SHA-256), default localhost-only |
| SQLi via API | Low | High | sqlc generates parameterized queries |
| XSS in task description | Medium | Medium | Templ auto-escapes, HTMX swap is safe |
| MCP token leak | Low | Medium | Tokens are not stored in logs |
| Path traversal in project paths | Medium | Medium | Path validation, walk-up has max depth |

### 14.2 Default security posture

- **Single-user MVP:** API server binds to `127.0.0.1` only
- **Multi-user future:** TLS required, tokens mandatory
- **No secrets in Git:** `.tasks/tasks.db` can go in Git, but configuration with tokens cannot

### 14.3 Audit trail

All write operations are logged with the `source` field (cli/mcp/ui/api). For v1.0+ we'll add a separate `audit_log` table.

---

## 15. Performance targets

### 15.1 Latency targets (local)

| Operation | Target | Strategy |
|---|---|---|
| `task add` | < 50ms | Direct SQLite write |
| `task list` (100 items) | < 100ms | Indexed query |
| `task list --all` (10 projects) | < 300ms | Parallel DB queries |
| MCP tool call | < 100ms | Same as above |
| Page load (dashboard) | < 200ms | Server-rendered, embedded assets |

### 15.2 Scalability

| Metric | Target |
|---|---|
| Tasks per project | 10,000+ |
| Projects per instance | 100+ |
| Concurrent users (multi-user mode) | 10+ |

SQLite handles these numbers without issues with proper indexing.

---

## 16. Distribution & Installation

### 16.1 Target platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64) — primary use case is WSL, but native binary also available

### 16.2 Installation methods

```bash
# Go users
go install github.com/tomas-chudjak/kleidi-task/cmd/klt@latest

# Curl install script
curl -fsSL https://kleidi-task.dev/install.sh | sh

# Homebrew (future)
brew install tomas-chudjak/tap/kleidi-task

# Manual download
# Download from github.com/tomas-chudjak/kleidi-task/releases
```

### 16.3 Binary size budget

Target: < 30MB with embedded assets.

Expected breakdown:
- Go runtime: ~3MB
- SQLite (modernc): ~5MB
- HTMX + Alpine + SortableJS (minified): ~50KB
- ahoylog-css: ~30KB
- Templates: ~100KB
- Application logic: ~5MB

Total: ~13-15MB. Comfortably under target.

---

## 17. Open questions / TODO for decisions

Things not resolved in this document that need to be decided during implementation:

1. **Timezones** — store UTC, display local? Configurable?
2. **Soft delete** — `task delete` actually deletes, or just marks? (Proposal: actually deletes in MVP, soft delete in v0.4+)
3. **Project rename** — ~~slug is primary key~~ Resolved: INTEGER PK + UNIQUE(slug), rename is just an UPDATE
4. **WSL path handling** — can `klt` handle `/mnt/c/...` paths? Yes, but in Claude Desktop config this needs to be accounted for.
5. **MCP discovery** — register on MCP marketplace when available?
6. **Telemetry** — none in MVP. Decision will be made post-release.
7. **Cross-project UX** — single-project view with switching vs. multi-project aggregation. Deferred to v0.2.

---

## 18. Success Metrics

After v1.0 release we'll measure:

- **GitHub stars:** target 500 in first year
- **Active users:** hard to measure locally, but GitHub releases downloads
- **Claude skill installs:** if published on marketplace
- **Community contributions:** PRs per quarter
- **Issue resolution time:** median < 7 days

---

## 19. Decision log

| Date | Decision | Status |
|---|---|---|
| 2026-04-27 | Per-project DB + registry (vs. global DB) | ✅ Approved |
| 2026-04-27 | Single-user MVP, multi-user-ready schema | ✅ Approved |
| 2026-04-27 | Minimalistic MVP (tasks + bugs + projects only) | ✅ Approved |
| 2026-04-27 | HTMX + ahoylog-css (vs. React/Vue + Tailwind) | ✅ Approved |
| 2026-04-27 | Go + SQLite + sqlc stack | ✅ Approved |
| 2026-04-27 | MIT license | ✅ Approved |
| 2026-04-27 | Project name `kleidi-task` + binary name `klt` | ✅ Approved |
| 2026-04-27 | Priority as INTEGER (higher = more important) | ✅ Approved |
| 2026-04-27 | Registry projects: INTEGER PK + UNIQUE(slug) | ✅ Approved |
| 2026-04-27 | Source field automatically set by entry point | ✅ Approved |
| 2026-04-27 | Single optimized trigger instead of two | ✅ Approved |
| 2026-04-27 | Rename kviky-css → ahoylog-css | ✅ Approved |
| 2026-04-27 | Rename .tasky → .tasks, tasky:// → tasks:// | ✅ Approved |
| 2026-04-27 | All code, comments, and docs in English | ✅ Approved |
| 2026-04-27 | 4 task types: task, bug, feature, hotfix + title prefix auto-detection | ✅ Approved |
| 2026-04-28 | Unified dynamic SQL query with sqlc.narg() (replaces 4 separate list queries) | ✅ Approved |
| 2026-04-28 | Multi-select filters via comma-separated values + SQL instr() | ✅ Approved |
| 2026-04-28 | Kanban board with SortableJS drag & drop | ✅ Approved |
| 2026-04-28 | Pagination with offset/limit + total count | ✅ Approved |
| 2026-04-28 | Symlink install instead of binary copy (avoids "Text file busy") | ✅ Approved |
| 2026-04-28 | `task setup` for dev environment bootstrapping | ✅ Approved |
| 2026-04-28 | MCP skill auto-parse: prefix triggers task creation before any work | ✅ Approved |
| TBD | Timezone strategy | ⏳ Open |
| TBD | Soft delete / archive strategy for completed tasks | ⏳ Open |
| TBD | Cross-project UX (switching vs. aggregation) | ⏳ Open |

---

## 20. Glossary

- **MCP** — Model Context Protocol, Anthropic's standard for AI tool integration
- **HTMX** — library for server-rendered interactivity via HTML attributes
- **Templ** — type-safe Go templating language
- **sqlc** — type-safe Go code generator from SQL
- **ahoylog-css** — Ahoylog design system (`@ahoylog/ahoylog-css`)
- **WAL mode** — SQLite Write-Ahead Logging, better performance for concurrent reads
- **FTS5** — SQLite full-text search engine
- **OOB swap** — HTMX out-of-band swap, update multiple DOM parts with one response

---

## Notes for AI agents

This document is the **authoritative source of truth** for the project architecture. During implementation:

1. **Start with the v0.1 roadmap.** Don't try to implement everything at once.
2. **Service layer first**, then CLI, then MCP, finally UI.
3. **Test each layer independently.** Service layer has unit tests, API has integration tests, UI has smoke tests.
4. **Follow the sqlc workflow:** edit SQL → `sqlc generate` → use generated code.
5. **All migrations are embedded** via `go:embed`. Never execute SQL ad-hoc.
6. **ahoylog-css component additions** are separate PRs to the `ahoylog/ahoylog-css` repo, not part of kleidi-task.
7. **For each new feature write an ADR** in `docs/adr/` — especially for decisions that contradict this document.

In case of conflict between this document and code, **code is truth** and the document should be updated. This document is living — versioned in Git and updated with major changes.
