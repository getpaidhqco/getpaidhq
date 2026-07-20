"use client"

import * as React from "react";
import {useState} from "react";
import {PlusIcon} from "lucide-react";
import {useRouter} from "next/navigation";

import {ProductCreateDialog} from "@/components/product-create-dialog";
import {Button} from "@/components/ui/button";
import {PageHeader} from "@/components/ui/page-header";
import {DataTable} from "@/app/(dashboard)/products/components/data-table";
import {columns} from "@/app/(dashboard)/products/components/columns";
import type {ProductResponse} from "@getpaidhq/sdk";

export default function Page({products}: { products: ProductResponse[] }) {
  const router = useRouter()
  const [dialogOpen, setDialogOpen] = useState(false)

  const onProductCreated = (product: ProductResponse) => {
    router.push(`/products/${product.id}`);
  }

  return (
    <>
      <ProductCreateDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onProductCreated={onProductCreated}
      />

      <div className="flex flex-1 flex-col gap-8">
        <PageHeader
          title="Products"
          actions={
            <Button size="sm" onClick={() => setDialogOpen(true)}>
              <PlusIcon data-icon="inline-start" />
              New product
            </Button>
          }
        />
        <DataTable data={products} columns={columns} />
      </div>
    </>
  )
}
