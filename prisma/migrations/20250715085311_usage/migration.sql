/*
  Warnings:

  - You are about to drop the column `next_retry` on the `subscriptions` table. All the data in the column will be lost.
  - You are about to drop the column `price_id` on the `subscriptions` table. All the data in the column will be lost.
  - You are about to drop the column `product_id` on the `subscriptions` table. All the data in the column will be lost.
  - You are about to drop the column `retries` on the `subscriptions` table. All the data in the column will be lost.
  - You are about to drop the column `variant_id` on the `subscriptions` table. All the data in the column will be lost.

*/
-- CreateEnum
CREATE TYPE "SubscriptionItemStatus" AS ENUM ('active', 'paused', 'cancelled', 'pending');

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
CREATE TYPE "ConfigStatus" AS ENUM ('active', 'inactive', 'archived');

-- AlterEnum
-- This migration adds more than one value to an enum.
-- With PostgreSQL versions 11 and earlier, this is not possible
-- in a single migration. This can be worked around by creating
-- multiple migrations, each migration adding only one value to
-- the enum.


ALTER TYPE "PriceCategory" ADD VALUE 'usage';
ALTER TYPE "PriceCategory" ADD VALUE 'hybrid';

-- AlterEnum
ALTER TYPE "SubscriptionStatus" ADD VALUE 'non_renewing';

-- AlterTable
ALTER TABLE "customers" ADD COLUMN     "dunning_preferences" JSONB;

-- AlterTable
ALTER TABLE "payments" ADD COLUMN     "dunning_attempt_number" INTEGER,
ADD COLUMN     "dunning_campaign_id" TEXT,
ADD COLUMN     "is_dunning_recovery" BOOLEAN NOT NULL DEFAULT false,
ALTER COLUMN "order_id" DROP NOT NULL;

-- AlterTable
ALTER TABLE "prices" ADD COLUMN     "aggregation_type" TEXT,
ADD COLUMN     "fixed_fee" INTEGER,
ADD COLUMN     "has_usage" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "included_usage" INTEGER,
ADD COLUMN     "meter_id" TEXT,
ADD COLUMN     "overage_unit_price" INTEGER,
ADD COLUMN     "percentage_rate" DOUBLE PRECISION,
ADD COLUMN     "unit_type" TEXT,
ADD COLUMN     "usage_limit" INTEGER,
ADD COLUMN     "usage_type" TEXT;

-- AlterTable
ALTER TABLE "subscriptions" DROP COLUMN "next_retry",
DROP COLUMN "price_id",
DROP COLUMN "product_id",
DROP COLUMN "retries",
DROP COLUMN "variant_id",
ADD COLUMN     "active_dunning_campaign_id" TEXT,
ADD COLUMN     "dunning_active" BOOLEAN NOT NULL DEFAULT false,
ALTER COLUMN "order_id" DROP NOT NULL,
ALTER COLUMN "order_item_id" DROP NOT NULL,
ALTER COLUMN "amount" DROP NOT NULL;

-- CreateTable
CREATE TABLE "meters" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "event_name" TEXT NOT NULL,
    "event_filter" JSONB,
    "aggregation_type" TEXT NOT NULL,
    "value_property" TEXT,
    "unit_type" TEXT NOT NULL,
    "display_name" TEXT NOT NULL,
    "window_size" TEXT,
    "reset_interval" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "meters_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "price_tiers" (
    "org_id" TEXT NOT NULL,
    "price_id" TEXT NOT NULL,
    "tier" INTEGER NOT NULL,
    "from_qty" INTEGER NOT NULL,
    "to_qty" INTEGER,
    "unit_price" INTEGER NOT NULL,
    "description" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "price_tiers_pkey" PRIMARY KEY ("org_id","price_id","tier")
);

