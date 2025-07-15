-- CreateTable
CREATE TABLE "usage_events" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "meter_id" TEXT NOT NULL,
    "spec_version" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "event_id" TEXT NOT NULL,
    "time" TIMESTAMPTZ NOT NULL,
    "source" TEXT NOT NULL,
    "subject" TEXT NOT NULL,
    "data" JSONB NOT NULL,
    "received_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stored_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "usage_events_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "usage_processing_status" (
    "org_id" TEXT NOT NULL,
    "subscription_item_id" TEXT NOT NULL,
    "billing_period" TEXT NOT NULL,
    "total_quantity" DECIMAL(15,4) NOT NULL DEFAULT 0,
    "total_amount" BIGINT NOT NULL DEFAULT 0,
    "event_count" INTEGER NOT NULL DEFAULT 0,
    "processed" BOOLEAN NOT NULL DEFAULT false,
    "processed_at" TIMESTAMPTZ,
    "invoice_id" TEXT,
    "first_event_time" TIMESTAMPTZ NOT NULL,
    "last_event_time" TIMESTAMPTZ NOT NULL,
    "last_updated" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "usage_processing_status_pkey" PRIMARY KEY ("org_id","subscription_item_id","billing_period")
);

-- CreateTable
CREATE TABLE "usage_event_log" (
    "id" TEXT NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "org_id" TEXT NOT NULL,
    "event_type" TEXT NOT NULL,
    "subscription_id" TEXT,
    "subscription_item_id" TEXT,
    "customer_id" TEXT,
    "invoice_id" TEXT,
    "amount" BIGINT,
    "quantity" DECIMAL(15,4),
    "event_count" INTEGER,
    "billing_period" TEXT,
    "triggered_by" TEXT,
    "reason" TEXT,
    "metadata" JSONB,

    CONSTRAINT "usage_event_log_pkey" PRIMARY KEY ("id")
);
