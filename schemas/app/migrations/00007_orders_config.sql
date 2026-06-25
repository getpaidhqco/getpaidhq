-- +goose Up
ALTER TABLE "orders" ADD COLUMN "config" JSONB;
-- +goose Down
ALTER TABLE "orders" DROP COLUMN "config";
