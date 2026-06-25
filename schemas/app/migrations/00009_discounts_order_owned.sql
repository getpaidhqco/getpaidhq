-- +goose Up
-- Discounts are order-owned: order_id is always set; subscription_id is optional
-- (set when the discount targets a subscription's recurring invoices). The old
-- XOR constraint (exactly one of subscription/order) no longer holds.
ALTER TABLE "discounts" DROP CONSTRAINT IF EXISTS "discounts_target_xor";
ALTER TABLE "discounts" ADD CONSTRAINT "discounts_order_id_nn" CHECK (order_id IS NOT NULL);
-- +goose Down
ALTER TABLE "discounts" DROP CONSTRAINT IF EXISTS "discounts_order_id_nn";
ALTER TABLE "discounts" ADD CONSTRAINT "discounts_target_xor" CHECK (
  (subscription_id IS NOT NULL AND order_id IS NULL) OR
  (subscription_id IS NULL AND order_id IS NOT NULL));
