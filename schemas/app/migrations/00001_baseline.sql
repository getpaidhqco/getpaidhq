-- +goose Up

CREATE SCHEMA IF NOT EXISTS "public";

CREATE TYPE "OrgStatus" AS ENUM ('active', 'trial', 'demo', 'inactive', 'deleted');

CREATE TYPE "InvoiceStatus" AS ENUM ('draft', 'open', 'paid', 'uncollectible', 'void');

CREATE TYPE "InvoiceLineItemKind" AS ENUM ('base', 'usage');

CREATE TYPE "Role" AS ENUM ('owner', 'admin', 'user');

CREATE TYPE "ProductStatus" AS ENUM ('active', 'archived');

CREATE TYPE "PriceCategory" AS ENUM ('one_time', 'subscription', 'free', 'variable');

CREATE TYPE "PriceScheme" AS ENUM ('fixed', 'tiered', 'volume', 'graduated', 'package');

CREATE TYPE "BillingInterval" AS ENUM ('none', 'minute', 'hour', 'day', 'week', 'month', 'year');

CREATE TYPE "OrderStatus" AS ENUM ('pending', 'failed', 'refunded', 'partial_refund', 'completed', 'expired', 'cancelled', 'fraudulent');

CREATE TYPE "PaymentMethodStatus" AS ENUM ('active', 'expired');

CREATE TYPE "SubscriptionStatus" AS ENUM ('trial', 'active', 'retry', 'past_due', 'paused', 'unpaid', 'cancelled', 'pending', 'expired', 'completed');

CREATE TYPE "PaymentStatus" AS ENUM ('pending', 'failed', 'succeeded', 'refunded', 'partial_refund', 'cancelled', 'expired', 'fraudulent');

CREATE TYPE "DunningStatus" AS ENUM ('active', 'paused', 'recovered', 'failed', 'cancelled', 'expired');

CREATE TYPE "DunningAttemptType" AS ENUM ('immediate', 'progressive', 'manual', 'triggered');

CREATE TYPE "CommunicationChannel" AS ENUM ('email', 'sms', 'push', 'webhook', 'in_app');

CREATE TYPE "CommunicationStatus" AS ENUM ('pending', 'sent', 'delivered', 'failed', 'bounced');

CREATE TYPE "TokenStatus" AS ENUM ('active', 'expired', 'revoked', 'max_uses_reached');

CREATE TYPE "DunningConfigScope" AS ENUM ('organization', 'customer_segment', 'subscription_tier', 'customer', 'ab_test');

CREATE TYPE "DunningConfigStatus" AS ENUM ('active', 'inactive', 'archived');

CREATE TYPE "DiscountType" AS ENUM ('percentage', 'fixed');

CREATE TYPE "Duration" AS ENUM ('once', 'repeating', 'forever');

CREATE TYPE "DiscountStatus" AS ENUM ('active', 'completed', 'cancelled');

CREATE TABLE "api_keys" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT,
    "key_hash" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "api_keys_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "orgs" (
    "id" TEXT NOT NULL,
    "country" TEXT NOT NULL,
    "status" "OrgStatus" NOT NULL DEFAULT 'active',
    "name" TEXT NOT NULL,
    "timezone" TEXT NOT NULL DEFAULT 'UTC',
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "orgs_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "invoices" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "subscription_id" TEXT,
    "customer_id" TEXT NOT NULL,
    "order_id" TEXT NOT NULL,
    "status" "InvoiceStatus" NOT NULL,
    "currency" TEXT NOT NULL,
    "subtotal" INTEGER NOT NULL DEFAULT 0,
    "total" INTEGER NOT NULL DEFAULT 0,
    "discount_total" INTEGER NOT NULL DEFAULT 0,
    "cycle" INTEGER NOT NULL,
    "number" BIGINT NOT NULL DEFAULT 0,
    "reference" TEXT NOT NULL DEFAULT '',
    "period_start" TIMESTAMP(3),
    "period_end" TIMESTAMP(3),
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "invoices_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "inv_discount_nn" CHECK ("discount_total" >= 0)
);

