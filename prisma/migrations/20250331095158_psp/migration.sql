/*
  Warnings:

  - You are about to drop the `payment_service_providers` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropTable
DROP TABLE "payment_service_providers";

-- CreateTable
CREATE TABLE "gateways" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "psp_id" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "gateways_pkey" PRIMARY KEY ("org_id","id")
);
