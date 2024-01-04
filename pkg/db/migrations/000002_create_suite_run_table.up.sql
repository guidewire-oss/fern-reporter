-- CREATE TABLE IF NOT EXISTS suite_runs (
--   id SERIAL PRIMARY KEY,
--   test_run_id INT,
--   start_time TIMESTAMP,
--   end_time TIMESTAMP,
--   FOREIGN KEY(test_run_id)
--     REFERENCES test_runs (id)
-- )


CREATE TABLE public.suite_runs (
    id bigserial PRIMARY KEY,
    suite_name text,
    test_run_id bigint,
    test_run_seed bigint,
    start_time timestamp with time zone,
    end_time timestamp with time zone,
    FOREIGN KEY (test_run_id, test_run_seed)
    REFERENCES test_runs(id, test_seed)
    ON UPDATE CASCADE ON DELETE CASCADE
);
