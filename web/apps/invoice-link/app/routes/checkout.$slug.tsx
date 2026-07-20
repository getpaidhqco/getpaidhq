import { useState } from "react";
import { useLoaderData } from "react-router";
import { ArrowLeft, ChevronDown, Clock } from "lucide-react";
import type { Route } from "./+types/checkout.$slug";
import { Button } from "@/components/ui/button";
import { OrderSummary } from "@/components/checkout/order-summary";
import { PaymentForm } from "@/components/checkout/payment-form";
import {
  type AppliedPromo,
  computeTotals,
  demoSession,
  formatMoney,
} from "~/lib/checkout";

export function meta({ data }: Route.MetaArgs) {
  return [
    { title: `${data?.session.merchant.name ?? "Checkout"} – Checkout` },
    { name: "description", content: "Complete your subscription securely" },
  ];
}

const PREVIEW_STATES = ["success", "declined", "expired"] as const;
type PreviewState = (typeof PREVIEW_STATES)[number];

export async function loader({ params, request }: Route.LoaderArgs) {
  const url = new URL(request.url);
  const state = url.searchParams.get("state");
  return {
    session: demoSession(params.slug),
    initialState: PREVIEW_STATES.includes(state as PreviewState)
      ? (state as PreviewState)
      : null,
  };
}

function ExpiredSession({ merchant }: { merchant: { name: string; returnUrl: string } }) {
  return (
    <div className="flex flex-col items-start gap-5">
      <Clock className="size-6 shrink-0 text-muted-foreground" />
      <div>
        <h2 className="text-lg font-semibold text-balance text-foreground">
          This checkout session has expired
        </h2>
        <p className="mt-1 text-base/7 text-pretty text-muted-foreground sm:text-sm/6">
          For your security, checkout sessions expire after a period of
          inactivity. Return to {merchant.name} to start a new one.
        </p>
      </div>
      <Button asChild size="lg" className="w-full">
        <a href={merchant.returnUrl}>Return to {merchant.name}</a>
      </Button>
    </div>
  );
}

function Footer() {
  return (
    <div className="flex items-center gap-4 text-sm text-muted-foreground">
      <p>
        Powered by <span className="font-medium">GetPaidHQ</span>
      </p>
      <span className="h-3 w-px bg-black/10" aria-hidden="true" />
      <a href="https://getpaidhq.com/terms" className="hover:text-foreground">
        Terms
      </a>
      <a href="https://getpaidhq.com/privacy" className="hover:text-foreground">
        Privacy
      </a>
    </div>
  );
}

export default function CheckoutSlug() {
  const { session, initialState } = useLoaderData<typeof loader>();
  const [promo, setPromo] = useState<AppliedPromo | null>(null);
  const totals = computeTotals(session, promo);
  const { merchant } = session;

  const applyPromo = (code: string) => {
    const percentOff = session.promoCodes[code.trim().toUpperCase()];
    if (!percentOff) return false;
    setPromo({ code: code.trim().toUpperCase(), percentOff });
    return true;
  };
  const removePromo = () => setPromo(null);

  const summaryProps = {
    session,
    totals,
    promo,
    onApplyPromo: applyPromo,
    onRemovePromo: removePromo,
  };

  return (
    <main className="isolate min-h-dvh bg-background">
      <div className="mx-auto flex min-h-dvh max-w-4xl flex-col gap-8 px-4 py-6 sm:px-6 md:gap-12 md:py-16">
        <div className="grid flex-1 content-start grid-cols-1 gap-8 md:grid-cols-2 md:gap-12 lg:gap-16">
          {/* Summary side */}
          <div className="flex flex-col gap-6">
            <header>
              <a
                href={merchant.returnUrl}
                className="group inline-flex items-center gap-3"
              >
                <ArrowLeft className="size-4 shrink-0 text-muted-foreground group-hover:text-foreground" />
                {merchant.logoUrl && (
                  <img src={merchant.logoUrl} alt="" className="size-7 shrink-0" />
                )}
                <span className="text-base font-medium text-foreground">
                  {merchant.name}
                </span>
              </a>
            </header>

            <div>
              <p className="text-base text-muted-foreground">
                Subscribe to {session.items[0]?.name}
              </p>
              <p className="mt-1 text-4xl font-semibold tracking-tight text-foreground tabular-nums">
                {formatMoney(totals.total, session)}
              </p>
              <p className="mt-1 text-base text-muted-foreground sm:text-sm">
                Due today, then {formatMoney(totals.recurring, session)} plus
                usage per {session.interval}
              </p>
            </div>

            {/* Mobile: collapsed summary accordion */}
            <details className="group rounded-xl bg-card shadow-xs ring-1 ring-black/5 md:hidden">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-4 p-4 [&::-webkit-details-marker]:hidden">
                <span className="flex items-center gap-2 text-base text-muted-foreground">
                  Order summary
                  <ChevronDown className="size-4 shrink-0 transition-transform group-open:rotate-180" />
                </span>
                <span className="text-base font-semibold text-foreground tabular-nums">
                  {formatMoney(totals.total, session)}
                </span>
              </summary>
              <div className="border-t border-black/5 p-4">
                <OrderSummary {...summaryProps} />
              </div>
            </details>

            {/* Desktop: full summary */}
            <div className="max-md:hidden">
              <OrderSummary {...summaryProps} />
            </div>

            <div className="mt-auto max-md:hidden">
              <Footer />
            </div>
          </div>

          {/* Payment side */}
          <div>
            <div className="rounded-2xl bg-card p-6 shadow-sm ring-1 ring-black/5 sm:p-8">
              {initialState === "expired" ? (
                <ExpiredSession merchant={merchant} />
              ) : (
                <PaymentForm
                  session={session}
                  totals={totals}
                  initialStatus={
                    initialState === "success"
                      ? "succeeded"
                      : initialState === "declined"
                        ? "declined"
                        : "idle"
                  }
                />
              )}
            </div>
          </div>
        </div>

        <div className="md:hidden">
          <Footer />
        </div>
      </div>
    </main>
  );
}
