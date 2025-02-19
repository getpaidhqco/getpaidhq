INSERT INTO orgs (id, name, country, created_at, updated_at)
VALUES ('mollie', 'Mollie', 'ZA', '2025-01-26 13:18:47.991', '2025-01-26 15:18:47.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'Tjops', 'ZA', '2025-01-26 13:18:47.991', '2025-01-26 15:18:47.000')
ON CONFLICT DO NOTHING;

INSERT INTO products (org_id, id, name, description, metadata, created_at, updated_at)
VALUES ('mollie', 'prod-1', 'Pro plan', null, null, '2025-01-26 13:18:47.991', '2025-01-26 15:18:47.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'prod-1', 'Pro plan', null, null, '2025-01-26 13:18:47.991',
        '2025-01-26 15:18:47.000')
ON CONFLICT DO NOTHING;

INSERT INTO public.variants (org_id, id, product_id, name, description, metadata, created_at, updated_at)
VALUES ('mollie', 'var-1', 'prod-1', 'Subscription', 'Sub', null, '2025-01-26 13:18:53.260', '2025-01-26 15:17:52.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'var-1', 'prod-1', 'Subscription', 'Sub', null, '2025-01-26 13:18:53.260',
        '2025-01-26 15:17:52.000')
ON CONFLICT DO NOTHING;

INSERT INTO prices (org_id, id, variant_id, category, scheme, cycles, currency, unit_price, billing_interval,
                    billing_interval_qty, trial_interval, trial_interval_qty, min_price, suggested_price,
                    tax_code, metadata, created_at, updated_at)
VALUES ('mollie', 'price-1', 'var-1', 'subscription', 'fixed', 0, 'ZAR', 10000.000000000000000000000000000000, 'month',
        1,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('mollie', 'price-2', 'var-1', 'subscription', 'fixed', 0, 'ZAR', 10000, 'hour', 1,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('mollie', 'price-3', 'var-1', 'subscription', 'fixed', 0, 'ZAR', 100, 'minute', 2,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('mollie', 'freetrial', 'var-1', 'subscription', 'fixed', 0, 'ZAR', 100, 'hour', 1,
        'hour', 2, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('mollie', 'cyc-1', 'var-1', 'subscription', 'fixed', 3, 'ZAR', 100, 'minute', 1,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'price-1', 'var-1', 'subscription', 'fixed', 0, 'ZAR',
        10000.000000000000000000000000000000, 'month',
        1,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'price-2', 'var-1', 'subscription', 'fixed', 0, 'ZAR', 10000, 'hour', 1,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'price-3', 'var-1', 'subscription', 'fixed', 0, 'ZAR', 100, 'minute', 2,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'freetrial', 'var-1', 'subscription', 'fixed', 0, 'ZAR', 100, 'hour', 1,
        'hour', 2, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000'),
       ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'cyc-1', 'var-1', 'subscription', 'fixed', 3, 'ZAR', 100, 'minute', 1,
        'none', 0, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000')
ON CONFLICT DO NOTHING;

INSERT INTO settings (org_id, parent_id, id, value, value_type, created_at, updated_at)
VALUES ('mollie', 'payment_processors', 'Paystack', '{"api_key": "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"}','PaystackConfig',NOW(),NOW()),
 ('org_2syb0uTnhuKtQTaLO6EAk1iIUnu', 'payment_processors', 'Paystack', '{"api_key": "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"}','PaystackConfig',NOW(),NOW())
ON CONFLICT DO NOTHING;