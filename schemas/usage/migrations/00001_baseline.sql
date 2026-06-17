-- +goose Up
-- Baseline generated from schemas/usage/schema.prisma via 'prisma migrate diff'.
-- Reproduces the usage-event store schema exactly as of the Prisma->Goose cutover.

-- CreateSchema
CREATE SCHEMA IF NOT EXISTS "public";

-- CreateTable
CREATE TABLE "meter_events" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "customer_id" TEXT,
    "external_customer_id" TEXT,
    "metric_code" TEXT NOT NULL,
    "subscription_id" TEXT,
    "external_id" TEXT,
    "metadata" JSONB,
    "value" DECIMAL(38,9) NOT NULL DEFAULT 0,
    "timestamp" TIMESTAMP(3) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "meter_events_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateIndex
CREATE INDEX "meter_events_org_id_customer_id_metric_code_timestamp_idx" ON "meter_events"("org_id", "customer_id", "metric_code", "timestamp");

-- CreateIndex
CREATE INDEX "meter_events_org_id_external_customer_id_metric_code_timest_idx" ON "meter_events"("org_id", "external_customer_id", "metric_code", "timestamp");

-- CreateIndex
CREATE UNIQUE INDEX "meter_events_org_id_external_id_key" ON "meter_events"("org_id", "external_id");


-- +goose Down

-- DropTable
DROP TABLE "public"."meter_events";