CREATE TABLE "invoice_line_items" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "price_id" TEXT NOT NULL,
    "kind" "InvoiceLineItemKind" NOT NULL,
    "description" TEXT NOT NULL DEFAULT '',
    "quantity" DECIMAL(38,9) NOT NULL,
    "unit_amount" DECIMAL(38,9) NOT NULL,
    "total" INTEGER NOT NULL DEFAULT 0,
    "discount_total" INTEGER NOT NULL DEFAULT 0,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "invoice_line_items_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "ili_discount_nn" CHECK ("discount_total" >= 0),
    CONSTRAINT "ili_discount_cap" CHECK ("discount_total" <= "total")
);

CREATE TABLE "gateways" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "psp_id" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "config" TEXT NOT NULL DEFAULT '',
    "credentials" TEXT NOT NULL DEFAULT '',
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "gateways_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "settings" (
    "org_id" TEXT NOT NULL,
    "parent_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "value_type" TEXT NOT NULL,
    "value" JSONB NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "settings_pkey" PRIMARY KEY ("org_id","parent_id","id")
);

CREATE TABLE "users" (
    "id" TEXT NOT NULL,
    "name" TEXT,
    "email" TEXT NOT NULL,
    "email_verified" TIMESTAMP(3),
    "image" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "user_orgs" (
    "user_id" TEXT NOT NULL,
    "org_id" TEXT NOT NULL,
    "role" "Role" NOT NULL DEFAULT 'user',

    CONSTRAINT "user_orgs_pkey" PRIMARY KEY ("user_id","org_id")
);

CREATE TABLE "carts" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "data" JSONB,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "carts_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "products" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" "ProductStatus" NOT NULL DEFAULT 'active',
    "archived_at" TIMESTAMP(3),
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "products_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "variants" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "product_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "variants_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "prices" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "variant_id" TEXT NOT NULL,
    "category" "PriceCategory" NOT NULL,
    "scheme" "PriceScheme" NOT NULL,
    "label" TEXT,
    "currency" TEXT NOT NULL,
    "unit_price" INTEGER NOT NULL,
    "unit_count" INTEGER NOT NULL DEFAULT 1,
    "cycles" INTEGER,
    "billing_interval" "BillingInterval",
    "billing_interval_qty" INTEGER,
    "trial_interval" "BillingInterval",
    "trial_interval_qty" INTEGER,
    "min_price" INTEGER,
    "suggested_price" INTEGER,
    "tax_code" TEXT,
    "billable_metric_id" TEXT,
    "tiers" JSONB,
    "filter_field" TEXT,
    "filter_value" TEXT,
    "prorate_on_increase" BOOLEAN NOT NULL DEFAULT false,
    "credit_on_decrease" BOOLEAN NOT NULL DEFAULT false,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "prices_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "sessions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "cart_id" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "sessions_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "orders" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "status" "OrderStatus" NOT NULL,
    "reference" TEXT NOT NULL,
    "session_id" TEXT NOT NULL,
    "currency" TEXT NOT NULL,
    "total" INTEGER NOT NULL,
    "metadata" JSONB NOT NULL,
    "cart_id" TEXT NOT NULL,
    "payment_session" JSONB,
    "config" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "orders_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "order_items" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "order_id" TEXT NOT NULL,
    "product_id" TEXT NOT NULL,
    "price_id" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "quantity" INTEGER NOT NULL,
    "sub_total" INTEGER NOT NULL DEFAULT 0,
    "total" INTEGER NOT NULL DEFAULT 0,
    "tax_total" INTEGER NOT NULL DEFAULT 0,
    "discount_total" INTEGER NOT NULL DEFAULT 0,
    "metadata" JSONB NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,
    "variant_id" TEXT,
    "subscription_id" TEXT,

    CONSTRAINT "order_items_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "customers" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "external_id" TEXT,
    "email" TEXT NOT NULL,
    "first_name" TEXT,
    "last_name" TEXT,
    "phone" TEXT,
    "billing_address" JSONB,
    "metadata" JSONB,
    "default_payment_method_id" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "customer_primary_key" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "cohorts" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "type" TEXT NOT NULL DEFAULT 'signup',
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "cohort_primary_key" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "customer_cohorts" (
    "org_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "cohort_id" TEXT NOT NULL,
    "cohort_value" TEXT NOT NULL,
    "joined_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "customer_cohorts_pkey" PRIMARY KEY ("org_id","customer_id","cohort_id")
);

CREATE TABLE "payment_methods" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "status" "PaymentMethodStatus" NOT NULL DEFAULT 'active',
    "psp" TEXT NOT NULL,
    "name" TEXT,
    "customer_id" TEXT NOT NULL,
    "billing_address" JSONB,
    "details" JSONB,
    "type" TEXT NOT NULL,
    "token" TEXT NOT NULL,
    "metadata" JSONB,
    "expire_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "paymentmethod_pk" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "subscriptions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp_id" TEXT NOT NULL,
    "status" "SubscriptionStatus" NOT NULL,
    "order_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "payment_method_id" TEXT,
    "start_date" TIMESTAMP(3),
    "end_date" TIMESTAMP(3),
    "billing_interval" "BillingInterval" NOT NULL,
    "billing_interval_qty" INTEGER NOT NULL,
    "cycles" INTEGER NOT NULL,
    "billing_anchor" INTEGER NOT NULL,
    "trial_interval" "BillingInterval" NOT NULL DEFAULT 'none',
    "trial_interval_qty" INTEGER NOT NULL DEFAULT 0,
    "trial_ends_at" TIMESTAMP(3),
    "cancel_at" TIMESTAMP(3),
    "ends_at" TIMESTAMP(3),
    "last_charge" TIMESTAMP(3),
    "renews_at" TIMESTAMP(3),
    "current_period_start" TIMESTAMP(3),
    "current_period_end" TIMESTAMP(3),
    "retries" INTEGER,
    "next_retry" TIMESTAMP(3),
    "currency" TEXT NOT NULL,
    "metadata" JSONB NOT NULL,
    "cycles_processed" INTEGER NOT NULL,
    "total_revenue" INTEGER NOT NULL,
    "cancelled_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "subscriptions_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "payments" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp" TEXT NOT NULL,
    "psp_id" TEXT,
    "reference" TEXT NOT NULL DEFAULT '',
    "recurring" BOOLEAN NOT NULL DEFAULT false,
    "order_id" TEXT NOT NULL,
    "subscription_id" TEXT,
    "invoice_id" TEXT,
    "amount" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "status" "PaymentStatus" NOT NULL,
    "psp_fee" INTEGER NOT NULL,
    "platform_fee" INTEGER NOT NULL,
    "net_amount" INTEGER NOT NULL,
    "refunded_amount" INTEGER NOT NULL DEFAULT 0,
    "metadata" JSONB,
    "completed_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "payments_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "refunds" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp_refund_id" TEXT,
    "payment_id" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "reason" TEXT,
    "refunded_at" TIMESTAMP(3) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "refunds_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "idempotency_keys" (
    "id" TEXT NOT NULL,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "idempotency_keys_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "webhook_subscriptions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "events" TEXT[],
    "url" TEXT NOT NULL,
    "secret" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "webhook_subscriptions_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "metadata_store" (
    "org_id" TEXT NOT NULL,
    "parent_id" TEXT NOT NULL,
    "parent_type" TEXT,
    "key" TEXT NOT NULL,
    "value" TEXT NOT NULL,
    "namespace" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "metadata_store_pkey" PRIMARY KEY ("org_id","parent_id","key")
);

