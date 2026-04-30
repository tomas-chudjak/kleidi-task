# Changelog

All notable changes to kvik-tasks are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-04-30

Initial release.

### Added

**Core**
- Single Go binary with embedded assets (templates, CSS, JS)
- Per-project SQLite databases with global registry at `~/.tasks/registry.db`
- Service layer shared across all interfaces (CLI, MCP, REST API, Web UI)
- 4 built-in task types: task, bug, feature, hotfix
- Title prefix auto-detection (BUG:, FEAT:, FEATURE:, HOTFIX:, TODO:)
- Task priorities (higher number = more important)
- Full-text search with SQLite FTS5

**CLI**
- `kvt init` — project initialization
- `kvt add` — create tasks with type flags and prefix detection
- `kvt list` — list tasks with status/type filters
- `kvt show`, `kvt done`, `kvt update`, `kvt delete` — task CRUD
- `kvt advance` — advance task to next workflow phase
- `kvt archive` / `kvt unarchive` — archive management
- `kvt suggest` — scan source code for TODO/FIXME/HACK/XXX comments
- `kvt export` / `kvt import` — JSON and Markdown formats
- `kvt backup` — database backup
- `kvt serve` — HTTP server with `--host` and `--port` flags
- `kvt mcp` — MCP stdio server
- `kvt user add` / `kvt user list` — user management
- `kvt project list` / `kvt project stats` — project management
- `kvt version` — version info

**MCP Server**
- stdio transport for Claude Desktop, Cursor, VS Code
- Tools: task_create, task_list, task_get, task_update, task_complete, task_delete, task_archive, task_unarchive, task_search, task_advance, task_history, task_suggest, task_bulk_update, task_bulk_complete
- Tools: project_list, project_current, project_stats, project_stats_extended, project_backup
- Tools: category_list, category_create
- Resources: tasks:// URI scheme for project and task data

**REST API**
- Full CRUD for tasks and projects under `/api/v1`
- Project stats endpoint
- Task filtering with status, type, priority, date range, pagination
- Health check and version endpoints

**Web UI**
- Dashboard with project overview and live statistics
- Project view with task table, inline editing, and sidebar creation form
- Kanban board with SortableJS drag-and-drop
- Task detail page with markdown editor (EasyMDE)
- Multi-select filters for status, type, and category
- Date range and priority filtering with collapsible panel
- Pagination with page navigation
- Vim-like keyboard shortcuts (j/k navigate, x complete, e edit, n new)
- Light/dark mode toggle
- Settings page: categories, configuration, workflows, templates, hooks
- Workflow editor with phase management, AI prompts, and trigger configuration
- Dedicated workflow history page per task

**Task Workflows**
- Phase-based workflows per task type
- Built-in workflows for task, bug, feature, and hotfix types
- Custom task types with configurable workflows, badge colors, and prefixes
- AI prompts per phase (returned via MCP for context-aware guidance)
- Before/after triggers (shell commands or built-in skills)
- Workflow history with execution output and duration tracking

**Categories and Templates**
- Custom categories with colors for organizing tasks
- Task templates with predefined type, priority, and description
- Template-based task creation in web UI

**Git Integration**
- Commit to task linking via message references (#ID, kvt:ID, fixes #ID, etc.)
- Git activity displayed on task detail page

**Script Hooks**
- Shell hooks on task lifecycle events: create, complete, update, delete, archive
- Async execution with 30s timeout
- Task data via environment variables and stdin JSON
- Configuration in `.tasks/hooks.json`, manageable via Settings UI

**Infrastructure**
- Docker deployment with multi-stage Dockerfile and docker-compose.yml
- HTTP Basic Auth with bcrypt password hashing
- User management via CLI
- Database backup command
- Conversation/session linking for MCP-created tasks
- Project configuration: default priority, default type, auto-archive days

**VS Code Extension**
- Task sidebar with project tree view
- Status bar widget showing todo/doing counts
- Basic filtering (todo, doing, all)
- Expandable task details with description
- Actions: insert to terminal, open in browser, copy reference
- Basic Auth support via URL credentials
- Auto-refresh polling

**Developer Tools**
- Source code TODO/FIXME/HACK/XXX scanning with duplicate detection
- Claude skill (`claude-skill/SKILL.md`) with auto-parse title/description
- Taskfile for build orchestration (build, test, sqlc, templ, setup)
- goreleaser configuration for multi-platform releases
