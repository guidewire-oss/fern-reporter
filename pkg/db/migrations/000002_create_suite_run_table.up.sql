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
    test_run_id bigint,
    start_time timestamp with time zone,
    end_time timestamp with time zone,
    FOREIGN KEY (test_run_id)
    REFERENCES public.test_runs(id)
    ON UPDATE CASCADE ON DELETE CASCADE
);
