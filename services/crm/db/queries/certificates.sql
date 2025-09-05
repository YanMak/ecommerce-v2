-- name: UpsertCertificate :exec
INSERT INTO certificates (
  id, xml_id, title,
  created_by, updated_by, moved_by,
  created_time, updated_time, moved_time,
  category_id, opened, previous_stage_id,
  begindate, closedate,
  company_id, contact_id,
  opportunity, is_manual_opportunity, tax_value,
  currency_id, opportunity_account, tax_value_account, account_currency_id,
  mycompany_id,
  source_id, source_description, webform_id,
  uf_uuid, uf_inn, uf_company_name, uf_number,
  uf_start_date, uf_contract_date, uf_end_date,
  uf_status, uf_ids_documents,
  assigned_by_id, last_activity_by, last_activity_time,
  utm_source, utm_medium, utm_campaign, utm_content, utm_term,
  observers, contact_ids,
  entity_type_id
) VALUES (
  $1, $2, $3,
  $4, $5, $6,
  $7, $8, $9,
  $10, $11, $12,
  $13, $14,
  $15, $16,
  $17, $18, $19,
  $20, $21, $22, $23,
  $24,
  $25, $26, $27,
  $28, $29, $30, $31,
  $32, $33, $34,
  $35, $36,
  $37, $38, $39,
  $40, $41, $42, $43, $44,
  $45, $46,
  $47
)
ON CONFLICT (id) DO UPDATE SET
  xml_id = EXCLUDED.xml_id,
  title = EXCLUDED.title,
  created_by = EXCLUDED.created_by,
  updated_by = EXCLUDED.updated_by,
  moved_by = EXCLUDED.moved_by,
  created_time = EXCLUDED.created_time,
  updated_time = EXCLUDED.updated_time,
  moved_time = EXCLUDED.moved_time,
  category_id = EXCLUDED.category_id,
  opened = EXCLUDED.opened,
  previous_stage_id = EXCLUDED.previous_stage_id,
  begindate = EXCLUDED.begindate,
  closedate = EXCLUDED.closedate,
  company_id = EXCLUDED.company_id,
  contact_id = EXCLUDED.contact_id,
  opportunity = EXCLUDED.opportunity,
  is_manual_opportunity = EXCLUDED.is_manual_opportunity,
  tax_value = EXCLUDED.tax_value,
  currency_id = EXCLUDED.currency_id,
  opportunity_account = EXCLUDED.opportunity_account,
  tax_value_account = EXCLUDED.tax_value_account,
  account_currency_id = EXCLUDED.account_currency_id,
  mycompany_id = EXCLUDED.mycompany_id,
  source_id = EXCLUDED.source_id,
  source_description = EXCLUDED.source_description,
  webform_id = EXCLUDED.webform_id,
  uf_uuid = EXCLUDED.uf_uuid,
  uf_inn = EXCLUDED.uf_inn,
  uf_company_name = EXCLUDED.uf_company_name,
  uf_number = EXCLUDED.uf_number,
  uf_start_date = EXCLUDED.uf_start_date,
  uf_contract_date = EXCLUDED.uf_contract_date,
  uf_end_date = EXCLUDED.uf_end_date,
  uf_status = EXCLUDED.uf_status,
  uf_ids_documents = EXCLUDED.uf_ids_documents,
  assigned_by_id = EXCLUDED.assigned_by_id,
  last_activity_by = EXCLUDED.last_activity_by,
  last_activity_time = EXCLUDED.last_activity_time,
  utm_source = EXCLUDED.utm_source,
  utm_medium = EXCLUDED.utm_medium,
  utm_campaign = EXCLUDED.utm_campaign,
  utm_content = EXCLUDED.utm_content,
  utm_term = EXCLUDED.utm_term,
  observers = EXCLUDED.observers,
  contact_ids = EXCLUDED.contact_ids,
  entity_type_id = EXCLUDED.entity_type_id;

-- name: UpsertDocument :exec
INSERT INTO certificate_documents (id, certificate_id, url, url_machine)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
  certificate_id = EXCLUDED.certificate_id,
  url = EXCLUDED.url,
  url_machine = EXCLUDED.url_machine;

-- name: DeleteDocumentsByCertificate :exec
DELETE FROM certificate_documents
WHERE certificate_id = $1;

-- name: GetCertificate :one
SELECT *
FROM certificates
WHERE id = $1;

-- name: ListCertificatesUpdatedSince :many
SELECT *
FROM certificates
WHERE updated_time >= $1
ORDER BY updated_time ASC
LIMIT $2 OFFSET $3;

-- name: SearchCertificates :many
-- Поиск по номеру сертификата (частичное совпадение, case-insensitive) и/или ИНН.
SELECT *
FROM certificates
WHERE ($1::TEXT IS NULL OR uf_number ILIKE '%' || $1 || '%')
  AND ($2::TEXT IS NULL OR uf_inn = $2)
ORDER BY updated_time DESC
LIMIT $3 OFFSET $4;

-- name: ListDocumentsForCertificate :many
SELECT *
FROM certificate_documents
WHERE certificate_id = $1
ORDER BY id ASC;