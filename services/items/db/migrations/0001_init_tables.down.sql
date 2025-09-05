-- базовые таблицы для сервиса items

CREATE TABLE IF NOT EXISTS items (
    id           BIGSERIAL PRIMARY KEY,
    sku          TEXT        NOT NULL UNIQUE,
    name         TEXT        NOT NULL,
    description  TEXT,
    price_cents  INTEGER     NOT NULL DEFAULT 0, -- можно поменять на NUMERIC(12,2), если хочешь деньги в decimal
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- простая модель остатков (одна строка на товар; при необходимости потом расширим под склады)
CREATE TABLE IF NOT EXISTS stocks (
    id         BIGSERIAL PRIMARY KEY,
    item_id    BIGINT      NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    qty        INTEGER     NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_id)
);