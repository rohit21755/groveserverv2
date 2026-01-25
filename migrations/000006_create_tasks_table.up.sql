-- Create tasks table
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    xp INTEGER NOT NULL DEFAULT 0,
    type VARCHAR(50) NOT NULL,
    proof_type VARCHAR(50) NOT NULL,
    priority VARCHAR(50) NOT NULL DEFAULT 'normal',
    start_at TIMESTAMP,
    end_at TIMESTAMP,
    is_flash BOOLEAN NOT NULL DEFAULT false,
    is_weekly BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_start_at ON tasks(start_at);
CREATE INDEX idx_tasks_end_at ON tasks(end_at);
CREATE INDEX idx_tasks_is_flash ON tasks(is_flash);
CREATE INDEX idx_tasks_is_weekly ON tasks(is_weekly);
