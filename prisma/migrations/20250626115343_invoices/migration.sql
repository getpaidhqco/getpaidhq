/*
  Warnings:

  - You are about to alter the column `quantity` on the `order_items` table. The data in that column could be lost. The data in that column will be cast from `Integer` to `Decimal(10,3)`.

*/
-- CreateEnum
CREATE TYPE "InvoiceStatus" AS ENUM ('draft', 'sent', 'paid', 'overdue', 'cancelled', 'refunded');

-- CreateEnum
CREATE TYPE "CreditNoteReason" AS ENUM ('correction', 'cancellation', 'refund', 'adjustment', 'tax_adjustment');

-- CreateEnum
CREATE TYPE "InvoiceType" AS ENUM ('initial', 'recurring', 'usage', 'adjustment', 'setup', 'cancellation', 'refund');

-- CreateEnum
CREATE TYPE "InvoiceHistoryAction" AS ENUM ('created', 'updated', 'sent', 'viewed', 'paid', 'partial_paid', 'overdue', 'reminded', 'voided', 'credited', 'refunded', 'disputed', 'adjusted');

-- CreateEnum
CREATE TYPE "DocumentType" AS ENUM ('invoice', 'proforma', 'quote', 'receipt', 'statement');

-- CreateEnum
CREATE TYPE "RefundStatus" AS ENUM ('pending', 'completed', 'error');

-- AlterTable
ALTER TABLE "order_items" ALTER COLUMN "quantity" SET DATA TYPE DECIMAL(10,3);

-- AlterTable
ALTER TABLE "payments" ADD COLUMN     "invoice_id" TEXT;

-- AlterTable
ALTER TABLE "refunds" ADD COLUMN     "completed_at" TIMESTAMP(3),
ADD COLUMN     "status" "RefundStatus" NOT NULL DEFAULT 'pending';

-- CreateTable
CREATE TABLE "metadata_store" (
    "org_id" TEXT NOT NULL,
    "parent_id" TEXT NOT NULL,
    "parent_type" TEXT,
    "key" TEXT NOT NULL,
    "value" TEXT NOT NULL,
    "namespace" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "metadata_store_pkey" PRIMARY KEY ("org_id","parent_id","key")
);

-- CreateTable
CREATE TABLE "invoices" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "customer_id" TEXT,
    "order_id" TEXT,
    "subscription_id" TEXT,
    "sequence_id" TEXT NOT NULL,
    "doc_number" TEXT NOT NULL,
    "type" "DocumentType" NOT NULL DEFAULT 'invoice',
    "invoice_type" "InvoiceType" NOT NULL,
    "status" "InvoiceStatus" NOT NULL DEFAULT 'draft',
    "is_immutable" BOOLEAN NOT NULL DEFAULT false,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "sub_total" INTEGER NOT NULL,
    "tax_total" INTEGER NOT NULL,
    "discount_total" INTEGER NOT NULL DEFAULT 0,
    "total" INTEGER NOT NULL,
    "amount_paid" INTEGER NOT NULL DEFAULT 0,
    "amount_due" INTEGER NOT NULL,
    "tax_provider" TEXT,
    "tax_transaction_id" TEXT,
    "tax_breakdown" JSONB,
    "issued_at" TIMESTAMP(3),
    "due_at" TIMESTAMP(3),
    "schedule_at" TIMESTAMP(3),
    "finalize_at" TIMESTAMP(3),
    "paid_at" TIMESTAMP(3),
    "notes" TEXT,
    "customer_notes" TEXT,
    "metadata" JSONB,
    "exchange_rate" INTEGER,
    "base_currency" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "invoices_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "invoice_line_items" (
    "org_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "product_id" TEXT,
    "variant_id" TEXT,
    "price_id" TEXT,
    "description" TEXT NOT NULL,
    "category" TEXT,
    "quantity" DECIMAL(10,3) NOT NULL,
    "unit_price" INTEGER NOT NULL,
    "line_total" INTEGER NOT NULL,
    "discount_type" TEXT,
    "discount_value" INTEGER,
    "discount_total" INTEGER NOT NULL DEFAULT 0,
    "tax_code" TEXT,
    "tax_rate" INTEGER,
    "tax_amount" INTEGER,
    "tax_exempt" BOOLEAN NOT NULL DEFAULT false,
    "seq" INTEGER,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "invoice_line_items_pkey" PRIMARY KEY ("org_id","invoice_id","id")
);

-- CreateTable
CREATE TABLE "credit_notes" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "sequence_id" TEXT NOT NULL,
    "doc_number" TEXT NOT NULL,
    "invoice_id" TEXT,
    "reason" "CreditNoteReason" NOT NULL,
    "reason_note" TEXT,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "amount" INTEGER NOT NULL,
    "tax_amount" INTEGER,
    "status" TEXT NOT NULL DEFAULT 'issued',
    "applied_at" TIMESTAMP(3),
    "notes" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "credit_notes_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "credit_note_line_items" (
    "org_id" TEXT NOT NULL,
    "credit_note_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "quantity" DECIMAL(10,3) NOT NULL,
    "unit_price" INTEGER NOT NULL,
    "amount" INTEGER NOT NULL,
    "tax_amount" INTEGER,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "credit_note_line_items_pkey" PRIMARY KEY ("org_id","credit_note_id","id")
);

