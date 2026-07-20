export type CheckoutLineItem = {
  id: string;
  name: string;
  description?: string;
  kind: "flat" | "metered";
  quantity?: number;
  /** Unit amount in minor units (cents). For metered items this is the per-unit rate. */
  unitAmount: number;
  /** Human label for the metered rate, e.g. "per 1,000 requests". */
  unitLabel?: string;
};

export type CheckoutSession = {
  slug: string;
  merchant: {
    name: string;
    logoUrl?: string;
    returnUrl: string;
  };
  currency: string;
  locale: string;
  interval: "month" | "year";
  items: CheckoutLineItem[];
  /** Fractional tax rate applied after discounts, e.g. 0.0875. */
  taxRate: number;
  taxLabel: string;
  collectsShipping: boolean;
  /** Map of promotion code -> percent off the first payment. */
  promoCodes: Record<string, number>;
};

export type AppliedPromo = {
  code: string;
  percentOff: number;
};

export type CheckoutTotals = {
  subtotal: number;
  discount: number;
  tax: number;
  total: number;
  /** Recurring flat amount per interval, before usage. */
  recurring: number;
};

export function computeTotals(
  session: CheckoutSession,
  promo: AppliedPromo | null,
): CheckoutTotals {
  const subtotal = session.items
    .filter((item) => item.kind === "flat")
    .reduce((sum, item) => sum + item.unitAmount * (item.quantity ?? 1), 0);
  const discount = promo
    ? Math.round((subtotal * promo.percentOff) / 100)
    : 0;
  const tax = Math.round((subtotal - discount) * session.taxRate);

  return {
    subtotal,
    discount,
    tax,
    total: subtotal - discount + tax,
    recurring: subtotal,
  };
}

export function formatMoney(
  minorUnits: number,
  session: Pick<CheckoutSession, "currency" | "locale">,
): string {
  return new Intl.NumberFormat(session.locale, {
    style: "currency",
    currency: session.currency,
  }).format(minorUnits / 100);
}

export function demoSession(slug: string): CheckoutSession {
  return {
    slug,
    merchant: {
      name: "Driftwave",
      logoUrl: "https://assets.ui.sh/marks/1.svg?color=blue-600",
      returnUrl: "https://driftwave.example.com",
    },
    currency: "USD",
    locale: "en-US",
    interval: "month",
    items: [
      {
        id: "li_scale",
        name: "Driftwave Scale",
        description: "5 seats",
        kind: "flat",
        quantity: 5,
        unitAmount: 1200,
      },
      {
        id: "li_api",
        name: "API requests",
        description: "Billed monthly based on usage",
        kind: "metered",
        unitAmount: 80,
        unitLabel: "per 1,000 requests",
      },
    ],
    taxRate: 0.0875,
    taxLabel: "Sales tax (8.75%)",
    collectsShipping: false,
    promoCodes: { SAVE20: 20 },
  };
}
