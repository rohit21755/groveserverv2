-- Create task_feed_reactions table
CREATE TABLE task_feed_reactions (
    feed_id UUID NOT NULL REFERENCES completed_task_feed(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reaction VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (feed_id, user_id)
);

-- Create indexes
CREATE INDEX idx_task_feed_reactions_feed_id ON task_feed_reactions(feed_id);
CREATE INDEX idx_task_feed_reactions_user_id ON task_feed_reactions(user_id);
