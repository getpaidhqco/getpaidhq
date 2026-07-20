"use client"

import * as React from "react"
import { useEffect, useState } from "react"
import { Archive, ArchiveRestore, Copy } from "lucide-react"
import { format } from "date-fns"
import { toast } from "sonner"
import { useArchiveProduct, useUnarchiveProduct } from "@getpaidhq/react-sdk"

import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { KpiRow } from "@/components/ui/kpi-row"
import { PageHeader } from "@/components/ui/page-header"
import { StatusTag } from "@/components/ui/status-tag"
import { useEditProduct } from "@/app/(dashboard)/products/[id]/context/product-context"
import { useBreadcrumb } from "@/context/breadcrumb-context"
import { formatCurrency } from "@/lib/currency"
import type { PriceResponse } from "@getpaidhq/sdk"

import { DetailsCard } from "./details-card"
import MetadataCard from "./metadata-card"
import VariantsCard from "./variants-card"

function formatPrice(p: PriceResponse | undefined) {
  if (!p) return "—"
  return formatCurrency(p.currency, p.unit_price)
}

export function ProductEditPage() {
  const { product, isSubscription, refreshProduct } = useEditProduct()
  const { setItems } = useBreadcrumb()
  const [showArchiveDialog, setShowArchiveDialog] = useState(false)

  const isArchived = product.status === "archived"

  const archiveProduct = useArchiveProduct({
    onSuccess: () => {
      toast.success(`Product "${product.name}" archived`)
      setShowArchiveDialog(false)
      void refreshProduct()
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Please try again."
      toast.error("Failed to archive product", { description: message })
    },
  })

  const unarchiveProduct = useUnarchiveProduct({
    onSuccess: () => {
      toast.success(`Product "${product.name}" restored`)
      void refreshProduct()
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Please try again."
      toast.error("Failed to restore product", { description: message })
    },
  })

  useEffect(() => {
    setItems([
      { label: "Products", href: "/products" },
      { label: product.name },
    ])
    return () => setItems(null)
  }, [product.name, setItems])

  const variant = product.variants?.[0]
  const prices = variant?.prices ?? []
  const defaultPrice = prices[0]
  const sub = isSubscription()

  const copyId = () => {
    navigator.clipboard.writeText(product.id)
    toast.success("Product ID copied")
  }

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader
        title={
          <span className="inline-flex items-center gap-3">
            {product.name}
            {isArchived ? <StatusTag tone="neutral">Archived</StatusTag> : null}
          </span>
        }
        description={product.description || undefined}
        actions={
          <>
            <Button variant="outline" size="sm" onClick={copyId}>
              <Copy data-icon="inline-start" />
              Copy ID
            </Button>
            {isArchived ? (
              <Button
                variant="ghost"
                size="sm"
                disabled={unarchiveProduct.isPending}
                onClick={() => unarchiveProduct.mutate(product.id)}
              >
                <ArchiveRestore data-icon="inline-start" />
                {unarchiveProduct.isPending ? "Restoring…" : "Unarchive"}
              </Button>
            ) : (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowArchiveDialog(true)}
              >
                <Archive data-icon="inline-start" />
                Archive
              </Button>
            )}
          </>
        }
      />

      <AlertDialog open={showArchiveDialog} onOpenChange={setShowArchiveDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Archive product?</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="font-medium text-foreground">{product.name}</span> will be hidden
              from listings and can no longer be sold. Existing orders, subscriptions and history
              are kept, and active subscriptions keep billing. You can unarchive it at any time.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={archiveProduct.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={archiveProduct.isPending}
              onClick={(event) => {
                event.preventDefault()
                archiveProduct.mutate(product.id)
              }}
            >
              {archiveProduct.isPending ? "Archiving…" : "Archive"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <div className="space-y-10 lg:col-span-2">
          <DetailsCard />
          <VariantsCard />
        </div>
        <div className="space-y-10">
          <MetadataCard />
        </div>
      </div>
    </div>
  )
}
