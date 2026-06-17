-- Coupon/discount invariants Prisma can't express. Idempotent: safe to re-run.
DO $$
BEGIN
  -- coupons
  BEGIN ALTER TABLE coupons ADD CONSTRAINT coupons_amount_off_pos CHECK (amount_off > 0); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE coupons ADD CONSTRAINT coupons_currency_len CHECK (currency IS NULL OR char_length(currency) = 3); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE coupons ADD CONSTRAINT coupons_percent_off_range CHECK (percent_off > 0 AND percent_off <= 100); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE coupons ADD CONSTRAINT coupons_max_redemptions_nn CHECK (max_redemptions >= 0); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE coupons ADD CONSTRAINT coupons_discount_type_xor CHECK (
    (amount_off IS NOT NULL AND currency IS NOT NULL AND percent_off IS NULL) OR
    (amount_off IS NULL AND currency IS NULL AND percent_off IS NOT NULL)); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE coupons ADD CONSTRAINT coupons_repeating_cycles CHECK (
    (duration = 'repeating' AND duration_in_cycles >= 1) OR
    (duration <> 'repeating' AND duration_in_cycles IS NULL)); EXCEPTION WHEN duplicate_object THEN END;

  -- coupon_codes
  BEGIN ALTER TABLE coupon_codes ADD CONSTRAINT codes_max_redemptions_nn CHECK (max_redemptions >= 0); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE coupon_codes ADD CONSTRAINT codes_times_redeemed_nn CHECK (times_redeemed >= 0); EXCEPTION WHEN duplicate_object THEN END;

  -- discounts
  BEGIN ALTER TABLE discounts ADD CONSTRAINT discounts_target_xor CHECK (
    (subscription_id IS NOT NULL AND order_id IS NULL) OR
    (subscription_id IS NULL AND order_id IS NOT NULL)); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE discounts ADD CONSTRAINT discounts_start_cycle_nn CHECK (start_cycle >= 0); EXCEPTION WHEN duplicate_object THEN END;

  -- invoice line discount sanity
  BEGIN ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_nn CHECK (discount_total >= 0); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE invoice_line_items ADD CONSTRAINT ili_discount_cap CHECK (discount_total <= total); EXCEPTION WHEN duplicate_object THEN END;
  BEGIN ALTER TABLE invoices ADD CONSTRAINT inv_discount_nn CHECK (discount_total >= 0); EXCEPTION WHEN duplicate_object THEN END;
END $$;

-- Coupon immutability: only name, active, metadata, updated_at may change.
CREATE OR REPLACE FUNCTION coupons_block_term_update() RETURNS trigger AS $$
BEGIN
  IF (NEW.discount_type, NEW.amount_off, NEW.currency, NEW.percent_off,
      NEW.duration, NEW.duration_in_cycles, NEW.applies_to_products,
      NEW.redeem_by, NEW.max_redemptions, NEW.once_per_customer)
   IS DISTINCT FROM
     (OLD.discount_type, OLD.amount_off, OLD.currency, OLD.percent_off,
      OLD.duration, OLD.duration_in_cycles, OLD.applies_to_products,
      OLD.redeem_by, OLD.max_redemptions, OLD.once_per_customer)
  THEN RAISE EXCEPTION 'coupon terms are immutable (only name/active/metadata may change)';
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS coupons_immutable ON coupons;
CREATE TRIGGER coupons_immutable BEFORE UPDATE ON coupons
  FOR EACH ROW EXECUTE FUNCTION coupons_block_term_update();
