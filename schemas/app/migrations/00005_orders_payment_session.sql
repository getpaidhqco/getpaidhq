-- +goose Up
ALTER TABLE "orders" ADD COLUMN "payment_session" JSONB;

-- +goose Down
ALTER TABLE "orders" DROP COLUMN "payment_session";
