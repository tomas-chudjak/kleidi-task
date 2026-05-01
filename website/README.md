# kvik-tasks Website

Marketing and documentation site built with [Hugo](https://gohugo.io/) + [Hextra](https://imfing.github.io/hextra/) theme.

## Local development

```bash
cd website
hugo server -D
# Open http://localhost:1313
```

## How it works

- Landing page: `website/content/_index.md`
- Docs index: `website/content/docs/_index.md`
- All documentation: mounted from `docs/` (project root) via Hugo module mounts

Edit docs in the root `docs/` directory — changes appear both on GitHub and on the website.

## Deploy to Cloudflare Pages

### Option A: Git integration (recommended)

1. Connect your GitHub repo to Cloudflare Pages
2. Configure:
   - **Build command:** `cd website && hugo --minify`
   - **Build output directory:** `website/public`
   - **Environment variable:** `HUGO_VERSION` = `0.160.1`

### Option B: Direct upload

```bash
cd website
hugo --minify
npx wrangler pages deploy public --project-name=kvik-tasks
```

## Adding documentation

1. Create/edit a markdown file in `docs/` (project root)
2. Add YAML frontmatter with `title` and `weight` (for ordering)
3. The file automatically appears on the website via Hugo module mount

Example:

```markdown
---
title: My New Page
weight: 12
---

# My New Page

Content here...
```
