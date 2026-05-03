---
title: Authentication
weight: 8
---

kleidi-task supports optional HTTP Basic Auth for team deployments. When no users are configured, the app runs in open-access mode — no credentials needed.

## How it works

1. Admin creates users via CLI: `klt user add <username>`
2. Server detects users with passwords exist → enables auth
3. Browser shows native username/password popup on first request
4. Credentials are verified against bcrypt hashes in the registry database

No login pages, no registration forms, no sessions — the browser handles the UI natively.

## Managing users

### Add a user

```bash
klt user add tomas
# Password for tomas: ****
# User "tomas" created (ID: 2)
```

The password is read securely (hidden input) and stored as a bcrypt hash.

### List users

```bash
klt user list
# ID  Username  Created
# 2   tomas     2026-04-30
# 3   peter     2026-04-30
```

The default `local` user (ID 1) is hidden from the list — it's used internally when auth is disabled.

## When is auth active?

Auth is **automatically enabled** when at least one user with a password exists in the registry database. If no users have passwords, the app runs without authentication.

This means:
- **Fresh install** → no auth (open access)
- **After `klt user add`** → auth enabled (browser popup)
- **Delete all users from DB** → back to open access

## Docker deployment

For Docker deployments, create users after the container starts:

```bash
# Enter the running container
docker exec -it klt-container klt user add tomas

# Or use docker compose
docker compose exec klt klt user add tomas
```

Alternatively, create users in a startup script or init container.

## Task attribution

When auth is enabled, tasks are attributed to the authenticated user:

- `created_by` field stores the user ID of whoever created the task
- The default `local` user (ID 1) is used when auth is disabled

Future enhancements (not yet implemented):
- Task assignment (`assigned_to` field exists in schema)
- Filter by "my tasks"
- User avatars/display names

## Technical details

- Passwords are hashed with **bcrypt** (default cost)
- Users are stored in the **registry database** (`~/.tasks/registry.db`), shared across all projects
- Auth middleware runs on all routes (REST API, UI, static assets)
- MCP stdio transport is **not affected** — auth only applies to HTTP

## Security considerations

- HTTP Basic Auth sends credentials in base64 (not encrypted). For production use over the internet, always put kleidi-task behind a **reverse proxy with TLS** (Nginx, Caddy, Traefik).
- For local network / VPN use, Basic Auth provides sufficient protection.
- There is no rate limiting on auth attempts. For public-facing deployments, use a reverse proxy with rate limiting.
