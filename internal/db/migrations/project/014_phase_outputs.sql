-- +goose Up

-- Maps a phase to the task description section its work must land in.
-- JSON: {"phase_name": "Section heading"}
--
-- A phase that produces analysis needs somewhere durable to put it. Without
-- this the design lives in the chat, dies with the session, and the next
-- session opens a task that looks exactly as it did before the phase ran.
ALTER TABLE workflows ADD COLUMN phase_outputs TEXT NOT NULL DEFAULT '{}';

UPDATE workflows SET phase_outputs = '{"research":"Design"}' WHERE task_type = 'feature';
UPDATE workflows SET phase_outputs = '{"reproducing":"Root cause"}' WHERE task_type = 'bug';

-- The research phase now writes its conclusion into the description.
UPDATE workflows SET phase_prompts = '{"research":"Analyze the requirements. Explore the codebase for relevant patterns and decide on an implementation approach. Write the conclusion into the task description under \"## Design\" via task_update — a design left in the chat is lost when the session ends. Propose the update and let the author accept it before advancing.","implementation":"Implement the feature following the approach recorded under \"## Design\". Write clean, well-structured code.","review":"Review the implementation for correctness, code quality, and test coverage. Run all tests."}'
WHERE task_type = 'feature';

UPDATE workflows SET phase_prompts = '{"reproducing":"Analyze the bug report. Reproduce the issue and identify the root cause. Record the root cause in the task description under \"## Root cause\" via task_update, along with the exact steps that reproduce it.","fixing":"Implement the fix based on the recorded root cause. Keep changes minimal and focused.","verifying":"Run tests to verify the fix works. Check for regressions. Ensure the original issue is resolved."}'
WHERE task_type = 'bug';

-- +goose Down
ALTER TABLE workflows DROP COLUMN phase_outputs;
