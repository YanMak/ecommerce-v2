-- init tables for items (synced with services/items/db/schema.sql)
CREATE TABLE IF NOT EXISTS items (
    id           BIGSERIAL PRIMARY KEY,
    slug         TEXT        NOT NULL UNIQUE,
    name         TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    price_cents  BIGINT      NOT NULL CHECK (price_cents >= 0),
    tags         TEXT[]      NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);