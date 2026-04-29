-- +goose Up

CREATE TABLE task_templates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL DEFAULT 'task',
    priority    INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Default templates
INSERT INTO task_templates (name, type, priority, description) VALUES
('Task', 'task', 0, '## Objective

## Acceptance criteria
- [ ]

## Notes
');

INSERT INTO task_templates (name, type, priority, description) VALUES
('Bug', 'bug', 5, '## Steps to reproduce
1.

## Expected behavior

## Actual behavior

## Environment
- OS:
- Version:
');

INSERT INTO task_templates (name, type, priority, description) VALUES
('Feature', 'feature', 3, '## Problem

## Proposed solution

## Acceptance criteria
- [ ]

## Out of scope
');

INSERT INTO task_templates (name, type, priority, description) VALUES
('Hotfix', 'hotfix', 8, '## Issue

## Root cause

## Fix

## Verification
- [ ] Fix verified locally
- [ ] No regressions
');

-- +goose Down
DROP TABLE IF EXISTS task_templates;
