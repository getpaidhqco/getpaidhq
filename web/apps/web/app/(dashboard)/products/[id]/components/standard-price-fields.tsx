"use client"

import React from "react"

import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import { Switch } from "@/components/ui/switch"

import {
  BillingCadenceField,
  InlineCurrencySelect,
  MoneyInput,
  type PriceFieldsProps,
} from "./price-form-shared"

// Category is HOW a non-metered price charges. Server PriceCategory: one_time |
// subscription | free | variable. (Metered billing is not a category — see the
// metered toggle in the drawer.)
const priceCategoryOptions = [
  { value: "one_time", label: "One Time" },
  { value: "subscription", label: "Subscription" },
  { value: "free", label: "Free" },
  { value: "variable", label: "Variable (customer chooses)" },
] as const

/**
 * Fields for a non-metered price: the category, then the amount that category
 * needs (a plain price, min/suggested for variable, nothing for free), plus the
 * cadence and payment count for subscriptions. The scheme is always "fixed", so
 * no scheme picker appears.
 */
export function StandardPriceFields({ form, symbol }: PriceFieldsProps) {
  const category = form.watch("category")
  // "Payment plan" is purely a UI fork over cycles: 0 = charge until cancelled,
  // > 0 = stop after that many payments. Deriving it from the value (instead of
  // separate state) makes edit mode initialise itself.
  const cycles = form.watch("cycles")
  const paymentPlan = (cycles ?? 0) > 0

  return (
    <>
      <FormField
        control={form.control}
        name="category"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Category</FormLabel>
            <Select value={field.value} onValueChange={field.onChange}>
              <FormControl>
                <SelectTrigger>
                  <SelectValue placeholder="Select category" />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                {priceCategoryOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <FormMessage />
          </FormItem>
        )}
      />

      {category === "free" ? (
        <FormDescription>Free prices charge nothing — no amount to configure.</FormDescription>
      ) : category === "variable" ? (
        <>
          <FormField
            control={form.control}
            name="min_price"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Minimum price</FormLabel>
                <div className="flex gap-2">
                  <div className="flex-1">
                    <MoneyInput symbol={symbol} value={field.value} onChange={field.onChange} />
                  </div>
                  <InlineCurrencySelect form={form} />
                </div>
                <FormDescription>
                  The least a customer may pay. Leave empty for no minimum.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="suggested_price"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Suggested price</FormLabel>
                <MoneyInput symbol={symbol} value={field.value} onChange={field.onChange} />
                <FormDescription>Pre-filled at checkout; the customer can change it.</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </>
      ) : (
        <FormField
          control={form.control}
          name="unit_price"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Price</FormLabel>
              <div className="flex gap-2">
                <div className="flex-1">
                  <MoneyInput
                    symbol={symbol}
                    value={field.value}
                    onChange={field.onChange}
                    emptyAsZero
                  />
                </div>
                <InlineCurrencySelect form={form} />
              </div>
              <FormMessage />
            </FormItem>
          )}
        />
      )}

      {category === "subscription" ? (
        <>
          <BillingCadenceField form={form} />

          <FormItem className="flex items-start gap-3">
            <Switch
              checked={paymentPlan}
              onCheckedChange={(on) => form.setValue("cycles", on ? 12 : 0)}
              aria-label="Payment plan"
            />
            <div className="space-y-1">
              <FormLabel>Payment plan</FormLabel>
              <FormDescription>
                {paymentPlan
                  ? "Ends automatically after the number of payments below."
                  : "Charges until the subscription is cancelled."}
              </FormDescription>
            </div>
          </FormItem>

          {paymentPlan ? (
            <FormField
              control={form.control}
              name="cycles"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Number of payments</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={1}
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      value={field.value ?? ""}
                      onChange={(e) =>
                        field.onChange(e.target.value === "" ? undefined : e.target.valueAsNumber)
                      }
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : null}
        </>
      ) : null}
    </>
  )
}
