import {Metadata} from "next"
import * as React from "react";

import {fetchData} from "./components/data";
import {loadAuthProvider} from "@getpaidhq/auth/server";
import {PageHeader} from "@/components/ui/page-header";
import {DataTable} from "@/app/(dashboard)/invoices/components/data-table";
import {columns} from "@/app/(dashboard)/invoices/components/columns";

export const metadata: Metadata = {
  title: "Invoices",
}

export default async function InvoicePage() {
  const authProvider = loadAuthProvider();
  const invoices = await fetchData(await authProvider.getAuthHeader(), {page: 0, limit: 10});

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader title="Invoices" />
      <DataTable data={invoices.data} columns={columns} />
    </div>
  )
}
