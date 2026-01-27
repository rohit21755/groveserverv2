-- Update tasks table to support admin-created tasks
-- Remove foreign key constraint on created_by since it can reference either users or admins
-- Make created_by nullable to allow system-generated tasks

-- Drop the existing foreign key constraint
-- PostgreSQL typically names it tasks_created_by_fkey, but we'll find and drop it dynamically
DO $$
DECLARE
    constraint_name text;
    created_by_attnum smallint;
BEGIN
    -- Get the attribute number for created_by column
    SELECT attnum INTO created_by_attnum
    FROM pg_attribute
    WHERE attrelid = 'tasks'::regclass
      AND attname = 'created_by';
    
    -- Find the foreign key constraint on created_by column that references users
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'tasks'::regclass
      AND contype = 'f'
      AND created_by_attnum = ANY(conkey);
    
    -- Drop the constraint if it exists
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE tasks DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

-- Make created_by nullable (it was NOT NULL, but we want to allow NULL for system tasks)
ALTER TABLE tasks ALTER COLUMN created_by DROP NOT NULL;

-- Add a comment to document that created_by can reference either users.id or admins.id
COMMENT ON COLUMN tasks.created_by IS 'UUID of the user or admin who created the task. Can be NULL for system-generated tasks.';
