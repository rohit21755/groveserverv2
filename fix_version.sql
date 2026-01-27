-- Fix dirty migration state
-- This will reset the migration version to 31 (before the failed migration 32)
-- and mark it as clean (dirty = false)

UPDATE schema_migrations SET version = 31, dirty = false;

-- Verify the fix
SELECT version, dirty FROM schema_migrations;
