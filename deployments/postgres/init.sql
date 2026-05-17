CREATE TABLE ads (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    region TEXT NOT NULL,
    price BIGINT NOT NULL CHECK (price >= 0),
    status TEXT NOT NULL CHECK (status IN ('draft', 'moderation_pending', 'published', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_ads_status ON ads (status);
CREATE INDEX idx_ads_category_region_price ON ads (category, region, price);

CREATE TABLE ad_media (
    id UUID PRIMARY KEY,
    ad_id UUID NOT NULL REFERENCES ads (id) ON DELETE CASCADE,
    storage_key TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    position SMALLINT NOT NULL CHECK (position >= 0 AND position < 8),
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (ad_id, position)
);

CREATE INDEX idx_ad_media_ad_id ON ad_media (ad_id);

CREATE TABLE promotions (
    id UUID PRIMARY KEY,
    ad_id UUID NOT NULL REFERENCES ads (id),
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_promotions_ad_id ON promotions (ad_id);
