-- Create engagement_events table
CREATE TABLE engagement_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    start_at TIMESTAMP NOT NULL,
    end_at TIMESTAMP NOT NULL,
    rules TEXT NOT NULL,
    reward TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_engagement_events_type ON engagement_events(type);
CREATE INDEX idx_engagement_events_start_at ON engagement_events(start_at);
CREATE INDEX idx_engagement_events_end_at ON engagement_events(end_at);
