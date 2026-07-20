"use client"

import React, { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import {
  priceResolvers,
  schemeRequiresTiers,
  schemeSupportsUnitCount,
  type CreatePriceFormValues,
} from "@getpaidhq/react-sdk"
import type { PriceResponse } from "@getpaidhq/sdk"
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
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { centsToCurrency, currencyToCents } from "@/lib/currency"

import { MeteredPriceFields } from "./metered-price-fields"
import { StandardPriceFields } from "./standard-price-fields"

const CREATE_DEFAULTS: Partial<CreatePriceFormValues> = {
  variant_id: "pending",
  label: "",
  category: "subscription",
  scheme: "fixed",
  currency: "USD",
  billing_interval: "month",
  billing_interval_qty: 1,
  cycles: 0,
  unit_price: 1000,
  unit_count: 1,
}

type PriceDrawerProps = {
  open: boolean
  onClose: () => void
  price?: PriceResponse
  onSubmit: (v: CreatePriceFormValues) => Promise<void>
}

export function PriceDrawer({ open, onClose, price, onSubmit }: PriceDrawerProps) {
  const [symbol, setSymbol] = useState<string>("")
  // Metered (usage-based) vs standard is the top-level fork of the form: it isn't
  // a server field — a price is metered iff it carries a billable_metric_id — but
  // it decides which field set renders and how the payload is normalised.
  const [metered, setMetered] = useState(false)

  // variant_id is supplied by the parent on submit (a default variant may be
  // created on the fly), so it's not edited here — seed with a placeholder that
  // satisfies the resolver and is overwritten before the API call.
  const form = useForm<CreatePriceFormValues>({
    resolver: priceResolvers.create,
    defaultValues: { ...CREATE_DEFAULTS, variant_id: price?.variant_id || "pending" },
  })

  const currency = form.watch("currency")

  // Reset the form whenever the drawer opens: to the edited price's values, or
  // back to create defaults — otherwise "Add price" after an edit would silently
  // reuse the previous price's values (meter included).
  useEffect(() => {
    if (!open) return
    if (!price) {
      form.reset(CREATE_DEFAULTS as CreatePriceFormValues)
      setMetered(false)
      return
    }
    form.reset({
      variant_id: price.variant_id,
      label: price.label,
      category: price.category,
      scheme: price.scheme || "fixed",
      currency: price.currency,
      unit_price: price.unit_price,
      unit_count: price.unit_count || 1,
      min_price: price.min_price,
      suggested_price: price.suggested_price,
      billing_interval: price.billing_interval,
      billing_interval_qty: price.billing_interval_qty,
      cycles: price.cycles,
      billable_metric_id: price.billable_metric_id,
      prorate_on_increase: price.prorate_on_increase,
      credit_on_decrease: price.credit_on_decrease,
      // Server tiers are cents; the editor works in dollars. to_value "0" = unbounded.
      tiers: (price.tiers ?? []).map((t) => ({
        from_value: t.from_value ?? "",
        to_value: t.to_value && t.to_value !== "0" ? t.to_value : "",
        per_unit_amount: t.per_unit_amount
          ? String(centsToCurrency(Number(t.per_unit_amount)))
          : "",
        flat_amount: t.flat_amount ? String(centsToCurrency(t.flat_amount)) : "0",
      })),
    } as CreatePriceFormValues)
    setMetered(Boolean(price.billable_metric_id))
  }, [price, open, form])

  useEffect(() => {
    setSymbol(getSymbolFromCurrency(currency))
  }, [currency])

  const onMeteredChange = (on: boolean) => {
    setMetered(on)
    if (on) {
      // A metered price is always a subscription.
      form.setValue("category", "subscription")
    } else {
      // Standard prices are plain fixed: drop everything metering-specific.
      form.setValue("billable_metric_id", undefined)
      form.setValue("scheme", "fixed")
      form.setValue("unit_count", 1)
      form.setValue("prorate_on_increase", undefined)
      form.setValue("credit_on_decrease", undefined)
    }
  }

  const onFormSubmit = async (data: CreatePriceFormValues) => {
    if (metered && !data.billable_metric_id) {
      form.setError("billable_metric_id", {
        message: "Choose the meter this price bills from",
      })
      return
    }

    const freeOrVariable = data.category === "free" || data.category === "variable"
    // The server treats ANY price with an interval as recurring, so the hidden
    // interval defaults must be scrubbed for one-time/free/variable prices.
    const recurring = metered || data.category === "subscription"

    const payload = {
      ...data,
      category: metered ? "subscription" : data.category,
      // The scheme picker only exists for metered prices; standard prices are fixed.
      scheme: metered ? data.scheme : "fixed",
      billable_metric_id: metered ? data.billable_metric_id : undefined,
      billing_interval: recurring ? data.billing_interval : "none",
      billing_interval_qty: recurring ? data.billing_interval_qty : undefined,
      // Metered usage bills until the subscription ends — no payment count.
      cycles: !metered && data.category === "subscription" ? data.cycles : undefined,
      // Free charges nothing and variable amounts come from min/suggested.
      unit_price: freeOrVariable ? 0 : data.unit_price,
      // unit_count belongs to the metered fixed/package schemes only.
      unit_count: metered && schemeSupportsUnitCount(data.scheme) ? data.unit_count : undefined,
      min_price: data.category === "variable" ? data.min_price : undefined,
      suggested_price: data.category === "variable" ? data.suggested_price : undefined,
      prorate_on_increase: metered ? data.prorate_on_increase : undefined,
      credit_on_decrease: metered ? data.credit_on_decrease : undefined,
      // The server tier contract is cents: per_unit_amount is a cents string
      // (sub-cent allowed), flat_amount is whole cents. The editor collects
      // dollars, so convert here. An empty to_value is the unbounded last tier.
      tiers:
        metered && schemeRequiresTiers(data.scheme)
          ? (data.tiers ?? []).map((t) => ({
              from_value: t.from_value?.trim() || "0",
              to_value: t.to_value?.trim() || "0",
              // Keep sub-cent precision (e.g. $0.0001/unit = 0.01¢) — don't round
              // to whole cents like flat_amount.
              per_unit_amount: String(Number((Number(t.per_unit_amount || 0) * 100).toFixed(6))),
              flat_amount: currencyToCents(Number(t.flat_amount || 0)),
            }))
          : undefined,
    }
    await onSubmit(payload as CreatePriceFormValues)
    form.reset(CREATE_DEFAULTS as CreatePriceFormValues)
    setMetered(false)
  }

  return (
    <Sheet open={open} onOpenChange={onClose}>
      <SheetContent className="w-[600px] sm:max-w-[600px]">
        <SheetHeader>
          <SheetTitle>{price ? "Edit" : "Create"} Price</SheetTitle>
          <SheetDescription>Set up the pricing for this variant.</SheetDescription>
        </SheetHeader>

        <Form {...form}>
          <form className="grid flex-1 auto-rows-min gap-6 overflow-y-auto px-4">
            <FormField
              control={form.control}
              name="label"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Label</FormLabel>
                  <FormControl>
                    <Input {...field} value={field.value ?? ""} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormItem className="flex items-start gap-3">
              <Switch
                checked={metered}
                onCheckedChange={onMeteredChange}
                aria-label="Usage-based pricing"
              />
              <div className="space-y-1">
                <FormLabel>Usage-based (metered)</FormLabel>
                <FormDescription>
                  {metered
                    ? "Bills a subscription from metered usage each period."
                    : "Charges a set price — turn on to bill from a usage meter instead."}
                </FormDescription>
              </div>
            </FormItem>

            {metered ? (
              <MeteredPriceFields form={form} symbol={symbol} />
            ) : (
              <StandardPriceFields form={form} symbol={symbol} />
            )}
          </form>
        </Form>

        <SheetFooter className="mt-8 justify-end">
          <SheetClose asChild>
            <Button type="button" variant="ghost">
              Go back
            </Button>
          </SheetClose>
          <Button
            type="button"
            disabled={form.formState.isSubmitting}
            onClick={form.handleSubmit(onFormSubmit)}
            className="ml-2"
          >
            {form.formState.isSubmitting ? "Saving…" : "Save"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
