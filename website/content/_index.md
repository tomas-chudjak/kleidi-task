---
title: kvik-tasks
layout: hextra-home
---

{{< hextra/hero-badge >}}
  <div class="hx-w-2 hx-h-2 hx-rounded-full hx-bg-primary-400"></div>
  <span>Open Source — MIT License</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx-mt-6 hx-mb-6">
{{< hextra/hero-headline >}}
  Task tracking built for&nbsp;<br class="sm:hx-block hx-hidden" />AI-assisted development
{{< /hextra/hero-headline >}}
</div>

<div class="hx-mb-12">
{{< hextra/hero-subtitle >}}
  Local-first, single-binary task tracker designed for developers&nbsp;<br class="sm:hx-block hx-hidden" />who work alongside AI assistants like Claude, Cursor, and Copilot.
{{< /hextra/hero-subtitle >}}
</div>

<div style="margin-top:1rem;">
{{< hextra/hero-button text="Get Started" link="docs/installation" >}}
{{< hextra/hero-button text="GitHub" link="https://github.com/tomas-chudjak/kvik-tasks" style="alt" >}}
</div>

<div style="margin-top: 2rem; margin-bottom: 2rem;">
{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="MCP-First Design"
    subtitle="The AI integration is a primary interface, not a wrapper. Claude, Cursor, and Copilot connect directly via Model Context Protocol."
    link="docs/mcp-usage"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(59,130,246,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Single Binary, Zero Dependencies"
    subtitle="One Go binary with embedded SQLite. No Docker required, no external services, no configuration files needed to start."
    link="docs/installation"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-lg:hx-min-h-[340px]"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(16,185,129,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Per-Project Storage"
    subtitle="Each project gets its own SQLite database in .tasks/ — tasks live with your code. No cloud sync, no vendor lock-in."
    link="docs/configuration"
    class="hx-aspect-auto md:hx-aspect-[1.1/1] max-md:hx-min-h-[340px]"
    style="background: radial-gradient(ellipse at 50% 80%,rgba(245,158,11,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="4 Interfaces, 1 Service Layer"
    subtitle="CLI, REST API, Web UI (HTMX + Kanban), and MCP server — all sharing one service layer. Use whichever fits your workflow."
    link="docs/cli"
  >}}
  {{< hextra/feature-card
    title="Phase-Based Workflows"
    subtitle="Each task type has a workflow with phases, AI prompts, and triggers. Bugs go through reported → reproducing → fixing → verifying → done."
    link="docs/workflows"
  >}}
  {{< hextra/feature-card
    title="Developer-Native"
    subtitle="Git commit linking, TODO/FIXME scanning, VS Code extension, script hooks on task events, full-text search. Built by developers, for developers."
    link="docs/git-integration"
  >}}
{{< /hextra/feature-grid >}}
</div>

## Quick Start

```bash
# Install
go install github.com/tomas-chudjak/kvik-tasks/cmd/kvt@latest

# Initialize in your project
cd my-project
kvt init

# Add tasks — AI or human
kvt add "Implement user authentication"
kvt add "BUG: Login fails on Firefox"

# Connect your AI assistant
claude mcp add kvik-tasks -- kvt mcp
```

## How It Works

```
You (CLI / Browser / Claude / Cursor)
    │
  kvt binary (single process)
    │
  Service Layer (TaskService, ProjectService, WorkflowService)
    │
  SQLite (per-project .tasks/tasks.db + global ~/.tasks/registry.db)
```

Tasks are stored locally in your project directory. A global registry maps projects by slug. Cross-project queries aggregate on demand.

## Why kvik-tasks?

Existing task managers are designed for humans clicking buttons. **kvik-tasks** is designed for the way modern developers actually work — with AI assistants in the loop.

- Say `"task: implement search"` in Claude and the task is created instantly
- AI assistants read your task context and follow phase-specific instructions
- No context switching between your editor and a task management app
- Everything runs locally — your tasks never leave your machine
