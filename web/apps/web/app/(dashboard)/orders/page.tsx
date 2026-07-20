import {Metadata} from "next"

import {columns} from "./components/columns"
import {DataTable} from "./components/data-table"
import {fetchData} from "./data/data";
import {loadAuthProvider} from "@getpaidhq/auth/server";
import {PageHeader} from "@/components/ui/page-header";

export const metadata: Metadata = {
  title: "Orders",
}


export default async function OrdersPage() {
  const authProvider = loadAuthProvider();
  const data = await fetchData({pageIndex: 0, pageSize: 10}, await authProvider.getAuthHeader());

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader title="Orders" />
      <DataTable data={data.data} columns={columns} />
    </div>
  )
}
