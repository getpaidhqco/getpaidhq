-- +goose Up
-- Baseline generated from schemas/app/schema.prisma via 'prisma migrate diff'.
-- Reproduces the operational schema exactly as of the Prisma->Goose cutover.

-- CreateSchema
CREATE SCHEMA IF NOT EXISTS "public";

-- CreateEnum
CREATE TYPE "OrgStatus" AS ENUM ('active', 'trial', 'demo', 'inactive', 'deleted');

-- CreateEnum
CREATE TYPE "InvoiceStatus" AS ENUM ('draft', 'open', 'paid', 'unpaid', 'void');

-- CreateEnum
CREATE TYPE "InvoiceLineItemKind" AS ENUM ('base', 'usage');

-- CreateEnum
CREATE TYPE "Role" AS ENUM ('owner', 'admin', 'user');

-- CreateEnum
CREATE TYPE "ProductStatus" AS ENUM ('active', 'archived');

-- CreateEnum
CREATE TYPE "PriceCategory" AS ENUM ('one_time', 'subscription', 'free', 'variable');

-- CreateEnum
CREATE TYPE "PriceScheme" AS ENUM ('fixed', 'tiered', 'volume', 'graduated', 'package');

-- CreateEnum
CREATE TYPE "BillingInterval" AS ENUM ('none', 'minute', 'hour', 'day', 'week', 'month', 'year');

-- CreateEnum
CREATE TYPE "OrderStatus" AS ENUM ('pending', 'failed', 'refunded', 'partial_refund', 'completed', 'expired', 'cancelled', 'fraudulent');

-- CreateEnum
CREATE TYPE "PaymentMethodStatus" AS ENUM ('active', 'expired');

-- CreateEnum
CREATE TYPE "SubscriptionStatus" AS ENUM ('trial', 'active', 'retry', 'past_due', 'paused', 'unpaid', 'cancelled', 'pending', 'expired', 'completed');

-- CreateEnum
CREATE TYPE "PaymentStatus" AS ENUM ('pending', 'failed', 'succeeded', 'refunded', 'partial_refund', 'cancelled', 'expired', 'fraudulent');

-- CreateEnum
CREATE TYPE "DunningStatus" AS ENUM ('active', 'paused', 'recovered', 'failed', 'cancelled', 'expired');

-- CreateEnum
CREATE TYPE "DunningAttemptType" AS ENUM ('immediate', 'progressive', 'manual', 'triggered');

-- CreateEnum
CREATE TYPE "CommunicationChannel" AS ENUM ('email', 'sms', 'push', 'webhook', 'in_app');

-- CreateEnum
CREATE TYPE "CommunicationStatus" AS ENUM ('pending', 'sent', 'delivered', 'failed', 'bounced');

-- CreateEnum
CREATE TYPE "TokenStatus" AS ENUM ('active', 'expired', 'revoked', 'max_uses_reached');

-- CreateEnum
CREATE TYPE "DunningConfigScope" AS ENUM ('organization', 'customer_segment', 'subscription_tier', 'customer', 'ab_test');

-- CreateEnum
CREATE TYPE "DunningConfigStatus" AS ENUM ('active', 'inactive', 'archived');

-- CreateTable
CREATE TABLE "api_keys" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT,
    "key_hash" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "api_keys_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
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

-- CreateTable
CREATE TABLE "invoices" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "subscription_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "order_id" TEXT NOT NULL,
    "status" "InvoiceStatus" NOT NULL,
    "currency" TEXT NOT NULL,
    "subtotal" INTEGER NOT NULL DEFAULT 0,
    "total" INTEGER NOT NULL DEFAULT 0,
    "cycle" INTEGER NOT NULL,
    "period_start" TIMESTAMP(3),
    "period_end" TIMESTAMP(3),
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "invoices_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
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
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "invoice_line_items_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
CREATE TABLE "user_orgs" (
    "user_id" TEXT NOT NULL,
    "org_id" TEXT NOT NULL,
    "role" "Role" NOT NULL DEFAULT 'user',

    CONSTRAINT "user_orgs_pkey" PRIMARY KEY ("user_id","org_id")
);

