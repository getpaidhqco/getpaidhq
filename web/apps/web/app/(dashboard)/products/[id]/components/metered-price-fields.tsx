"use client"

import React, { useState } from "react"
import { useFieldArray } from "react-hook-form"
import { Trash2 } from "lucide-react"
import {
  SELECTABLE_PRICE_SCHEMES,
  schemeRequiresTiers,
  useMeters,
} from "@getpaidhq/react-sdk"
import type { MeterResponse } from "@getpaidhq/sdk"

import { Button } from "@/components/ui/button"
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
import { MeterSearch } from "@/components/meter-search"
import { formatAmount, formatUnits } from "@/lib/price-display"

import {
  BillingCadenceField,
  humanize,
  InlineCurrencySelect,
  MoneyInput,
  type PriceFieldsProps,
} from "./price-form-shared"

const SCHEME_DESCRIPTIONS: Record<string, string> = {
  fixed: "Usage bills at a single rate — set the price per unit.",
  package:
    "Usage bills per started block of units: any partial block is charged as a full block (rounded up).",
  graduated: "Each unit bills at the rate of the band it falls into, summed.",
  volume: "All units bill at the single band the total quantity reaches.",
}

/**
 * Concrete explanation of what a fixed/package price charges, with a worked
 * example so the unit_price/unit_count interaction is unambiguous: unit_price
 * buys unit_count units; fixed prorates a partial block, package bills it in full.
 */
function unitPricingExplainer(
  scheme: string,
  unitPriceCents: number,
  unitCount: number,
  currency: string,
): string {
  const count = Math.max(unitCount || 1, 1)
  const price = formatAmount(currency, unitPriceCents || 0)
  if (scheme === "package") {
    const exampleUnits = Math.round(count * 1.5)
    return (
      `Charges ${price} for every started block of ${formatUnits(count)} units. ` +
      `Usage is rounded UP to whole blocks — e.g. ${formatUnits(exampleUnits)} units = 2 blocks = ` +
      `${formatAmount(currency, (unitPriceCents || 0) * 2)}.`
    )
  }
  if (count > 1) {
    return (
      `Charges ${price} per ${formatUnits(count)} units — an effective ` +
      `${formatAmount(currency, (unitPriceCents || 0) / count)} per unit.`
    )
  }
  return `Charges ${price} per unit of metered usage. Increase “units” to price a block (e.g. ${price} per 1,000 units).`
}

/**
 * Fields for a usage-based (metered) price: the meter it bills from, the pricing
 * scheme, the rate(s), the cadence, and the carry-over proration switches.
 * Metered prices are always subscriptions and have no payment count (usage bills
 * until the subscription ends), so neither field appears here.
 */
