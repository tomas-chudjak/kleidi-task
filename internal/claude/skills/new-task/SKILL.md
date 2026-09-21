---
name: new-task
description: Create a new kleidi-task item. Use when user invokes /new-task with a type and title.
disable-model-invocation: true
argument-hint: <type> <title...>
---

# Create New Task

Create a new task in the current kleidi-task project using MCP tools.

## Arguments

`$ARGUMENTS` format: `<type> <title and context...>`

- **First word** = task type: `task`, `bug`, `feature`, `hotfix`
  - If omitted or unrecognized, default to `task`
- **Remaining words** = title and context for the task

Examples:
- `/new-task bug Login page crashes on Safari with double submit`
- `/new-task feature Add dark mode support`
- `/new-task hotfix Fix production database connection timeout`
- `/new-task Refactor authentication module` (type defaults to `task`)

## Procedure

Follow these steps exactly, in order:

### 1. Parse arguments

From `$ARGUMENTS`, extract:
- **type**: first word if it matches `task|bug|feature|hotfix`, otherwise default to `task` and treat the entire input as title/context
- **raw_input**: everything after the type (or the full input if no type matched)

### 2. Detect project

Call `project_current` MCP tool to get the current project slug. If no project is detected, tell the user to run `klt init` first and stop.

### 3. Fetch template

Call `template_get` MCP tool with the detected type to get the structured description template for this task type.

The template contains section headings (e.g. `## Steps to reproduce`, `## Expected behavior` for bugs). Your job is to **fill in every section** based on the user's input.

### 4. Generate title and description

- **title**: Extract a short, clear summary (max ~10 words) from the raw input. Make it actionable.
- **description**: Take the template from step 3 and fill in each section with content derived from the user's input. Write in English regardless of user's language.

**Rules for filling the template:**
- Fill every section you have information for — even partial info is better than empty
- Anything you cannot answer from the user's input goes under `## Open questions` as a question. Never leave a placeholder like "To be determined" inside another section — a buried unknown is one the next session walks past with an invented assumption
- Be specific and actionable — use the user's exact details, don't generalize
- Never return the template with empty sections if the user gave you enough context to fill them
- `template_get` also returns **agent rules** for the type. Follow them, and never copy them into the description
- Every section heading from the template must appear in the description. When the project runs strict template enforcement, `task_create` rejects a description that is missing one

**Example — `/new-task bug hello world nefunguje, vypise error "undefined variable" na riadku 42`:**

Template from `template_get(type="bug")`:
```
## Steps to reproduce
1.

## Expected behavior

## Actual behavior

## Environment
- OS:
- Version:
```

Filled description:
```
## Steps to reproduce
1. Run the hello world program

## Expected behavior
Program prints "Hello, World!" to stdout

## Actual behavior
Program fails with error "undefined variable" on line 42

## Environment
- OS: To be determined
- Version: To be determined
```

(The `bug` template has no Open questions section — unknown environment values stay as
placeholders there. For `task` and `feature`, which do have the section, an unknown becomes
a question under `## Open questions` instead.)

### 5. Check for duplicates

Call `task_list` with `status=todo` and `status=doing` to check if a similar task already exists. If a clear duplicate exists, inform the user and stop.

### 6. Create the task

Call `task_create` with:
- `project`: current project slug
- `title`: the generated title
- `type`: the detected type
- `description`: the filled template

### 7. Confirm

Report to the user: `Created <type> #<id>: <title>`

If the task has a workflow, briefly mention the first phase.
