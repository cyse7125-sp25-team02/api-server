-- migrations/001_create_health_check_table.sql
CREATE TABLE api.health_check (
    check_id BIGSERIAL PRIMARY KEY,
    datetime TIMESTAMP NOT NULL
);
