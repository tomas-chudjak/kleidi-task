-- +goose Up

-- Enforcement policy for template-driven task creation.
-- off    — no validation
-- warn   — log missing sections, create the task anyway
-- strict — reject programmatic creates (mcp, api) that skip the template
INSERT INTO project_config (key, value) VALUES ('template_enforcement', 'warn');

-- +goose Down
DELETE FROM project_config WHERE key = 'template_enforcement';