export function MeteredPriceFields({ form, symbol }: PriceFieldsProps) {
  const currency = form.watch("currency")
  const scheme = form.watch("scheme")
  const unitPrice = form.watch("unit_price")
  const unitCount = form.watch("unit_count")
  const billableMetricId = form.watch("billable_metric_id")
  const isTiered = schemeRequiresTiers(scheme)

  // The selected meter's details decide which extra settings apply. MeterSearch
  // hands the full meter over on pick; for edit mode (only the id is known) fall
  // back to the meters list.
  const [pickedMeter, setPickedMeter] = useState<MeterResponse | undefined>()
  const { data: metersData } = useMeters()
  const metersList = (
    Array.isArray(metersData) ? metersData : (metersData?.data ?? [])
  ) as MeterResponse[]
  const meter = billableMetricId
    ? pickedMeter?.id === billableMetricId
      ? pickedMeter
      : metersList.find((m) => m.id === billableMetricId)
    : undefined
  // The proration switches shape per-unit time fractions, which only exist on a
  // time-weighted (weighted_sum) carry-over meter — see gphq-server
  // docs/internal/billing-model/seat-billing/mapping.md §1 Axis 2.
  const isTimeWeighted = Boolean(meter?.carry_over) && meter?.aggregation === "weighted_sum"

  const tiers = useFieldArray({ control: form.control, name: "tiers" })

  // A default starter band: from 0, unbounded, no rate yet.
  const addTier = () =>
    tiers.append({ from_value: "0", to_value: "", per_unit_amount: "", flat_amount: "0" })

  return (
    <>
      <FormField
        control={form.control}
        name="billable_metric_id"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Meter</FormLabel>
            <MeterSearch
              value={field.value}
              onValueChange={(value, m) => {
                field.onChange(value)
                setPickedMeter(m)
              }}
            />
            <FormDescription>
              {meter
                ? `${meter.name} aggregates by ${meter.aggregation}${meter.carry_over ? " and carries its level across periods (stock meter)" : ""}.`
                : "The meter whose aggregated usage this price bills, in arrears each period."}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="scheme"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Pricing scheme</FormLabel>
            <Select
              value={field.value}
              onValueChange={(v) => {
                field.onChange(v)
                // Tiered schemes need at least one band — seed one when switching in.
                if (schemeRequiresTiers(v) && tiers.fields.length === 0) addTier()
              }}
            >
              <FormControl>
                <SelectTrigger>
                  <SelectValue placeholder="Select scheme" />
                </SelectTrigger>
              </FormControl>
              <SelectContent>
                {SELECTABLE_PRICE_SCHEMES.map((s) => (
                  <SelectItem key={s} value={s}>
                    {humanize(s)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <FormDescription>{SCHEME_DESCRIPTIONS[scheme] ?? null}</FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      {isTiered ? (
        <FormItem>
          <div className="flex items-center justify-between">
            <FormLabel>Tiers</FormLabel>
            <InlineCurrencySelect form={form} />
          </div>
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-[1fr_1fr_1fr_1fr_auto] gap-2 text-xs text-muted-foreground">
              <span>From (units)</span>
              <span>To (units)</span>
              <span>Per unit ({symbol})</span>
              <span>Flat ({symbol})</span>
              <span className="sr-only">Remove</span>
            </div>
            {tiers.fields.map((row, index) => {
              const isLast = index === tiers.fields.length - 1
              return (
                <div
                  key={row.id}
                  className="grid grid-cols-[1fr_1fr_1fr_1fr_auto] items-center gap-2"
                >
                  <Input
                    type="number"
                    min="0"
                    step="1"
                    placeholder="0"
                    {...form.register(`tiers.${index}.from_value`)}
                  />
                  <Input
                    type="number"
                    min="0"
                    step="1"
                    placeholder={isLast ? "∞" : "0"}
                    {...form.register(`tiers.${index}.to_value`)}
                  />
                  <Input
                    type="number"
                    min="0"
                    step="any"
                    placeholder="0.00"
                    {...form.register(`tiers.${index}.per_unit_amount`)}
                  />
                  <Input
                    type="number"
                    min="0"
                    step="0.01"
                    placeholder="0.00"
                    {...form.register(`tiers.${index}.flat_amount`)}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => tiers.remove(index)}
                    disabled={tiers.fields.length === 1}
                    aria-label="Remove tier"
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </div>
              )
            })}
          </div>
          <Button type="button" variant="outline" size="sm" className="mt-1 w-fit" onClick={addTier}>
            Add tier
          </Button>
          {form.formState.errors.tiers ? (
            <p className="text-sm text-destructive">
              {form.formState.errors.tiers.message as string}
            </p>
          ) : null}
          <FormDescription>Leave the last tier&apos;s “To” empty for an unbounded band.</FormDescription>
        </FormItem>
      ) : (
        <div className="flex flex-col gap-2">
          <div className="grid grid-cols-[1fr_auto_auto_6rem_auto] items-end gap-2">
            <FormField
              control={form.control}
              name="unit_price"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{scheme === "package" ? "Block price" : "Unit price"}</FormLabel>
                  <MoneyInput
                    symbol={symbol}
                    value={field.value}
                    onChange={field.onChange}
                    emptyAsZero
                  />
                </FormItem>
              )}
            />
            <InlineCurrencySelect form={form} />
            <span className="pb-2.5 text-sm text-muted-foreground">per</span>
            <FormField
              control={form.control}
              name="unit_count"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Units</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={1}
                      step={1}
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      value={field.value ?? 1}
                      onChange={(e) =>
                        field.onChange(
                          e.target.value === "" ? 1 : Math.max(1, e.target.valueAsNumber),
                        )
                      }
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <span className="pb-2.5 text-sm text-muted-foreground">
              unit{(unitCount ?? 1) === 1 ? "" : "s"}
            </span>
          </div>
          <FormDescription>
            {unitPricingExplainer(scheme, unitPrice ?? 0, unitCount ?? 1, currency)}
          </FormDescription>
          {form.formState.errors.unit_price ? (
            <p className="text-sm text-destructive">
              {form.formState.errors.unit_price.message as string}
            </p>
          ) : null}
          {form.formState.errors.unit_count ? (
            <p className="text-sm text-destructive">
              {form.formState.errors.unit_count.message as string}
            </p>
          ) : null}
        </div>
      )}

      <BillingCadenceField form={form} note="Usage is billed in arrears each period." />

      {isTimeWeighted ? (
        <div className="flex flex-col gap-4 border-t border-border pt-5">
          <div className="space-y-1">
            <FormLabel>Mid-period changes</FormLabel>
            <FormDescription>
              This meter time-weights a standing level (e.g. seats): each unit bills for the
              fraction of the period it is active. These switches shape that fraction.
            </FormDescription>
          </div>
          <FormField
            control={form.control}
            name="prorate_on_increase"
            render={({ field }) => (
              <FormItem className="flex items-start gap-3">
                <FormControl>
                  <Switch checked={field.value ?? false} onCheckedChange={field.onChange} />
                </FormControl>
                <div className="space-y-1">
                  <FormLabel>Prorate additions</FormLabel>
                  <FormDescription>
                    A unit added mid-period accrues from its add date — otherwise it bills for the
                    full period.
                  </FormDescription>
                </div>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="credit_on_decrease"
            render={({ field }) => (
              <FormItem className="flex items-start gap-3">
                <FormControl>
                  <Switch checked={field.value ?? false} onCheckedChange={field.onChange} />
                </FormControl>
                <div className="space-y-1">
                  <FormLabel>Credit removals</FormLabel>
                  <FormDescription>
                    A unit removed mid-period stops accruing at its removal date — otherwise it
                    stays billable to period end.
                  </FormDescription>
                </div>
              </FormItem>
            )}
          />
        </div>
      ) : null}
    </>
  )
}
