-- +goose Up
INSERT INTO orgs (id, name, country, created_at, updated_at)
VALUES ('mollie', 'Mollie', 'ZA', now(), now())
    ON CONFLICT DO NOTHING;

INSERT INTO api_keys (org_id, id, key_hash, created_at, updated_at)
VALUES ('mollie', 'apikey-101', 'sk_23456789' , NOW(), NOW())ON CONFLICT DO NOTHING;


-- +goose Down
DELETE FROM orgs WHERE id = 'mollie';
DELETE FROM api_keys WHERE org_id='mollie' and id = 'apikey-101';