-- CreateTable
CREATE TABLE "subscription_items" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "subscription_id" TEXT NOT NULL,
    "price_id" TEXT NOT NULL,
    "product_id" TEXT,
    "variant_id" TEXT,
    "meter_id" TEXT,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" "SubscriptionItemStatus" NOT NULL DEFAULT 'active',
    "quantity" INTEGER NOT NULL DEFAULT 1,
    "amount" INTEGER,
    "currency" TEXT NOT NULL,
    "percentage_rate" DOUBLE PRECISION,
    "fixed_fee" INTEGER,
    "unit_price" INTEGER,
    "overage_unit_price" INTEGER,
    "included_usage" INTEGER,
    "usage_limit" INTEGER,
    "price_snapshot" JSONB,
    "has_usage" BOOLEAN NOT NULL DEFAULT false,
    "usage_type" TEXT,
    "aggregation_type" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "subscription_items_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "dunning_campaigns" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "subscription_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "temporal_workflow_id" TEXT NOT NULL,
    "temporal_run_id" TEXT NOT NULL,
    "parent_workflow_id" TEXT,
    "status" "DunningStatus" NOT NULL DEFAULT 'active',
    "failed_amount" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "initial_failure_reason" TEXT,
    "total_attempts" INTEGER NOT NULL DEFAULT 0,
    "immediate_attempts" INTEGER NOT NULL DEFAULT 0,
    "progressive_attempts" INTEGER NOT NULL DEFAULT 0,
    "started_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "last_attempt_at" TIMESTAMP(3),
    "next_attempt_at" TIMESTAMP(3),
    "completed_at" TIMESTAMP(3),
    "recovery_method" TEXT,
    "recovered_amount" INTEGER,
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
    "amount" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "payment_method_id" TEXT,
    "status" "PaymentStatus" NOT NULL,
    "failure_reason" TEXT,
    "failure_code" TEXT,
    "processor_response" JSONB,
    "processing_time_ms" INTEGER,
    "attempted_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    "status" "CommunicationStatus" NOT NULL DEFAULT 'pending',
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
    "token_data" JSONB NOT NULL,
    "signature" TEXT NOT NULL,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "max_uses" INTEGER NOT NULL DEFAULT 5,
    "used_count" INTEGER NOT NULL DEFAULT 0,
    "status" "TokenStatus" NOT NULL DEFAULT 'active',
    "allowed_actions" JSONB NOT NULL,
    "admin_generated" BOOLEAN NOT NULL DEFAULT false,
    "admin_user_id" TEXT,
    "admin_reason" TEXT,
    "admin_notes" TEXT,
    "created_by" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "last_used_at" TIMESTAMP(3),
    "last_used_ip" TEXT,

    CONSTRAINT "payment_update_tokens_pkey" PRIMARY KEY ("org_id","token_id")
);

-- CreateTable
CREATE TABLE "payment_token_usage" (
    "org_id" TEXT NOT NULL,
    "token_id" TEXT NOT NULL,
    "used_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "ip_address" TEXT,
    "user_agent" TEXT,
    "action_taken" TEXT,
    "success" BOOLEAN,

    CONSTRAINT "payment_token_usage_pkey" PRIMARY KEY ("org_id","token_id","used_at")
);

-- CreateTable
CREATE TABLE "dunning_configurations" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "applies_to" "DunningConfigScope" NOT NULL DEFAULT 'organization',
    "target_rules" JSONB,
    "config" JSONB NOT NULL,
    "status" "ConfigStatus" NOT NULL DEFAULT 'active',
    "is_ab_test" BOOLEAN NOT NULL DEFAULT false,
    "ab_test_percentage" DECIMAL(65,30),
    "created_by" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "dunning_configurations_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "dunning_analytics_daily" (
    "org_id" TEXT NOT NULL,
    "date" DATE NOT NULL,
    "campaigns_started" INTEGER NOT NULL DEFAULT 0,
    "campaigns_completed" INTEGER NOT NULL DEFAULT 0,
    "total_attempts" INTEGER NOT NULL DEFAULT 0,
    "immediate_recoveries" INTEGER NOT NULL DEFAULT 0,
    "progressive_recoveries" INTEGER NOT NULL DEFAULT 0,
    "manual_recoveries" INTEGER NOT NULL DEFAULT 0,
    "total_recoveries" INTEGER NOT NULL DEFAULT 0,
    "amount_at_risk" INTEGER NOT NULL DEFAULT 0,
    "amount_recovered" INTEGER NOT NULL DEFAULT 0,
    "amount_lost" INTEGER NOT NULL DEFAULT 0,
    "emails_sent" INTEGER NOT NULL DEFAULT 0,
    "sms_sent" INTEGER NOT NULL DEFAULT 0,
    "total_communications" INTEGER NOT NULL DEFAULT 0,
    "avg_recovery_time_hours" DECIMAL(65,30),
    "avg_attempts_to_recovery" DECIMAL(65,30),
    "customer_segment" TEXT,
    "subscription_tier" TEXT,
    "failure_reason_category" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "dunning_analytics_daily_pkey" PRIMARY KEY ("org_id","date")
);