CREATE TABLE "dunning_campaigns" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "subscription_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "workflow_id" TEXT NOT NULL DEFAULT '',
    "workflow_run_id" TEXT NOT NULL DEFAULT '',
    "parent_workflow_id" TEXT,
    "status" "DunningStatus" NOT NULL,
    "failed_amount" BIGINT NOT NULL,
    "currency" TEXT NOT NULL,
    "initial_failure_reason" TEXT,
    "total_attempts" INTEGER NOT NULL DEFAULT 0,
    "immediate_attempts" INTEGER NOT NULL DEFAULT 0,
    "progressive_attempts" INTEGER NOT NULL DEFAULT 0,
    "started_at" TIMESTAMP(3) NOT NULL,
    "last_attempt_at" TIMESTAMP(3),
    "next_attempt_at" TIMESTAMP(3),
    "completed_at" TIMESTAMP(3),
    "recovery_method" TEXT,
    "recovered_amount" BIGINT,
    "recovered_at" TIMESTAMP(3),
    "final_failure_reason" TEXT,
    "config_snapshot" JSONB,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "dunning_campaigns_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "dunning_attempts" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "dunning_campaign_id" TEXT NOT NULL,
    "subscription_id" TEXT NOT NULL,
    "attempt_number" INTEGER NOT NULL,
    "attempt_type" "DunningAttemptType" NOT NULL,
    "amount" BIGINT NOT NULL,
    "currency" TEXT NOT NULL,
    "payment_method_id" TEXT,
    "status" TEXT NOT NULL,
    "failure_reason" TEXT,
    "failure_code" TEXT,
    "processor_response" JSONB,
    "processing_time_ms" INTEGER,
    "attempted_at" TIMESTAMP(3) NOT NULL,
    "completed_at" TIMESTAMP(3),
    "triggered_by" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "dunning_attempts_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "dunning_communications" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "dunning_campaign_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "channel" "CommunicationChannel" NOT NULL,
    "template_id" TEXT NOT NULL,
    "attempt_number" INTEGER NOT NULL,
    "subject" TEXT,
    "content_preview" TEXT,
    "personalization_data" JSONB,
    "sent_at" TIMESTAMP(3),
    "delivered_at" TIMESTAMP(3),
    "opened_at" TIMESTAMP(3),
    "clicked_at" TIMESTAMP(3),
    "bounced_at" TIMESTAMP(3),
    "provider" TEXT NOT NULL,
    "provider_message_id" TEXT,
    "provider_response" JSONB,
    "status" "CommunicationStatus" NOT NULL,
    "failure_reason" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "dunning_communications_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "payment_update_tokens" (
    "org_id" TEXT NOT NULL,
    "token_id" TEXT NOT NULL,
    "subscription_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "dunning_campaign_id" TEXT,
    "token_data" JSONB,
    "signature" TEXT NOT NULL,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "max_uses" INTEGER NOT NULL DEFAULT 5,
    "used_count" INTEGER NOT NULL DEFAULT 0,
    "status" "TokenStatus" NOT NULL,
    "allowed_actions" JSONB,
    "admin_generated" BOOLEAN NOT NULL DEFAULT false,
    "admin_user_id" TEXT,
    "admin_reason" TEXT,
    "admin_notes" TEXT,
    "created_by" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "last_used_at" TIMESTAMP(3),
    "last_used_ip" TEXT,

    CONSTRAINT "payment_update_tokens_pkey" PRIMARY KEY ("org_id","token_id")
);

