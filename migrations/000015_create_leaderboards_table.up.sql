-- Create leaderboards table
CREATE TABLE leaderboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope VARCHAR(50) NOT NULL,
    scope_value VARCHAR(255),
    period VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_leaderboards_scope ON leaderboards(scope);
CREATE INDEX idx_leaderboards_scope_value ON leaderboards(scope_value);
CREATE INDEX idx_leaderboards_period ON leaderboards(period);
CREATE UNIQUE INDEX idx_leaderboards_unique ON leaderboards(scope, scope_value, period);
