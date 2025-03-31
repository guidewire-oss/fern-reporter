-- Drop the single-column index on project_id
DROP INDEX IF EXISTS idx_test_runs_project_id;

-- Drop the composite index on id and project_id
DROP INDEX IF EXISTS idx_test_runs_id_project_id;

ALTER TABLE public.test_runs
DROP CONSTRAINT IF EXISTS fk_test_runs_project_id,
    DROP COLUMN IF EXISTS project_id;