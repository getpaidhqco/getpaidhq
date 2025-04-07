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
    "currency" TEXT NOT NULL,
    "arr" INTEGER NOT NULL,
    "mrr" INTEGER NOT NULL,
    "customer_count" INTEGER NOT NULL,
    "churn_rate" DOUBLE PRECISION NOT NULL,
    "arpu" DOUBLE PRECISION NOT NULL,
    "cltv" DOUBLE PRECISION NOT NULL,
    "successful_payments" INTEGER NOT NULL,
    "failed_payments" INTEGER NOT NULL,
    "refunds" DOUBLE PRECISION NOT NULL,
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
CREATE TABLE "refunds" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "payment_id" TEXT NOT NULL,
    "currency" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "date" TIMESTAMP(3) NOT NULL,
    "reason" TEXT,
    "usd_amount" INTEGER,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "refunds_pkey" PRIMARY KEY ("org_id","id")
);
