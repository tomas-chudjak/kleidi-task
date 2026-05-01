---
title: REST API
weight: 4
---

The REST API is available when `kvt serve` is running. All endpoints are prefixed with `/api/v1`.

Base URL: `http://localhost:7842/api/v1`

## Authentication

If Basic Auth is enabled (users with passwords exist), all API requests require credentials:

```bash
curl -u username:password http://localhost:7842/api/v1/projects
```

When no users are configured, the API is open access.

## Projects

### List projects

```bash
curl http://localhost:7842/api/v1/projects
```

Response:

```json
[
  {
    "id": 1,
    "slug": "kvik-tasks",
    "name": "kvik-tasks",
    "path": "/root/projects/kvik-tasks",
    "cached_todo_count": 3,
    "cached_doing_count": 1,
    "cached_total_count": 42
  }
]
```

### Get project

```bash
curl http://localhost:7842/api/v1/projects/kvik-tasks
```

### Get project stats

```bash
curl http://localhost:7842/api/v1/projects/kvik-tasks/stats
```

Response:

```json
{
  "todo": 3,
  "doing": 1,
  "done": 38,
  "bugs_open": 0
}
```

## Tasks

### List tasks

```bash
curl http://localhost:7842/api/v1/projects/kvik-tasks/tasks
```

Query parameters:

| Parameter | Description | Example |
|-----------|-------------|---------|
| `status` | Filter by status | `todo`, `doing`, `done` |
| `type` | Filter by type | `task`, `bug`, `feature` |
| `limit` | Max results (default 50) | `100` |
| `offset` | Pagination offset | `50` |
| `min_priority` | Minimum priority | `3` |
| `created_after` | ISO 8601 date | `2026-04-01` |
| `created_before` | ISO 8601 date | `2026-04-30` |

Example with filters:

```bash
curl "http://localhost:7842/api/v1/projects/kvik-tasks/tasks?status=todo&type=bug&limit=10"
```

Response:

```json
{
  "tasks": [
    {
      "id": 42,
      "type": "bug",
      "title": "Login fails on Firefox",
      "description": "Steps to reproduce...",
      "status": "todo",
      "priority": 5,
      "category": "frontend",
      "phase": "",
      "is_archived": false,
      "source": "mcp",
      "created_at": "2026-04-28T10:00:00Z",
      "updated_at": "2026-04-28T10:00:00Z",
      "created_by": 1
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0,
  "total_pages": 1,
  "page": 1
}
```

### Create task

```bash
curl -X POST http://localhost:7842/api/v1/projects/kvik-tasks/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Fix login bug", "type": "bug", "priority": 5}'
```

### Get task

```bash
curl http://localhost:7842/api/v1/projects/kvik-tasks/tasks/42
```

### Update task

```bash
curl -X PATCH http://localhost:7842/api/v1/projects/kvik-tasks/tasks/42 \
  -H "Content-Type: application/json" \
  -d '{"status": "doing", "priority": 8}'
```

All fields are optional — only provided fields are updated.

### Delete task

```bash
curl -X DELETE http://localhost:7842/api/v1/projects/kvik-tasks/tasks/42
```

### Complete task

```bash
curl -X POST http://localhost:7842/api/v1/projects/kvik-tasks/tasks/42/complete
```

### Archive / Unarchive task

```bash
curl -X POST http://localhost:7842/api/v1/projects/kvik-tasks/tasks/42/archive
curl -X POST http://localhost:7842/api/v1/projects/kvik-tasks/tasks/42/unarchive
```

## System

### Health check

```bash
curl http://localhost:7842/api/v1/health
```

### Version

```bash
curl http://localhost:7842/api/v1/version
```
