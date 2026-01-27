-- Update submissions table to support admin-reviewed submissions
-- Remove foreign key constraint on reviewed_by since it can reference either users or admins
-- Keep reviewed_by nullable to allow NULL for pending submissions

-- Drop the existing foreign key constraint
DO $$
DECLARE
    constraint_name text;
    reviewed_by_attnum smallint;
BEGIN
    -- Get the attribute number for reviewed_by column
    SELECT attnum INTO reviewed_by_attnum
    FROM pg_attribute
    WHERE attrelid = 'submissions'::regclass
      AND attname = 'reviewed_by';
    
    -- Find the foreign key constraint on reviewed_by column that references users
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'submissions'::regclass
      AND contype = 'f'
      AND reviewed_by_attnum = ANY(conkey);
    
    -- Drop the constraint if it exists
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE submissions DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

-- Add a comment to document that reviewed_by can reference either users.id or admins.id
COMMENT ON COLUMN submissions.reviewed_by IS 'UUID of the user or admin who reviewed the submission. Can be NULL for pending submissions.';
