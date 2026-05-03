---
title: Installation
weight: 1
---

## Prerequisites

- **Go 1.22+** (for building from source)
- **Node.js 18+** (only for VS Code extension)

## Install via Go

```bash
go install github.com/tomas-chudjak/kleidi-task/cmd/klt@latest
```

Verify:

```bash
klt version
```

## Build from source

```bash
git clone https://github.com/tomas-chudjak/kleidi-task.git
cd kleidi-task
```

### Automated setup

```bash
task setup
```

This installs all dev tools (templ, sqlc, goose, air), builds the binary, and symlinks it to `/usr/local/bin/klt`.

### Manual setup

```bash
# Install build tools
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Build
templ generate
go build -o klt ./cmd/klt

# Optional: symlink to PATH
ln -sf $(pwd)/klt /usr/local/bin/klt
```

## Docker

No Go installation needed. See [docker.md](docker.md) for full details.

```bash
git clone https://github.com/tomas-chudjak/kleidi-task.git
cd kleidi-task
docker compose up -d
```

## Verify installation

```bash
# Check binary
klt version

# Initialize a project
cd ~/my-project
klt init

# Start web UI
klt serve
# Open http://localhost:7842
```

## Uninstall

```bash
# Remove binary
rm $(which klt)

# Remove global data (optional)
rm -rf ~/.tasks

# Remove project data (per project, optional)
rm -rf .tasks
```
