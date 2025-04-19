-- App User Table
CREATE TABLE app_users
(
    id       SERIAL PRIMARY KEY,
    is_dark  BOOLEAN,
    timezone VARCHAR(100),
    cookie   VARCHAR(100)
);

-- User Project Groups Table
CREATE TABLE project_groups
(
    group_id   SERIAL PRIMARY KEY,
    user_id    INT          NOT NULL,
    group_name VARCHAR(255) NOT NULL,
    CONSTRAINT unique_group_per_user UNIQUE (user_id, group_name),
    CONSTRAINT fk_user
        FOREIGN KEY (user_id)
            REFERENCES app_users (id)
            ON DELETE CASCADE
);

-- User Preferred Projects Table
CREATE TABLE preferred_projects
(
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL,
    project_id INT NOT NULL, -- References project_details.project_id
    group_id   INT,          -- References user_project_groups.group_id (NULL for ungrouped)
    CONSTRAINT fk_user
        FOREIGN KEY (user_id)
            REFERENCES app_users (id)
            ON DELETE CASCADE,
    CONSTRAINT fk_project
        FOREIGN KEY (project_id)
            REFERENCES project_details (id)
            ON DELETE CASCADE,
    CONSTRAINT fk_group
        FOREIGN KEY (group_id)
            REFERENCES project_groups (group_id)
            ON DELETE SET NULL
);


-- Index
CREATE UNIQUE INDEX idx_user_project ON preferred_projects (user_id, project_id);
