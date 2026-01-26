-- Add password field to users table
ALTER TABLE users 
    ADD COLUMN password_hash VARCHAR(255);

-- Create index for password (if needed for lookups, though we typically search by email)
-- Note: We don't index password_hash as it's not used for lookups
