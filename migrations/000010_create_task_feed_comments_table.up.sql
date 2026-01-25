-- Create task_feed_comments table
CREATE TABLE task_feed_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL REFERENCES completed_task_feed(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_task_feed_comments_feed_id ON task_feed_comments(feed_id);
CREATE INDEX idx_task_feed_comments_user_id ON task_feed_comments(user_id);
CREATE INDEX idx_task_feed_comments_created_at ON task_feed_comments(created_at);
