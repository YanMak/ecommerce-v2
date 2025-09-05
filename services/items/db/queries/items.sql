-- name: CreateItem :one
INSERT INTO items (slug, name, description, price_cents, tags)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: PatchItemOptimistic :one
UPDATE items AS i
SET
  name        = COALESCE(sqlc.narg(name_)::text,        i.name),
  description = COALESCE(sqlc.narg(description_)::text, i.description),
  price_cents = COALESCE(sqlc.narg(price_)::bigint,     i.price_cents),
  tags        = COALESCE(sqlc.narg(tags_)::text[],      i.tags),
  updated_at  = now()
WHERE i.id = sqlc.arg(id)
  AND i.updated_at = sqlc.arg(prev_updated_at)::timestamptz
RETURNING *;

-- name: UpsertItemBySlug :one
INSERT INTO items (slug, name, description, price_cents, tags)
VALUES (
  sqlc.arg(slug),
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(price_cents),
  sqlc.arg(tags)::text[]
)
ON CONFLICT (slug) DO UPDATE
SET
  name        = EXCLUDED.name,
  description = EXCLUDED.description,
  price_cents = EXCLUDED.price_cents,
  tags        = EXCLUDED.tags,
  updated_at  = now()
RETURNING *;

-- name: GetItemByID :one
SELECT * FROM items WHERE id = $1 LIMIT 1;

-- name: GetItemByIDForUpdate :one
SELECT * FROM items WHERE id = $1 LIMIT 1 FOR UPDATE;

-- name: SearchItemsKeysetNext :many
SELECT id, slug, name, description, price_cents, tags, created_at, updated_at
FROM items
WHERE
  (sqlc.narg(name_)::text          IS NULL OR name ILIKE '%' || sqlc.narg(name_)::text || '%')
  AND (sqlc.narg(min_price)::bigint IS NULL OR price_cents >= sqlc.narg(min_price)::bigint)
  AND (sqlc.narg(max_price)::bigint IS NULL OR price_cents <= sqlc.narg(max_price)::bigint)
  AND (
    sqlc.narg(after_created_at)::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg(after_created_at)::timestamptz, sqlc.narg(after_id)::bigint)
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_);

-- name: SearchItemsKeysetPrev :many
SELECT id, slug, name, description, price_cents, tags, created_at, updated_at
FROM items
WHERE
  (sqlc.narg(name_)::text          IS NULL OR name ILIKE '%' || sqlc.narg(name_)::text || '%')
  AND (sqlc.narg(min_price)::bigint IS NULL OR price_cents >= sqlc.narg(min_price)::bigint)
  AND (sqlc.narg(max_price)::bigint IS NULL OR price_cents <= sqlc.narg(max_price)::bigint)
  AND (
    sqlc.narg(before_created_at)::timestamptz IS NULL
    OR (created_at, id) > (sqlc.narg(before_created_at)::timestamptz, sqlc.narg(before_id)::bigint)
  )
ORDER BY created_at ASC, id ASC
LIMIT sqlc.arg(limit_);

-- name: SearchItems :many
SELECT id, slug, name, description, price_cents, tags, created_at, updated_at
FROM items
WHERE 
  (sqlc.narg(name_)::text     IS NULL OR name ILIKE '%' || sqlc.narg(name_)::text || '%') 
  AND (sqlc.narg(min_price)::bigint IS NULL OR price_cents >= sqlc.narg(min_price)::bigint)
  AND (sqlc.narg(max_price)::bigint IS NULL OR price_cents <= sqlc.narg(max_price)::bigint)
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_) OFFSET sqlc.arg(offset_);

-- name: CountItems :one
SELECT COUNT(*) AS total
FROM items
WHERE
  (sqlc.narg(name_)::text       IS NULL OR name ILIKE '%' || sqlc.narg(name_)::text || '%')
  AND (sqlc.narg(min_price)::bigint IS NULL OR price_cents >= sqlc.narg(min_price)::bigint)
  AND (sqlc.narg(max_price)::bigint IS NULL OR price_cents <= sqlc.narg(max_price)::bigint); 