-- CreateTable
CREATE TABLE "carts" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "data" JSONB,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "carts_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
CREATE TABLE "sessions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "cart_id" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "sessions_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
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
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "orders_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
CREATE TABLE "idempotency_keys" (
    "id" TEXT NOT NULL,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "idempotency_keys_pkey" PRIMARY KEY ("id")
);

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateTable
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

-- CreateIndex
CREATE UNIQUE INDEX "api_keys_key_hash_key" ON "api_keys"("key_hash");

-- CreateIndex
CREATE UNIQUE INDEX "invoices_org_id_subscription_id_cycle_key" ON "invoices"("org_id", "subscription_id", "cycle");

-- CreateIndex
CREATE UNIQUE INDEX "users_email_key" ON "users"("email");

-- CreateIndex
CREATE INDEX "products_org_id_status_created_at_idx" ON "products"("org_id", "status", "created_at");

-- CreateIndex
CREATE INDEX "customers_org_id_email_idx" ON "customers"("org_id", "email");

-- CreateIndex
CREATE UNIQUE INDEX "customers_org_id_email_key" ON "customers"("org_id", "email");

-- CreateIndex
CREATE UNIQUE INDEX "customers_org_id_external_id_key" ON "customers"("org_id", "external_id");

-- CreateIndex
CREATE INDEX "idempotency_keys_id_expires_at_idx" ON "idempotency_keys"("id", "expires_at");

-- CreateIndex
CREATE INDEX "metadata_store_org_id_key_value_idx" ON "metadata_store"("org_id", "key", "value");

-- CreateIndex
CREATE INDEX "metadata_store_org_id_parent_id_idx" ON "metadata_store"("org_id", "parent_id");

-- CreateIndex
CREATE INDEX "metadata_store_org_id_parent_type_key_idx" ON "metadata_store"("org_id", "parent_type", "key");

-- CreateIndex
CREATE INDEX "metadata_store_parent_id_idx" ON "metadata_store"("parent_id");

-- CreateIndex
CREATE INDEX "dunning_campaigns_org_id_subscription_id_idx" ON "dunning_campaigns"("org_id", "subscription_id");

-- CreateIndex
CREATE INDEX "dunning_campaigns_org_id_customer_id_idx" ON "dunning_campaigns"("org_id", "customer_id");

-- CreateIndex
CREATE INDEX "dunning_campaigns_org_id_status_idx" ON "dunning_campaigns"("org_id", "status");

-- CreateIndex
CREATE INDEX "dunning_attempts_org_id_campaign_id_idx" ON "dunning_attempts"("org_id", "dunning_campaign_id");

-- CreateIndex
CREATE INDEX "dunning_comms_org_id_campaign_id_idx" ON "dunning_communications"("org_id", "dunning_campaign_id");

-- CreateIndex
CREATE INDEX "payment_update_tokens_org_id_subscription_id_idx" ON "payment_update_tokens"("org_id", "subscription_id");

-- CreateIndex
CREATE INDEX "payment_update_tokens_org_id_campaign_id_idx" ON "payment_update_tokens"("org_id", "dunning_campaign_id");

-- CreateIndex
CREATE INDEX "dunning_configurations_org_id_status_priority_idx" ON "dunning_configurations"("org_id", "status", "priority");

-- CreateIndex
CREATE UNIQUE INDEX "billable_metrics_org_id_code_key" ON "billable_metrics"("org_id", "code");

