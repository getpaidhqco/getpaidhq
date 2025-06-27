-- AlterTable
ALTER TABLE "subscriptions" ADD COLUMN     "price_id" TEXT,
ADD COLUMN     "product_id" TEXT,
ADD COLUMN     "variant_id" TEXT;

-- CreateTable
CREATE TABLE "subscription_plan_changes" (
    "id" TEXT NOT NULL,
    "org_id" TEXT NOT NULL,
    "subscription_id" TEXT NOT NULL,
    "from_product_id" TEXT NOT NULL,
    "from_variant_id" TEXT NOT NULL,
    "from_price_id" TEXT NOT NULL,
    "from_amount" BIGINT NOT NULL,
    "to_product_id" TEXT NOT NULL,
    "to_variant_id" TEXT NOT NULL,
    "to_price_id" TEXT NOT NULL,
    "to_amount" BIGINT NOT NULL,
    "change_type" TEXT NOT NULL,
    "effective_date" TIMESTAMP(3) NOT NULL,
    "proration_mode" TEXT NOT NULL,
    "proration_amount" BIGINT NOT NULL,
    "reason" TEXT,
    "initiated_by" TEXT NOT NULL,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "subscription_plan_changes_pkey" PRIMARY KEY ("org_id","id")
);

-- AddForeignKey
ALTER TABLE "subscription_plan_changes" ADD CONSTRAINT "subscription_plan_changes_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;
