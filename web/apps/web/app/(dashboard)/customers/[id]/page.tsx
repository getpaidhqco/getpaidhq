import { Metadata } from "next"
import { notFound } from "next/navigation"
import type { CustomerResponse } from "@getpaidhq/sdk"
import { fetchData } from "./data"
import { loadAuthProvider } from "@getpaidhq/auth/server"
import { CustomerOverview } from "@/app/(dashboard)/customers/[id]/components/customer-overview"

export const metadata: Metadata = {
  title: "Customer Details",
}

export default async function CustomerPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  const authProvider = loadAuthProvider()
  const customer: CustomerResponse = await fetchData(id, await authProvider.getAuthHeader())

  if (!customer) {
    notFound()
  }

  return <CustomerOverview customerId={id} />
}
