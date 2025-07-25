CREATE TABLE public.suite_run_tags (
    suite_run_id BIGINT,
    tag_id BIGINT,
    FOREIGN KEY (suite_run_id)
    REFERENCES suite_runs (id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (tag_id)
    REFERENCES tags (id) ON UPDATE CASCADE ON DELETE CASCADE,
    PRIMARY KEY (suite_run_id, tag_id)
);
