-- Create user_referrals table to track referral relationships
CREATE TABLE user_referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referred_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referral_code VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(referred_id) -- A user can only be referred once
);

-- Create indexes
CREATE INDEX idx_user_referrals_referrer_id ON user_referrals(referrer_id);
CREATE INDEX idx_user_referrals_referred_id ON user_referrals(referred_id);
CREATE INDEX idx_user_referrals_referral_code ON user_referrals(referral_code);

-- Add referred_by_id to users table for quick lookup
ALTER TABLE users 
    ADD COLUMN referred_by_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_users_referred_by_id ON users(referred_by_id);
