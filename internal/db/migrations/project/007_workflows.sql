-- +goose Up

-- Add phase column to tasks (nullable — null means "use status as phase")
ALTER TABLE tasks ADD COLUMN phase TEXT;

-- Workflow definitions per task type
CREATE TABLE workflows (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    task_type   TEXT NOT NULL UNIQUE,
    phases      TEXT NOT NULL,  -- JSON array of phase names
    triggers    TEXT NOT NULL DEFAULT '{}'  -- JSON: {"phase_name": {"before": [...], "after": [...]}}
);

-- Default workflows
INSERT INTO workflows (task_type, phases, triggers) VALUES
('task', '["todo","doing","done"]', '{}');

INSERT INTO workflows (task_type, phases, triggers) VALUES
('bug', '["reported","reproducing","fixing","verifying","done"]', '{"reproducing":{"before":["gather-context"]},"fixing":{"after":["lint","type-check"]},"verifying":{"before":["run-tests"]}}');

INSERT INTO workflows (task_type, phases, triggers) VALUES
('feature', '["todo","research","implementation","review","done"]', '{"research":{"before":["research","architecture-review"]},"implementation":{"after":["lint","type-check"]},"review":{"before":["run-tests","code-review"]}}');

INSERT INTO workflows (task_type, phases, triggers) VALUES
('hotfix', '["reported","fixing","verifying","done"]', '{"fixing":{"before":["root-cause-analysis"]},"verifying":{"before":["run-tests","smoke-test"]}}');

-- +goose Down
ALTER TABLE tasks DROP COLUMN phase;
DROP TABLE IF EXISTS workflows;
