-- Runs once when the data directory is first created.
--
-- Schemas are NOT created here: each service owns its schema and creates it in
-- its own first migration, so a service is deployable without touching this
-- file.

-- gen_random_uuid() for primary keys.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Case-insensitive text, for emails and similar.
CREATE EXTENSION IF NOT EXISTS citext;

-- Revoke the default public-schema rights so a service cannot accidentally
-- create tables outside its own schema.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