-- AddForeignKey
ALTER TABLE "api_keys" ADD CONSTRAINT "api_keys_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoices" ADD CONSTRAINT "invoices_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoice_line_items" ADD CONSTRAINT "invoice_line_items_org_id_invoice_id_fkey" FOREIGN KEY ("org_id", "invoice_id") REFERENCES "invoices"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoice_line_items" ADD CONSTRAINT "invoice_line_items_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "settings" ADD CONSTRAINT "settings_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_orgs" ADD CONSTRAINT "user_orgs_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_orgs" ADD CONSTRAINT "user_orgs_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "variants" ADD CONSTRAINT "variants_org_id_product_id_fkey" FOREIGN KEY ("org_id", "product_id") REFERENCES "products"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "variants" ADD CONSTRAINT "variants_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "prices" ADD CONSTRAINT "prices_org_id_variant_id_fkey" FOREIGN KEY ("org_id", "variant_id") REFERENCES "variants"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "prices" ADD CONSTRAINT "prices_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "sessions" ADD CONSTRAINT "sessions_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_cart_id_fkey" FOREIGN KEY ("org_id", "cart_id") REFERENCES "carts"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_variant_id_fkey" FOREIGN KEY ("org_id", "variant_id") REFERENCES "variants"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "customers" ADD CONSTRAINT "customers_org_id_default_payment_method_id_fkey" FOREIGN KEY ("org_id", "default_payment_method_id") REFERENCES "payment_methods"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "customers" ADD CONSTRAINT "customers_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "customer_cohorts" ADD CONSTRAINT "customer_cohorts_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "customer_cohorts" ADD CONSTRAINT "customer_cohorts_org_id_cohort_id_fkey" FOREIGN KEY ("org_id", "cohort_id") REFERENCES "cohorts"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment_methods" ADD CONSTRAINT "payment_methods_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_payment_method_id_fkey" FOREIGN KEY ("org_id", "payment_method_id") REFERENCES "payment_methods"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "refunds" ADD CONSTRAINT "refunds_org_id_payment_id_fkey" FOREIGN KEY ("org_id", "payment_id") REFERENCES "payments"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "metadata_store" ADD CONSTRAINT "metadata_store_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_attempts" ADD CONSTRAINT "dunning_attempts_org_id_dunning_campaign_id_fkey" FOREIGN KEY ("org_id", "dunning_campaign_id") REFERENCES "dunning_campaigns"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_communications" ADD CONSTRAINT "dunning_communications_org_id_dunning_campaign_id_fkey" FOREIGN KEY ("org_id", "dunning_campaign_id") REFERENCES "dunning_campaigns"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "billable_metrics" ADD CONSTRAINT "billable_metrics_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;


-- +goose Down

-- DropForeignKey
ALTER TABLE "public"."api_keys" DROP CONSTRAINT "api_keys_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."invoices" DROP CONSTRAINT "invoices_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."invoice_line_items" DROP CONSTRAINT "invoice_line_items_org_id_invoice_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."invoice_line_items" DROP CONSTRAINT "invoice_line_items_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."settings" DROP CONSTRAINT "settings_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."user_orgs" DROP CONSTRAINT "user_orgs_user_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."user_orgs" DROP CONSTRAINT "user_orgs_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."variants" DROP CONSTRAINT "variants_org_id_product_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."variants" DROP CONSTRAINT "variants_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."prices" DROP CONSTRAINT "prices_org_id_variant_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."prices" DROP CONSTRAINT "prices_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."sessions" DROP CONSTRAINT "sessions_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."orders" DROP CONSTRAINT "orders_org_id_customer_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."orders" DROP CONSTRAINT "orders_org_id_cart_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."orders" DROP CONSTRAINT "orders_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."order_items" DROP CONSTRAINT "order_items_org_id_order_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."order_items" DROP CONSTRAINT "order_items_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."order_items" DROP CONSTRAINT "order_items_org_id_variant_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."order_items" DROP CONSTRAINT "order_items_org_id_subscription_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."customers" DROP CONSTRAINT "customers_org_id_default_payment_method_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."customers" DROP CONSTRAINT "customers_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."customer_cohorts" DROP CONSTRAINT "customer_cohorts_org_id_customer_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."customer_cohorts" DROP CONSTRAINT "customer_cohorts_org_id_cohort_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."payment_methods" DROP CONSTRAINT "payment_methods_org_id_customer_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."subscriptions" DROP CONSTRAINT "subscriptions_org_id_order_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."subscriptions" DROP CONSTRAINT "subscriptions_org_id_customer_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."subscriptions" DROP CONSTRAINT "subscriptions_org_id_payment_method_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."subscriptions" DROP CONSTRAINT "subscriptions_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."payments" DROP CONSTRAINT "payments_org_id_order_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."payments" DROP CONSTRAINT "payments_org_id_subscription_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."payments" DROP CONSTRAINT "payments_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."refunds" DROP CONSTRAINT "refunds_org_id_payment_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."metadata_store" DROP CONSTRAINT "metadata_store_org_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."dunning_attempts" DROP CONSTRAINT "dunning_attempts_org_id_dunning_campaign_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."dunning_communications" DROP CONSTRAINT "dunning_communications_org_id_dunning_campaign_id_fkey";

