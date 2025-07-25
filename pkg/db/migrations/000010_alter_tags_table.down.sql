-- Drop indexes
DROP INDEX IF EXISTS idx_suite_run_tags_tag_id;
DROP INDEX IF EXISTS idx_spec_run_tags_tag_id;
DROP INDEX IF EXISTS idx_tags_category_value;
DROP INDEX IF EXISTS idx_tags_category;

-- Drop columns from tags table
ALTER TABLE public.tags
DROP COLUMN IF EXISTS category,
DROP COLUMN IF EXISTS value;