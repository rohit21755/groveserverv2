-- Add deadline_at column to tasks table (if not already exists as end_at)
-- Note: end_at already exists, but we'll add a comment to clarify it's the deadline
COMMENT ON COLUMN tasks.end_at IS 'Task deadline - submissions are not accepted after this time';

-- Note: Badge images will be stored in the existing profile bucket or task proof bucket
-- We'll use the profile bucket for badge images by default
-- This is handled in the application code, no schema change needed
