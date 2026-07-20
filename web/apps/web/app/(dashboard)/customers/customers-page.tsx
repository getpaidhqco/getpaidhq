"use client"

import * as React from "react";
import {useState} from "react";
import {PlusIcon} from "lucide-react";
import {useRouter} from "next/navigation";

import {CustomerDialog} from "@/components/customer-dialog";
import {DataTable} from "@/app/(dashboard)/customers/components/data-table";
import {columns} from "@/app/(dashboard)/customers/components/columns";
import type {CustomerResponse} from "@getpaidhq/sdk";
import {Button} from "@/components/ui/button";
import {PageHeader} from "@/components/ui/page-header";


export default function Page({customers}: { customers: CustomerResponse[] }) {
  const router = useRouter()
  const [dialogOpen, setDialogOpen] = useState(false)

  const onCustomerSaved = (customer: CustomerResponse) => {
    router.push(`/customers/${customer.id}`);
  }

  return (
    <>
      <CustomerDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onCustomerSaved={onCustomerSaved}
      />

      <div className="flex flex-1 flex-col gap-8">
        <PageHeader
          title="Customers"
          actions={
            <Button size="sm" onClick={() => setDialogOpen(true)}>
              <PlusIcon data-icon="inline-start" /> New customer
            </Button>
          }
        />
        <DataTable data={customers} columns={columns} />
      </div>
    </>
  );
}
