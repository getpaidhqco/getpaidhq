/*
  Warnings:

  - You are about to drop the column `description` on the `orgs` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "orgs" DROP COLUMN "description",
ADD COLUMN     "timezone" TEXT NOT NULL DEFAULT 'UTC';
