---
title: Docker Deployment
weight: 10
---

kleidi-task can run as a Docker container for team deployments.

## Quick start

```bash
git clone https://github.com/tomas-chudjak/kleidi-task.git
cd kleidi-task
docker compose up -d
```

Web UI available at http://localhost:7842.

## docker-compose.yml

```yaml
services:
  klt:
    build: .
    ports:
      - "7842:7842"
    volumes:
      - klt-data:/data
    restart: unless-stopped

volumes:
  klt-data:
```

## Data persistence

Task databases and configuration are stored in a named volume (`klt-data`). This persists across container restarts and rebuilds.

The container uses `/data` as the home directory, so:
- Registry: `/data/.tasks/registry.db`
- Config: `/data/.tasks/config.json`

## Custom port

```yaml
services:
  klt:
    build: .
    ports:
      - "8080:8080"
    command: ["serve", "--host", "0.0.0.0", "--port", "8080"]
```

## Adding users

Create users for Basic Auth after the container is running:

```bash
docker compose exec klt klt user add tomas
```

See [authentication.md](authentication.md) for details.

## Production deployment

For production, put kleidi-task behind a reverse proxy with TLS:

### Caddy

```
tasks.example.com {
    reverse_proxy klt:7842
}
```

### Nginx

```nginx
server {
    listen 443 ssl;
    server_name tasks.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://klt:7842;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Building the image

The Dockerfile uses a multi-stage build:

1. **Builder stage** — golang:1.25-alpine, installs templ, compiles the binary
2. **Runtime stage** — alpine:3.21, copies only the binary (~30MB image)

```bash
docker build -t klt .
docker run -p 7842:7842 -v klt-data:/data klt
```

## Initializing projects

Projects are created by running `klt init` inside a directory. In Docker, projects can be created via the MCP server or REST API — the web UI dashboard shows all registered projects.

To mount a host directory as a project:

```yaml
services:
  klt:
    build: .
    ports:
      - "7842:7842"
    volumes:
      - klt-data:/data
      - ./my-project:/projects/my-project
```

Then initialize it:

```bash
docker compose exec -w /projects/my-project klt klt init
```
