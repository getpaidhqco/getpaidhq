-- +goose Up
ALTER TABLE "invoices" ADD COLUMN "reference" TEXT NOT NULL DEFAULT '';
CREATE INDEX "invoices_org_id_reference_idx" ON "invoices" ("org_id", "reference") WHERE "reference" <> '';
ALTER TABLE "invoices" ALTER COLUMN "subscription_id" DROP NOT NULL;
-- +goose Down
ALTER TABLE "invoices" ALTER COLUMN "subscription_id" SET NOT NULL;
DROP INDEX "invoices_org_id_reference_idx";
ALTER TABLE "invoices" DROP COLUMN "reference";
