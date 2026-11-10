-- One row per ad published through POST /ads, used by the publication quota.
-- Imports do not write here, and a row is kept when its ad is deleted.
CREATE TABLE ad_publications (
    ad_id        BIGINT      PRIMARY KEY,
    user_id      UUID        NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ad_publications_user_id_published_at_idx ON ad_publications (user_id, published_at);
