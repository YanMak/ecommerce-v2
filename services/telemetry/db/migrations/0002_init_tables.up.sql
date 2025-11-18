BEGIN;

CREATE TABLE IF NOT EXISTS certificates (
  id                   BIGINT PRIMARY KEY,
  xml_id               TEXT,
  title                TEXT NOT NULL,
  created_by           BIGINT NOT NULL DEFAULT 0,
  updated_by           BIGINT NOT NULL DEFAULT 0,
  moved_by             BIGINT NOT NULL DEFAULT 0,
  created_time         TIMESTAMPTZ NOT NULL,
  updated_time         TIMESTAMPTZ NOT NULL,
  moved_time           TIMESTAMPTZ NOT NULL,
  category_id          BIGINT NOT NULL,
  opened               BOOLEAN NOT NULL,
  previous_stage_id    TEXT,
  begindate            TIMESTAMPTZ,
  closedate            TIMESTAMPTZ,
  company_id           BIGINT,
  contact_id           BIGINT,
  opportunity          NUMERIC(18,2),
  is_manual_opportunity BOOLEAN NOT NULL,
  tax_value            NUMERIC(18,2),
  currency_id          TEXT,
  opportunity_account  NUMERIC(18,2),
  tax_value_account    NUMERIC(18,2),
  account_currency_id  TEXT,
  mycompany_id         BIGINT,
  source_id            TEXT,
  source_description   TEXT,
  webform_id           BIGINT,
  uf_uuid              UUID,
  uf_inn               TEXT,
  uf_company_name      TEXT,
  uf_number            TEXT,
  uf_start_date        TIMESTAMPTZ,
  uf_contract_date     TIMESTAMPTZ,
  uf_end_date          TIMESTAMPTZ,
  uf_status            BIGINT,
  uf_ids_documents     TEXT,
  assigned_by_id       BIGINT,
  last_activity_by     BIGINT,
  last_activity_time   TIMESTAMPTZ,
  utm_source           TEXT,
  utm_medium           TEXT,
  utm_campaign         TEXT,
  utm_content          TEXT,
  utm_term             TEXT,
  observers            INT[] NOT NULL DEFAULT '{}',
  contact_ids          INT[] NOT NULL DEFAULT '{}',
  entity_type_id       BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS certificates_uf_uuid_uq
  ON certificates (uf_uuid);
CREATE INDEX IF NOT EXISTS certificates_uf_number_idx
  ON certificates (uf_number);
CREATE INDEX IF NOT EXISTS certificates_company_id_idx
  ON certificates (company_id);
CREATE INDEX IF NOT EXISTS certificates_assigned_by_id_idx
  ON certificates (assigned_by_id);
CREATE INDEX IF NOT EXISTS certificates_updated_time_idx
  ON certificates (updated_time);
CREATE INDEX IF NOT EXISTS certificates_last_activity_time_idx
  ON certificates (last_activity_time);

CREATE TABLE IF NOT EXISTS certificate_documents (
  id              BIGINT PRIMARY KEY,
  certificate_id  BIGINT NOT NULL REFERENCES certificates(id) ON DELETE CASCADE,
  url             TEXT NOT NULL,
  url_machine     TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS certificate_documents_cert_id_idx
  ON certificate_documents (certificate_id);

COMMIT;