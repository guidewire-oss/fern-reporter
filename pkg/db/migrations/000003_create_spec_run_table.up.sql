-- CREATE TABLE IF NOT EXISTS spec_runs (
--   id SERIAL PRIMARY KEY,
--   suite_id INT,
--   spec_description VARCHAR(100),
--   status VARCHAR(10),
--   Message VARCHAR(255),
--   start_time TIMESTAMP,
--   end_time TIMESTAMP,
--   FOREIGN KEY (suite_id)
--     REFERENCES suite_runs(id)
-- );
--

CREATE TABLE public.spec_runs (
    id bigserial PRIMARY KEY,
    suite_id bigint,
    spec_description text,
    status text,
    message text,
    start_time timestamp with time zone,
    end_time timestamp with time zone,
    FOREIGN KEY (suite_id)
    REFERENCES public.suite_runs(id)
    ON UPDATE CASCADE ON DELETE CASCADE
);
