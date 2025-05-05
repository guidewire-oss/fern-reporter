-- Drop indexes first
DROP INDEX IF EXISTS idx_user_project_group;
DROP INDEX IF EXISTS idx_app_user_cookie;
DROP INDEX IF EXISTS idx_user_id_group_id;
DROP INDEX IF EXISTS idx_user_id;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS preferred_projects;
DROP TABLE IF EXISTS project_groups;
DROP TABLE IF EXISTS app_users;