-- DropForeignKey
ALTER TABLE "public"."billable_metrics" DROP CONSTRAINT "billable_metrics_org_id_fkey";

-- DropTable
DROP TABLE "public"."api_keys";

-- DropTable
DROP TABLE "public"."orgs";

-- DropTable
DROP TABLE "public"."invoices";

-- DropTable
DROP TABLE "public"."invoice_line_items";

-- DropTable
DROP TABLE "public"."gateways";

-- DropTable
DROP TABLE "public"."settings";

-- DropTable
DROP TABLE "public"."users";

-- DropTable
DROP TABLE "public"."user_orgs";

-- DropTable
DROP TABLE "public"."carts";

-- DropTable
DROP TABLE "public"."products";

-- DropTable
DROP TABLE "public"."variants";

-- DropTable
DROP TABLE "public"."prices";

-- DropTable
DROP TABLE "public"."sessions";

-- DropTable
DROP TABLE "public"."orders";

-- DropTable
DROP TABLE "public"."order_items";

-- DropTable
DROP TABLE "public"."customers";

-- DropTable
DROP TABLE "public"."cohorts";

-- DropTable
DROP TABLE "public"."customer_cohorts";

-- DropTable
DROP TABLE "public"."payment_methods";

-- DropTable
DROP TABLE "public"."subscriptions";

-- DropTable
DROP TABLE "public"."payments";

-- DropTable
DROP TABLE "public"."refunds";

-- DropTable
DROP TABLE "public"."idempotency_keys";

-- DropTable
DROP TABLE "public"."webhook_subscriptions";

-- DropTable
DROP TABLE "public"."metadata_store";

-- DropTable
DROP TABLE "public"."dunning_campaigns";

-- DropTable
DROP TABLE "public"."dunning_attempts";

-- DropTable
DROP TABLE "public"."dunning_communications";

-- DropTable
DROP TABLE "public"."payment_update_tokens";

-- DropTable
DROP TABLE "public"."dunning_configurations";

-- DropTable
DROP TABLE "public"."customer_dunning_history";

-- DropTable
DROP TABLE "public"."billable_metrics";

-- DropEnum
DROP TYPE "public"."OrgStatus";

-- DropEnum
DROP TYPE "public"."InvoiceStatus";

-- DropEnum
DROP TYPE "public"."InvoiceLineItemKind";

-- DropEnum
DROP TYPE "public"."Role";

-- DropEnum
DROP TYPE "public"."ProductStatus";

-- DropEnum
DROP TYPE "public"."PriceCategory";

-- DropEnum
DROP TYPE "public"."PriceScheme";

-- DropEnum
DROP TYPE "public"."BillingInterval";

-- DropEnum
DROP TYPE "public"."OrderStatus";

-- DropEnum
DROP TYPE "public"."PaymentMethodStatus";

-- DropEnum
DROP TYPE "public"."SubscriptionStatus";

-- DropEnum
DROP TYPE "public"."PaymentStatus";

-- DropEnum
DROP TYPE "public"."DunningStatus";

-- DropEnum
DROP TYPE "public"."DunningAttemptType";

-- DropEnum
DROP TYPE "public"."CommunicationChannel";

-- DropEnum
DROP TYPE "public"."CommunicationStatus";

-- DropEnum
DROP TYPE "public"."TokenStatus";

-- DropEnum
DROP TYPE "public"."DunningConfigScope";

-- DropEnum
DROP TYPE "public"."DunningConfigStatus";

