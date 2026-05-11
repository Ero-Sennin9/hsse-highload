CREATE TABLE ads (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'moderation_pending', 'published', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_ads_status ON ads (status);

CREATE TABLE promotions (
    id UUID PRIMARY KEY,
    ad_id UUID NOT NULL REFERENCES ads (id),
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_promotions_ad_id ON promotions (ad_id);
