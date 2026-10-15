-- One-time setup of the bakery manager inside a shared Supabase project.
-- Run it as the admin role (postgres) from the SQL editor, replacing the placeholder
-- __MANAGER_APP_PASSWORD__ with a long random password first. Never commit the real value.
-- It can be run again: the role is reused and the last password wins.

-- Needed by gen_random_uuid() in the migrations. A non-superuser cannot create it, so the
-- admin installs it here (no-op if Supabase already has it).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'manager_app') THEN
    CREATE ROLE manager_app LOGIN PASSWORD '__MANAGER_APP_PASSWORD__'
      NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  ELSE
    ALTER ROLE manager_app WITH LOGIN PASSWORD '__MANAGER_APP_PASSWORD__'
      NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
END
$$;

CREATE SCHEMA IF NOT EXISTS manager AUTHORIZATION manager_app;

-- Every connection of the role lands in its own schema, so neither the app nor migrate
-- needs a search_path in the connection string.
ALTER ROLE manager_app SET search_path = manager;

-- Least privilege: the role owns "manager" and can do nothing in "public".
REVOKE ALL ON SCHEMA public FROM manager_app;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

-- The Data API roles of Supabase must not see this schema.
REVOKE ALL ON SCHEMA manager FROM PUBLIC;
DO $$
DECLARE r text;
BEGIN
  FOREACH r IN ARRAY ARRAY['anon', 'authenticated', 'service_role'] LOOP
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) THEN
      EXECUTE format('REVOKE ALL ON SCHEMA manager FROM %I', r);
      EXECUTE format('ALTER DEFAULT PRIVILEGES FOR ROLE manager_app IN SCHEMA manager REVOKE ALL ON TABLES FROM %I', r);
    END IF;
  END LOOP;
END
$$;
