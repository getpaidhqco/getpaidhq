-- AlterTable
ALTER TABLE "payments" ADD COLUMN     "refunded_amount" INTEGER NOT NULL DEFAULT 0;

-- CreateTable
CREATE TABLE "cohorts" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "type" TEXT NOT NULL DEFAULT 'signup',
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "cohort_primary_key" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "customer_cohorts" (
    "org_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "cohort_id" TEXT NOT NULL,
    "cohort_value" TEXT NOT NULL,
    "joined_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "customer_cohorts_pkey" PRIMARY KEY ("org_id","customer_id","cohort_id")
);

-- CreateTable
CREATE TABLE "refunds" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "payment_id" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "reason" TEXT,
    "usd_amount" INTEGER,
    "date" TIMESTAMP(3) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "refunds_pkey" PRIMARY KEY ("org_id","id")
);

-- AddForeignKey
ALTER TABLE "customer_cohorts" ADD CONSTRAINT "customer_cohorts_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "customer_cohorts" ADD CONSTRAINT "customer_cohorts_org_id_cohort_id_fkey" FOREIGN KEY ("org_id", "cohort_id") REFERENCES "cohorts"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "refunds" ADD CONSTRAINT "refunds_org_id_payment_id_fkey" FOREIGN KEY ("org_id", "payment_id") REFERENCES "payments"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;
