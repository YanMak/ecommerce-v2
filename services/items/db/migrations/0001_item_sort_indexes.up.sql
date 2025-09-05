-- создаём индексы для сортировок (каждый отдельной командой)
CREATE INDEX  IF NOT EXISTS idx_items_created_id_desc
  ON items (created_at DESC, id DESC);

-- пример под сортировку по цене (возрастание)
CREATE INDEX  IF NOT EXISTS idx_items_price_id_asc
  ON items (price_cents ASC, id ASC);