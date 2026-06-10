-- Kavach Database Schema
-- Migration 001: Initial schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    pass_hash VARCHAR(255), -- NULL if using OAuth
    name VARCHAR(100),
    company VARCHAR(200),
    plan VARCHAR(20) DEFAULT 'free', -- free, pro, team
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tokens table (canary tokens)
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL, -- url, document, api_key, dns, email
    description TEXT,
    payload TEXT NOT NULL, -- The token value (URL, fake key, etc.)
    trigger_url VARCHAR(500) NOT NULL, -- Internal trigger endpoint
    trigger_id VARCHAR(64) UNIQUE NOT NULL, -- Short ID used in trigger URLs
    is_active BOOLEAN DEFAULT true,
    trigger_count INTEGER DEFAULT 0,
    last_triggered TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Attackers table (profiled threat actors)
CREATE TABLE IF NOT EXISTS attackers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint VARCHAR(64) UNIQUE NOT NULL, -- Unique device fingerprint hash
    ip_address INET,
    country VARCHAR(2),
    city VARCHAR(100),
    region VARCHAR(100),
    isp VARCHAR(200),
    asn VARCHAR(50),
    user_agent TEXT,
    browser VARCHAR(50),
    browser_version VARCHAR(20),
    os VARCHAR(50),
    os_version VARCHAR(20),
    tls_fingerprint VARCHAR(100), -- JA3/JA4 hash
    is_vpn BOOLEAN DEFAULT false,
    is_tor BOOLEAN DEFAULT false,
    is_proxy BOOLEAN DEFAULT false,
    threat_level VARCHAR(20) DEFAULT 'low', -- low, medium, high, critical
    trigger_count INTEGER DEFAULT 0,
    tokens_triggered INTEGER DEFAULT 0,
    first_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    notes TEXT
);

-- Trigger events table (individual trigger occurrences)
CREATE TABLE IF NOT EXISTS trigger_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_id UUID NOT NULL REFERENCES tokens(id) ON DELETE CASCADE,
    attacker_id UUID REFERENCES attackers(id),
    ip_address INET NOT NULL,
    user_agent TEXT,
    referrer TEXT,
    country VARCHAR(2),
    city VARCHAR(100),
    fingerprint VARCHAR(64),
    headers JSONB, -- Full request headers
    metadata JSONB, -- Additional context
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Alert configurations
CREATE TABLE IF NOT EXISTS alert_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- email, slack, webhook
    config JSONB NOT NULL, -- Channel-specific config (webhook URL, email, etc.)
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_tokens_user_id ON tokens(user_id);
CREATE INDEX idx_tokens_trigger_id ON tokens(trigger_id);
CREATE INDEX idx_tokens_is_active ON tokens(is_active);
CREATE INDEX idx_trigger_events_token_id ON trigger_events(token_id);
CREATE INDEX idx_trigger_events_attacker_id ON trigger_events(attacker_id);
CREATE INDEX idx_trigger_events_created_at ON trigger_events(created_at DESC);
CREATE INDEX idx_attackers_fingerprint ON attackers(fingerprint);
CREATE INDEX idx_attackers_ip_address ON attackers(ip_address);
CREATE INDEX idx_attackers_last_seen ON attackers(last_seen_at DESC);
CREATE INDEX idx_alert_configs_user_id ON alert_configs(user_id);
