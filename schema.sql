-- SDS Referral Dashboard schema

CREATE TABLE IF NOT EXISTS jobs (
    greenhouse_id BIGINT PRIMARY KEY,
    title TEXT NOT NULL,
    department TEXT,
    team TEXT,
    status TEXT DEFAULT 'open',
    is_priority BOOLEAN DEFAULT FALSE,
    referral_count INTEGER DEFAULT 0,
    location TEXT,
    last_synced_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS referrals (
    id SERIAL PRIMARY KEY,
    candidate_name TEXT NOT NULL,
    linkedin_url TEXT,
    role TEXT,
    job_greenhouse_id BIGINT REFERENCES jobs(greenhouse_id),
    referrer_name TEXT,
    referrer_slack_id TEXT,
    stage TEXT DEFAULT 'submitted',
    greenhouse_candidate_id BIGINT,
    greenhouse_application_id BIGINT,
    slack_channel_id TEXT,
    slack_message_ts TEXT,
    source TEXT DEFAULT 'slack',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_referrals_stage ON referrals(stage);
CREATE INDEX IF NOT EXISTS idx_referrals_role ON referrals(role);
CREATE INDEX IF NOT EXISTS idx_referrals_job ON referrals(job_greenhouse_id);
CREATE INDEX IF NOT EXISTS idx_referrals_referrer ON referrals(referrer_slack_id);
CREATE INDEX IF NOT EXISTS idx_referrals_created ON referrals(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_referrals_gh_candidate ON referrals(greenhouse_candidate_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(is_priority);
CREATE INDEX IF NOT EXISTS idx_jobs_department ON jobs(department);

CREATE TABLE IF NOT EXISTS sync_log (
    id SERIAL PRIMARY KEY,
    sync_type TEXT NOT NULL,
    status TEXT NOT NULL,
    records_synced INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP DEFAULT NOW(),
    finished_at TIMESTAMP
);
