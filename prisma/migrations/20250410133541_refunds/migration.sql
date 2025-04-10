/*
  Warnings:

  - You are about to drop the column `date` on the `refunds` table. All the data in the column will be lost.
  - You are about to drop the column `usd_amount` on the `refunds` table. All the data in the column will be lost.
  - Added the required column `refunded_at` to the `refunds` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "refunds" DROP COLUMN "date",
DROP COLUMN "usd_amount",
ADD COLUMN     "psp_refund_id" TEXT,
ADD COLUMN     "refunded_at" TIMESTAMP(3) NOT NULL;
