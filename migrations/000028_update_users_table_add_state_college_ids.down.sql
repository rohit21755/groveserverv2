-- Revert changes: restore state and college columns
ALTER TABLE users 
    ADD COLUMN state VARCHAR(100),
    ADD COLUMN college VARCHAR(255);

-- Restore indexes
CREATE INDEX idx_users_state ON users(state);
CREATE INDEX idx_users_college ON users(college);

-- Drop new columns and indexes
DROP INDEX IF EXISTS idx_users_state_id;
DROP INDEX IF EXISTS idx_users_college_id;
ALTER TABLE users 
    DROP COLUMN IF EXISTS state_id,
    DROP COLUMN IF EXISTS college_id;
