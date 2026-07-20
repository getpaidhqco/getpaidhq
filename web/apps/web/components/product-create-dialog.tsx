"use client"

import * as React from "react"
import { useState } from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateProduct,
  productResolvers,
  PRICE_CATEGORIES,
  type CreateProductFormValues,
} from "@getpaidhq/react-sdk"
import type { CreateProductRequest, ProductResponse } from "@getpaidhq/sdk"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { SectionHead } from "@/components/ui/section-head"
import { currencyToCents } from "@/lib/currency"

// "one_time" -> "One time"
const humanize = (v: string) => v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ")

const RECURRING_CATEGORIES = ["subscription"]

interface ProductCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onProductCreated: (product: ProductResponse) => void
}

const CURRENCIES = [
  { value: "USD", label: "USD — US Dollar" },
  { value: "EUR", label: "EUR — Euro" },
  { value: "GBP", label: "GBP — British Pound" },
  { value: "ZAR", label: "ZAR — South African Rand" },
] as const

export function ProductCreateDialog({
  open,
  onOpenChange,
  onProductCreated,
}: ProductCreateDialogProps) {
  const form = useForm<CreateProductFormValues>({
    resolver: productResolvers.create,
    defaultValues: {
      name: "",
      description: "",
      variants: [{ name: "Default" }],
    },
  })

  // Local default-price state (the product resolver doesn't model prices).
  const [price, setPrice] = useState({
    label: "Standard",
    category: "subscription",
    currency: "USD",
    unit_price: 0,
  })

  const createProduct = useCreateProduct({
    onSuccess: (newProduct: ProductResponse) => {
      toast.success("Product created")
      onProductCreated(newProduct)
      onOpenChange(false)
      form.reset()
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create product", { duration: 8000 })
    },
  })

  const onSubmit = (data: CreateProductFormValues) => {
    const isRecurring = RECURRING_CATEGORIES.includes(price.category)
    const payload: CreateProductRequest = {
      name: data.name,
      description: data.description,
      metadata: data.metadata,
      variants: [
        {
          name: data.variants[0]?.name || "Default",
          prices: [
            {
              label: price.label,
              category: price.category,
              scheme: "fixed",
              currency: price.currency,
              unit_price: price.unit_price,
              ...(isRecurring
                ? { billing_interval: "month", billing_interval_qty: 1 }
                : {}),
            },
          ],
        },
      ],
    }
    createProduct.mutate(payload)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>New product</DialogTitle>
          <DialogDescription>
            Create a product with its default price. You can add variants and
            additional prices later.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            <div className="space-y-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g. Pro plan" {...field} value={field.value ?? ""} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Description</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder="What customers get when they buy this."
                        size="sm"
                        {...field}
                        value={field.value ?? ""}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className="space-y-4">
              <SectionHead
                size="sm"
                title="Default price"
                subtitle="The first price attached to this product. You can add more later."
              />

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <FormItem>
                  <FormLabel>Label</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Standard"
                      value={price.label}
                      onChange={(e) => setPrice((p) => ({ ...p, label: e.target.value }))}
                    />
                  </FormControl>
                </FormItem>

                <FormItem>
                  <FormLabel>Category</FormLabel>
                  <Select
                    value={price.category}
                    onValueChange={(v) => setPrice((p) => ({ ...p, category: v }))}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select category" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {PRICE_CATEGORIES.map((c) => (
                        <SelectItem key={c} value={c}>
                          {humanize(c)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormItem>

                <FormItem>
                  <FormLabel>Currency</FormLabel>
                  <Select
                    value={price.currency}
                    onValueChange={(v) => setPrice((p) => ({ ...p, currency: v }))}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select currency" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {CURRENCIES.map((c) => (
                        <SelectItem key={c.value} value={c.value}>
                          {c.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormItem>

                <FormItem>
                  <FormLabel>Amount</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min="0"
                      step="0.01"
                      inputMode="decimal"
                      placeholder="0.00"
                      className="tabular-nums"
                      onChange={(e) =>
                        setPrice((p) => ({
                          ...p,
                          unit_price: currencyToCents(parseFloat(e.target.value) || 0),
                        }))
                      }
                    />
                  </FormControl>
                </FormItem>
              </div>
            </div>

            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createProduct.isPending}>
                {createProduct.isPending ? "Creating…" : "Create product"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
