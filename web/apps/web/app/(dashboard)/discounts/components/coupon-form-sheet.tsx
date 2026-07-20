"use client"

import * as React from "react"
import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useCreateCoupon,
  useUpdateCoupon,
  couponResolvers,
  DISCOUNT_TYPES,
  COUPON_DURATIONS,
  type CreateCouponFormValues,
  type UpdateCouponFormValues,
} from "@getpaidhq/react-sdk"
import type {
  CouponResponse,
  CreateCouponInput,
  UpdateCouponInput,
} from "@getpaidhq/sdk"

import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { DatePicker } from "@/components/ui/date-picker"
import { format, parseISO } from "date-fns"
import { currencyToCents, centsToCurrency } from "@/lib/currency"

// "fixed" -> "Fixed", "once_per_customer" -> "Once per customer"
const humanize = (v: string) =>
  v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ")

const CURRENCIES = [
  { value: "USD", label: "USD — US Dollar" },
  { value: "EUR", label: "EUR — Euro" },
  { value: "GBP", label: "GBP — British Pound" },
  { value: "ZAR", label: "ZAR — South African Rand" },
] as const

interface CouponFormSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** When set, the sheet edits this coupon; otherwise it creates a new one. */
  coupon?: CouponResponse | null
}

export function CouponFormSheet({ open, onOpenChange, coupon }: CouponFormSheetProps) {
  const isEdit = Boolean(coupon)

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex w-[600px] flex-col sm:max-w-[600px]">
        <SheetHeader>
          <SheetTitle>{isEdit ? "Edit discount" : "New discount"}</SheetTitle>
          <SheetDescription>
            {isEdit
              ? "Rename the discount or change whether it's active. The discount type and amount are fixed once created."
              : "Create a coupon that can be applied to orders and subscriptions."}
          </SheetDescription>
        </SheetHeader>

        {/* Remount per coupon/mode so form state never leaks between opens. */}
        {isEdit && coupon ? (
          <EditCouponForm
            key={coupon.id}
            coupon={coupon}
            onDone={() => onOpenChange(false)}
          />
        ) : (
          <CreateCouponForm key="create" onDone={() => onOpenChange(false)} />
        )}
      </SheetContent>
    </Sheet>
  )
}

