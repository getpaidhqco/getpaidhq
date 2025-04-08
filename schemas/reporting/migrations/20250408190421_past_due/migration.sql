/*
  Warnings:

  - You are about to drop the column `arpu` on the `daily_metrics` table. All the data in the column will be lost.
  - You are about to drop the column `cltv` on the `daily_metrics` table. All the data in the column will be lost.
  - You are about to drop the column `refunds` on the `daily_metrics` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "daily_metrics" DROP COLUMN "arpu",
DROP COLUMN "cltv",
DROP COLUMN "refunds",
ADD COLUMN     "ave_revenue_per_user" DOUBLE PRECISION NOT NULL DEFAULT 0.0,
ADD COLUMN     "customer_lifetime_value" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "past_due_count" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "past_due_total" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "refund_count" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "refund_total" INTEGER NOT NULL DEFAULT 0,
ALTER COLUMN "arr" SET DEFAULT 0,
ALTER COLUMN "mrr" SET DEFAULT 0,
ALTER COLUMN "customer_count" SET DEFAULT 0,
ALTER COLUMN "churn_rate" SET DEFAULT 0.0,
ALTER COLUMN "successful_payments" SET DEFAULT 0,
ALTER COLUMN "failed_payments" SET DEFAULT 0;
