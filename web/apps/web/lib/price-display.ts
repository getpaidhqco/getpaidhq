import type { PriceResponse } from "@getpaidhq/sdk"

import { getBrowserLocale } from "@/lib/currency"

/**
 * Display helpers for prices. A price's headline amount depends on its category
 * and scheme: fixed/package prices carry a unit price (possibly per block of
 * unit_count units), graduated/volume prices are rate bands, variable prices are
 * chosen by the customer at checkout, and free prices have no amount at all.
 */

/**
 * Cents → localized currency string. Fractional cents (sub-cent per-unit rates,
 * e.g. tier rates like 0.5¢) keep up to 6 decimals instead of rounding to $0.01.
 */
export function formatAmount(currency: string, cents: number): string {
  const opts: Intl.NumberFormatOptions = { style: "currency", currency }
  if (!Number.isInteger(cents)) opts.maximumFractionDigits = 6
  return new Intl.NumberFormat(getBrowserLocale() || "en", opts).format(cents / 100)
}

/** Localized unit quantity ("1,000"). */
export function formatUnits(n: number): string {
  return new Intl.NumberFormat(getBrowserLocale() || "en").format(n)
}

export type PriceAmountParts = {
  /** Headline: "$10.00", "$5.00 / 1,000 units", "$0.005–$0.01 / unit", "Free"… */
  main: string
  /** Qualifier: "per started block, rounded up", "3 graduated tiers", "min $5.00"… */
  detail?: string
}

const TIERED = new Set(["graduated", "volume", "tiered"])

/** The amount cell for a price row, correct for every category/scheme combination. */
export function priceAmountParts(p: PriceResponse): PriceAmountParts {
  if (p.category === "free") return { main: "Free" }

  if (p.category === "variable") {
    const min = p.min_price ? `min ${formatAmount(p.currency, p.min_price)}` : undefined
    if (p.suggested_price) {
      return {
        main: formatAmount(p.currency, p.suggested_price),
        detail: ["suggested", min].filter(Boolean).join(" · "),
      }
    }
    return { main: "Customer chooses", detail: min }
  }

  if (TIERED.has(p.scheme)) {
    const kind = p.scheme === "volume" ? "volume" : "graduated"
    const count = p.tiers?.length ?? 0
    const detail = `${count} ${kind} tier${count === 1 ? "" : "s"}`
    const rates = (p.tiers ?? [])
      .map((t) => Number(t.per_unit_amount))
      .filter((n) => Number.isFinite(n))
    if (rates.length === 0) return { main: "Tiered", detail }
    const lo = Math.min(...rates)
    const hi = Math.max(...rates)
    const main =
      lo === hi
        ? `${formatAmount(p.currency, lo)} / unit`
        : `${formatAmount(p.currency, lo)}–${formatAmount(p.currency, hi)} / unit`
    return { main, detail }
  }

  const unitCount = Math.max(p.unit_count ?? 1, 1)

  if (p.scheme === "package") {
    return {
      main: `${formatAmount(p.currency, p.unit_price)} / ${formatUnits(unitCount)} units`,
      detail: "per started block, rounded up",
    }
  }

  // Fixed: a block price prorates, so show the effective per-unit rate.
  if (unitCount > 1) {
    return {
      main: `${formatAmount(p.currency, p.unit_price)} / ${formatUnits(unitCount)} units`,
      detail: `${formatAmount(p.currency, p.unit_price / unitCount)} per unit, prorated`,
    }
  }

  if (p.billable_metric_id) {
    return { main: `${formatAmount(p.currency, p.unit_price)} / unit` }
  }
  return { main: formatAmount(p.currency, p.unit_price) }
}

const INTERVAL_ADVERB: Record<string, string> = {
  hour: "Hourly",
  day: "Daily",
  week: "Weekly",
  month: "Monthly",
  year: "Yearly",
}

/** "Monthly", "Every 3 months", "Every 10 minutes", or "One-time" for no interval. */
export function billingCadenceLabel(interval?: string, qty: number = 1): string {
  if (!interval || interval === "none") return "One-time"
  if (qty > 1) return `Every ${qty} ${interval}s`
  return INTERVAL_ADVERB[interval] ?? `Every ${interval}`
}

const APPROX_DAYS: Record<string, number> = {
  second: 1 / 86400,
  minute: 1 / 1440,
  day: 1,
  week: 7,
  month: 30,
  year: 365,
}

/**
 * The billing cadence for a price row: "One-time", "Monthly", "Every 3 months"…
 * Mirrors the server's Price.SubscriptionCadence: a metered price always recurs
 * and is capped at monthly, so cadences longer than a month (or none) display as
 * Monthly for metered prices.
 */
export function priceBillingLabel(p: PriceResponse): string {
  const interval = p.billing_interval
  const qty = p.billing_interval_qty || 1
  const recurring = interval && interval !== "none"
  if (p.billable_metric_id) {
    if (!recurring || (APPROX_DAYS[interval] ?? 0) * qty > 31) return "Monthly"
  }
  return billingCadenceLabel(interval, qty)
}