CREATE TABLE "dunning_configurations" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "applies_to" "DunningConfigScope" NOT NULL,
    "target_rules" JSONB,
    "config" JSONB NOT NULL,
    "status" "DunningConfigStatus" NOT NULL,
    "is_ab_test" BOOLEAN NOT NULL DEFAULT false,
    "ab_test_percentage" DOUBLE PRECISION,
    "created_by" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "dunning_configurations_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "customer_dunning_history" (
    "org_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "total_dunning_campaigns" INTEGER NOT NULL DEFAULT 0,
    "successful_recoveries" INTEGER NOT NULL DEFAULT 0,
    "failed_campaigns" INTEGER NOT NULL DEFAULT 0,
    "total_amount_at_risk" BIGINT NOT NULL DEFAULT 0,
    "total_amount_recovered" BIGINT NOT NULL DEFAULT 0,
    "total_amount_lost" BIGINT NOT NULL DEFAULT 0,
    "avg_recovery_time_hours" DOUBLE PRECISION,
    "preferred_recovery_method" TEXT,
    "most_responsive_channel" "CommunicationChannel",
    "payment_reliability_score" DOUBLE PRECISION,
    "dunning_risk_tier" TEXT,
    "first_dunning_at" TIMESTAMP(3),
    "last_dunning_at" TIMESTAMP(3),
    "last_recovery_at" TIMESTAMP(3),
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "customer_dunning_history_pkey" PRIMARY KEY ("org_id","customer_id")
);

CREATE TABLE "billable_metrics" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "aggregation" TEXT NOT NULL,
    "field_name" TEXT,
    "carry_over" BOOLEAN NOT NULL DEFAULT false,
    "rounding_mode" TEXT,
    "rounding_scale" INTEGER NOT NULL DEFAULT 0,
    "filters" JSONB,
    "group_by" JSONB,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "billable_metrics_pkey" PRIMARY KEY ("org_id","id")
);

