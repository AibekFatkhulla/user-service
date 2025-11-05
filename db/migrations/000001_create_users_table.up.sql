CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    
    coins_balance BIGINT DEFAULT 0 CHECK (coins_balance >= 0),
    total_coins_purchased BIGINT DEFAULT 0 CHECK (total_coins_purchased >= 0),
    
    is_trial BOOLEAN DEFAULT true,
    trial_ends_at TIMESTAMPTZ NOT NULL,
    
    has_subscription BOOLEAN DEFAULT false,
    subscription_ends_at TIMESTAMPTZ,
    
    status TEXT DEFAULT 'active',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);
CREATE INDEX IF NOT EXISTS idx_users_is_trial ON users (is_trial);
CREATE INDEX IF NOT EXISTS idx_users_has_subscription ON users (has_subscription);
CREATE INDEX IF NOT EXISTS idx_users_subscription_ends_at ON users (subscription_ends_at);