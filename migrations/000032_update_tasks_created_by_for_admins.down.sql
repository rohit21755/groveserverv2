-- Revert changes to tasks table
-- Restore foreign key constraint to users table

-- Make created_by NOT NULL again
-- Note: This will fail if there are any NULL values in created_by
ALTER TABLE tasks ALTER COLUMN created_by SET NOT NULL;

-- Restore foreign key constraint (assuming it references users)
-- Note: This will fail if there are any admin IDs in created_by that don't exist in users table
ALTER TABLE tasks 
    ADD CONSTRAINT tasks_created_by_fkey 
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
