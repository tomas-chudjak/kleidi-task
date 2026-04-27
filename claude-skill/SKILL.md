---
name: kvik-tasks
description: Use when user mentions tasks, bugs, todos, or wants to track work across projects. Connects to local kvik-tasks instance via MCP.
---

# kvik-tasks Integration

## When to use
- User says "task: ...", "bug: ...", "todo: ..."
- User asks "what am I working on", "show my tasks", "list bugs"
- User wants to mark something as done
- User mentions tracking work in current project

## Prerequisites
The user must have kvik-tasks installed and either:
- MCP server configured in Claude Desktop config, OR
- `kvt serve` running locally

## How to use
Use the MCP tools `task_create`, `task_list`, `task_complete`, etc.

**Always check `task_list` before creating** to avoid duplicates.

**Project detection:**
1. If user mentions specific project, use that slug
2. If unclear, call `project_current` (uses cwd) or `project_list` and ask

## Available tools

| Tool | Description |
|---|---|
| `task_create` | Create a new task or bug |
| `task_list` | List tasks with optional filters |
| `task_get` | Get task details by ID |
| `task_update` | Update task fields |
| `task_complete` | Mark task as done |
| `task_delete` | Permanently delete a task |
| `project_list` | List all registered projects |
| `project_current` | Get current project (based on cwd) |
| `project_stats` | Get todo/doing/done/bugs counts |

## Example workflows

### "Bug: login fails on Firefox"
1. Call `project_current` to get current project
2. Call `task_create(project=current, title="login fails on Firefox", type="bug")`
3. Confirm: "Created bug #42 in project webapp"

### "What am I working on?"
1. Call `task_list(status="doing")`
2. Format as readable list with project, type, title

### "Show me all open bugs"
1. Call `task_list(type="bug", status="todo")`
2. Format as prioritized list
