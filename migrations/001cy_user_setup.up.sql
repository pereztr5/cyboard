BEGIN;

-- Create a regular (non-superuser) role & a user with login in that role.
-- The DB admin can add other regular users to the cyboard_role to grant the same access.
DO $$
BEGIN
  BEGIN CREATE ROLE cyboard_role WITH NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END;
  BEGIN CREATE USER cyboard IN ROLE cyboard_role; EXCEPTION WHEN duplicate_object THEN NULL; END;
END
$$;


-- Create a schema to namespace all our tables & stuff to.
-- The 'cyboard' user will automatically use the 'cyboard' schema by default.
-- Other users must add the 'cyboard' schema to their search path.
CREATE SCHEMA IF NOT EXISTS cyboard;

GRANT USAGE ON SCHEMA cyboard TO cyboard_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA cyboard
  GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE ON TABLES TO cyboard_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA cyboard
  GRANT USAGE ON SEQUENCES TO cyboard_role;

--ALTER ROLE cyboard SET search_path = cyboard, "$user", public;

COMMIT;
