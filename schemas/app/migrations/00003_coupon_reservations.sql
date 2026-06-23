-- +goose Up
CREATE TABLE "coupon_reservations" (
    "org_id"              TEXT NOT NULL,
    "id"                  TEXT NOT NULL,
    "coupon_id"           TEXT NOT NULL,
    "coupon_code_id"      TEXT,
    "customer_id"         TEXT,
    "checkout_session_id" TEXT,
    "order_id"            TEXT,
    "expires_at"          TIMESTAMP(3) NOT NULL,
    "created_at"          TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "coupon_reservations_pkey" PRIMARY KEY ("org_id","id"),
    CONSTRAINT "coupon_reservations_has_holder" CHECK ("checkout_session_id" IS NOT NULL OR "order_id" IS NOT NULL),
    CONSTRAINT "coupon_reservations_coupon_fkey" FOREIGN KEY ("org_id","coupon_id") REFERENCES "coupons"("org_id","id") ON DELETE CASCADE
);
CREATE UNIQUE INDEX "coupon_reservations_org_coupon_order_key"   ON "coupon_reservations"("org_id","coupon_id","order_id")            WHERE "order_id" IS NOT NULL;
CREATE UNIQUE INDEX "coupon_reservations_org_coupon_session_key" ON "coupon_reservations"("org_id","coupon_id","checkout_session_id") WHERE "checkout_session_id" IS NOT NULL;
CREATE INDEX "coupon_reservations_org_coupon_idx" ON "coupon_reservations"("org_id","coupon_id");
CREATE INDEX "coupon_reservations_org_code_idx"   ON "coupon_reservations"("org_id","coupon_code_id");
CREATE INDEX "coupon_reservations_expires_idx"    ON "coupon_reservations"("expires_at");

-- +goose Down
DROP TABLE "coupon_reservations";
