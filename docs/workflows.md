# Task Workflows

Every task type has a phase-based workflow defining the stages it goes through from creation to completion.

## Built-in workflows

| Type | Phases |
|------|--------|
| task | todo -> doing -> done |
| bug | reported -> reproducing -> fixing -> verifying -> done |
| feature | todo -> research -> implementation -> review -> done |
| hotfix | reported -> fixing -> verifying -> done |

## How phases work

A task's phase determines its position in the workflow. The status (todo/doing/done) is derived from the phase:
- **First phase** maps to `todo`
- **Last phase** maps to `done`
- **All other phases** map to `doing`

### Advancing a task

Move a task to the next phase:

```bash
kvt advance 42
```

Via MCP:

```
task_advance(id: 42)
```

The web UI shows an "Advance" button on the task detail page with a progress bar indicating the current phase.

## Phase prompts

Each phase can have an AI prompt — instructions that guide an AI assistant through that workflow step. For example, a bug's "reproducing" phase might have:

> Analyze the bug report. Try to reproduce the issue and identify the root cause. Document steps to reproduce.

Phase prompts are returned by the `task_advance` and `task_get` MCP tools, giving AI assistants context-aware guidance.

## Triggers

Each phase can have **before** and **after** triggers — shell commands or skill names that execute automatically when a task enters that phase.

Built-in skills:
- `run-tests` — `go test ./...`
- `lint` — `go vet ./...`
- `type-check` — `go build ./...`
- `smoke-test` — `go test -short ./...`
- `regression-test` — `go test -run TestRegression ./...`

Any unrecognized trigger name is treated as an AI prompt.

Example: a "verifying" phase with `run-tests` as a before trigger will automatically run the test suite when a task reaches that phase.

## Custom task types

Create custom types beyond the built-in four via the Workflows page in the web UI (`/p/{slug}/workflows`):

1. Enter a type name (lowercase, e.g. `spike`, `research`)
2. Choose a badge color
3. Optionally set a title prefix for auto-detection (e.g. `SPIKE`)
4. The type gets a default workflow (todo -> doing -> done)
5. Edit the workflow to add custom phases, prompts, and triggers

Custom types appear in all type dropdowns throughout the UI, CLI, and MCP.

## Editing workflows

The workflow editor is accessible at `/p/{slug}/workflows/{type}` in the web UI. You can:

- Add, remove, and reorder phases
- Set AI prompts per phase
- Configure before/after triggers per phase

Changes are saved to the project database and take effect immediately.

## Workflow history

Every phase transition is recorded in the workflow history. View it on the task detail page under "Workflow Timeline", or on the dedicated history page at `/p/{slug}/tasks/{id}/history`.

History entries include:
- Phase transition (previous -> current)
- Trigger execution results (output, success/failure, duration)
- Timestamps

Via MCP:

```
task_history(id: 42)
```
