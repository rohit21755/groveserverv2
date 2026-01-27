-- Revert changes to submissions table
-- Restore foreign key constraint to users table

-- Restore foreign key constraint (assuming it references users)
-- Note: This will fail if there are any admin IDs in reviewed_by that don't exist in users table
ALTER TABLE submissions 
    ADD CONSTRAINT submissions_reviewed_by_fkey 
    FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL;
