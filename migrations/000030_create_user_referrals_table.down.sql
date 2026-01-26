-- Drop user_referrals table and related columns
DROP TABLE IF EXISTS user_referrals;

ALTER TABLE users 
    DROP COLUMN IF EXISTS referred_by_id;

DROP INDEX IF EXISTS idx_users_referred_by_id;
