"use client"

import * as React from "react"
import {Dialog, DialogContent, DialogHeader, DialogTitle} from "@/components/ui/dialog"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {Switch} from "@/components/ui/switch"
import {Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form"
import {useForm} from "react-hook-form"
import {toast} from "sonner"
import {
  useCreateMeter,
  meterResolvers,
  AGGREGATION_TYPES,
  isCarryOverAggregation,
  requiresCarryOver,
  type CreateMeterFormValues,
  type MeterAggregation,
} from "@getpaidhq/react-sdk"
import type {MeterResponse} from "@getpaidhq/sdk"
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"

// "weighted_sum" -> "Weighted sum"
const humanize = (v: string) => v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ")

interface MeterCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onMeterCreated: (meter: MeterResponse) => void
}

export function MeterCreateDialog({
                                    open,
                                    onOpenChange,
                                    onMeterCreated,
                                  }: MeterCreateDialogProps) {
  const form = useForm<CreateMeterFormValues>({
    resolver: meterResolvers.create,
    defaultValues: {
      code: "",
      name: "",
      aggregation: "count",
      field_name: "",
      carry_over: false,
    },
  })

  const aggregation = form.watch("aggregation")
  const carryOver = form.watch("carry_over")

  // Carry-over (stock) meters only support standing-level aggregations; weighted_sum
  // in turn requires carry_over. Mirrors the server's validateCarryOver rules.
  const aggregationOptions = carryOver
    ? AGGREGATION_TYPES.filter(isCarryOverAggregation)
    : AGGREGATION_TYPES
  const carryOverLocked = requiresCarryOver(aggregation)

  function handleAggregationChange(value: MeterAggregation) {
    form.setValue("aggregation", value, {shouldValidate: true})
    if (requiresCarryOver(value)) {
      form.setValue("carry_over", true, {shouldValidate: true})
    }
  }

  function handleCarryOverChange(value: boolean) {
    form.setValue("carry_over", value, {shouldValidate: true})
    if (value && !isCarryOverAggregation(form.getValues("aggregation"))) {
      form.setValue("aggregation", "latest", {shouldValidate: true})
    }
  }

  const createMeter = useCreateMeter({
    onSuccess: (newMeter: MeterResponse) => {
      toast.success("Meter created successfully")
      onMeterCreated(newMeter)
      onOpenChange(false)
      form.reset()
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create meter", {duration: 8000})
      console.error("Error creating meter:", error)
    },
  })

  const onSubmit = (data: CreateMeterFormValues) => {
    createMeter.mutate(data)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Create New Meter</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
            <FormField
              control={form.control}
              name="code"
              render={({field}) => (
                <FormItem>
                  <FormLabel>Code</FormLabel>
                  <FormControl>
                    <Input placeholder="api_requests" {...field} />
                  </FormControl>
                  <FormDescription>
                    Unique identifier that usage events reference. Cannot be changed later.
                  </FormDescription>
                  <FormMessage/>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="name"
              render={({field}) => (
                <FormItem>
                  <FormLabel>Display Name</FormLabel>
                  <FormControl>
                    <Input placeholder="API Requests" {...field} />
                  </FormControl>
                  <FormMessage/>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="aggregation"
              render={({field}) => (
                <FormItem>
                  <FormLabel>Aggregation</FormLabel>
                  <Select onValueChange={handleAggregationChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select aggregation"/>
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {aggregationOptions.map((type) => (
                        <SelectItem key={type} value={type}>
                          {humanize(type)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage/>
                </FormItem>
              )}
            />

            {aggregation !== "count" && (
              <FormField
                control={form.control}
                name="field_name"
                render={({field}) => (
                  <FormItem>
                    <FormLabel>Field Name</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="bytes"
                        {...field}
                        value={field.value ?? ""}
                      />
                    </FormControl>
                    <FormDescription>
                      Event metadata key to read the value from.
                    </FormDescription>
                    <FormMessage/>
                  </FormItem>
                )}
              />
            )}

            <FormField
              control={form.control}
              name="carry_over"
              render={({field}) => (
                <FormItem className="flex flex-row items-center justify-between gap-6 rounded-lg border p-4">
                  <div className="space-y-0.5">
                    <FormLabel>Carry over</FormLabel>
                    <FormDescription>
                      Treat usage as a standing level (seats, active resources) that carries
                      across billing periods, instead of resetting each period.
                      {carryOverLocked ? ` Required for ${humanize(aggregation)}.` : null}
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value ?? false}
                      onCheckedChange={handleCarryOverChange}
                      disabled={carryOverLocked}
                    />
                  </FormControl>
                </FormItem>
              )}
            />

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMeter.isPending}>
                {createMeter.isPending ? "Creating…" : "Create Meter"}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
