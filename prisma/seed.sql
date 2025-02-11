INSERT INTO orgs (id, name, country, created_at, updated_at)
VALUES ('mollie', 'Mollie', 'ZA', '2025-01-26 13:18:47.991', '2025-01-26 15:18:47.000');

INSERT INTO products (org_id, id, name, description, metadata, created_at, updated_at)
VALUES ('mollie', 'prod-1', 'Pro plan', null, null, '2025-01-26 13:18:47.991', '2025-01-26 15:18:47.000');

INSERT INTO public.variants (org_id, id, product_id, name, description, metadata, created_at, updated_at)
VALUES ('mollie', 'var-1', 'prod-1', 'Subscription', 'Sub', null, '2025-01-26 13:18:53.260', '2025-01-26 15:17:52.000');

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
       ('mollie', 'cyc-1', 'var-1', 'subscription', 'fixed', 2, 'ZAR', 100, 'hour', 1,
        'hour', 2, null, null, null, null, '2025-01-26 13:18:56.354', '2025-01-26 15:17:14.000')
ON CONFLICT DO NOTHING;
