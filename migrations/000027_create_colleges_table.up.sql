-- Create colleges table
CREATE TABLE colleges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    state_id UUID NOT NULL REFERENCES states(id) ON DELETE RESTRICT,
    city VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, state_id)
);

-- Create indexes
CREATE INDEX idx_colleges_name ON colleges(name);
CREATE INDEX idx_colleges_state_id ON colleges(state_id);
CREATE INDEX idx_colleges_city ON colleges(city);
