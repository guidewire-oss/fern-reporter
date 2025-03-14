ALTER TABLE public.test_runs

ADD COLUMN project_id INT,
ADD CONSTRAINT fk_test_runs_project_id
    FOREIGN KEY (project_id)
        REFERENCES project_details (id);

-- Index for project_id to optimize lookups based on project_id
CREATE INDEX idx_test_runs_project_id ON test_runs (project_id);

-- Composite index for id and project_id to optimize queries using both columns
CREATE INDEX idx_test_runs_id_project_id ON test_runs (id, project_id);
