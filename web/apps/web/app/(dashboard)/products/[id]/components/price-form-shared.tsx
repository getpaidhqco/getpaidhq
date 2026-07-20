"use client"

import React, { useState } from "react"
import type { UseFormReturn } from "react-hook-form"
import { BILLING_INTERVALS, type CreatePriceFormValues } from "@getpaidhq/react-sdk"

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
import { CurrencySelect } from "@/components/currency-select"
import { cn } from "@/lib/utils"
import { centsToCurrency, currencyToCents } from "@/lib/currency"

/** Props shared by the metered / standard price field components. */
export type PriceFieldsProps = {
  form: UseFormReturn<CreatePriceFormValues>
  symbol: string
}

// "month" -> "Month"
export const humanize = (v: string) => v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ")

type MoneyInputProps = {
  symbol: string
  /** Amount in cents; undefined = empty input. */
  value: number | undefined
  onChange: (cents: number | undefined) => void
  /** Treat an emptied input as 0 instead of undefined (for required amounts). */
  emptyAsZero?: boolean
}

/**
 * Currency input that edits dollars text over a cents value. The raw text is
 * local state so a half-typed decimal ("10.") isn't normalised away on every
 * keystroke; external value changes (form reset, switching to edit) re-seed it.
 */
export function MoneyInput({ symbol, value, onChange, emptyAsZero = false }: MoneyInputProps) {
  const [text, setText] = useState<string>(value != null ? String(centsToCurrency(value)) : "")
  // Re-seed the text when the value changed from outside (form reset / edit load)
  // rather than from typing — i.e. when the current text no longer parses to it.
  const [seenValue, setSeenValue] = useState(value)
  if (value !== seenValue) {
    setSeenValue(value)
    const parsed = text === "" ? (emptyAsZero ? 0 : undefined) : currencyToCents(Number(text))
    if (parsed !== value) setText(value != null ? String(centsToCurrency(value)) : "")
  }

  return (
    <div className="relative">
      <span className="z-10 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
        {symbol}
      </span>
      <FormControl>
        <Input
          type="number"
          min="0"
          step="0.01"
          placeholder="0.00"
          className={cn(symbol?.length > 2 ? "pl-12" : "pl-10")}
          value={text}
          onChange={(e) => {
            const t = e.target.value
            setText(t)
            onChange(t === "" ? (emptyAsZero ? 0 : undefined) : currencyToCents(Number(t)))
          }}
        />
      </FormControl>
    </div>
  )
}

/**
 * Compact currency picker (code only) for placing inline next to an amount
 * input, instead of as a standalone form row.
 */
export function InlineCurrencySelect({ form }: { form: UseFormReturn<CreatePriceFormValues> }) {
  return (
    <FormField
      control={form.control}
      name="currency"
      render={({ field }) => (
        <FormItem>
          <CurrencySelect
            variant="small"
            value={field.value}
            onValueChange={field.onChange}
            placeholder="USD"
          />
        </FormItem>
      )}
    />
  )
}

/**
 * The billing cadence on one line: "every [qty] [interval]", with a plain-words
 * summary underneath.
 */
export function BillingCadenceField({
  form,
  note,
}: {
  form: UseFormReturn<CreatePriceFormValues>
  note?: string
}) {
  const qty = form.watch("billing_interval_qty") ?? 1
  const interval = form.watch("billing_interval")
  const summary =
    !interval || interval === "none"
      ? "No recurring interval."
      : qty === 1
        ? `Charges every ${interval}.`
        : `Charges every ${qty} ${interval}s.`

  return (
    <FormItem>
      <FormLabel>Billing</FormLabel>
      <div className="flex items-center gap-2">
        <span className="text-sm text-muted-foreground">every</span>
        <FormField
          control={form.control}
          name="billing_interval_qty"
          render={({ field }) => (
            <FormItem className="w-20">
              <FormControl>
                <Input
                  type="number"
                  min={1}
                  aria-label="Billing frequency"
                  name={field.name}
                  ref={field.ref}
                  onBlur={field.onBlur}
                  value={field.value ?? ""}
                  onChange={(e) =>
                    field.onChange(e.target.value === "" ? undefined : e.target.valueAsNumber)
                  }
                />
              </FormControl>
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="billing_interval"
          render={({ field }) => (
            <FormItem className="flex-1">
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger aria-label="Billing interval">
                    <SelectValue placeholder="Interval" />
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
          )}
        />
      </div>
      <FormDescription>
        {summary}
        {note ? ` ${note}` : ""}
      </FormDescription>
      <FormMessage />
    </FormItem>
  )
}
