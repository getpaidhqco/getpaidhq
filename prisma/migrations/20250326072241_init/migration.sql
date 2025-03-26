-- CreateEnum
CREATE TYPE "OrgStatus" AS ENUM ('active', 'trial', 'demo', 'inactive', 'deleted');

-- CreateEnum
CREATE TYPE "Role" AS ENUM ('owner', 'admin', 'user');

-- CreateEnum
CREATE TYPE "PriceCategory" AS ENUM ('one_time', 'subscription', 'free', 'variable');

-- CreateEnum
CREATE TYPE "PriceScheme" AS ENUM ('fixed', 'tiered', 'volume', 'graduated');

-- CreateEnum
CREATE TYPE "BillingInterval" AS ENUM ('none', 'minute', 'hour', 'day', 'week', 'month', 'year');

-- CreateEnum
CREATE TYPE "OrderStatus" AS ENUM ('pending', 'failed', 'refunded', 'partial_refund', 'completed', 'expired', 'cancelled', 'fraudulent');

-- CreateEnum
CREATE TYPE "SubscriptionStatus" AS ENUM ('trial', 'active', 'retry', 'past_due', 'paused', 'unpaid', 'cancelled', 'pending', 'expired', 'completed');

-- CreateEnum
CREATE TYPE "PaymentStatus" AS ENUM ('pending', 'failed', 'succeeded', 'refunded', 'partial_refund', 'cancelled', 'expired', 'fraudulent');

-- CreateTable
CREATE TABLE "api_keys" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "key" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "api_keys_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "orgs" (
    "id" TEXT NOT NULL,
    "country" TEXT NOT NULL,
    "status" "OrgStatus" NOT NULL DEFAULT 'active',
    "name" TEXT NOT NULL,
    "description" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "orgs_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "settings" (
    "org_id" TEXT NOT NULL,
    "parent_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "value_type" TEXT NOT NULL,
    "value" JSONB NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "settings_pkey" PRIMARY KEY ("org_id","parent_id","id")
);

-- CreateTable
CREATE TABLE "users" (
    "id" TEXT NOT NULL,
    "name" TEXT,
    "email" TEXT NOT NULL,
    "email_verified" TIMESTAMP(3),
    "image" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_orgs" (
    "user_id" TEXT NOT NULL,
    "org_id" TEXT NOT NULL,
    "role" "Role" NOT NULL DEFAULT 'user',

    CONSTRAINT "user_orgs_pkey" PRIMARY KEY ("user_id","org_id")
);

-- CreateTable
CREATE TABLE "carts" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "data" JSONB,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "carts_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "products" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "products_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "variants" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "product_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "variants_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "prices" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "variant_id" TEXT NOT NULL,
    "category" "PriceCategory" NOT NULL,
    "scheme" "PriceScheme" NOT NULL,
    "label" TEXT,
    "currency" TEXT NOT NULL,
    "unit_price" INTEGER NOT NULL,
    "cycles" INTEGER,
    "billing_interval" "BillingInterval",
    "billing_interval_qty" INTEGER,
    "trial_interval" "BillingInterval",
    "trial_interval_qty" INTEGER,
    "min_price" INTEGER,
    "suggested_price" INTEGER,
    "tax_code" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "prices_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "sessions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "cart_id" TEXT,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "sessions_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "orders" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "status" "OrderStatus" NOT NULL,
    "reference" TEXT NOT NULL,
    "session_id" TEXT NOT NULL,
    "currency" TEXT NOT NULL,
    "total" INTEGER NOT NULL,
    "metadata" JSONB NOT NULL,
    "cart_id" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "orders_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "order_items" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "order_id" TEXT NOT NULL,
    "product_id" TEXT NOT NULL,
    "price_id" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "quantity" INTEGER NOT NULL,
    "sub_total" INTEGER NOT NULL DEFAULT 0,
    "total" INTEGER NOT NULL DEFAULT 0,
    "tax_total" INTEGER NOT NULL DEFAULT 0,
    "discount_total" INTEGER NOT NULL DEFAULT 0,
    "metadata" JSONB NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,
    "variant_id" TEXT,

    CONSTRAINT "order_items_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "customers" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "first_name" TEXT,
    "last_name" TEXT,
    "phone" TEXT,
    "billing_address" JSONB,
    "metadata" JSONB,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "customer_primary_key" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "payment_methods" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp" TEXT NOT NULL,
    "name" TEXT,
    "customer_id" TEXT NOT NULL,
    "is_default" BOOLEAN NOT NULL,
    "billing_address" JSONB,
    "details" JSONB,
    "type" TEXT NOT NULL,
    "token" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "paymentmethod_pk" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "subscriptions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp_id" TEXT NOT NULL,
    "status" "SubscriptionStatus" NOT NULL,
    "order_id" TEXT NOT NULL,
    "order_item_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "payment_method_id" TEXT,
    "start_date" TIMESTAMP(3),
    "end_date" TIMESTAMP(3),
    "billing_interval" "BillingInterval" NOT NULL,
    "billing_interval_qty" INTEGER NOT NULL,
    "cycles" INTEGER NOT NULL,
    "billing_anchor" INTEGER NOT NULL,
    "trial_ends_at" TIMESTAMP(3),
    "cancel_at" TIMESTAMP(3),
    "ends_at" TIMESTAMP(3),
    "last_charge" TIMESTAMP(3),
    "renews_at" TIMESTAMP(3),
    "current_period_start" TIMESTAMP(3),
    "current_period_end" TIMESTAMP(3),
    "retries" INTEGER,
    "next_retry" TIMESTAMP(3),
    "currency" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "metadata" JSONB NOT NULL,
    "cycles_processed" INTEGER NOT NULL,
    "total_revenue" INTEGER NOT NULL,
    "cancelled_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "subscriptions_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "payments" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "psp" TEXT NOT NULL,
    "psp_id" TEXT NOT NULL,
    "reference" TEXT NOT NULL DEFAULT '',
    "recurring" BOOLEAN NOT NULL DEFAULT false,
    "order_id" TEXT NOT NULL,
    "subscription_id" TEXT,
    "amount" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "status" "PaymentStatus" NOT NULL,
    "psp_fee" INTEGER NOT NULL,
    "platform_fee" INTEGER NOT NULL,
    "net_amount" INTEGER NOT NULL,
    "metadata" JSONB,
    "completed_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "payments_pkey" PRIMARY KEY ("org_id","id")
);