function CreateCouponForm({ onDone }: { onDone: () => void }) {
  const form = useForm<CreateCouponFormValues>({
    resolver: couponResolvers.create,
    defaultValues: {
      name: "",
      discount_type: "percentage",
      duration: "once",
      currency: "USD",
      once_per_customer: false,
    },
  })

  const discountType = form.watch("discount_type")
  const duration = form.watch("duration")

  const createCoupon = useCreateCoupon({
    onSuccess: () => {
      toast.success("Discount created")
      onDone()
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create discount", { duration: 8000 })
    },
  })

  const onSubmit = (data: CreateCouponFormValues) => {
    const payload: CreateCouponInput = {
      name: data.name,
      discount_type: data.discount_type,
      duration: data.duration,
      ...(data.discount_type === "percentage"
        ? { percent_off: data.percent_off }
        : { amount_off: data.amount_off, currency: data.currency }),
      ...(data.duration === "repeating"
        ? { duration_in_cycles: data.duration_in_cycles }
        : {}),
      ...(data.max_redemptions != null ? { max_redemptions: data.max_redemptions } : {}),
      once_per_customer: data.once_per_customer ?? false,
      ...(data.redeem_by ? { redeem_by: new Date(data.redeem_by).toISOString() } : {}),
    }
    createCoupon.mutate(payload)
  }

  return (
    <Form {...form}>
      <form className="grid flex-1 auto-rows-min gap-6 overflow-y-auto px-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input placeholder="e.g. Launch 20% off" {...field} value={field.value ?? ""} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="discount_type"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Type</FormLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Select type" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {DISCOUNT_TYPES.map((t) => (
                    <SelectItem key={t} value={t}>
                      {humanize(t)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />

        {discountType === "percentage" ? (
          <FormField
            control={form.control}
            name="percent_off"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Percentage off</FormLabel>
                <FormControl>
                  <div className="relative">
                    <Input
                      type="number"
                      min="0"
                      max="100"
                      step="0.01"
                      inputMode="decimal"
                      placeholder="20"
                      className="tabular-nums pr-8"
                      value={field.value ?? ""}
                      onChange={(e) =>
                        field.onChange(e.target.value === "" ? undefined : parseFloat(e.target.value))
                      }
                    />
                    <span className="text-muted-foreground pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm">
                      %
                    </span>
                  </div>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        ) : (
          <div className="grid grid-cols-2 gap-4">
            <FormField
              control={form.control}
              name="currency"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Currency</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
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
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="amount_off"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Amount off</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min="0"
                      step="0.01"
                      inputMode="decimal"
                      placeholder="0.00"
                      className="tabular-nums"
                      value={field.value != null ? centsToCurrency(field.value) : ""}
                      onChange={(e) =>
                        field.onChange(
                          e.target.value === ""
                            ? undefined
                            : currencyToCents(parseFloat(e.target.value)),
                        )
                      }
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        )}

        <FormField
          control={form.control}
          name="duration"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Duration</FormLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Select duration" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {COUPON_DURATIONS.map((d) => (
                    <SelectItem key={d} value={d}>
                      {humanize(d)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormDescription>
                How long the discount applies on a subscription — a single invoice
                (once), a set number of cycles (repeating) or for the lifetime (forever).
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        {duration === "repeating" ? (
          <FormField
            control={form.control}
            name="duration_in_cycles"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Number of cycles</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    min="1"
                    step="1"
                    inputMode="numeric"
                    placeholder="3"
                    className="tabular-nums"
                    value={field.value ?? ""}
                    onChange={(e) =>
                      field.onChange(e.target.value === "" ? undefined : parseInt(e.target.value, 10))
                    }
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        ) : null}

        <FormField
          control={form.control}
          name="max_redemptions"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Max redemptions</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  min="1"
                  step="1"
                  inputMode="numeric"
                  placeholder="Unlimited"
                  className="tabular-nums"
                  value={field.value ?? ""}
                  onChange={(e) =>
                    field.onChange(e.target.value === "" ? undefined : parseInt(e.target.value, 10))
                  }
                />
              </FormControl>
              <FormDescription>Leave empty for unlimited redemptions.</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="redeem_by"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Redeem by</FormLabel>
              <FormControl>
                <DatePicker
                  placeholder="Pick an expiry date"
                  value={field.value ? parseISO(field.value) : undefined}
                  minDate={new Date()}
                  onChange={(date) =>
                    field.onChange(date ? format(date, "yyyy-MM-dd") : undefined)
                  }
                />
              </FormControl>
              <FormDescription>Optional expiry date for the discount.</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="once_per_customer"
          render={({ field }) => (
            <FormItem className="flex items-start gap-3">
              <Switch
                checked={field.value ?? false}
                onCheckedChange={field.onChange}
                aria-label="Once per customer"
              />
              <div className="space-y-1">
                <FormLabel>Once per customer</FormLabel>
                <FormDescription>
                  Limit each customer to redeeming this discount a single time.
                </FormDescription>
              </div>
            </FormItem>
          )}
        />
      </form>

      <SheetFooter className="mt-8 justify-end">
        <SheetClose asChild>
          <Button type="button" variant="ghost">
            Cancel
          </Button>
        </SheetClose>
        <Button
          type="button"
          disabled={createCoupon.isPending}
          onClick={form.handleSubmit(onSubmit)}
          className="ml-2"
        >
          {createCoupon.isPending ? "Creating…" : "Create discount"}
        </Button>
      </SheetFooter>
    </Form>
  )
}

function EditCouponForm({ coupon, onDone }: { coupon: CouponResponse; onDone: () => void }) {
  const form = useForm<UpdateCouponFormValues>({
    resolver: couponResolvers.update,
    defaultValues: {
      name: coupon.name,
      active: coupon.active,
    },
  })

  // Keep the form in sync if the coupon prop changes underneath us.
  useEffect(() => {
    form.reset({ name: coupon.name, active: coupon.active })
  }, [coupon, form])

  const updateCoupon = useUpdateCoupon(coupon.id, {
    onSuccess: () => {
      toast.success("Discount updated")
      onDone()
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to update discount", { duration: 8000 })
    },
  })

  const onSubmit = (data: UpdateCouponFormValues) => {
    const payload: UpdateCouponInput = { name: data.name, active: data.active }
    updateCoupon.mutate(payload)
  }

  return (
    <Form {...form}>
      <form className="grid flex-1 auto-rows-min gap-6 overflow-y-auto px-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ""} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="active"
          render={({ field }) => (
            <FormItem className="flex items-start gap-3">
              <Switch
                checked={field.value}
                onCheckedChange={field.onChange}
                aria-label="Active"
              />
              <div className="space-y-1">
                <FormLabel>Active</FormLabel>
                <FormDescription>
                  Inactive discounts can no longer be redeemed.
                </FormDescription>
              </div>
            </FormItem>
          )}
        />
      </form>

      <SheetFooter className="mt-8 justify-end">
        <SheetClose asChild>
          <Button type="button" variant="ghost">
            Cancel
          </Button>
        </SheetClose>
        <Button
          type="button"
          disabled={updateCoupon.isPending}
          onClick={form.handleSubmit(onSubmit)}
          className="ml-2"
        >
          {updateCoupon.isPending ? "Saving…" : "Save changes"}
        </Button>
      </SheetFooter>
    </Form>
  )
}