CREATE TABLE "coupons" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "metadata" JSONB,
    "active" BOOLEAN NOT NULL DEFAULT true,
    "discount_type" "DiscountType" NOT NULL,
    "amount_off" INTEGER,
    "currency" TEXT,
    "percent_off" DECIMAL(5,2),
    "duration" "Duration" NOT NULL,
    "duration_in_cycles" INTEGER,
    "redeem_by" TIMESTAMP(3),
    "applies_to_products" TEXT[],
    "max_redemptions" INTEGER NOT NULL DEFAULT 0,
    "once_per_customer" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "coupons_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "coupons_amount_off_pos" CHECK ("amount_off" > 0),
    CONSTRAINT "coupons_currency_len" CHECK ("currency" IS NULL OR char_length("currency") = 3),
    CONSTRAINT "coupons_percent_off_range" CHECK ("percent_off" > 0 AND "percent_off" <= 100),
    CONSTRAINT "coupons_max_redemptions_nn" CHECK ("max_redemptions" >= 0),
    CONSTRAINT "coupons_discount_type_xor" CHECK (
        ("amount_off" IS NOT NULL AND "currency" IS NOT NULL AND "percent_off" IS NULL) OR
        ("amount_off" IS NULL AND "currency" IS NULL AND "percent_off" IS NOT NULL)),
    CONSTRAINT "coupons_repeating_cycles" CHECK (
        ("duration" = 'repeating' AND "duration_in_cycles" >= 1) OR
        ("duration" <> 'repeating' AND "duration_in_cycles" IS NULL))
);

CREATE TABLE "coupon_codes" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "coupon_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT true,
    "customer_id" TEXT,
    "expires_at" TIMESTAMP(3),
    "max_redemptions" INTEGER NOT NULL DEFAULT 0,
    "times_redeemed" INTEGER NOT NULL DEFAULT 0,
    "restrictions" JSONB,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "coupon_codes_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "codes_max_redemptions_nn" CHECK ("max_redemptions" >= 0),
    CONSTRAINT "codes_times_redeemed_nn" CHECK ("times_redeemed" >= 0)
);

CREATE TABLE "discounts" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "coupon_id" TEXT NOT NULL,
    "coupon_code_id" TEXT,
    "customer_id" TEXT NOT NULL,
    "subscription_id" TEXT,
    "order_id" TEXT,
    "start_cycle" INTEGER NOT NULL DEFAULT 0,
    "status" "DiscountStatus" NOT NULL DEFAULT 'active',
    "redeemed_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "ended_at" TIMESTAMP(3),
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "discounts_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "discounts_order_id_nn" CHECK ("order_id" IS NOT NULL),
    CONSTRAINT "discounts_start_cycle_nn" CHECK ("start_cycle" >= 0)
);

CREATE TABLE "coupon_reservations" (
    "org_id"              TEXT NOT NULL,
    "id"                  TEXT NOT NULL,
    "coupon_id"           TEXT NOT NULL,
    "coupon_code_id"      TEXT,
    "customer_id"         TEXT,
    "checkout_session_id" TEXT,
    "order_id"            TEXT,
    "expires_at"          TIMESTAMP(3) NOT NULL,
    "created_at"          TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "coupon_reservations_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "coupon_reservations_has_holder" CHECK ("checkout_session_id" IS NOT NULL OR "order_id" IS NOT NULL),
    CONSTRAINT "coupon_reservations_coupon_fkey" FOREIGN KEY ("org_id","coupon_id") REFERENCES "coupons"("org_id","id") ON DELETE CASCADE
);

CREATE TABLE "idempotency_requests" (
    "key"              TEXT        NOT NULL,
    "request_hash"     TEXT        NOT NULL,
    "state"            TEXT        NOT NULL,
    "token"            TEXT        NOT NULL,
    "response_code"    INTEGER,
    "response_headers" BYTEA,
    "response_body"    BYTEA,
    "expires_at"       TIMESTAMP(3) NOT NULL,
    "created_at"       TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"       TIMESTAMP(3) NOT NULL,

    CONSTRAINT "idempotency_requests_pkey" PRIMARY KEY ("key")
);

CREATE TABLE "invoice_counters" (
    "org_id" TEXT NOT NULL,
    "value" BIGINT NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "invoice_counters_pkey" PRIMARY KEY ("org_id"),
    CONSTRAINT "invoice_counters_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT "invoice_counters_value_nonnegative" CHECK ("value" >= 0)
);

