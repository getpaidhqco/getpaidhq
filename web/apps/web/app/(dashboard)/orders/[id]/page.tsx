import { Metadata } from "next"
import { notFound } from "next/navigation"
import { loadAuthProvider } from "@getpaidhq/auth/server"
import type {
  InvoiceResponse,
  OrderResponse,
  PaymentResponse,
  SubscriptionResponse,
} from "@getpaidhq/sdk"

import OrderDetailPage from "@/app/(dashboard)/orders/[id]/components/order-page"

export const metadata: Metadata = {
  title: "Order",
}

async function fetchJson<T>(path: string, headers: Record<string, string>): Promise<T | undefined> {
  try {
    const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${path}`, { headers })
    if (!rsp.ok) throw new Error(`${rsp.status} ${rsp.statusText}`)
    return (await rsp.json()) as T
  } catch (e) {
    console.error(`GET ${path} failed:`, e)
  }
}

export default async function OrderPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  const authProvider = loadAuthProvider()
  const authHeaders = await authProvider.getAuthHeader()

  const [order, subscriptions] = await Promise.all([
    fetchJson<OrderResponse>(`/api/orders/${id}`, authHeaders),
    fetchJson<SubscriptionResponse[]>(`/api/orders/${id}/subscriptions`, authHeaders),
  ])

  if (!order) {
    notFound()
  }

  // Payments and invoices hang off the order's subscriptions (there is no
  // order-scoped list endpoint), so fan out per subscription and merge.
  const subs = subscriptions ?? []
  const perSub = await Promise.all(
    subs.map(async (s) => {
      const [payments, invoices] = await Promise.all([
        fetchJson<{ data: PaymentResponse[] }>(
          `/api/subscriptions/${s.id}/payments?page=0&limit=50&sort_order=desc&sort_by=created_at`,
          authHeaders,
        ),
        fetchJson<{ data: InvoiceResponse[] }>(
          `/api/subscriptions/${s.id}/invoices?page=0&limit=50`,
          authHeaders,
        ),
      ])
      return { payments: payments?.data ?? [], invoices: invoices?.data ?? [] }
    }),
  )

  const payments = perSub
    .flatMap((r) => r.payments)
    .sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at))
  const invoices = perSub
    .flatMap((r) => r.invoices)
    .sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at))

  return (
    <OrderDetailPage
      order={order}
      subscriptions={subs}
      payments={payments}
      invoices={invoices}
    />
  )
}
