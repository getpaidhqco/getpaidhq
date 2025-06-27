INSERT INTO orgs (id, name, country, created_at, updated_at)
VALUES ('mollie', 'Mollie', 'ZA', '2025-01-26 13:18:47.991', '2025-01-26 15:18:47.000')
ON CONFLICT DO NOTHING;

INSERT INTO settings (org_id, parent_id, id, value, value_type, created_at, updated_at)
VALUES ('mollie', 'Paystack', 'settings', '{
  "api_key": "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"
}', 'PaystackConfig', NOW(), NOW()),

       ('mollie', 'CheckoutDotCom', 'settings', '{
         "secret_key": "sk_sbox_g2dxr775jvhnwbvwqbl5qon6kux"
       }', 'CheckoutDotComConfig', NOW(), NOW())
ON CONFLICT DO NOTHING;


INSERT INTO api_keys (org_id, id, key, created_at, updated_at)
VALUES ('mollie', 'apikey-101', 'sk_23456789' , NOW(), NOW())ON CONFLICT DO NOTHING;


INSERT INTO gateways (org_id, id, name,psp_id,active, created_at, updated_at)
VALUES ('mollie', 'Paystack', 'Paystack' , 'Paystack',true,NOW(), NOW()),
       ('mollie', 'CheckoutDotCom', 'CheckoutDotCom' , 'CheckoutDotCom',true,NOW(), NOW())
ON CONFLICT DO NOTHING;

