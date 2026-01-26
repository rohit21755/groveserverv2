-- Add state_id and college_id columns to users table
ALTER TABLE users 
    ADD COLUMN state_id UUID REFERENCES states(id) ON DELETE SET NULL,
    ADD COLUMN college_id UUID REFERENCES colleges(id) ON DELETE SET NULL;

-- Create indexes for foreign keys
CREATE INDEX idx_users_state_id ON users(state_id);
CREATE INDEX idx_users_college_id ON users(college_id);

-- Migrate existing data if any (optional - only if you have existing data)
-- This assumes state names match exactly
-- UPDATE users u SET state_id = (SELECT id FROM states s WHERE s.name = u.state);
-- UPDATE users u SET college_id = (SELECT id FROM colleges c WHERE c.name = u.college);

-- Drop old columns and indexes
DROP INDEX IF EXISTS idx_users_state;
DROP INDEX IF EXISTS idx_users_college;
ALTER TABLE users 
    DROP COLUMN IF EXISTS state,
    DROP COLUMN IF EXISTS college;