CREATE TABLE outbox_events (
    id              BIGSERIAL PRIMARY KEY,
    event_id        TEXT        NOT NULL,
    org_id          TEXT        NOT NULL,
    topic           TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    attempts        INT         NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    last_error      TEXT,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX "api_keys_key_hash_key" ON "api_keys"("key_hash");

CREATE UNIQUE INDEX "invoices_org_id_subscription_id_cycle_key" ON "invoices"("org_id", "subscription_id", "cycle");

CREATE UNIQUE INDEX "invoices_org_id_number_key" ON "invoices"("org_id", "number") WHERE "number" > 0;

CREATE INDEX "invoices_org_id_reference_idx" ON "invoices" ("org_id", "reference") WHERE "reference" <> '';

CREATE UNIQUE INDEX "users_email_key" ON "users"("email");

CREATE INDEX "products_org_id_status_created_at_idx" ON "products"("org_id", "status", "created_at");

CREATE INDEX "customers_org_id_email_idx" ON "customers"("org_id", "email");

CREATE UNIQUE INDEX "customers_org_id_email_key" ON "customers"("org_id", "email");

CREATE UNIQUE INDEX "customers_org_id_external_id_key" ON "customers"("org_id", "external_id");

CREATE INDEX "idempotency_keys_id_expires_at_idx" ON "idempotency_keys"("id", "expires_at");

CREATE INDEX "metadata_store_org_id_key_value_idx" ON "metadata_store"("org_id", "key", "value");

CREATE INDEX "metadata_store_org_id_parent_id_idx" ON "metadata_store"("org_id", "parent_id");

CREATE INDEX "metadata_store_org_id_parent_type_key_idx" ON "metadata_store"("org_id", "parent_type", "key");

CREATE INDEX "metadata_store_parent_id_idx" ON "metadata_store"("parent_id");

CREATE INDEX "dunning_campaigns_org_id_subscription_id_idx" ON "dunning_campaigns"("org_id", "subscription_id");

CREATE INDEX "dunning_campaigns_org_id_customer_id_idx" ON "dunning_campaigns"("org_id", "customer_id");

CREATE INDEX "dunning_campaigns_org_id_status_idx" ON "dunning_campaigns"("org_id", "status");

CREATE INDEX "dunning_attempts_org_id_campaign_id_idx" ON "dunning_attempts"("org_id", "dunning_campaign_id");

CREATE INDEX "dunning_comms_org_id_campaign_id_idx" ON "dunning_communications"("org_id", "dunning_campaign_id");

CREATE INDEX "payment_update_tokens_org_id_subscription_id_idx" ON "payment_update_tokens"("org_id", "subscription_id");

CREATE INDEX "payment_update_tokens_org_id_campaign_id_idx" ON "payment_update_tokens"("org_id", "dunning_campaign_id");

CREATE INDEX "dunning_configurations_org_id_status_priority_idx" ON "dunning_configurations"("org_id", "status", "priority");

CREATE UNIQUE INDEX "billable_metrics_org_id_code_key" ON "billable_metrics"("org_id", "code");

CREATE UNIQUE INDEX "coupon_codes_org_id_code_key" ON "coupon_codes"("org_id", "code");

CREATE INDEX "discounts_org_id_coupon_id_idx" ON "discounts"("org_id", "coupon_id");

CREATE INDEX "discounts_org_id_subscription_id_idx" ON "discounts"("org_id", "subscription_id");

CREATE UNIQUE INDEX "discounts_org_id_coupon_id_subscription_id_key" ON "discounts"("org_id", "coupon_id", "subscription_id");

CREATE UNIQUE INDEX "coupon_reservations_org_coupon_order_key"   ON "coupon_reservations"("org_id","coupon_id","order_id")            WHERE "order_id" IS NOT NULL;

CREATE UNIQUE INDEX "coupon_reservations_org_coupon_session_key" ON "coupon_reservations"("org_id","coupon_id","checkout_session_id") WHERE "checkout_session_id" IS NOT NULL;

CREATE INDEX "coupon_reservations_org_coupon_idx" ON "coupon_reservations"("org_id","coupon_id");

CREATE INDEX "coupon_reservations_org_code_idx"   ON "coupon_reservations"("org_id","coupon_code_id");

CREATE INDEX "coupon_reservations_expires_idx"    ON "coupon_reservations"("expires_at");

CREATE INDEX "idempotency_requests_expires_at" ON "idempotency_requests" ("expires_at");

CREATE INDEX outbox_events_pending_idx ON outbox_events (id) WHERE published_at IS NULL;

ALTER TABLE "api_keys" ADD CONSTRAINT "api_keys_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "invoices" ADD CONSTRAINT "invoices_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "invoice_line_items" ADD CONSTRAINT "invoice_line_items_org_id_invoice_id_fkey" FOREIGN KEY ("org_id", "invoice_id") REFERENCES "invoices"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "invoice_line_items" ADD CONSTRAINT "invoice_line_items_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "settings" ADD CONSTRAINT "settings_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "user_orgs" ADD CONSTRAINT "user_orgs_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "user_orgs" ADD CONSTRAINT "user_orgs_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "variants" ADD CONSTRAINT "variants_org_id_product_id_fkey" FOREIGN KEY ("org_id", "product_id") REFERENCES "products"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "variants" ADD CONSTRAINT "variants_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "prices" ADD CONSTRAINT "prices_org_id_variant_id_fkey" FOREIGN KEY ("org_id", "variant_id") REFERENCES "variants"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "prices" ADD CONSTRAINT "prices_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "sessions" ADD CONSTRAINT "sessions_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_cart_id_fkey" FOREIGN KEY ("org_id", "cart_id") REFERENCES "carts"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_variant_id_fkey" FOREIGN KEY ("org_id", "variant_id") REFERENCES "variants"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "customers" ADD CONSTRAINT "customers_org_id_default_payment_method_id_fkey" FOREIGN KEY ("org_id", "default_payment_method_id") REFERENCES "payment_methods"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "customers" ADD CONSTRAINT "customers_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "customer_cohorts" ADD CONSTRAINT "customer_cohorts_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "customer_cohorts" ADD CONSTRAINT "customer_cohorts_org_id_cohort_id_fkey" FOREIGN KEY ("org_id", "cohort_id") REFERENCES "cohorts"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "payment_methods" ADD CONSTRAINT "payment_methods_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_payment_method_id_fkey" FOREIGN KEY ("org_id", "payment_method_id") REFERENCES "payment_methods"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "refunds" ADD CONSTRAINT "refunds_org_id_payment_id_fkey" FOREIGN KEY ("org_id", "payment_id") REFERENCES "payments"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "metadata_store" ADD CONSTRAINT "metadata_store_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "dunning_attempts" ADD CONSTRAINT "dunning_attempts_org_id_dunning_campaign_id_fkey" FOREIGN KEY ("org_id", "dunning_campaign_id") REFERENCES "dunning_campaigns"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "dunning_communications" ADD CONSTRAINT "dunning_communications_org_id_dunning_campaign_id_fkey" FOREIGN KEY ("org_id", "dunning_campaign_id") REFERENCES "dunning_campaigns"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "billable_metrics" ADD CONSTRAINT "billable_metrics_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "coupon_codes" ADD CONSTRAINT "coupon_codes_org_id_coupon_id_fkey" FOREIGN KEY ("org_id", "coupon_id") REFERENCES "coupons"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "discounts" ADD CONSTRAINT "discounts_org_id_coupon_id_fkey" FOREIGN KEY ("org_id", "coupon_id") REFERENCES "coupons"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- +goose StatementBegin
CREATE FUNCTION coupons_block_term_update() RETURNS trigger AS $$
BEGIN
  IF (NEW.discount_type, NEW.amount_off, NEW.currency, NEW.percent_off,
      NEW.duration, NEW.duration_in_cycles, NEW.applies_to_products,
      NEW.redeem_by, NEW.max_redemptions, NEW.once_per_customer)
   IS DISTINCT FROM
     (OLD.discount_type, OLD.amount_off, OLD.currency, OLD.percent_off,
      OLD.duration, OLD.duration_in_cycles, OLD.applies_to_products,
      OLD.redeem_by, OLD.max_redemptions, OLD.once_per_customer)
  THEN RAISE EXCEPTION 'coupon terms are immutable (only name/active/metadata may change)';
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER coupons_immutable BEFORE UPDATE ON coupons
  FOR EACH ROW EXECUTE FUNCTION coupons_block_term_update();

-- +goose Down
