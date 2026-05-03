---
title: MCP Integration
weight: 3
---

## Setup

Add to your Claude Desktop or Claude Code configuration:

```json
{
  "mcpServers": {
    "kleidi-task": {
      "command": "klt",
      "args": ["mcp"]
    }
  }
}
```

Or via Claude Code CLI:

```bash
claude mcp add kleidi-task -- klt mcp
```

## Available Tools

### Projects

#### `project_current`
Get the current project based on your working directory.

```
No parameters required.
```

**Example:** "What project am I in?"

---

#### `project_list`
List all registered projects across the system.

```
No parameters required.
```

**Example:** "Show me all my projects"

---

#### `project_stats`
Get task statistics (todo/doing/done counts, open bugs) for a project.

| Parameter | Type   | Required | Description                     |
|-----------|--------|----------|---------------------------------|
| `slug`    | string | No       | Project slug (default: current) |

**Example:** "How many open tasks do I have?"

---

### Tasks

#### `task_create`
Create a new task, bug, feature, or hotfix.

| Parameter     | Type    | Required | Description                                    |
|---------------|---------|----------|------------------------------------------------|
| `project`     | string  | **Yes**  | Project slug or `"current"`                    |
| `title`       | string  | **Yes**  | Task title                                     |
| `description` | string  | No       | Detailed description                           |
| `type`        | string  | No       | `task` / `bug` / `feature` / `hotfix`          |
| `priority`    | integer | No       | Higher number = higher priority                |

**Title prefix auto-detection:** The title prefix sets the type automatically:
- `BUG: ...` → bug
- `FEAT: ...` or `FEATURE: ...` → feature
- `HOTFIX: ...` → hotfix
- `TASK: ...` → task

**Examples:**
- "Create a task to refactor the auth module"
- "Bug: login page crashes on Safari"
- "I need a feature for dark mode"

---

#### `task_list`
List tasks with optional filters.

| Parameter | Type    | Required | Description                           |
|-----------|---------|----------|---------------------------------------|
| `project` | string  | No       | Project slug or `"current"`           |
| `status`  | string  | No       | `todo` / `doing` / `done`            |
| `type`    | string  | No       | `task` / `bug` / `feature` / `hotfix` |
| `limit`   | integer | No       | Max results (default: 50)             |

**Examples:**
- "What am I working on?" → `task_list(status="doing")`
- "Show all open bugs" → `task_list(type="bug", status="todo")`
- "List everything in project X" → `task_list(project="x")`

---

#### `task_get`
Get detailed information about a specific task.

| Parameter | Type    | Required | Description                 |
|-----------|---------|----------|-----------------------------|
| `id`      | integer | **Yes**  | Task ID                     |
| `project` | string  | No       | Project slug or `"current"` |

**Example:** "Show me task #5"

---

#### `task_update`
Update an existing task's fields. Only provided fields are changed.

| Parameter     | Type    | Required | Description                           |
|---------------|---------|----------|---------------------------------------|
| `id`          | integer | **Yes**  | Task ID                               |
| `project`     | string  | No       | Project slug or `"current"`           |
| `title`       | string  | No       | New title                             |
| `description` | string  | No       | New description                       |
| `status`      | string  | No       | `todo` / `doing` / `done`            |
| `type`        | string  | No       | `task` / `bug` / `feature` / `hotfix` |
| `priority`    | integer | No       | New priority                          |

**Examples:**
- "Start working on task #3" → `task_update(id=3, status="doing")`
- "Change task #7 to a bug" → `task_update(id=7, type="bug")`
- "Bump priority of #12" → `task_update(id=12, priority=10)`

---

#### `task_complete`
Mark a task as done.

| Parameter | Type    | Required | Description                 |
|-----------|---------|----------|-----------------------------|
| `id`      | integer | **Yes**  | Task ID                     |
| `project` | string  | No       | Project slug or `"current"` |

**Example:** "Done with task #5"

---

#### `task_delete`
Permanently delete a task.

| Parameter | Type    | Required | Description                 |
|-----------|---------|----------|-----------------------------|
| `id`      | integer | **Yes**  | Task ID                     |
| `project` | string  | No       | Project slug or `"current"` |

**Example:** "Delete task #8"

---

## Common Workflows

### Track a bug while coding
> "Bug: the search bar doesn't handle special characters"

Claude will call `task_create(project="current", title="search bar doesn't handle special characters", type="bug")`.

### Morning standup
> "What's on my plate?"

Claude will call `task_list(status="doing")` and `task_list(status="todo")` to show in-progress and upcoming work.

### Sprint review
> "Show me project stats"

Claude will call `project_stats()` to show todo/doing/done/bug counts.

### Quick capture during conversation
> "task: add rate limiting to the API"

Claude will create a task in the current project without interrupting the conversation flow.

### Cross-project overview
> "List all my projects and their stats"

Claude will call `project_list()` then `project_stats()` for each project.

## Natural Language Tips

You don't need to use exact commands. These all work:

| You say                          | Claude does                                    |
|----------------------------------|------------------------------------------------|
| "task: refactor auth"            | Creates a task                                 |
| "bug: login broken"             | Creates a bug                                  |
| "what am I working on?"         | Lists doing tasks                              |
| "show all bugs"                 | Lists bugs                                     |
| "done with #5"                  | Marks task 5 as done                           |
| "start #3"                      | Updates task 3 status to doing                 |
| "how's the project going?"      | Shows project stats                            |
| "delete #8, it's a duplicate"   | Deletes task 8                                 |
