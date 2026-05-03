---
name: kleidi-task
description: Use when user mentions tasks, bugs, todos, or wants to track work across projects. Connects to local kleidi-task instance via MCP.
---

# kleidi-task Integration

## CRITICAL — Prefix triggers

When a user message starts with `task:`, `bug:`, `feat:`, `feature:`, `hotfix:`, or `todo:`, this is a **BLOCKING REQUIREMENT**: you MUST call `task_create` via MCP as your **very first action** — BEFORE reading files, writing code, researching, planning, or doing ANY other work. The prefix is a trigger for task creation, not an instruction to start implementing.

**The flow is always:**
1. Parse the message → extract title + description
2. Call `task_create` via MCP immediately
3. Confirm to the user: "Created [type] #[id]: [title]"
4. Only THEN, if the user also asked for implementation, proceed with the work

**Do NOT:**
- Interpret the prefixed message as an implementation instruction
- Start coding or editing files before creating the task
- Skip task creation because "you'll do it later"
- Ask the user whether they want to create a task — just do it

## When to use
- User says "task: ...", "bug: ...", "todo: ..." → **always create task first** (see above)
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

## Task creation — title & description parsing

When a user wants to create a task, ALWAYS parse their message into two parts:

1. **title** — A short, clear summary (max ~10 words). Extract the core intent.
2. **description** — The user's full original message, verbatim. Preserves all context, reasoning, and details.

### Parsing rules
- Strip the trigger prefix (`task:`, `bug:`, `feat:`, `hotfix:`, `todo:`) before parsing
- The title should be actionable and concise (e.g. "Add rate limiting to REST API")
- The description is the user's **exact original message** (after prefix removal) — do NOT summarize or rewrite it
- If the message is already short enough to be a title (under ~10 words, no extra context), use it as both title and description
- Detect type from prefix: `bug:` → bug, `feat:`/`feature:` → feature, `hotfix:` → hotfix, otherwise → task

### Examples

**User:** "task: Pri vytvoreni tasku potrebujem, aby claude cez skill automaticky preparsoval uzivatelovu spravu a vytvoril z nej title pre task a spravu vlozil ako description do tasku"
```
title: "Auto-parse user message into task title and description"
description: "Pri vytvoreni tasku potrebujem, aby claude cez skill automaticky preparsoval uzivatelovu spravu a vytvoril z nej title pre task a spravu vlozil ako description do tasku"
type: task
```

**User:** "bug: login page crashes on Safari when user clicks the submit button twice rapidly, the form sends duplicate requests and the second one returns a 500 error"
```
title: "Login page crashes on Safari with double submit"
description: "login page crashes on Safari when user clicks the submit button twice rapidly, the form sends duplicate requests and the second one returns a 500 error"
type: bug
```

**User:** "feat: dark mode"
```
title: "Dark mode"
description: "dark mode"
type: feature
```

## Template-based description generation

When you create a task via `task_create` without a description, the response may include a **template** — a structured outline for that task type (e.g. Bug has "Steps to reproduce / Expected / Actual", Feature has "Problem / Proposed solution / Acceptance criteria").

**When you receive a template, you MUST:**
1. Read the template structure
2. Fill in each section based on what you know from the user's message and conversation context
3. Call `task_update` to set the generated description on the task

**Rules for filling templates:**
- Fill every section you have information for — even partial info is better than empty
- If you don't have enough info for a section, leave the heading with a brief placeholder (e.g. "To be determined" or a question)
- Always write in English regardless of the user's language — all task content must be in English
- Keep it concise but specific — the description should be actionable
- Do NOT leave the template empty or just echo it back unchanged

**Example flow:**
1. User: "task: Aktualizujme layout pre login page"
2. You call `task_create(title="Aktualizujme layout pre login page")`
3. Response includes template: "## Objective\n\n## Acceptance criteria\n- [ ]\n..."
4. You generate description filling in the sections based on context
5. You call `task_update(id=X, description="## Objective\nPrepracovať layout login stránky...")
6. Confirm: "Created task #X: Aktualizujme layout pre login page (description generated from template)"

## Workflow-driven task execution

When you start working on a task, always call `task_get` first. The response includes **workflow context**:
- Current phase and phase instruction (AI prompt)
- Next phase
- Phase count

**When working on a task, you MUST:**
1. Read the phase instruction and follow it
2. When the phase work is complete, call `task_advance` to move to the next phase
3. Read the new phase instruction and continue
4. Repeat until the task reaches the final phase (done)

**Example flow:**
1. User: "work on task #42"
2. You call `task_get(id=42)` → response shows phase "research" with instruction "Analyze requirements and propose approach"
3. You do the research, propose an approach
4. You call `task_advance(id=42)` → moves to "implementation" with new instruction
5. You implement, then advance again to "review"
6. Continue until done

**Do NOT skip phases.** Each phase exists for a reason. If a phase seems unnecessary for a specific task, still advance through it with a brief note.

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
| `task_suggest` | Scan code for TODO/FIXME comments and suggest tasks |
| `task_archive` | Archive a completed task |
| `task_bulk_update` | Update multiple tasks at once |
| `task_bulk_complete` | Complete multiple tasks at once |
| `task_advance` | Advance task to next workflow phase |

## Example workflows

### "Bug: login fails on Firefox"
1. Call `project_current` to get current project
2. Parse: title="Login fails on Firefox", description="login fails on Firefox"
3. Call `task_create(project=current, title="Login fails on Firefox", description="login fails on Firefox", type="bug")`
4. Confirm: "Created bug #42 in project webapp"

### "task: I need to refactor the authentication module because the current implementation mixes session handling with JWT validation and it's causing issues when we try to add OAuth providers"
1. Call `project_current` to get current project
2. Parse: title="Refactor authentication module", description="I need to refactor the authentication module because the current implementation mixes session handling with JWT validation and it's causing issues when we try to add OAuth providers"
3. Call `task_create(project=current, title="Refactor authentication module", description="I need to refactor the authentication module because the current implementation mixes session handling with JWT validation and it's causing issues when we try to add OAuth providers")`
4. Confirm: "Created task #43 in project webapp"

### "What am I working on?"
1. Call `task_list(status="doing")`
2. Format as readable list with project, type, title

### "Show me all open bugs"
1. Call `task_list(type="bug", status="todo")`
2. Format as prioritized list
