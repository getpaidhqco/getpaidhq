"use client"

import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { SettingsRow } from "@/app/(dashboard)/settings/_components/settings-row"
import { useSettingsForm } from "../../settings-context"

export default function InvoiceSettings() {
  const { form, isLoading, submitForm } = useSettingsForm()

  const onSubmit = async () => {
    try {
      await submitForm()
      toast.success("Settings saved")
    } catch (error: any) {
      console.error("Failed to update invoice settings:", error)
      toast.error(error.message || "Failed to save settings")
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <div>
          <SettingsRow
            title="Invoice numbering"
            description="A short prefix prepended to every invoice number."
          >
            <FormField
              control={form.control}
              name="invoice_prefix"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Prefix</FormLabel>
                  <FormControl>
                    <Input className="max-w-32" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsRow>

          <SettingsRow
            title="PDF invoices"
            description="Attach PDF copies to invoice emails and customer payment pages."
          >
            <FormField
              control={form.control}
              name="enable_invoice_pdfs"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center gap-3">
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormLabel className="m-0 font-normal">
                    Include PDF links and attachments
                  </FormLabel>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsRow>
        </div>

        <div className="mt-8 flex justify-end gap-2">
          <Button type="reset" variant="ghost">
            Reset
          </Button>
          <Button
            type="submit"
            disabled={form.formState.isSubmitting || isLoading}
          >
            {form.formState.isSubmitting || isLoading ? "Saving…" : "Save changes"}
          </Button>
        </div>
      </form>
    </Form>
  )
}
