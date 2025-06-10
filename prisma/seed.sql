INSERT INTO settings (org_id, parent_id, id, value, value_type, created_at, updated_at)
VALUES ('org_2yJ2WAf2tzJoxlPK3D8eewP068p', 'Paystack', 'settings', '{
  "api_key": "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"
}', 'PaystackConfig', NOW(), NOW())
ON CONFLICT DO NOTHING;

INSERT INTO gateways (org_id, id, name,psp_id,active, created_at, updated_at)
VALUES ('org_2yJ2WAf2tzJoxlPK3D8eewP068p', 'Paystack', 'Paystack' , 'Paystack',true,NOW(), NOW())
ON CONFLICT DO NOTHING;

