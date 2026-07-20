"use client"

import {useForm} from "react-hook-form"
import {toast} from "sonner"
import {useRouter} from "next/navigation"
import {
  useCreateMeter,
  meterResolvers,
  AGGREGATION_TYPES,
  ROUNDING_MODES,
  isCarryOverAggregation,
  requiresCarryOver,
  type CreateMeterFormValues,
  type MeterAggregation,
} from "@getpaidhq/react-sdk"
import {Button} from "@/components/ui/button"
import {Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage,} from "@/components/ui/form"
import {FormLayout, FormSection, FormActions} from "@/components/ui/form-section"
import {Input} from "@/components/ui/input"
import {Switch} from "@/components/ui/switch"
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"

// "weighted_sum" -> "Weighted sum"
const humanize = (v: string) => v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ")

const ROUNDING_NONE = "none"

export function MeterForm() {
  const router = useRouter()

  // Validation comes from the react-sdk resolver (mirrors the server contract) — no
  // local schema, no inline rules. The form is typed by the SDK-derived form values.
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

  // Need the chosen aggregation + carry_over to drive the cross-field constraints
  // (field_name requirement, and the carry_over ↔ aggregation rules the server enforces).
  const aggregation = form.watch("aggregation")
  const carryOver = form.watch("carry_over")
  const roundingMode = form.watch("rounding_mode")

  // A carry-over (stock) meter only supports standing-level aggregations.
  const aggregationOptions = carryOver
    ? AGGREGATION_TYPES.filter(isCarryOverAggregation)
    : AGGREGATION_TYPES
  // weighted_sum (a time-average) is a standing level, so it forces carry_over on.
  const carryOverLocked = requiresCarryOver(aggregation)

  // Picking an aggregation that requires carry_over auto-enables it.
  function handleAggregationChange(value: MeterAggregation) {
    form.setValue("aggregation", value, {shouldValidate: true})
    if (requiresCarryOver(value)) {
      form.setValue("carry_over", true, {shouldValidate: true})
    }
  }

  // Toggling carry_over on must drop any flow-only aggregation (count/sum).
  function handleCarryOverChange(value: boolean) {
    form.setValue("carry_over", value, {shouldValidate: true})
    if (value && !isCarryOverAggregation(form.getValues("aggregation"))) {
      form.setValue("aggregation", "latest", {shouldValidate: true})
    }
  }

  const createMeter = useCreateMeter({
    onSuccess: () => {
      toast.success("Meter created successfully")
      router.push("/meters")
    },
    onError: (error: Error) => {
      toast.error("Failed to create meter", {
        description: error.message || "An unknown error occurred",
      })
    },
  })

  function onSubmit(values: CreateMeterFormValues) {
    createMeter.mutate(values)
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <FormLayout className="mx-0">
          <FormSection
            title="Meter details"
            description="What this meter measures and how events reference it."
          >
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
                  <FormDescription>
                    Human-friendly name shown in the dashboard.
                  </FormDescription>
                  <FormMessage/>
                </FormItem>
              )}
            />
          </FormSection>

          <FormSection
            title="Aggregation"
            description="How raw usage events are turned into a billable quantity."
          >
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
                  <FormDescription>
                    How usage values are combined over the billing period.
                    {carryOver ? " Carry-over meters read a standing level." : null}
                  </FormDescription>
                  <FormMessage/>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="field_name"
              render={({field}) => (
                <FormItem>
                  <FormLabel>
                    Field Name{aggregation === "count" ? " (optional)" : ""}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder="bytes"
                      {...field}
                      value={field.value ?? ""}
                      disabled={aggregation === "count"}
                    />
                  </FormControl>
                  <FormDescription>
                    Event metadata key to read the value from. Required for all
                    aggregations except <strong>count</strong>.
                  </FormDescription>
                  <FormMessage/>
                </FormItem>
              )}
            />
          </FormSection>

          <FormSection
            title="Rounding & behaviour"
            description="Optional controls for how the computed quantity is rounded, and whether it carries across periods."
          >
            <FormField
              control={form.control}
              name="rounding_mode"
              render={({field}) => (
                <FormItem>
                  <FormLabel>Rounding Mode</FormLabel>
                  <Select
                    onValueChange={(v) => {
                      if (v === ROUNDING_NONE) {
                        field.onChange(undefined)
                        // Scale is meaningless without a mode — clear it.
                        form.setValue("rounding_scale", undefined)
                      } else {
                        field.onChange(v)
                      }
                    }}
                    value={field.value ?? ROUNDING_NONE}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue/>
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value={ROUNDING_NONE}>None</SelectItem>
                      {ROUNDING_MODES.map((mode) => (
                        <SelectItem key={mode} value={mode}>
                          {humanize(mode)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    How computed usage is rounded before billing.
                  </FormDescription>
                  <FormMessage/>
                </FormItem>
              )}
            />

            {roundingMode ? (
              <FormField
                control={form.control}
                name="rounding_scale"
                render={({field}) => (
                  <FormItem>
                    <FormLabel>Rounding Scale</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        max={18}
                        placeholder="0"
                        name={field.name}
                        ref={field.ref}
                        onBlur={field.onBlur}
                        value={field.value ?? ""}
                        onChange={(e) =>
                          field.onChange(e.target.value === "" ? undefined : e.target.valueAsNumber)
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      Decimal places to round to (0–18).
                    </FormDescription>
                    <FormMessage/>
                  </FormItem>
                )}
              />
            ) : null}

            <FormField
              control={form.control}
              name="carry_over"
              render={({field}) => (
                <FormItem className="flex flex-row items-center justify-between gap-6">
                  <div className="space-y-0.5">
                    <FormLabel>Carry over</FormLabel>
                    <FormDescription>
                      Treat usage as a standing level (seats, active resources) that carries
                      across billing periods, instead of resetting each period.
                      {carryOverLocked
                        ? ` Required for ${humanize(aggregation)}.`
                        : null}
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
          </FormSection>

          <FormActions>
            <Button type="button" variant="outline" onClick={() => router.push("/meters")}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMeter.isPending}>
              {createMeter.isPending ? "Creating…" : "Create Meter"}
            </Button>
          </FormActions>
        </FormLayout>
      </form>
    </Form>
  )
}