-- CreateTable
CREATE TABLE "customer_dunning_history" (
    "org_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "total_dunning_campaigns" INTEGER NOT NULL DEFAULT 0,
    "successful_recoveries" INTEGER NOT NULL DEFAULT 0,
    "failed_campaigns" INTEGER NOT NULL DEFAULT 0,
    "total_amount_at_risk" INTEGER NOT NULL DEFAULT 0,
    "total_amount_recovered" INTEGER NOT NULL DEFAULT 0,
    "total_amount_lost" INTEGER NOT NULL DEFAULT 0,
    "avg_recovery_time_hours" DECIMAL(65,30),
    "preferred_recovery_method" TEXT,
    "most_responsive_channel" "CommunicationChannel",
    "payment_reliability_score" DECIMAL(65,30),
    "dunning_risk_tier" TEXT,
    "first_dunning_at" TIMESTAMP(3),
    "last_dunning_at" TIMESTAMP(3),
    "last_recovery_at" TIMESTAMP(3),
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "customer_dunning_history_pkey" PRIMARY KEY ("org_id","customer_id")
);

-- CreateIndex
CREATE UNIQUE INDEX "meters_org_id_event_name_key" ON "meters"("org_id", "event_name");

-- CreateIndex
CREATE INDEX "price_tiers_org_id_price_id_idx" ON "price_tiers"("org_id", "price_id");

-- CreateIndex
CREATE INDEX "subscription_items_org_id_subscription_id_idx" ON "subscription_items"("org_id", "subscription_id");

-- CreateIndex
CREATE INDEX "subscription_items_org_id_subscription_id_status_idx" ON "subscription_items"("org_id", "subscription_id", "status");

-- AddForeignKey
ALTER TABLE "meters" ADD CONSTRAINT "meters_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "prices" ADD CONSTRAINT "prices_org_id_meter_id_fkey" FOREIGN KEY ("org_id", "meter_id") REFERENCES "meters"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "price_tiers" ADD CONSTRAINT "price_tiers_org_id_price_id_fkey" FOREIGN KEY ("org_id", "price_id") REFERENCES "prices"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "price_tiers" ADD CONSTRAINT "price_tiers_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscription_items" ADD CONSTRAINT "subscription_items_org_id_meter_id_fkey" FOREIGN KEY ("org_id", "meter_id") REFERENCES "meters"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscription_items" ADD CONSTRAINT "subscription_items_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscription_items" ADD CONSTRAINT "subscription_items_org_id_price_id_fkey" FOREIGN KEY ("org_id", "price_id") REFERENCES "prices"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscription_items" ADD CONSTRAINT "subscription_items_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_campaigns" ADD CONSTRAINT "dunning_campaigns_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_campaigns" ADD CONSTRAINT "dunning_campaigns_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_campaigns" ADD CONSTRAINT "dunning_campaigns_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_attempts" ADD CONSTRAINT "dunning_attempts_org_id_dunning_campaign_id_fkey" FOREIGN KEY ("org_id", "dunning_campaign_id") REFERENCES "dunning_campaigns"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_communications" ADD CONSTRAINT "dunning_communications_org_id_dunning_campaign_id_fkey" FOREIGN KEY ("org_id", "dunning_campaign_id") REFERENCES "dunning_campaigns"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_communications" ADD CONSTRAINT "dunning_communications_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment_update_tokens" ADD CONSTRAINT "payment_update_tokens_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment_update_tokens" ADD CONSTRAINT "payment_update_tokens_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment_update_tokens" ADD CONSTRAINT "payment_update_tokens_org_id_dunning_campaign_id_fkey" FOREIGN KEY ("org_id", "dunning_campaign_id") REFERENCES "dunning_campaigns"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment_token_usage" ADD CONSTRAINT "payment_token_usage_org_id_token_id_fkey" FOREIGN KEY ("org_id", "token_id") REFERENCES "payment_update_tokens"("org_id", "token_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "dunning_configurations" ADD CONSTRAINT "dunning_configurations_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "customer_dunning_history" ADD CONSTRAINT "customer_dunning_history_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;
