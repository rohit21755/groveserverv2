-- Remove indexes
DROP INDEX IF EXISTS idx_users_streak_days;
DROP INDEX IF EXISTS idx_badges_required_level;
DROP INDEX IF EXISTS idx_badges_is_streak_badge;

-- Remove columns from users table
ALTER TABLE users
DROP COLUMN IF EXISTS streak_days,
DROP COLUMN IF EXISTS streak_started_at;

-- Remove columns from badges table
ALTER TABLE badges
DROP COLUMN IF EXISTS is_streak_badge,
DROP COLUMN IF EXISTS image_url,
DROP COLUMN IF EXISTS required_level,
DROP COLUMN IF EXISTS xp;
