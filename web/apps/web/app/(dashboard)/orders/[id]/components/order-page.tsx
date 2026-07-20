"use client"

import { useEffect, useMemo } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { format } from "date-fns"
import { Copy } from "lucide-react"
import { toast } from "sonner"
import type {
  InvoiceResponse,
  OrderItem,
  OrderResponse,
  PaymentResponse,
  SubscriptionResponse,
} from "@getpaidhq/sdk"

import { Button } from "@/components/ui/button"
import { type Column, DataTable, PaginatedDataTable } from "@/components/ui/data-table"
import { PageHeader } from "@/components/ui/page-header"
import { SectionHead } from "@/components/ui/section-head"
import { StatusTag, type StatusTone } from "@/components/ui/status-tag"
import { Surface, SurfaceContent } from "@/components/ui/surface"
import { useBreadcrumb } from "@/context/breadcrumb-context"
import { formatCurrency } from "@/lib/currency"
import { billingCadenceLabel } from "@/lib/price-display"

const ORDER_STATUS: Record<string, { label: string; tone: StatusTone }> = {
  completed: { label: "Completed", tone: "success" },
  paid: { label: "Paid", tone: "success" },
  processing: { label: "Processing", tone: "info" },
  pending: { label: "Pending", tone: "info" },
  failed: { label: "Failed", tone: "danger" },
  cancelled: { label: "Cancelled", tone: "neutral" },
  expired: { label: "Expired", tone: "neutral" },
}

const PAYMENT_TONE: Record<string, StatusTone> = {
  succeeded: "success",
  pending: "warn",
  failed: "danger",
  refunded: "info",
  cancelled: "danger",
}

const INVOICE_TONE: Record<string, StatusTone> = {
  paid: "success",
  pending: "warn",
  partially_paid: "info",
  open: "info",
  draft: "neutral",
  overdue: "danger",
  void: "neutral",
  cancelled: "danger",
  uncollectible: "warn",
}

const SUBSCRIPTION_TONE: Record<string, StatusTone> = {
  active: "success",
  trial: "info",
  pending: "warn",
  paused: "warn",
  non_renewing: "warn",
  completed: "success",
  retry: "danger",
  past_due: "danger",
  error: "danger",
  cancelled: "neutral",
}

// "partially_paid" -> "Partially paid"
const humanize = (v?: string) =>
  v ? v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ") : ""

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[max-content_1fr] gap-x-6 py-2 text-sm">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0 truncate text-right text-foreground">{children}</dd>
    </div>
  )
}

function MetaRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 gap-x-6 py-2.5 sm:grid-cols-[12rem_1fr]">
      <dt className="text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="text-sm text-foreground">{value}</dd>
    </div>
  )
}

