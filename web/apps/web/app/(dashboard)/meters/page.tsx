import { Metadata } from "next"
import Link from "next/link";
import { PlusIcon } from "lucide-react";

import { fetchData } from "./components/data";
import { loadAuthProvider } from "@getpaidhq/auth/server";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";
import { DataTable } from "@/app/(dashboard)/meters/components/data-table";
import { columns } from "@/app/(dashboard)/meters/components/columns";

export const metadata: Metadata = {
  title: "Meters",
}

export default async function MetersPage() {
  const authProvider = loadAuthProvider();
  const meters = await fetchData(await authProvider.getAuthHeader(), { page: 0, limit: 10 });

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader
        title="Usage meters"
        actions={
          <Button size="sm" asChild>
            <Link href="/meters/create">
              <PlusIcon data-icon="inline-start" />
              New meter
            </Link>
          </Button>
        }
      />
      <DataTable data={meters.data} columns={columns} />
    </div>
  )
}
