import { Metadata } from "next"
import type { PaymentResponse } from "@getpaidhq/sdk"
import { notFound } from 'next/navigation'
import { loadAuthProvider } from "@getpaidhq/auth/server"
import { AuthHeader } from "@getpaidhq/auth"
import { PaymentProvider } from "./payment-context"
import PaymentPage from "./components/payment-page"

export const metadata: Metadata = {
  title: "Payment Details",
}

async function fetchData(id: string, authHeaders: AuthHeader): Promise<PaymentResponse | undefined> {
  try {
    const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/payments/${id}`, {
      headers: authHeaders
    })
    if (!rsp.ok) throw new Error(`${rsp.status} ${rsp.statusText}`)
    return (await rsp.json()) as PaymentResponse
  } catch (e: unknown) {
    console.error(e)
  }
}

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const authProvider = loadAuthProvider()
  const { id } = await params
  const authHeaders = await authProvider.getAuthHeader()
  const payment = await fetchData(id, authHeaders)

  if (!payment) {
    notFound()
  }

  return (
    <PaymentProvider payment={payment}>
      <PaymentPage payment={payment} />
    </PaymentProvider>
  )
}
