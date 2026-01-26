-- Remove password field from users table
ALTER TABLE users 
    DROP COLUMN IF EXISTS password_hash;
