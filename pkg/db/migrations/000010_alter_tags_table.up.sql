ALTER TABLE public.tags
ADD COLUMN category TEXT,
ADD COLUMN value TEXT;

CREATE INDEX idx_tags_category ON tags (category);
CREATE INDEX idx_tags_category_value ON tags (category, value);
CREATE INDEX idx_spec_run_tags_tag_id ON spec_run_tags (tag_id);
CREATE INDEX idx_suite_run_tags_tag_id ON suite_run_tags (tag_id);
