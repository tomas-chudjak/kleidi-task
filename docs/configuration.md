---
title: Configuration
weight: 5
---

kleidi-task uses a layered configuration system. Project config overrides global config.

## Config files

| File | Scope | Description |
|------|-------|-------------|
| `~/.tasks/config.json` | Global | Applies to all projects |
| `<project>/.tasks/config.json` | Project | Overrides global for this project |

## Global config

```json
{
  "port": 7842,
  "default_priority": 0,
  "default_project": "my-project"
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `port` | `7842` | HTTP server port for `klt serve` |
| `default_priority` | `0` | Default priority for new tasks |
| `default_project` | `""` | Default project slug for MCP when not in a project directory |

## Project config (via Settings UI)

Project-level settings are stored in the project database and managed through the web UI at `/p/{slug}/settings`:

| Option | Default | Description |
|--------|---------|-------------|
| Default priority | `0` | Default priority for new tasks in this project |
| Default task type | `task` | Default type when creating tasks |
| Auto-archive days | `0` | Auto-archive completed tasks after N days (0 = disabled) |

These settings can also be edited via the REST API:

```bash
curl -X POST http://localhost:7842/p/my-project/settings/config \
  -H "Content-Type: application/json" \
  -d '{"default_priority": 3, "default_type": "task", "auto_archive_days": 30}'
```

## Custom task types

Custom types beyond the built-in task/bug/feature/hotfix can be created via the Workflows page in the web UI. Each custom type gets its own workflow definition with configurable phases, colors, and title prefixes.

See [workflows.md](workflows.md) for details.

## Data locations

| Path | Content |
|------|---------|
| `~/.tasks/registry.db` | Global registry (projects, users) |
| `~/.tasks/config.json` | Global configuration |
| `<project>/.tasks/tasks.db` | Project task database |
| `<project>/.tasks/config.json` | Project configuration |
| `<project>/.tasks/hooks.json` | Project script hooks |
