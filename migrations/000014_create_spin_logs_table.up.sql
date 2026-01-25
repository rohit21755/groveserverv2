-- Create spin_logs table
CREATE TABLE spin_logs (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    spin_date DATE NOT NULL DEFAULT CURRENT_DATE,
    reward TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, spin_date)
);

-- Create indexes
CREATE INDEX idx_spin_logs_user_id ON spin_logs(user_id);
CREATE INDEX idx_spin_logs_spin_date ON spin_logs(spin_date);
