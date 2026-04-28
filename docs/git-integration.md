# Git Integration

kvik-tasks automatically links git commits to tasks by scanning commit messages for task references. No configuration or hooks needed — it works out of the box in any git repository.

## How it works

When you open a task detail page, kvik-tasks runs `git log` against the project repository and finds all commits that reference that task ID. Results are displayed in the "Git Activity" card on the task detail page.

This is **on-demand** — there is no background sync, no database table, and no git hooks to install. Every time you view a task, the git log is scanned fresh, so results are always up to date.

## Reference format

Include a task reference anywhere in your commit message. Supported formats:

| Format | Example | Description |
|--------|---------|-------------|
| `#ID` | `#15` | Basic reference |
| `kvt:ID` | `kvt:15` | Explicit kvik-tasks reference |
| `fixes #ID` | `fixes #15` | Marks commit as a fix |
| `closes #ID` | `closes #15` | Marks commit as closing the task |
| `refs #ID` | `refs #15` | General reference |
| `re #ID` | `re #15` | Short reference |

Prefixes (`fixes`, `closes`, `refs`, `re`) work with both `#ID` and `kvt:ID` notation:

```
fixes kvt:15
closes #42
refs #7
```

### Multiple references

A single commit can reference multiple tasks:

```
feat: add search bar and keyboard shortcuts #7 #10
```

### Regex pattern

The full pattern used for matching: `(?:(?:fixes|closes|refs|re)\s+)?(?:#|kvt:)(\d+)`

## Usage examples

```bash
# Basic reference
git commit -m "add pagination to task list #19"

# Fix reference
git commit -m "fixes #24 — date inputs were autofilled by password manager"

# Multiple tasks
git commit -m "feat: filtering UI with multi-select #8 #21"

# Explicit kvik-tasks reference (useful if #ID is ambiguous with GitHub issues)
git commit -m "refactor task service kvt:12"
```

## What you see

On the task detail page, the "Git Activity" card shows:

- **Commit hash** (7-char short hash, highlighted)
- **Commit message** (subject line)
- **Author** and **date**

Commits are sorted newest-first (git log default).

## When no commits are found

If no commits reference the task, the card shows a hint:

> No commits reference this task. Use `#15` in commit messages.

## Requirements

- The project directory must be a git repository (contains `.git`)
- `git` must be available on the system PATH
- Scans all branches (`--all` flag)

If either condition is not met, the Git Activity card silently shows no results.

## Architecture notes

- **No database storage** — commits are not persisted in SQLite. This avoids sync/staleness issues.
- **On-demand parsing** — `git log --grep` does the initial filtering, then Go-side regex confirms the exact task ID match (prevents `#1` matching `#10`, `#100`, etc.)
- **Performance** — for typical single-developer repositories, `git log --grep` returns in milliseconds. No caching is needed.
- **Service**: `GitService.CommitsForTask(ctx, projectPath, taskID)` in `internal/core/git.go`
