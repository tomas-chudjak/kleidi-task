-- +goose Up

-- Maintenance rules handed to the agent filling a template. Kept out of the
-- description on purpose: these instruct the agent, they are not part of the
-- task text a human reads in the UI, CLI or kanban.
ALTER TABLE task_templates ADD COLUMN agent_rules TEXT NOT NULL DEFAULT '';

-- Open questions gives unresolved decisions a visible home. Without it an
-- agent buries uncertainty inside a section as a placeholder, and the next
-- session walks past it with an invented assumption.
UPDATE task_templates
SET description = description || '
## Open questions
'
WHERE type IN ('task', 'feature')
  AND description NOT LIKE '%## Open questions%';

UPDATE task_templates SET agent_rules = 'Final designs only — describe the target state, not the history of how it was decided. Do not track changes or weigh rejected alternatives in the description.
Use future tense for the target design ("X will use Y") or imperative for work to do ("Update X to Y"). Present tense describes only what is already implemented.
Anything you cannot answer from the available context goes under "## Open questions" as a question. Never leave a placeholder like "TBD" inside another section.
Keep entries brief and specific. Prefer short paragraphs and lists over blocks of prose.
Do not nest headings deeper than h3.
Every section of this template must be present in the description.'
WHERE agent_rules = '';

-- +goose Down
ALTER TABLE task_templates DROP COLUMN agent_rules;
UPDATE task_templates SET description = REPLACE(description, '
## Open questions
', '') WHERE type IN ('task', 'feature');
