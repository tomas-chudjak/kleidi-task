---
title: VS Code Extension
weight: 11
---

kvik-tasks includes a VS Code extension that brings your tasks directly into the editor. It works in both VS Code and Cursor.

## Prerequisites

The extension communicates with the kvt HTTP server. Make sure it's running before using the extension:

```bash
kvt serve
```

By default, the server runs on `http://localhost:7842`.

## Installation

### Step 1: Build the VSIX package

The extension needs to be built from source. You need Node.js (18+) installed.

```bash
cd vscode-extension
npm install          # install dependencies
npm run build        # compile TypeScript → JavaScript
```

Then create the `.vsix` package:

```bash
npx @vscode/vsce package --allow-missing-repository
```

This produces a file like `kvik-tasks-0.1.0.vsix` in the `vscode-extension/` directory.

### Step 2: Install into VS Code or Cursor

There are three ways to install the `.vsix` file:

**Option A — Command Palette (VS Code & Cursor)**

1. Open VS Code or Cursor
2. Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on Mac)
3. Type "Extensions: Install from VSIX..."
4. Navigate to `vscode-extension/kvik-tasks-0.1.0.vsix` and select it
5. Reload the window when prompted

**Option B — CLI**

```bash
# VS Code
code --install-extension vscode-extension/kvik-tasks-0.1.0.vsix

# Cursor
cursor --install-extension vscode-extension/kvik-tasks-0.1.0.vsix
```

**Option C — Drag and drop**

Open the Extensions panel (`Ctrl+Shift+X`), then drag the `.vsix` file onto the panel.

### Verify installation

After installation, you should see:
- A **Kvik Tasks** icon in the activity bar (left sidebar)
- A **kvt: offline** or **kvt: X todo · Y doing** entry in the status bar (bottom)

If the activity bar icon doesn't appear, reload the window: `Ctrl+Shift+P` → "Developer: Reload Window".

### Uninstall

To remove the extension:
1. Open Extensions panel (`Ctrl+Shift+X`)
2. Find "Kvik Tasks"
3. Click "Uninstall"

Or via CLI: `code --uninstall-extension ahoylog.kvik-tasks`

### Development mode

For contributing or testing changes without creating a `.vsix`:

```bash
cd vscode-extension
npm install
npm run build
code --extensionDevelopmentPath=$(pwd)
```

This opens a new VS Code window with the extension loaded. Changes require rebuilding (`npm run build`) and reloading the window (`Ctrl+Shift+P` → "Developer: Reload Window").

Use `npm run watch` for automatic recompilation on file save.

## Features

### Task sidebar

After installation, a new **Kvik Tasks** icon appears in the activity bar (left side). Clicking it opens the task sidebar showing all tasks from your projects.

- **Single project:** Tasks are listed directly
- **Multiple projects:** Tasks are grouped by project name, each expandable

Each task shows:
- Task title
- ID, type, and priority in the description line
- Status icon: `○` todo, `▶` doing, `✓` done

### Status bar

The bottom status bar shows a summary of your tasks across all projects:

```
☑ kvt: 3 todo · 2 doing
```

If the server is not running, it shows:

```
☑ kvt: offline
```

Click the status bar item to show all tasks in the sidebar.

### Filtering

Use the sidebar menu (three dots at the top of the Tasks panel) to filter:

- **Show Todo** — only tasks with status `todo`
- **Show Doing** — only tasks with status `doing`
- **Show All** — all tasks (default)

You can also trigger filters via Command Palette:
- `Kvik Tasks: Show Todo`
- `Kvik Tasks: Show Doing`
- `Kvik Tasks: Show All`

### Task actions

Right-click any task in the sidebar, or use the inline icon:

#### Insert to Terminal

Sends the task context into the active terminal without pressing Enter:

```
task: #42 Fix login bug (bug, P5, doing)
```

This is useful for AI chat workflows — paste the task into a Claude Code session, Copilot chat, or any terminal-based assistant. The `task:` prefix triggers kvik-tasks' auto-creation behavior in Claude.

If no terminal is open, a new one is created.

#### Open in Browser

Opens the task detail page in your default browser:

```
http://localhost:7842/p/my-project/t/42
```

This gives you access to the full web UI for editing description, viewing workflow timeline, git commits, and more.

#### Copy Reference

Copies the task reference `#42` to the clipboard. Useful for commit messages:

```
git commit -m "fix login validation refs #42"
```

### Refresh

Tasks auto-refresh every 10 seconds by default. You can also manually refresh:

- Click the refresh icon (↻) in the sidebar header
- Command Palette: `Kvik Tasks: Refresh Tasks`

## Configuration

Open VS Code Settings (`Ctrl+,`) and search for "Kvik Tasks":

| Setting | Default | Description |
|---------|---------|-------------|
| `kvikTasks.serverUrl` | `http://localhost:7842` | URL of the kvt server |
| `kvikTasks.refreshInterval` | `10` | Auto-refresh interval in seconds. Set to `0` to disable |

### Custom server URL

If you run kvt on a different port or host (e.g., Docker deployment):

```json
{
  "kvikTasks.serverUrl": "http://192.168.1.100:7842"
}
```

### Authentication

If Basic Auth is enabled on the server, include credentials in the URL:

```json
{
  "kvikTasks.serverUrl": "http://tomas:password@localhost:7842"
}
```

## Typical workflows

### Start of day

1. Open VS Code
2. Glance at the status bar — see how many tasks are todo/doing
3. Open the task sidebar to review your task list
4. Click a task → "Insert to Terminal" → paste into Claude Code to start working

### During development

1. Find a bug → use Claude: `bug: login fails on Firefox`
2. Task appears in sidebar within 10 seconds
3. Fix the bug, commit with `refs #42` in message
4. Mark as done via web UI or Claude

### Commit workflow

1. Right-click task → "Copy Reference"
2. Use `#42` in your commit message
3. Git integration in kvt links the commit to the task automatically

## Troubleshooting

**Sidebar shows "Cannot connect to kvt serve"**
- Make sure `kvt serve` is running
- Check the server URL in settings matches your setup
- Try opening `http://localhost:7842` in a browser

**Tasks not updating**
- Click the refresh button in the sidebar header
- Check if the refresh interval is set to 0 (disabled)

**Extension not appearing in activity bar**
- Reload VS Code window (`Ctrl+Shift+P` → "Developer: Reload Window")
- Check the Extensions panel to verify kvik-tasks is enabled
