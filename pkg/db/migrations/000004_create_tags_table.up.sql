CREATE TABLE public.tags (
    id bigserial PRIMARY KEY,
    name text
);


CREATE TABLE public.spec_run_tags (
    spec_run_id BIGINT,
    tag_id BIGINT,
    FOREIGN KEY (spec_run_id)
    REFERENCES spec_runs (id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (tag_id)
    REFERENCES tags (id) ON UPDATE CASCADE ON DELETE CASCADE,
    PRIMARY KEY (spec_run_id, tag_id)
);
