"use client"
import { useEffect, useMemo } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { format } from "date-fns"
import { Copy } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/ui/page-header"
import { SectionHead } from "@/components/ui/section-head"
import { StatusTag, type StatusTone } from "@/components/ui/status-tag"
import { Surface, SurfaceContent } from "@/components/ui/surface"
import { useBreadcrumb } from "@/context/breadcrumb-context"
import { formatCurrency } from "@/lib/currency"
import ItemsTable from "@/app/(dashboard)/invoices/[id]/components/items-table"
import { useInvoice } from "../invoice-context"

const STATUS_TONE: Record<string, StatusTone> = {
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

export default function ViewInvoice() {
  const router = useRouter()
  const { invoice, isLoading, error } = useInvoice()
  const { setItems } = useBreadcrumb()

  const amountLabel = invoice ? formatCurrency(invoice.currency, invoice.total) : ""

  useEffect(() => {
    if (!invoice) return
    setItems([{ label: "Invoices", href: "/invoices" }, { label: amountLabel }])
    return () => setItems(null)
  }, [invoice, amountLabel, setItems])

  // Surface fetch errors as a toast (in an effect — never during render).
  useEffect(() => {
    if (error) {
      toast.error("Failed to fetch invoice", {
        description: error instanceof Error ? error.message : "Unknown error",
        duration: 5000,
      })
      console.error("Error fetching invoice:", error)
    }
  }, [error])

  const meta = useMemo(() => Object.entries(invoice?.metadata ?? {}), [invoice?.metadata])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-foreground mx-auto"></div>
          <p className="mt-4 text-muted-foreground">Loading invoice...</p>
        </div>
      </div>
    )
  }

  if (error || !invoice) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <p className="text-destructive text-lg">Failed to load invoice</p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => router.push("/invoices")}
          >
            Back to Invoices
          </Button>
        </div>
      </div>
    )
  }

  const tone = STATUS_TONE[invoice.status] ?? "neutral"
  const period =
    invoice.period_start && invoice.period_end
      ? `${format(new Date(invoice.period_start), "MMM d, yyyy")} – ${format(new Date(invoice.period_end), "MMM d, yyyy")}`
      : null

  const copyId = () => {
    navigator.clipboard.writeText(invoice.id)
    toast.success("Invoice ID copied")
  }

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader
        eyebrow="Invoice"
        title={
          <span className="flex flex-wrap items-center gap-3">
            <span className="tabular">{amountLabel}</span>
            <StatusTag tone={tone}>{humanize(invoice.status)}</StatusTag>
          </span>
        }
        description={[period, invoice.cycle ? `cycle ${invoice.cycle}` : null]
          .filter(Boolean)
          .join(" · ")}
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
            <SectionHead title="Line items" />
            <ItemsTable />
          </section>

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
                  <code className="font-mono text-xs text-muted-foreground">{invoice.id}</code>
                </InfoRow>
                {invoice.customer_id ? (
                  <InfoRow label="Customer">
                    <Link
                      href={`/customers/${invoice.customer_id}`}
                      className="font-mono text-xs text-foreground hover:underline"
                    >
                      {invoice.customer_id}
                    </Link>
                  </InfoRow>
                ) : null}
                {invoice.subscription_id ? (
                  <InfoRow label="Subscription">
                    <Link
                      href={`/subscriptions/${invoice.subscription_id}`}
                      className="font-mono text-xs text-foreground hover:underline"
                    >
                      {invoice.subscription_id}
                    </Link>
                  </InfoRow>
                ) : null}
                {invoice.order_id ? (
                  <InfoRow label="Order">
                    <Link
                      href={`/orders/${invoice.order_id}`}
                      className="font-mono text-xs text-foreground hover:underline"
                    >
                      {invoice.order_id}
                    </Link>
                  </InfoRow>
                ) : null}
                <InfoRow label="Created">
                  <span className="font-mono text-xs text-muted-foreground tabular">
                    {format(new Date(invoice.created_at), "MMM d, yyyy 'at' HH:mm")}
                  </span>
                </InfoRow>
                <InfoRow label="Updated">
                  <span className="font-mono text-xs text-muted-foreground tabular">
                    {format(new Date(invoice.updated_at), "MMM d, yyyy 'at' HH:mm")}
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
