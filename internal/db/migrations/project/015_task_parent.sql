-- +goose Up

-- Parent/child decomposition: a large feature split into PR-sized deliverables.
-- Limited to one level (a child never becomes a parent) — deeper nesting
-- complicates every query and view for little gain. Enforced in the service.
ALTER TABLE tasks ADD COLUMN parent_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE;

-- Position among siblings. The point of splitting is the order the work lands in.
ALTER TABLE tasks ADD COLUMN child_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_id);

-- +goose Down
DROP INDEX IF EXISTS idx_tasks_parent;
ALTER TABLE tasks DROP COLUMN child_order;
ALTER TABLE tasks DROP COLUMN parent_id;
