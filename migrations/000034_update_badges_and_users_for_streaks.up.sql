-- Update badges table to add xp, required_level, and image_url
ALTER TABLE badges
ADD COLUMN IF NOT EXISTS xp INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS required_level INTEGER DEFAULT 1,
ADD COLUMN IF NOT EXISTS image_url TEXT,
ADD COLUMN IF NOT EXISTS is_streak_badge BOOLEAN DEFAULT false;

-- Update users table to add streak tracking fields
ALTER TABLE users
ADD COLUMN IF NOT EXISTS streak_started_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS streak_days INTEGER DEFAULT 0;

-- Create index for streak badges
CREATE INDEX IF NOT EXISTS idx_badges_is_streak_badge ON badges(is_streak_badge);
CREATE INDEX IF NOT EXISTS idx_badges_required_level ON badges(required_level);

-- Create index for user streaks
CREATE INDEX IF NOT EXISTS idx_users_streak_days ON users(streak_days);