export default function OrderDetailPage({
  order,
  subscriptions,
  payments,
  invoices,
}: {
  order: OrderResponse
  subscriptions: SubscriptionResponse[]
  payments: PaymentResponse[]
  invoices: InvoiceResponse[]
}) {
  const router = useRouter()
  const { setItems } = useBreadcrumb()

  const totalLabel = formatCurrency(order.currency, order.total)
  const customer = order.customer
  const customerName =
    customer?.name ||
    [customer?.first_name, customer?.last_name].filter(Boolean).join(" ") ||
    customer?.email ||
    "Customer"

  useEffect(() => {
    setItems([{ label: "Orders", href: "/orders" }, { label: totalLabel }])
    return () => setItems(null)
  }, [totalLabel, setItems])

  const status = ORDER_STATUS[order.status] ?? {
    label: humanize(order.status),
    tone: "neutral" as StatusTone,
  }

  const items = order.items ?? []
  const subtotal = items.reduce((acc, i) => acc + (i.sub_total ?? 0), 0)
  const discount = items.reduce((acc, i) => acc + (i.discount_total ?? 0), 0)
  const tax = items.reduce((acc, i) => acc + (i.tax_total ?? 0), 0)

  const meta = useMemo(() => Object.entries(order.metadata ?? {}), [order.metadata])

  const itemColumns: Column<OrderItem>[] = [
    {
      key: "description",
      header: "Product",
      render: (item) =>
        item.product_id ? (
          <Link
            href={`/products/${item.product_id}`}
            className="font-medium text-foreground hover:underline"
            onClick={(e) => e.stopPropagation()}
          >
            {item.description}
          </Link>
        ) : (
          <span className="font-medium text-foreground">{item.description}</span>
        ),
    },
    {
      key: "unit",
      header: "Unit price",
      align: "right",
      width: "120px",
      render: (item) => (
        <span className="tabular text-muted-foreground">
          {formatCurrency(item.price?.currency ?? order.currency, item.price?.unit_price ?? 0)}
        </span>
      ),
    },
    {
      key: "quantity",
      header: "Qty",
      align: "right",
      width: "70px",
      render: (item) => <span className="tabular">{item.quantity}</span>,
    },
    {
      key: "total",
      header: "Total",
      align: "right",
      width: "120px",
      render: (item) => (
        <span className="tabular font-medium">
          {formatCurrency(item.price?.currency ?? order.currency, item.total)}
        </span>
      ),
    },
  ]

  const subscriptionColumns: Column<SubscriptionResponse>[] = [
    {
      key: "cadence",
      header: "Billing",
      render: (s) => (
        <span className="font-medium text-foreground">
          {billingCadenceLabel(s.billing_interval, s.billing_interval_qty)}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      render: (s) => (
        <StatusTag tone={SUBSCRIPTION_TONE[s.status] ?? "neutral"}>{humanize(s.status)}</StatusTag>
      ),
    },
    {
      key: "renews",
      header: "Renews",
      render: (s) => (
        <span className="text-muted-foreground tabular">
          {s.renews_at ? format(new Date(s.renews_at), "MMM d, yyyy HH:mm") : "—"}
        </span>
      ),
    },
    {
      key: "revenue",
      header: "Revenue",
      align: "right",
      render: (s) => (
        <span className="tabular font-medium">
          {formatCurrency(s.currency ?? order.currency, s.total_revenue ?? 0)}
        </span>
      ),
    },
  ]

  const paymentColumns: Column<PaymentResponse>[] = [
    {
      key: "date",
      header: "Date",
      render: (p) => (
        <span className="tabular">{format(new Date(p.created_at), "MMM d, yyyy HH:mm")}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      render: (p) => (
        <StatusTag tone={PAYMENT_TONE[p.status] ?? "neutral"}>{humanize(p.status)}</StatusTag>
      ),
    },
    {
      key: "reference",
      header: "Reference",
      render: (p) => (
        <span className="block max-w-[16rem] truncate font-mono text-xs text-muted-foreground">
          {p.reference}
        </span>
      ),
    },
    {
      key: "amount",
      header: "Amount",
      align: "right",
      render: (p) => (
        <span className="tabular font-medium">{formatCurrency(p.currency, p.amount)}</span>
      ),
    },
  ]

  const invoiceColumns: Column<InvoiceResponse>[] = [
    {
      key: "period",
      header: "Period",
      render: (inv) => (
        <span className="tabular">
          {inv.period_start && inv.period_end
            ? `${format(new Date(inv.period_start), "MMM d, yyyy")} – ${format(new Date(inv.period_end), "MMM d, yyyy")}`
            : format(new Date(inv.created_at), "MMM d, yyyy")}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      render: (inv) => (
        <StatusTag tone={INVOICE_TONE[inv.status] ?? "neutral"}>{humanize(inv.status)}</StatusTag>
      ),
    },
    {
      key: "cycle",
      header: "Cycle",
      align: "right",
      width: "70px",
      render: (inv) => <span className="text-muted-foreground tabular">{inv.cycle ?? "—"}</span>,
    },
    {
      key: "total",
      header: "Total",
      align: "right",
      render: (inv) => (
        <span className="tabular font-medium">{formatCurrency(inv.currency, inv.total)}</span>
      ),
    },
  ]

  const copyId = () => {
    navigator.clipboard.writeText(order.id)
    toast.success("Order ID copied")
  }

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader
        eyebrow="Order"
        title={
          <span className="flex flex-wrap items-center gap-3">
            <span className="tabular">{totalLabel}</span>
            <StatusTag tone={status.tone}>{status.label}</StatusTag>
          </span>
        }
        description={`${format(new Date(order.created_at), "MMM d, yyyy 'at' HH:mm")} · ${items.length} item${items.length === 1 ? "" : "s"} · ${customerName}`}
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
            <SectionHead title="Items" />
            <div className="flex flex-col gap-4">
              <DataTable columns={itemColumns} rows={items} />
              <div className="flex justify-end">
                <dl className="flex w-full max-w-xs flex-col gap-2 text-sm">
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">Subtotal</dt>
                    <dd className="tabular">{formatCurrency(order.currency, subtotal)}</dd>
                  </div>
                  {discount > 0 ? (
                    <div className="flex justify-between">
                      <dt className="text-muted-foreground">Discount</dt>
                      <dd className="tabular">−{formatCurrency(order.currency, discount)}</dd>
                    </div>
                  ) : null}
                  {tax > 0 ? (
                    <div className="flex justify-between">
                      <dt className="text-muted-foreground">Tax</dt>
                      <dd className="tabular">{formatCurrency(order.currency, tax)}</dd>
                    </div>
                  ) : null}
                  <div className="flex justify-between border-t border-border pt-2 font-semibold">
                    <dt>Total</dt>
                    <dd className="tabular">{totalLabel}</dd>
                  </div>
                </dl>
              </div>
            </div>
          </section>

          {subscriptions.length > 0 ? (
            <section>
              <SectionHead title="Subscriptions" />
              <DataTable
                columns={subscriptionColumns}
                rows={subscriptions}
                onRowClick={(s) => router.push(`/subscriptions/${s.id}`)}
              />
            </section>
          ) : null}

          {payments.length > 0 ? (
            <section>
              <SectionHead title="Payments" />
              <PaginatedDataTable
                columns={paymentColumns}
                rows={payments}
                onRowClick={(p) => router.push(`/payments/${p.id}`)}
              />
            </section>
          ) : null}

          {invoices.length > 0 ? (
            <section>
              <SectionHead title="Invoices" />
              <PaginatedDataTable
                columns={invoiceColumns}
                rows={invoices}
                onRowClick={(inv) => router.push(`/invoices/${inv.id}`)}
              />
            </section>
          ) : null}

          {meta.length > 0 ? (
            <section>
              <SectionHead title="Metadata" />
              <dl className="divide-y divide-border">
                {meta.map(([k, v]) => (
                  <MetaRow key={k} label={k} value={String(v)} />
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
                  <code className="font-mono text-xs text-muted-foreground">{order.id}</code>
                </InfoRow>
                {order.reference ? (
                  <InfoRow label="Reference">
                    <code className="font-mono text-xs text-muted-foreground">
                      {order.reference}
                    </code>
                  </InfoRow>
                ) : null}
                <InfoRow label="Customer">
                  <Link
                    href={`/customers/${order.customer_id || customer?.id}`}
                    className="text-sm font-medium text-foreground hover:underline"
                  >
                    {customerName}
                  </Link>
                </InfoRow>
                <InfoRow label="Created">
                  <span className="font-mono text-xs text-muted-foreground tabular">
                    {format(new Date(order.created_at), "MMM d, yyyy 'at' HH:mm")}
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
