-- Create event_submissions table
CREATE TABLE event_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES engagement_events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_event_submissions_event_id ON event_submissions(event_id);
CREATE INDEX idx_event_submissions_user_id ON event_submissions(user_id);
CREATE INDEX idx_event_submissions_created_at ON event_submissions(created_at);
