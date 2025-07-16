/*
  Warnings:

  - You are about to drop the column `next_retry` on the `subscriptions` table. All the data in the column will be lost.
  - You are about to drop the column `retries` on the `subscriptions` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "subscriptions" DROP COLUMN "next_retry",
DROP COLUMN "retries",
ADD COLUMN     "active_dunning_campaign_id" TEXT,
ADD COLUMN     "dunning_active" BOOLEAN NOT NULL DEFAULT false;
