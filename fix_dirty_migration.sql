-- Fix dirty migration state
-- This script will:
-- 1. Check current migration state
-- 2. Fix the dirty flag
-- 3. Verify the tasks table state

-- First, let's check the current state
SELECT version, dirty FROM schema_migrations;

-- Force the version to 31 (clean state before migration 32)
UPDATE schema_migrations SET version = 31, dirty = false;

-- Verify the update
SELECT version, dirty FROM schema_migrations;

-- Check if the constraint still exists (if migration partially ran)
SELECT 
    conname as constraint_name,
    contype as constraint_type
FROM pg_constraint
WHERE conrelid = 'tasks'::regclass
  AND contype = 'f'
  AND 'created_by' = ANY(
    SELECT attname 
    FROM pg_attribute 
    WHERE attrelid = 'tasks'::regclass 
      AND attnum = ANY(conkey)
  );

-- Check if created_by is still NOT NULL
SELECT 
    column_name,
    is_nullable,
    data_type
FROM information_schema.columns
WHERE table_name = 'tasks' 
  AND column_name = 'created_by';
