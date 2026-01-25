-- Create completed_task_feed table
CREATE TABLE completed_task_feed (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    visibility VARCHAR(50) NOT NULL DEFAULT 'public',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_completed_task_feed_submission_id ON completed_task_feed(submission_id);
CREATE INDEX idx_completed_task_feed_user_id ON completed_task_feed(user_id);
CREATE INDEX idx_completed_task_feed_task_id ON completed_task_feed(task_id);
CREATE INDEX idx_completed_task_feed_visibility ON completed_task_feed(visibility);
CREATE INDEX idx_completed_task_feed_created_at ON completed_task_feed(created_at DESC);
