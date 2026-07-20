"use client"

import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"

import { Address } from "./address"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form"
import { countries } from "@/lib/countries"
import { SettingsRow } from "@/app/(dashboard)/settings/_components/settings-row"
import { useSettings } from "../settings-context"

const currencies = (() => {
  const seen = new Set<string>()
  return countries
    .filter((c) => {
      if (!c.currency_code || seen.has(c.currency_code)) return false
      seen.add(c.currency_code)
      return true
    })
    .sort((a, b) => a.currency_code.localeCompare(b.currency_code))
})()

const Schema = z.object({
  name: z.string().min(3),
  currency: z.string().optional().nullable(),
})
type SchemaType = z.infer<typeof Schema>

export default function GatewaySettings() {
  const { settings, updateSettings } = useSettings()

  const form = useForm<SchemaType>({
    resolver: zodResolver(Schema),
    defaultValues: {
      name: settings?.name || "org",
      currency: settings?.currency || "USD",
    },
  })

  useEffect(() => {
    form.reset({
      name: settings?.name || "org",
      currency: settings?.currency || "USD",
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [settings?.name, settings?.currency])

  const onSubmit = async (values: SchemaType) => {
    try {
      await updateSettings(values)
    } catch (error) {
      console.error("Failed to update gateway settings:", error)
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} onReset={() => form.reset()}>
        <div>
          <SettingsRow
            title="Organization name"
            description="Shown on your public profile and invoices."
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input {...field} value={field.value ?? ""} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsRow>

          <SettingsRow
            title="Currency"
            description="The dashboard's reporting currency. Changing this rebuilds all historic reports."
          >
            <FormField
              control={form.control}
              name="currency"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Select value={field.value ?? ""} onValueChange={field.onChange}>
                      <SelectTrigger aria-label="Currency">
                        <SelectValue placeholder="Select currency" />
                      </SelectTrigger>
                      <SelectContent>
                        {currencies.map((country) => (
                          <SelectItem
                            key={country.currency_code}
                            value={country.currency_code}
                          >
                            <span className="w-5 sm:w-4">{country.flag}</span>
                            &nbsp;&nbsp;
                            {country.currency_code} — {country.currency}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsRow>

          <SettingsRow
            title="Organization email"
            description="How customers reach you for support."
          >
            <Input
              type="email"
              aria-label="Organization email"
              name="email"
              defaultValue="info@example.com"
            />
            <div className="flex items-center gap-2">
              <Checkbox id="email_is_public" name="email_is_public" defaultChecked />
              <Label htmlFor="email_is_public">Show email on public profile</Label>
            </div>
          </SettingsRow>

          <SettingsRow
            title="Address"
            description="Where your organization is registered."
          >
            <Address />
          </SettingsRow>
        </div>

        <div className="mt-8 flex justify-end gap-2">
          <Button type="reset" variant="ghost">
            Reset
          </Button>
          <Button type="submit">Save changes</Button>
        </div>
      </form>
    </Form>
  )
}
