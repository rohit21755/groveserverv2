-- Create streaks table
CREATE TABLE streaks (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_streak INTEGER NOT NULL DEFAULT 0,
    last_active DATE NOT NULL DEFAULT CURRENT_DATE
);

-- Create index
CREATE INDEX idx_streaks_last_active ON streaks(last_active);
