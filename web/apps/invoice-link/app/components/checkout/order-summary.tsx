import { useState } from "react";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  type AppliedPromo,
  type CheckoutSession,
  type CheckoutTotals,
  formatMoney,
} from "~/lib/checkout";

type OrderSummaryProps = {
  session: CheckoutSession;
  totals: CheckoutTotals;
  promo: AppliedPromo | null;
  onApplyPromo: (code: string) => boolean;
  onRemovePromo: () => void;
};

function PromoCode({
  promo,
  onApplyPromo,
  onRemovePromo,
}: Pick<OrderSummaryProps, "promo" | "onApplyPromo" | "onRemovePromo">) {
  const [open, setOpen] = useState(false);
  const [code, setCode] = useState("");
  const [invalid, setInvalid] = useState(false);

  if (promo) {
    return (
      <div className="flex items-center justify-between gap-3">
        <span className="inline-flex items-center gap-1 rounded-md bg-black/5 py-1 pr-1 pl-2">
          <span className="text-sm font-medium text-foreground">
            {promo.code}
          </span>
          <button
            type="button"
            aria-label={`Remove promotion code ${promo.code}`}
            onClick={onRemovePromo}
            className="relative rounded-sm p-0.5 text-muted-foreground hover:text-foreground"
          >
            <X className="size-4 shrink-0" />
            <span
              className="absolute top-1/2 left-1/2 size-[max(100%,3rem)] -translate-1/2 pointer-fine:hidden"
              aria-hidden="true"
            />
          </button>
        </span>
        <p className="text-base text-muted-foreground sm:text-sm">
          {promo.percentOff}% off
        </p>
      </div>
    );
  }

  if (!open) {
    return (
      <div>
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="text-base font-medium text-primary hover:text-primary/80 sm:text-sm"
        >
          Add promotion code
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex gap-2">
        <Input
          name="promo-code"
          aria-label="Promotion code"
          placeholder="Promotion code"
          autoFocus
          value={code}
          aria-invalid={invalid || undefined}
          onChange={(event) => {
            setCode(event.target.value.toUpperCase());
            setInvalid(false);
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              setInvalid(!onApplyPromo(code));
            }
          }}
          className="h-9 flex-1"
        />
        <Button
          type="button"
          variant="secondary"
          size="sm"
          className="h-9"
          onClick={() => setInvalid(!onApplyPromo(code))}
        >
          Apply
        </Button>
      </div>
      {invalid && (
        <p className="text-sm text-destructive">
          This code is invalid or has expired.
        </p>
      )}
    </div>
  );
}

export function OrderSummary({
  session,
  totals,
  promo,
  onApplyPromo,
  onRemovePromo,
}: OrderSummaryProps) {
  const money = (amount: number) => formatMoney(amount, session);

  return (
    <div className="flex flex-col gap-6">
      <ul role="list" className="flex flex-col gap-5">
        {session.items.map((item) => (
          <li key={item.id} className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <p className="text-base font-medium text-foreground sm:text-sm">
                {item.name}
              </p>
              <p className="mt-0.5 text-base text-muted-foreground sm:text-sm">
                {item.kind === "metered"
                  ? `${money(item.unitAmount)} ${item.unitLabel}`
                  : (item.quantity ?? 1) > 1
                    ? `${item.quantity} × ${money(item.unitAmount)}`
                    : item.description}
              </p>
              {item.kind === "metered" && item.description && (
                <p className="text-base text-muted-foreground sm:text-sm">
                  {item.description}
                </p>
              )}
            </div>
            <p className="text-base text-foreground tabular-nums sm:text-sm">
              {item.kind === "metered"
                ? "Usage-based"
                : money(item.unitAmount * (item.quantity ?? 1))}
            </p>
          </li>
        ))}
      </ul>

      <div className="border-t border-black/10 pt-5">
        <PromoCode
          promo={promo}
          onApplyPromo={onApplyPromo}
          onRemovePromo={onRemovePromo}
        />
      </div>

      <div className="flex flex-col gap-2 border-t border-black/10 pt-5">
        <div className="flex items-baseline justify-between gap-4">
          <p className="text-base text-muted-foreground sm:text-sm">Subtotal</p>
          <p className="text-base text-foreground tabular-nums sm:text-sm">
            {money(totals.subtotal)}
          </p>
        </div>
        {totals.discount > 0 && promo && (
          <div className="flex items-baseline justify-between gap-4">
            <p className="text-base text-muted-foreground sm:text-sm">
              Discount
            </p>
            <p className="text-base text-foreground tabular-nums sm:text-sm">
              −{money(totals.discount)}
            </p>
          </div>
        )}
        <div className="flex items-baseline justify-between gap-4">
          <p className="text-base text-muted-foreground sm:text-sm">
            {session.taxLabel}
          </p>
          <p className="text-base text-foreground tabular-nums sm:text-sm">
            {money(totals.tax)}
          </p>
        </div>
        <div className="mt-2 flex items-baseline justify-between gap-4 border-t border-black/10 pt-4">
          <p className="text-base font-semibold text-foreground sm:text-sm">
            Total due today
          </p>
          <p className="text-base font-semibold text-foreground tabular-nums sm:text-sm">
            {money(totals.total)}
          </p>
        </div>
        <p className="text-sm text-pretty text-muted-foreground">
          Then {money(totals.recurring)} plus metered usage per{" "}
          {session.interval}.
        </p>
      </div>
    </div>
  );
}
