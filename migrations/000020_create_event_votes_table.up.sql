-- Create event_votes table
CREATE TABLE event_votes (
    event_id UUID NOT NULL REFERENCES engagement_events(id) ON DELETE CASCADE,
    submission_id UUID NOT NULL REFERENCES event_submissions(id) ON DELETE CASCADE,
    voter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (event_id, submission_id, voter_id)
);

-- Create indexes
CREATE INDEX idx_event_votes_event_id ON event_votes(event_id);
CREATE INDEX idx_event_votes_submission_id ON event_votes(submission_id);
CREATE INDEX idx_event_votes_voter_id ON event_votes(voter_id);
