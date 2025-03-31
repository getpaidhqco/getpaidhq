-- DropIndex
DROP INDEX "payments_org_id_psp_id_key";

-- AlterTable
ALTER TABLE "payments" ALTER COLUMN "psp_id" DROP NOT NULL;

-- CreateTable
CREATE TABLE "payment_service_providers" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "payment_service_providers_pkey" PRIMARY KEY ("org_id","id")
);