-- CreateTable
CREATE TABLE "idempotency_keys" (
    "id" TEXT NOT NULL,
    "expires_at" TIMESTAMP(3) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "idempotency_keys_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "webhook_subscriptions" (
    "org_id" TEXT NOT NULL,
    "id" TEXT NOT NULL,
    "events" TEXT[],
    "url" TEXT NOT NULL,
    "secret" TEXT,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "webhook_subscriptions_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "users_email_key" ON "users"("email");

-- CreateIndex
CREATE INDEX "customers_org_id_email_idx" ON "customers"("org_id", "email");

-- CreateIndex
CREATE UNIQUE INDEX "customers_org_id_email_key" ON "customers"("org_id", "email");

-- CreateIndex
CREATE UNIQUE INDEX "payment_methods_org_id_customer_id_token_key" ON "payment_methods"("org_id", "customer_id", "token");

-- CreateIndex
CREATE UNIQUE INDEX "payments_org_id_psp_id_key" ON "payments"("org_id", "psp_id");

-- CreateIndex
CREATE INDEX "idempotency_keys_id_expires_at_idx" ON "idempotency_keys"("id", "expires_at");

-- AddForeignKey
ALTER TABLE "api_keys" ADD CONSTRAINT "api_keys_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "settings" ADD CONSTRAINT "settings_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_orgs" ADD CONSTRAINT "user_orgs_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_orgs" ADD CONSTRAINT "user_orgs_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "variants" ADD CONSTRAINT "variants_org_id_product_id_fkey" FOREIGN KEY ("org_id", "product_id") REFERENCES "products"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "variants" ADD CONSTRAINT "variants_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "prices" ADD CONSTRAINT "prices_org_id_variant_id_fkey" FOREIGN KEY ("org_id", "variant_id") REFERENCES "variants"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "prices" ADD CONSTRAINT "prices_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "sessions" ADD CONSTRAINT "sessions_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_cart_id_fkey" FOREIGN KEY ("org_id", "cart_id") REFERENCES "carts"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "orders" ADD CONSTRAINT "orders_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "order_items" ADD CONSTRAINT "order_items_org_id_variant_id_fkey" FOREIGN KEY ("org_id", "variant_id") REFERENCES "variants"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "customers" ADD CONSTRAINT "customers_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment_methods" ADD CONSTRAINT "payment_methods_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_order_item_id_fkey" FOREIGN KEY ("org_id", "order_item_id") REFERENCES "order_items"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_customer_id_fkey" FOREIGN KEY ("org_id", "customer_id") REFERENCES "customers"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_payment_method_id_fkey" FOREIGN KEY ("org_id", "payment_method_id") REFERENCES "payment_methods"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "subscriptions" ADD CONSTRAINT "subscriptions_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_order_id_fkey" FOREIGN KEY ("org_id", "order_id") REFERENCES "orders"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_subscription_id_fkey" FOREIGN KEY ("org_id", "subscription_id") REFERENCES "subscriptions"("org_id", "id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payments" ADD CONSTRAINT "payments_org_id_fkey" FOREIGN KEY ("org_id") REFERENCES "orgs"("id") ON DELETE CASCADE ON UPDATE CASCADE;

