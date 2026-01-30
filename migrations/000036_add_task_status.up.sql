-- Add status column to tasks table
-- ongoing: submission window open; ended: time passed for submission (past end_at); completed: optional (e.g. admin closed)
ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'ongoing'
CHECK (status IN ('ongoing', 'ended', 'completed'));

COMMENT ON COLUMN tasks.status IS 'Task lifecycle: ongoing (open for submission), ended (past end_at), completed (e.g. admin closed)';

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
