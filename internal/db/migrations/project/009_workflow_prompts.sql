-- +goose Up

-- Add phase prompts to workflows (JSON: {"phase_name": "prompt text"})
ALTER TABLE workflows ADD COLUMN phase_prompts TEXT NOT NULL DEFAULT '{}';

-- Set default prompts for existing workflows
UPDATE workflows SET phase_prompts = '{"reproducing":"Analyze the bug report. Try to reproduce the issue and identify the root cause. Document steps to reproduce.","fixing":"Implement the fix based on root cause analysis. Keep changes minimal and focused.","verifying":"Run tests to verify the fix works. Check for regressions. Ensure the original issue is resolved."}' WHERE task_type = 'bug';

UPDATE workflows SET phase_prompts = '{"research":"Analyze the requirements. Explore the codebase for relevant patterns and propose an implementation approach.","implementation":"Implement the feature following the proposed approach. Write clean, well-structured code.","review":"Review the implementation for correctness, code quality, and test coverage. Run all tests."}' WHERE task_type = 'feature';

UPDATE workflows SET phase_prompts = '{"doing":"Work on this task. Follow project conventions and best practices."}' WHERE task_type = 'task';

UPDATE workflows SET phase_prompts = '{"fixing":"Identify and fix the critical issue. Prioritize speed and correctness over elegance.","verifying":"Run full test suite and smoke tests. Verify the fix resolves the issue without regressions."}' WHERE task_type = 'hotfix';

-- +goose Down
-- SQLite doesn't support DROP COLUMN before 3.35.0, but modernc.org/sqlite supports it
ALTER TABLE workflows DROP COLUMN phase_prompts;
