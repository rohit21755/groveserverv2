-- Create leaderboard_entries table
CREATE TABLE leaderboard_entries (
    leaderboard_id UUID NOT NULL REFERENCES leaderboards(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rank INTEGER NOT NULL,
    xp INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (leaderboard_id, user_id)
);

-- Create indexes
CREATE INDEX idx_leaderboard_entries_leaderboard_id ON leaderboard_entries(leaderboard_id);
CREATE INDEX idx_leaderboard_entries_user_id ON leaderboard_entries(user_id);
CREATE INDEX idx_leaderboard_entries_rank ON leaderboard_entries(leaderboard_id, rank);
