-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    college VARCHAR(255),
    state VARCHAR(100),
    role VARCHAR(50) NOT NULL DEFAULT 'student',
    xp INTEGER NOT NULL DEFAULT 0,
    level INTEGER NOT NULL DEFAULT 1,
    coins INTEGER NOT NULL DEFAULT 0,
    bio TEXT,
    avatar_url TEXT,
    resume_url TEXT,
    resume_visibility VARCHAR(50) NOT NULL DEFAULT 'private',
    referral_code VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_referral_code ON users(referral_code);
CREATE INDEX idx_users_state ON users(state);
CREATE INDEX idx_users_college ON users(college);
CREATE INDEX idx_users_role ON users(role);
