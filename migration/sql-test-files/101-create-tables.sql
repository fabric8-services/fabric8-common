CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    name text,
    age smallint
);
