DROP EXTENSION IF EXISTS "uuid-ossp";

-- Drop composite index on id and uuid
DROP INDEX IF EXISTS idx_project_details_id_uuid;

-- Drop index on uuid
DROP INDEX IF EXISTS idx_project_details_uuid;

DROP TABLE IF EXISTS project_details;

