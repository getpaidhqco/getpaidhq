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
