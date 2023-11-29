-- CREATE TABLE public.test_runs (
--     id SERIAL PRIMARY KEY,
--     start_time timestamp with time zone,
--     end_time timestamp with time zone
-- );

CREATE TABLE public.test_runs (
    id bigserial PRIMARY KEY,
    start_time timestamp with time zone,
    end_time timestamp with time zone
);

