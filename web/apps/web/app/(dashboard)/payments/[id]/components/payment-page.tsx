"use client"

import type { PaymentResponse } from "@getpaidhq/sdk"
import { useEffect, useMemo } from "react"
import Link from "next/link"
import { format } from "date-fns"
import { ArrowUpRight, Copy } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/ui/page-header"
import { SectionHead } from "@/components/ui/section-head"
import { StatusTag, type StatusTone } from "@/components/ui/status-tag"
import { Surface, SurfaceContent } from "@/components/ui/surface"
import { useBreadcrumb } from "@/context/breadcrumb-context"
import { formatCurrency } from "@/lib/currency"
import { usePayment } from "../payment-context"
import { statuses } from "../../data/data"

const COLOR_TONE: Record<string, StatusTone> = {
  success: "success",
  warning: "warn",
  destructive: "danger",
  info: "info",
}

function AmountRow({
  label,
  value,
  emphasis = false,
}: {
  label: string
  value: React.ReactNode
  emphasis?: boolean
}) {
  return (
    <div className="grid grid-cols-1 gap-x-6 py-2.5 sm:grid-cols-[12rem_1fr]">
      <dt className="text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className={`text-sm tabular ${emphasis ? "font-semibold text-foreground" : "text-foreground"}`}>
        {value}
      </dd>
    </div>
  )
}

function RelatedRow({ label, href }: { label: string; href: string }) {
  return (
    <div className="grid grid-cols-1 gap-x-6 py-2.5 sm:grid-cols-[12rem_1fr]">
      <dt className="text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="text-sm">
        <Link
          href={href}
          className="inline-flex items-center gap-1 font-medium text-foreground hover:underline"
        >
          View {label.toLowerCase()}
          <ArrowUpRight className="size-3.5 text-muted-foreground" />
        </Link>
      </dd>
    </div>
  )
}

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[max-content_1fr] gap-x-6 py-2 text-sm">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0 truncate text-right text-foreground">{children}</dd>
    </div>
  )
}

export default function PaymentPage({ payment }: { payment: PaymentResponse }) {
  const { payment: livePayment } = usePayment()
  const { setItems } = useBreadcrumb()

  // Use the live payment data from context if available
  const p = livePayment || payment

  const amountLabel = formatCurrency(p.currency, p.amount)

  useEffect(() => {
    setItems([{ label: "Payments", href: "/payments" }, { label: amountLabel }])
    return () => setItems(null)
  }, [amountLabel, setItems])

  const status = statuses.find((s) => s.value === p.status)
  const tone: StatusTone = status ? (COLOR_TONE[status.color] ?? "neutral") : "neutral"
  const hasRelated = Boolean(p.order_id || p.subscription_id || p.invoice_id)
  const meta = useMemo(() => Object.entries(p.metadata ?? {}), [p.metadata])

  const copyId = () => {
    navigator.clipboard.writeText(p.id)
    toast.success("Payment ID copied")
  }

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader
        eyebrow="Payment"
        title={
          <span className="flex flex-wrap items-center gap-3">
            <span className="tabular">{amountLabel}</span>
            <StatusTag tone={tone}>{status?.label ?? p.status}</StatusTag>
          </span>
        }
        description={`${format(new Date(p.created_at), "MMM d, yyyy 'at' HH:mm")} · ${p.currency.toUpperCase()}`}
        actions={
          <Button variant="outline" size="sm" onClick={copyId}>
            <Copy data-icon="inline-start" />
            Copy ID
          </Button>
        }
      />

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <div className="space-y-10 lg:col-span-2">
          <section>
            <SectionHead title="Breakdown" />
            <dl className="divide-y divide-border">
              <AmountRow label="Amount" value={formatCurrency(p.currency, p.amount)} />
              {p.psp_fee != null ? (
                <AmountRow label="PSP fee" value={formatCurrency(p.currency, p.psp_fee)} />
              ) : null}
              {p.platform_fee != null ? (
                <AmountRow label="Platform fee" value={formatCurrency(p.currency, p.platform_fee)} />
              ) : null}
              {p.net_amount != null ? (
                <AmountRow label="Net" value={formatCurrency(p.currency, p.net_amount)} emphasis />
              ) : null}
            </dl>
          </section>

          {hasRelated ? (
            <section>
              <SectionHead title="Related" />
              <dl className="divide-y divide-border">
                {p.order_id ? <RelatedRow label="Order" href={`/orders/${p.order_id}`} /> : null}
                {p.subscription_id ? (
                  <RelatedRow label="Subscription" href={`/subscriptions/${p.subscription_id}`} />
                ) : null}
                {p.invoice_id ? (
                  <RelatedRow label="Invoice" href={`/invoices/${p.invoice_id}`} />
                ) : null}
              </dl>
            </section>
          ) : null}

          {meta.length > 0 ? (
            <section>
              <SectionHead title="Metadata" />
              <dl className="divide-y divide-border">
                {meta.map(([k, v]) => (
                  <AmountRow key={k} label={k} value={String(v)} />
                ))}
              </dl>
            </section>
          ) : null}
        </div>

        <div className="space-y-10">
          <Surface>
            <SurfaceContent>
              <dl className="divide-y divide-border">
                <InfoRow label="ID">
                  <code className="font-mono text-xs text-muted-foreground">{p.id}</code>
                </InfoRow>
                {p.reference ? (
                  <InfoRow label="Reference">
                    <code className="font-mono text-xs text-muted-foreground">{p.reference}</code>
                  </InfoRow>
                ) : null}
                {p.psp_id ? (
                  <InfoRow label="PSP reference">
                    <code className="font-mono text-xs text-muted-foreground">{p.psp_id}</code>
                  </InfoRow>
                ) : null}
                <InfoRow label="Created">
                  <span className="font-mono text-xs text-muted-foreground tabular">
                    {format(new Date(p.created_at), "MMM d, yyyy 'at' HH:mm")}
                  </span>
                </InfoRow>
                <InfoRow label="Updated">
                  <span className="font-mono text-xs text-muted-foreground tabular">
                    {format(new Date(p.updated_at), "MMM d, yyyy 'at' HH:mm")}
                  </span>
                </InfoRow>
              </dl>
            </SurfaceContent>
          </Surface>
        </div>
      </div>
    </div>
  )
}
