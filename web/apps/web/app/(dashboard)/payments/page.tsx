import { Metadata } from "next"

import { fetchData } from "./data/data"
import { loadAuthProvider } from "@getpaidhq/auth/server"
import { PageHeader } from "@/components/ui/page-header"
import { columns } from "./components/columns"
import { DataTable } from "./components/data-table"

export const metadata: Metadata = {
  title: "Payments",
  description: "View and manage payment transactions.",
}

export default async function PaymentsPage() {
  const authProvider = loadAuthProvider();
  const data = await fetchData({ pageIndex: 0, pageSize: 10 }, await authProvider.getAuthHeader());

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader title="Payments" />
      <DataTable data={data.data} columns={columns} />
    </div>
  )
}
