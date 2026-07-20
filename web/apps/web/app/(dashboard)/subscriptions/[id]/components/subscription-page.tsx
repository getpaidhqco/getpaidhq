"use client"
import type { PaymentMethodResponse } from "@getpaidhq/sdk"
import { useEffect, useState } from "react"
import Link from "next/link"
import { format } from "date-fns"
import { Copy } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { KpiRow } from "@/components/ui/kpi-row"
import { PageHeader } from "@/components/ui/page-header"
import { SectionHead } from "@/components/ui/section-head"
import { StatusTag, type StatusTone } from "@/components/ui/status-tag"
import { Surface, SurfaceContent } from "@/components/ui/surface"
import { useBreadcrumb } from "@/context/breadcrumb-context"
import { formatCurrency } from "@/lib/currency"
import { billingCadenceLabel } from "@/lib/price-display"
import { PaymentsTable } from "@/app/(dashboard)/subscriptions/[id]/components/payments-table/payments-table"
import { columns } from "@/app/(dashboard)/subscriptions/[id]/components/payments-table/columns"
import ActionsDropdown from "@/app/(dashboard)/subscriptions/[id]/components/actions-dropdown"
import PauseAlert from "@/app/(dashboard)/subscriptions/[id]/components/pause-alert"
import ResumeAlert from "@/app/(dashboard)/subscriptions/[id]/components/resume-alert"
import CancelAlert from "@/app/(dashboard)/subscriptions/[id]/components/cancel-alert"
import UpdateSubscriptionDialog from "@/app/(dashboard)/subscriptions/[id]/components/update-subscription-dialog"
import { useSubscription } from "@/app/(dashboard)/subscriptions/[id]/subscription-context"

const STATUS: Record<string, { label: string; tone: StatusTone }> = {
  active: { label: "Active", tone: "success" },
  pending: { label: "Pending", tone: "warn" },
  paused: { label: "Paused", tone: "warn" },
  non_renewing: { label: "Not renewing", tone: "warn" },
  retry: { label: "Retry", tone: "danger" },
  trial: { label: "In trial", tone: "info" },
  completed: { label: "Completed", tone: "success" },
  cancelled: { label: "Cancelled", tone: "neutral" },
  past_due: { label: "Past due", tone: "danger" },
  error: { label: "Error", tone: "danger" },
}

const titleCase = (s?: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1) : "")

/** "Visa •••• 4242 · expires 12/2027" from whatever detail fields exist. */
function paymentMethodLabel(pm: PaymentMethodResponse): string {
  const details = pm.details ?? {}
  return [
    titleCase(details.brand || pm.type || "Card"),
    details.last4 ? `•••• ${details.last4}` : null,
    details.expiry_month && details.expiry_year
      ? `expires ${details.expiry_month}/${details.expiry_year}`
      : null,
  ]
    .filter(Boolean)
    .join(" · ")
}

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 gap-x-6 py-2.5 sm:grid-cols-[12rem_1fr]">
      <dt className="text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-sm text-foreground">{children}</dd>
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

