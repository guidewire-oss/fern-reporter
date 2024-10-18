-- CREATE TABLE public.test_runs (
--     id SERIAL PRIMARY KEY,
--     start_time timestamp with time zone,
--     end_time timestamp with time zone
-- );

CREATE TABLE public.test_runs (
    id bigserial,
    test_seed bigint,
    test_project_name text,
    start_time timestamp with time zone,
    end_time timestamp with time zone,
    enable_gemini_insights boolean NOT NULL DEFAULT true,
    PRIMARY KEY(id, test_seed)
);

