/*
  Warnings:

  - You are about to drop the column `is_default` on the `payment_methods` table. All the data in the column will be lost.

*/
-- CreateEnum
CREATE TYPE "PaymentMethodStatus" AS ENUM ('active', 'expired');

-- DropIndex
DROP INDEX "payment_methods_org_id_customer_id_token_key";

-- AlterTable
ALTER TABLE "customers" ADD COLUMN     "default_payment_method_id" TEXT;

-- AlterTable
ALTER TABLE "payment_methods" DROP COLUMN "is_default",
ADD COLUMN     "expire_at" TIMESTAMP(3),
ADD COLUMN     "metadata" JSONB,
ADD COLUMN     "status" "PaymentMethodStatus" NOT NULL DEFAULT 'active';

-- AddForeignKey
ALTER TABLE "customers" ADD CONSTRAINT "customers_org_id_default_payment_method_id_fkey" FOREIGN KEY ("org_id", "default_payment_method_id") REFERENCES "payment_methods"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;