function SubscriptionPageContent({ paymentMethod }: { paymentMethod?: PaymentMethodResponse }) {
  const [isOpen, setIsOpen] = useState({
    pause: false,
    resume: false,
    cancel: false,
    update: false,
  })
  const { pause, resume, cancel, refreshData, subscription } = useSubscription()
  const { setItems } = useBreadcrumb()

  const customer = subscription.customer
  const customerName =
    customer?.name ||
    [customer?.first_name, customer?.last_name].filter(Boolean).join(" ") ||
    customer?.email ||
    "Customer"

  useEffect(() => {
    setItems([{ label: "Subscriptions", href: "/subscriptions" }, { label: customerName }])
    return () => setItems(null)
  }, [customerName, setItems])

  const onSelect = (action: string) => {
    if (action === "pause" || action === "cancel" || action === "resume" || action === "update") {
      toggleShow(action)
    }
  }

  const toggleShow = (key: "pause" | "resume" | "cancel" | "update") => {
    setIsOpen((o) => ({ ...o, [key]: !o[key] }))
  }

  const onResume = async (
    resume_behavior: "continue_existing_billing_period" | "start_new_billing_period",
  ) => {
    try {
      await resume(subscription!.id, { resume_behavior })
      toast.success("Subscription resumed successfully")
      toggleShow("resume")
      refreshData()
      return Promise.resolve()
    } catch (err: unknown) {
      toast.error(
        `Failed to resume the subscription ${err instanceof Error ? err.message : ""}`,
      )
      return Promise.reject(err)
    }
  }

  const onPause = () => {
    toggleShow("pause")
    pause(subscription!.id, { reason: "Customer requested" })
      .then(() => {
        toast.success("The subscription is paused")
        refreshData()
      })
      .catch((err) => {
        toast.error(`${err?.message}`)
      })
  }

  const onCancel = async (reason: string) => {
    try {
      await cancel(subscription!.id, { reason })
      toast.success("The subscription is cancelled")
      toggleShow("cancel")
      return refreshData()
    } catch (err: unknown) {
      toast.error(
        `Failed to cancel the subscription ${err instanceof Error ? err.message : ""}`,
      )
    }
  }

  const status = STATUS[subscription.status] ?? {
    label: titleCase(subscription.status),
    tone: "neutral" as StatusTone,
  }
  const cadence = billingCadenceLabel(
    subscription.billing_interval,
    subscription.billing_interval_qty,
  )
  const startedAt = subscription.start_date || subscription.created_at

  const copyId = () => {
    navigator.clipboard.writeText(subscription.id)
    toast.success("Subscription ID copied")
  }

  return (
    <>
      <PauseAlert
        isOpen={isOpen.pause}
        onClose={() => setIsOpen((o) => ({ ...o, pause: false }))}
        onSubmit={onPause}
      />
      <ResumeAlert
        isOpen={isOpen.resume}
        onClose={() => setIsOpen((o) => ({ ...o, resume: false }))}
        onSubmit={onResume}
      />
      <CancelAlert
        isOpen={isOpen.cancel}
        onClose={() => setIsOpen((o) => ({ ...o, cancel: false }))}
        onSubmit={onCancel}
      />
      <UpdateSubscriptionDialog
        isOpen={isOpen.update}
        onClose={() => setIsOpen((o) => ({ ...o, update: false }))}
        subscriptionId={subscription.id}
        onSuccess={refreshData}
      />

      <div className="flex flex-1 flex-col gap-8">
        <PageHeader
          eyebrow="Subscription"
          title={
            <span className="flex flex-wrap items-center gap-3">
              {customerName}
              <StatusTag tone={status.tone}>{status.label}</StatusTag>
            </span>
          }
          description={`${cadence}${startedAt ? ` · started ${format(new Date(startedAt), "MMM d, yyyy")}` : ""}`}
          actions={
            <>
              <Button variant="outline" size="sm" onClick={copyId}>
                <Copy data-icon="inline-start" />
                Copy ID
              </Button>
              <ActionsDropdown subscription={subscription} onSelect={onSelect} />
            </>
          }
        />

        <KpiRow
          cols={3}
          items={[
            {
              label: "Total revenue",
              value: formatCurrency(subscription.currency, subscription.total_revenue),
            },
            {
              label: "Cycles processed",
              value: String(subscription.cycles_processed ?? 0),
              sub:
                subscription.cycles > 0
                  ? `of ${subscription.cycles}`
                  : "until cancelled",
            },
            {
              label: "Renews",
              value: subscription.renews_at
                ? format(new Date(subscription.renews_at), "MMM d, yyyy")
                : "—",
              sub: subscription.renews_at
                ? format(new Date(subscription.renews_at), "HH:mm")
                : undefined,
            },
          ]}
        />

        <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
          <div className="space-y-10 lg:col-span-2">
            <section>
              <SectionHead title="Details" />
              <dl className="divide-y divide-border">
                <DetailRow label="Customer">
                  <Link
                    href={`/customers/${customer?.id}`}
                    className="font-medium text-foreground hover:underline"
                  >
                    {customerName}
                  </Link>
                  {customer?.email && customer.email !== customerName ? (
                    <span className="ml-2 text-muted-foreground">{customer.email}</span>
                  ) : null}
                </DetailRow>
                <DetailRow label="Billing">{cadence}</DetailRow>
                {subscription.current_period_start && subscription.current_period_end ? (
                  <DetailRow label="Current period">
                    {format(new Date(subscription.current_period_start), "MMM d, yyyy")} –{" "}
                    {format(new Date(subscription.current_period_end), "MMM d, yyyy")}
                  </DetailRow>
                ) : null}
                {subscription.renews_at ? (
                  <DetailRow label="Renews">
                    {format(new Date(subscription.renews_at), "MMM d, yyyy 'at' HH:mm")}
                  </DetailRow>
                ) : null}
                {subscription.trial_ends_at ? (
                  <DetailRow label="Trial ends">
                    {format(new Date(subscription.trial_ends_at), "MMM d, yyyy")}
                  </DetailRow>
                ) : null}
                {paymentMethod ? (
                  <DetailRow label="Payment method">{paymentMethodLabel(paymentMethod)}</DetailRow>
                ) : null}
              </dl>
            </section>

            <section>
              <SectionHead title="Payments" />
              <PaymentsTable columns={columns} subscription={subscription} />
            </section>
          </div>

          <div className="space-y-10">
            <Surface>
              <SurfaceContent>
                <dl className="divide-y divide-border">
                  <InfoRow label="ID">
                    <code className="font-mono text-xs text-muted-foreground">
                      {subscription.id}
                    </code>
                  </InfoRow>
                  {subscription.order_id ? (
                    <InfoRow label="Order">
                      <Link
                        href={`/orders/${subscription.order_id}`}
                        className="font-mono text-xs text-foreground hover:underline"
                      >
                        {subscription.order_id}
                      </Link>
                    </InfoRow>
                  ) : null}
                  <InfoRow label="Created">
                    <span className="font-mono text-xs text-muted-foreground tabular">
                      {format(new Date(subscription.created_at), "MMM d, yyyy 'at' HH:mm")}
                    </span>
                  </InfoRow>
                  <InfoRow label="Updated">
                    <span className="font-mono text-xs text-muted-foreground tabular">
                      {format(new Date(subscription.updated_at), "MMM d, yyyy 'at' HH:mm")}
                    </span>
                  </InfoRow>
                </dl>
              </SurfaceContent>
            </Surface>
          </div>
        </div>
      </div>
    </>
  )
}

export default function SubscriptionPage({
  paymentMethod,
}: {
  paymentMethod?: PaymentMethodResponse
}) {
  return <SubscriptionPageContent paymentMethod={paymentMethod} />
}
