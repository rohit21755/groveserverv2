-- Create event_participants table
CREATE TABLE event_participants (
    event_id UUID NOT NULL REFERENCES engagement_events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (event_id, user_id)
);

-- Create indexes
CREATE INDEX idx_event_participants_event_id ON event_participants(event_id);
CREATE INDEX idx_event_participants_user_id ON event_participants(user_id);
CREATE INDEX idx_event_participants_score ON event_participants(event_id, score DESC);
