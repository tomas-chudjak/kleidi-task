-- +goose Up

-- The prompt task_review hands to the agent. Stored per template so it can be
-- edited without a rebuild, rather than hardcoded in Go.
ALTER TABLE task_templates ADD COLUMN review_instruction TEXT NOT NULL DEFAULT '';

UPDATE task_templates SET review_instruction = 'Review this task against the codebase and any relevant documents.

Are there any gaps in the requirements? Any conflicts — between sections, with existing code, or with project conventions? Any significant decisions still unmade?

Sort what you find into three groups:
- resolved: issues with a single obvious best answer. Say what the answer is and which section it belongs in.
- gaps: something required is missing or too vague to act on.
- conflicts: two parts of the task, or the task and the codebase, disagree.

Report your findings. Do not rewrite the task yourself — the author accepts them first. For anything accepted with a single obvious answer, update that section via task_update. Everything else becomes a question under "## Open questions".'
WHERE review_instruction = '';

-- +goose Down
ALTER TABLE task_templates DROP COLUMN review_instruction;
