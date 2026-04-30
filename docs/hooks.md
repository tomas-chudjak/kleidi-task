# Script Hooks

kvik-tasks supports project-level script hooks that run automatically on task lifecycle events. Hooks are shell commands executed asynchronously — they never block the task operation.

## Configuration

Hooks are stored in `.tasks/hooks.json` per project:

```json
{
  "hooks": [
    {
      "id": 1,
      "event": "task.complete",
      "command": "echo \"Task #$KVT_TASK_ID done: $KVT_TASK_TITLE\" >> ~/completed.log",
      "description": "Log completed tasks"
    },
    {
      "id": 2,
      "event": "task.create",
      "command": "./scripts/notify-slack.sh",
      "description": "Slack notification"
    }
  ]
}
```

You can edit this file directly, or manage hooks via the Settings UI.

## Events

| Event | Fires when |
|-------|-----------|
| `task.create` | A new task is created (CLI, MCP, UI, or API) |
| `task.update` | A task is modified (title, status, type, priority, etc.) |
| `task.complete` | A task is marked as done |
| `task.delete` | A task is permanently deleted |
| `task.archive` | A completed task is archived |

Multiple hooks can be registered for the same event. All matching hooks run in parallel.

## Environment variables

Every hook receives task data as environment variables:

| Variable | Example | Description |
|----------|---------|-------------|
| `KVT_EVENT` | `task.complete` | The event that triggered the hook |
| `KVT_TASK_ID` | `42` | Task ID |
| `KVT_TASK_TITLE` | `Fix login bug` | Task title |
| `KVT_TASK_TYPE` | `bug` | Task type (task, bug, feature, hotfix, or custom) |
| `KVT_TASK_STATUS` | `done` | Task status (todo, doing, done) |
| `KVT_TASK_PRIORITY` | `5` | Task priority (0 = default) |

Additionally, the full task JSON is piped to **stdin** for scripts that need more detail (description, metadata, timestamps, etc.):

```bash
#!/bin/bash
# Read full task JSON from stdin
TASK_JSON=$(cat)
DESCRIPTION=$(echo "$TASK_JSON" | jq -r '.description')
```

## Behavior

- Hooks run **asynchronously** in background goroutines — the task operation completes immediately regardless of hook execution
- Each hook has a **30 second timeout** — if the command doesn't finish in time, it's killed
- Hook failures are **logged** but never prevent the task operation from succeeding
- Hooks run with the **project directory** as working directory
- Hooks inherit the parent process environment plus the `KVT_*` variables

## Managing hooks

### Via Settings UI

Navigate to your project's Settings page and scroll to the **Hooks** section. You can:

- Add a hook by selecting an event, entering a shell command, and clicking **+**
- Remove a hook by clicking the trash icon

### Via hooks.json

Edit `.tasks/hooks.json` directly. Assign a unique `id` to each hook. The file is read on every event, so changes take effect immediately — no restart needed.

## Examples

### Log all task completions

```json
{
  "event": "task.complete",
  "command": "echo \"$(date -Iseconds) DONE #$KVT_TASK_ID $KVT_TASK_TITLE\" >> .tasks/activity.log"
}
```

### Run tests when a bug is created

```json
{
  "event": "task.create",
  "command": "[ \"$KVT_TASK_TYPE\" = \"bug\" ] && go test ./... > /dev/null 2>&1 || true"
}
```

### Send a desktop notification

```json
{
  "event": "task.complete",
  "command": "notify-send 'kvt' \"Task #$KVT_TASK_ID completed: $KVT_TASK_TITLE\""
}
```

### Post to a webhook

```json
{
  "event": "task.create",
  "command": "cat | curl -s -X POST -H 'Content-Type: application/json' -d @- https://hooks.example.com/tasks"
}
```

This pipes the full task JSON from stdin directly to the webhook endpoint.
