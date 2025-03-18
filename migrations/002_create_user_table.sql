-- migrations/002_create_user_table.sql
CREATE SCHEMA IF NOT EXISTS webapp;

CREATE TABLE webapp.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NULL,
    username VARCHAR(30) NOT NULL UNIQUE,
    password VARCHAR(60) NOT NULL, -- bcrypt hashes are typically 60 characters
    role VARCHAR(10) NOT NULL CHECK (role IN ('admin')),
    email VARCHAR(100) NOT NULL UNIQUE CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    account_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    account_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

