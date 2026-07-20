"use client"

import { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import { useRouter } from "next/navigation"
import {
  useCreateProduct,
  productResolvers,
  PRICE_CATEGORIES,
  BILLING_INTERVALS,
  type CreateProductFormValues,
} from "@getpaidhq/react-sdk"
import type { CreateProductRequest } from "@getpaidhq/sdk"
import { getSymbolFromCurrency } from "country-data-list"

import { Button } from "@/components/ui/button"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { FormLayout, FormSection, FormActions } from "@/components/ui/form-section"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { CurrencySelect } from "@/components/currency-select"
import { currencyToCents } from "@/lib/currency"

// "one_time" -> "One time"
const humanize = (v: string) => v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ")

// Categories that bill on a recurring interval.
const RECURRING_CATEGORIES = ["subscription"]

export default function ProductForm() {
  const router = useRouter()
  const [symbol, setSymbol] = useState("")

  // Validation comes from the react-sdk resolver (mirrors the server contract).
  // The form collects the product plus a single default price; on submit we build
  // the API payload with a default variant that carries that price.
  const form = useForm<CreateProductFormValues>({
    resolver: productResolvers.create,
    defaultValues: {
      name: "",
      description: "",
      variants: [{ name: "Default" }],
    },
  })

  // Local price state (the product resolver doesn't model prices).
  const [price, setPrice] = useState({
    label: "Monthly",
    category: "subscription",
    currency: "USD",
    unit_price: 0,
    billing_interval: "month",
    billing_interval_qty: 1,
  })

  useEffect(() => {
    setSymbol(getSymbolFromCurrency(price.currency))
  }, [price.currency])

  const isRecurring = RECURRING_CATEGORIES.includes(price.category)

  const createProduct = useCreateProduct({
    onSuccess: (data: { id: string }) => {
      toast.success("Product created successfully")
      router.push(`/products/${data.id}`)
    },
    onError: (error: Error) => {
      toast.error("Failed to create product", {
        description: error.message || "An unknown error occurred",
      })
    },
  })

  function onSubmit(values: CreateProductFormValues) {
    const payload: CreateProductRequest = {
      name: values.name,
      description: values.description,
      metadata: values.metadata,
      variants: [
        {
          name: values.variants[0]?.name || "Default",
          prices: [
            {
              label: price.label,
              category: price.category,
              scheme: "fixed",
              currency: price.currency,
              unit_price: price.unit_price,
              ...(isRecurring
                ? {
                    billing_interval: price.billing_interval,
                    billing_interval_qty: price.billing_interval_qty,
                  }
                : {}),
            },
          ],
        },
      ],
    }
    createProduct.mutate(payload)
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <FormLayout>
          <FormSection
            title="Product details"
            description="What you're selling and how it's described."
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="Pro plan" {...field} value={field.value ?? ""} />
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
                      rows={4}
                      {...field}
                      value={field.value ?? ""}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="variants.0.name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Variant name</FormLabel>
                  <FormControl>
                    <Input placeholder="Default" {...field} value={field.value ?? ""} />
                  </FormControl>
                  <FormDescription>
                    The initial variant created for this product.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </FormSection>

          <FormSection
            title="Pricing"
            description="The first price attached to this product's variant."
          >
            <FormItem>
              <FormLabel>Label</FormLabel>
              <FormControl>
                <Input
                  placeholder="Monthly"
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
              <CurrencySelect
                value={price.currency}
                onValueChange={(v) => setPrice((p) => ({ ...p, currency: v }))}
                placeholder="Currency"
                disabled={false}
              />
            </FormItem>

            <FormItem>
              <FormLabel>Price</FormLabel>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
                  {symbol}
                </span>
                <Input
                  type="number"
                  step="0.01"
                  min="0"
                  className="pl-12"
                  placeholder="0.00"
                  onChange={(e) =>
                    setPrice((p) => ({ ...p, unit_price: currencyToCents(Number(e.target.value)) }))
                  }
                />
              </div>
              <FormDescription>Amount charged in the selected currency.</FormDescription>
            </FormItem>

            {isRecurring ? (
              <>
                <FormItem>
                  <FormLabel>Billing interval</FormLabel>
                  <Select
                    value={price.billing_interval}
                    onValueChange={(v) => setPrice((p) => ({ ...p, billing_interval: v }))}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select interval" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {BILLING_INTERVALS.map((i) => (
                        <SelectItem key={i} value={i}>
                          {humanize(i)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormItem>

                <FormItem>
                  <FormLabel>Billing frequency</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={1}
                      value={price.billing_interval_qty}
                      onChange={(e) =>
                        setPrice((p) => ({
                          ...p,
                          billing_interval_qty: Number(e.target.value),
                        }))
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    Charge every {price.billing_interval_qty} {price.billing_interval}
                    {price.billing_interval_qty === 1 ? "" : "s"}.
                  </FormDescription>
                </FormItem>
              </>
            ) : null}
          </FormSection>

          <FormActions>
            <Button type="button" variant="outline" onClick={() => router.push("/products")}>
              Cancel
            </Button>
            <Button type="submit" disabled={createProduct.isPending}>
              {createProduct.isPending ? "Creating…" : "Create Product"}
            </Button>
          </FormActions>
        </FormLayout>
      </form>
    </Form>
  )
}
