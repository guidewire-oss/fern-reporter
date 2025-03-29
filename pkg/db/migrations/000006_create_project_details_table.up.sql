CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE project_details(
    id   SERIAL PRIMARY KEY,
    uuid UUID GENERATED ALWAYS AS (uuid_generate_v5('6ba7b812-9dad-11d1-80b4-00c04fd430c8', id::text)) STORED,
    name VARCHAR(255) NOT NULL,
    team_name    VARCHAR(100),
    comment      VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Composite index for id and uuid
CREATE INDEX idx_project_details_id_uuid ON project_details (id, uuid);

-- Index for uuid (to speed up lookups by uuid)
CREATE INDEX idx_project_details_uuid ON project_details (uuid);
