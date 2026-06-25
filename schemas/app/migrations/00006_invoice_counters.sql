-- +goose Up
CREATE TABLE "invoice_counters" (
    "org_id" TEXT NOT NULL,
    "value" BIGINT NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "invoice_counters_pkey" PRIMARY KEY ("org_id"),
    CONSTRAINT "invoice_counters_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT "invoice_counters_value_nonnegative" CHECK ("value" >= 0)
);

ALTER TABLE "invoices" ADD COLUMN "number" BIGINT NOT NULL DEFAULT 0;
CREATE UNIQUE INDEX "invoices_org_id_number_key" ON "invoices"("org_id", "number") WHERE "number" > 0;

-- +goose Down
DROP INDEX "invoices_org_id_number_key";
ALTER TABLE "invoices" DROP COLUMN "number";
DROP TABLE "invoice_counters";
