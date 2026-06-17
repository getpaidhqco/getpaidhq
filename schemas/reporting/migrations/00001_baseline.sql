-- +goose Up
-- Baseline generated from schemas/reporting/schema.prisma via 'prisma migrate diff'.
-- Reproduces the reporting schema exactly as of the Prisma->Goose cutover.

-- CreateSchema
CREATE SCHEMA IF NOT EXISTS "public";

-- CreateEnum
CREATE TYPE "PriceCategory" AS ENUM ('one_time', 'subscription', 'free', 'variable');

-- CreateEnum
CREATE TYPE "PriceScheme" AS ENUM ('fixed', 'tiered', 'volume', 'graduated');

-- CreateEnum
CREATE TYPE "BillingInterval" AS ENUM ('none', 'minute', 'hour', 'day', 'week', 'month', 'year');

-- CreateEnum
CREATE TYPE "SubscriptionStatus" AS ENUM ('trial', 'active', 'retry', 'past_due', 'paused', 'unpaid', 'cancelled', 'pending', 'expired', 'completed');

-- CreateEnum
CREATE TYPE "PaymentStatus" AS ENUM ('pending', 'failed', 'succeeded', 'refunded', 'partial_refund', 'cancelled', 'expired', 'fraudulent');

-- CreateTable
CREATE TABLE "daily_metrics" (
    "org_id" TEXT NOT NULL,
    "date" TIMESTAMP(3) NOT NULL,
    "timezone" TEXT NOT NULL,
    "day_start_utc" TIMESTAMP(3) NOT NULL,
    "day_end_utc" TIMESTAMP(3) NOT NULL,
    "currency" TEXT NOT NULL,
    "arr" INTEGER NOT NULL DEFAULT 0,
    "mrr" INTEGER NOT NULL DEFAULT 0,
    "past_due_total" INTEGER NOT NULL DEFAULT 0,
    "past_due_count" INTEGER NOT NULL DEFAULT 0,
    "customer_count" INTEGER NOT NULL DEFAULT 0,
    "churn_count" INTEGER NOT NULL DEFAULT 0,
    "churn_total" INTEGER NOT NULL DEFAULT 0,
    "ave_revenue_per_user" DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    "customer_lifetime_value" INTEGER NOT NULL DEFAULT 0,
    "successful_payments" INTEGER NOT NULL DEFAULT 0,
    "failed_payments" INTEGER NOT NULL DEFAULT 0,
    "refund_count" INTEGER NOT NULL DEFAULT 0,
    "refund_total" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "daily_metrics_pkey" PRIMARY KEY ("org_id","date")
);

-- CreateTable
CREATE TABLE "subscriptions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp_id" TEXT NOT NULL,
    "status" "SubscriptionStatus" NOT NULL,
    "order_id" TEXT NOT NULL,
    "order_item_id" TEXT,
    "order_item_name" TEXT,
    "customer_id" TEXT NOT NULL,
    "payment_method_id" TEXT,
    "payment_method_type" TEXT,
    "start_date" TIMESTAMP(3),
    "end_date" TIMESTAMP(3),
    "billing_interval" "BillingInterval" NOT NULL,
    "billing_interval_qty" INTEGER NOT NULL,
    "cycles" INTEGER NOT NULL,
    "billing_anchor" INTEGER NOT NULL,
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
    "amount" INTEGER NOT NULL,
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
    "amount" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "status" "PaymentStatus" NOT NULL,
    "psp_fee" INTEGER NOT NULL,
    "platform_fee" INTEGER NOT NULL,
    "net_amount" INTEGER NOT NULL,
    "metadata" JSONB,
    "completed_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "payments_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "customers" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "customers_pkey" PRIMARY KEY ("org_id","id")
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
CREATE TABLE "refunds" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp_refund_id" TEXT,
    "payment_id" TEXT NOT NULL,
    "currency" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "refunded_at" TIMESTAMP(3) NOT NULL,
    "reason" TEXT,
    "usd_amount" INTEGER,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "refunds_pkey" PRIMARY KEY ("org_id","id")
);


-- +goose Down

-- DropTable
DROP TABLE "public"."daily_metrics";

-- DropTable
DROP TABLE "public"."subscriptions";

-- DropTable
DROP TABLE "public"."payments";

-- DropTable
DROP TABLE "public"."customers";

-- DropTable
DROP TABLE "public"."customer_cohorts";

-- DropTable
DROP TABLE "public"."refunds";

-- DropEnum
DROP TYPE "public"."PriceCategory";

-- DropEnum
DROP TYPE "public"."PriceScheme";

-- DropEnum
DROP TYPE "public"."BillingInterval";

-- DropEnum
DROP TYPE "public"."SubscriptionStatus";

-- DropEnum
DROP TYPE "public"."PaymentStatus";

