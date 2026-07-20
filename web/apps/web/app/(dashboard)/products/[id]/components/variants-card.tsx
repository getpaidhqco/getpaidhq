"use client"

import * as React from "react"
import { useCallback, useState } from "react"
import { Edit, Plus, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { useForm } from "react-hook-form"
import {
  useCreatePrice,
  useUpdatePrice,
  useDeletePrice,
  useCreateProductVariant,
  useUpdateVariant,
  useDeleteVariant,
  useMeters,
  variantResolvers,
  type CreatePriceFormValues,
  type CreateVariantFormValues,
} from "@getpaidhq/react-sdk"
import type { PriceResponse, VariantResponse } from "@getpaidhq/sdk"

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
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { type Column, DataTable } from "@/components/ui/data-table"
import { SectionHead } from "@/components/ui/section-head"
import { Badge } from "@/components/ui/badge"
import { H4, Muted } from "@/components/ui/typography"
import { priceAmountParts, priceBillingLabel } from "@/lib/price-display"
import { formatApiError } from "@/lib/api-error"
import { useEditProduct } from "@/app/(dashboard)/products/[id]/context/product-context"

import { PriceDrawer } from "./price-drawer"

export default function VariantsCard() {
  const { product, refreshProduct } = useEditProduct()
  const variants = (product.variants ?? []) as VariantResponse[]

  // Price drawer / delete state — activeVariantId scopes a new price to a variant.
  const [priceOpen, setPriceOpen] = useState(false)
  const [editingPrice, setEditingPrice] = useState<PriceResponse | undefined>()
  const [activeVariantId, setActiveVariantId] = useState<string | undefined>()
  const [deletePriceOpen, setDeletePriceOpen] = useState(false)
  const [deletingPrice, setDeletingPrice] = useState<PriceResponse | undefined>()

  // Variant add/rename/delete state.
  const [variantDialogOpen, setVariantDialogOpen] = useState(false)
  const [editingVariant, setEditingVariant] = useState<VariantResponse | undefined>()
  const [deleteVariantOpen, setDeleteVariantOpen] = useState(false)
  const [deletingVariant, setDeletingVariant] = useState<VariantResponse | undefined>()

  const createPrice = useCreatePrice()
  const updatePrice = useUpdatePrice(editingPrice?.id ?? "")
  const deletePrice = useDeletePrice()
  const createVariant = useCreateProductVariant(product.id)
  const updateVariant = useUpdateVariant(editingVariant?.id ?? "")
  const deleteVariant = useDeleteVariant()

  // Meters, to label metered prices by their meter rather than a blank cell.
  const { data: metersData } = useMeters()
  const meterName = useCallback(
    (id: string): string | undefined => {
      const list = (Array.isArray(metersData) ? metersData : (metersData?.data ?? [])) as Array<{
        id: string
        name?: string
        code?: string
      }>
      const m = list.find((x) => x.id === id)
      return m?.name || m?.code
    },
    [metersData],
  )

  const variantForm = useForm<CreateVariantFormValues>({
    resolver: variantResolvers.create,
    defaultValues: { name: "" },
  })

  // ---- price handlers ----
  const onAddPrice = useCallback((variantId: string) => {
    setEditingPrice(undefined)
    setActiveVariantId(variantId)
    setPriceOpen(true)
  }, [])

  const onEditPrice = useCallback((price: PriceResponse) => {
    setEditingPrice(price)
    setPriceOpen(true)
  }, [])

  const onPriceSubmit = useCallback(
    async (price: CreatePriceFormValues) => {
      try {
        if (editingPrice) {
          await updatePrice.mutateAsync(price)
          toast.success("Price updated")
        } else {
          await createPrice.mutateAsync({ ...price, variant_id: activeVariantId })
          toast.success("Price created")
        }
        setPriceOpen(false)
        await refreshProduct()
      } catch (error) {
        const { message, description } = formatApiError(error)
        toast.error(message, description ? { description } : undefined)
        throw error
      }
    },
    [editingPrice, updatePrice, createPrice, activeVariantId, refreshProduct],
  )

  const onConfirmDeletePrice = useCallback(() => {
    if (!deletingPrice) return
    deletePrice.mutate(deletingPrice.id, {
      onSuccess: async () => {
        toast.success("Price deleted")
        setDeletePriceOpen(false)
        await refreshProduct()
      },
      onError: (error: Error) => toast.error(error.message || "Failed to delete price"),
    })
  }, [deletingPrice, deletePrice, refreshProduct])

  // ---- variant handlers ----
  const onAddVariant = useCallback(() => {
    setEditingVariant(undefined)
    variantForm.reset({ name: "" })
    setVariantDialogOpen(true)
  }, [variantForm])

  const onRenameVariant = useCallback((v: VariantResponse) => {
    setEditingVariant(v)
    variantForm.reset({ name: v.name })
    setVariantDialogOpen(true)
  }, [variantForm])

  const onVariantSubmit = useCallback(
    async (values: CreateVariantFormValues) => {
      try {
        if (editingVariant) {
          await updateVariant.mutateAsync(values)
          toast.success("Variant updated")
        } else {
          await createVariant.mutateAsync(values)
          toast.success("Variant added")
        }
        setVariantDialogOpen(false)
        await refreshProduct()
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to save variant")
      }
    },
    [editingVariant, updateVariant, createVariant, refreshProduct],
  )

  const onConfirmDeleteVariant = useCallback(() => {
    if (!deletingVariant) return
    deleteVariant.mutate(deletingVariant.id, {
      onSuccess: async () => {
        toast.success("Variant deleted")
        setDeleteVariantOpen(false)
        await refreshProduct()
      },
      onError: (error: Error) => toast.error(error.message || "Failed to delete variant"),
    })
  }, [deletingVariant, deleteVariant, refreshProduct])

  const priceColumns = useCallback(
    (): Column<PriceResponse>[] => [
      {
        key: "label",
        header: "Label",
        render: (p) => {
          // Metered prices are often unlabelled — fall back to the meter's name.
          const fallback = p.billable_metric_id
            ? meterName(p.billable_metric_id) || "Metered usage"
            : "—"
          return (
            <span className="text-sm font-medium text-foreground">{p.label || fallback}</span>
          )
        },
      },
      {
        key: "billing",
        header: "Billing",
        render: (p) => (
          <div className="flex items-center gap-1.5">
            <span className="text-sm text-muted-foreground">{priceBillingLabel(p)}</span>
            {p.billable_metric_id ? <Badge variant="info">Metered</Badge> : null}
          </div>
        ),
      },
      {
        key: "amount",
        header: "Price",
        align: "right",
        render: (p) => {
          const { main, detail } = priceAmountParts(p)
          return (
            <div className="flex flex-col items-end">
              <span className="tabular font-medium text-foreground">{main}</span>
              {detail ? <span className="text-xs text-muted-foreground">{detail}</span> : null}
            </div>
          )
        },
      },
      {
        key: "actions",
        header: "",
        align: "right",
        width: "40px",
        render: (p) => (
          <div className="flex items-center justify-end">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(e) => {
                // The row itself opens the edit drawer — don't let delete bubble into it.
                e.stopPropagation()
                setDeletingPrice(p)
                setDeletePriceOpen(true)
              }}
              aria-label="Delete price"
            >
              <Trash2 className="size-3.5 text-destructive" />
            </Button>
          </div>
        ),
      },
    ],
    [meterName],
  )

  return (
    <>
      <PriceDrawer open={priceOpen} onClose={() => setPriceOpen(false)} price={editingPrice} onSubmit={onPriceSubmit} />

      {/* Add / rename variant dialog */}
      <Dialog open={variantDialogOpen} onOpenChange={setVariantDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingVariant ? "Rename variant" : "Add variant"}</DialogTitle>
          </DialogHeader>
          <Form {...variantForm}>
            <form onSubmit={variantForm.handleSubmit(onVariantSubmit)} className="space-y-5">
              <FormField
                control={variantForm.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Variant name</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g. Standard, Pro, Annual" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setVariantDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={createVariant.isPending || updateVariant.isPending}>
                  {editingVariant ? "Save" : "Add variant"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      {/* Delete price confirm */}
      <AlertDialog open={deletePriceOpen} onOpenChange={setDeletePriceOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete price?</AlertDialogTitle>
            <AlertDialogDescription>
              Deleting <strong>{deletingPrice?.label || "this price"}</strong> removes it from this
              variant. Existing subscriptions on it are not affected.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={onConfirmDeletePrice}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Delete price
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete variant confirm */}
      <AlertDialog open={deleteVariantOpen} onOpenChange={setDeleteVariantOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete variant?</AlertDialogTitle>
            <AlertDialogDescription>
              Deleting <strong>{deletingVariant?.name || "this variant"}</strong> also removes its{" "}
              {deletingVariant?.prices?.length ?? 0} price(s). This can&apos;t be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={onConfirmDeleteVariant}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              Delete variant
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <section>
        <SectionHead
          title="Variants & pricing"
          action={
            <Button size="sm" variant="outline" onClick={onAddVariant}>
              <Plus data-icon="inline-start" />
              Add variant
            </Button>
          }
        />

        {variants.length === 0 ? (
          <div className="py-6 text-center text-sm text-muted-foreground">
            No variants yet — add one to start pricing this product.
          </div>
        ) : (
          <div className="space-y-8">
            {variants.map((v) => (
              <div key={v.id} className="rounded-lg border border-border p-4">
                <div className="mb-3 flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <H4>{v.name}</H4>
                    <Muted className="text-xs">
                      {v.prices?.length ?? 0} price{(v.prices?.length ?? 0) === 1 ? "" : "s"}
                    </Muted>
                  </div>
                  <div className="flex items-center gap-0.5">
                    <Button variant="ghost" size="icon-sm" onClick={() => onRenameVariant(v)} aria-label="Rename variant">
                      <Edit className="size-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => {
                        setDeletingVariant(v)
                        setDeleteVariantOpen(true)
                      }}
                      aria-label="Delete variant"
                    >
                      <Trash2 className="size-3.5 text-destructive" />
                    </Button>
                  </div>
                </div>
                <DataTable
                  columns={priceColumns()}
                  rows={v.prices ?? []}
                  onRowClick={onEditPrice}
                  empty={<div className="py-4 text-center text-sm text-muted-foreground">No prices on this variant yet.</div>}
                />
                <div className="mt-3">
                  <Button size="sm" variant="ghost" onClick={() => onAddPrice(v.id)}>
                    <Plus data-icon="inline-start" />
                    Add price
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </>
  )
}