-- CreateTable
CREATE TABLE "invoice_history" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "action" TEXT NOT NULL,
    "field" TEXT,
    "old_value" JSONB,
    "new_value" JSONB,
    "user_id" TEXT,
    "user_email" TEXT,
    "ip_address" TEXT,
    "user_agent" TEXT,
    "reason" TEXT,
    "metadata" JSONB,
    "timestamp" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "invoice_history_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "documents" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "invoice_id" TEXT,
    "credit_note_id" TEXT,
    "filename" TEXT NOT NULL,
    "original_name" TEXT NOT NULL,
    "content_type" TEXT NOT NULL,
    "size" INTEGER NOT NULL,
    "storage_provider" TEXT NOT NULL,
    "storage_key" TEXT NOT NULL,
    "url" TEXT,
    "type" TEXT NOT NULL,
    "purpose" TEXT,
    "is_public" BOOLEAN NOT NULL DEFAULT false,
    "access_token" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "documents_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "doc_sequences" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "value" INTEGER NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "doc_sequences_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateIndex
CREATE INDEX "metadata_store_org_id_key_value_idx" ON "metadata_store"("org_id", "key", "value");

-- CreateIndex
CREATE INDEX "metadata_store_org_id_parent_id_idx" ON "metadata_store"("org_id", "parent_id");

-- CreateIndex
CREATE INDEX "metadata_store_org_id_parent_type_key_idx" ON "metadata_store"("org_id", "parent_type", "key");

-- CreateIndex
CREATE INDEX "metadata_store_parent_id_idx" ON "metadata_store"("parent_id");

-- CreateIndex
CREATE INDEX "invoices_org_id_status_idx" ON "invoices"("org_id", "status");

-- CreateIndex
CREATE INDEX "invoices_org_id_customer_id_idx" ON "invoices"("org_id", "customer_id");

-- CreateIndex
CREATE INDEX "invoices_org_id_order_id_idx" ON "invoices"("org_id", "order_id");

-- CreateIndex
CREATE INDEX "invoices_org_id_subscription_id_idx" ON "invoices"("org_id", "subscription_id");

-- CreateIndex
CREATE INDEX "invoices_org_id_due_at_idx" ON "invoices"("org_id", "due_at");

-- CreateIndex
CREATE INDEX "invoices_org_id_issued_at_idx" ON "invoices"("org_id", "issued_at");

-- CreateIndex
CREATE INDEX "invoices_org_id_invoice_type_idx" ON "invoices"("org_id", "invoice_type");

-- CreateIndex
CREATE UNIQUE INDEX "invoices_org_id_sequence_id_doc_number_key" ON "invoices"("org_id", "sequence_id", "doc_number");

-- CreateIndex
CREATE INDEX "invoice_line_items_org_id_product_id_idx" ON "invoice_line_items"("org_id", "product_id");

-- CreateIndex
CREATE INDEX "invoice_line_items_org_id_category_idx" ON "invoice_line_items"("org_id", "category");

-- CreateIndex
CREATE INDEX "credit_notes_org_id_invoice_id_idx" ON "credit_notes"("org_id", "invoice_id");

-- CreateIndex
CREATE INDEX "credit_notes_org_id_status_idx" ON "credit_notes"("org_id", "status");

-- CreateIndex
CREATE UNIQUE INDEX "credit_notes_org_id_sequence_id_doc_number_key" ON "credit_notes"("org_id", "sequence_id", "doc_number");

-- CreateIndex
CREATE INDEX "invoice_history_org_id_invoice_id_timestamp_idx" ON "invoice_history"("org_id", "invoice_id", "timestamp");

-- CreateIndex
CREATE INDEX "invoice_history_org_id_action_timestamp_idx" ON "invoice_history"("org_id", "action", "timestamp");

-- CreateIndex
CREATE INDEX "documents_org_id_invoice_id_idx" ON "documents"("org_id", "invoice_id");

-- CreateIndex
CREATE INDEX "documents_org_id_credit_note_id_idx" ON "documents"("org_id", "credit_note_id");

-- CreateIndex
CREATE INDEX "documents_org_id_type_idx" ON "documents"("org_id", "type");

-- CreateIndex
CREATE INDEX "documents_org_id_purpose_idx" ON "documents"("org_id", "purpose");

-- CreateIndex
CREATE INDEX "doc_sequences_org_id_type_idx" ON "doc_sequences"("org_id", "type");

-- CreateIndex
CREATE INDEX "payments_org_id_invoice_id_idx" ON "payments"("org_id", "invoice_id");

-- CreateIndex
CREATE INDEX "payments_org_id_subscription_id_idx" ON "payments"("org_id", "subscription_id");

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_invoice_id_fkey" FOREIGN KEY ("org_id", "invoice_id") REFERENCES "invoices"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "metadata_store" ADD CONSTRAINT "metadata_store_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoices" ADD CONSTRAINT "invoices_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoices" ADD CONSTRAINT "invoices_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoices" ADD CONSTRAINT "invoices_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoice_line_items" ADD CONSTRAINT "invoice_line_items_org_id_invoice_id_fkey" FOREIGN KEY ("org_id", "invoice_id") REFERENCES "invoices"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "credit_notes" ADD CONSTRAINT "credit_notes_org_id_invoice_id_fkey" FOREIGN KEY ("org_id", "invoice_id") REFERENCES "invoices"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "credit_note_line_items" ADD CONSTRAINT "credit_note_line_items_org_id_credit_note_id_fkey" FOREIGN KEY ("org_id", "credit_note_id") REFERENCES "credit_notes"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "invoice_history" ADD CONSTRAINT "invoice_history_org_id_invoice_id_fkey" FOREIGN KEY ("org_id", "invoice_id") REFERENCES "invoices"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "documents" ADD CONSTRAINT "documents_org_id_invoice_id_fkey" FOREIGN KEY ("org_id", "invoice_id") REFERENCES "invoices"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "documents" ADD CONSTRAINT "documents_org_id_credit_note_id_fkey" FOREIGN KEY ("org_id", "credit_note_id") REFERENCES "credit_notes"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;
