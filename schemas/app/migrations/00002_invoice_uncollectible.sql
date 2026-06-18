-- +goose Up
-- Postgres can't drop an enum value, so swap the type. Existing 'unpaid'
-- invoices map to 'open' (still collectible).
ALTER TYPE "InvoiceStatus" RENAME TO "InvoiceStatus_old";
CREATE TYPE "InvoiceStatus" AS ENUM ('draft', 'open', 'paid', 'uncollectible', 'void');
ALTER TABLE "invoices" ALTER COLUMN "status" TYPE "InvoiceStatus"
  USING (CASE "status"::text WHEN 'unpaid' THEN 'open' ELSE "status"::text END)::"InvoiceStatus";
DROP TYPE "InvoiceStatus_old";

-- +goose Down
ALTER TYPE "InvoiceStatus" RENAME TO "InvoiceStatus_new";
CREATE TYPE "InvoiceStatus" AS ENUM ('draft', 'open', 'paid', 'unpaid', 'void');
ALTER TABLE "invoices" ALTER COLUMN "status" TYPE "InvoiceStatus"
  USING (CASE "status"::text WHEN 'uncollectible' THEN 'unpaid' ELSE "status"::text END)::"InvoiceStatus";
DROP TYPE "InvoiceStatus_new";
