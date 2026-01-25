-- Create xp_logs table
CREATE TABLE xp_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source VARCHAR(100) NOT NULL,
    source_id UUID,
    xp INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_xp_logs_user_id ON xp_logs(user_id);
CREATE INDEX idx_xp_logs_source ON xp_logs(source);
CREATE INDEX idx_xp_logs_source_id ON xp_logs(source_id);
CREATE INDEX idx_xp_logs_created_at ON xp_logs(created_at DESC);
