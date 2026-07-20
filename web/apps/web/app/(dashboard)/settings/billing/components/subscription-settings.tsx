"use client"

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
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { SettingsRow } from "@/app/(dashboard)/settings/_components/settings-row"
import { useSettingsForm } from "../../settings-context"

export default function SubscriptionSettings() {
  const { form, isLoading, submitForm } = useSettingsForm()

  const onSubmit = async () => {
    try {
      await submitForm()
    } catch (error) {
      console.error("Failed to update subscription settings:", error)
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <div>
          <SettingsRow
            title="Renewal reminders"
            description="Email customers before a subscription renews."
          >
            <FormField
              control={form.control}
              name="email_reminders"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center gap-3">
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormLabel className="m-0 font-normal">
                    Send upcoming-renewal emails
                  </FormLabel>
                </FormItem>
              )}
            />

            {form.watch("email_reminders") ? (
              <FormField
                control={form.control}
                name="reminder_days"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center gap-2">
                    <FormDescription>Send</FormDescription>
                    <FormControl>
                      <Input
                        className="w-20"
                        type="number"
                        {...field}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormDescription>days before renewal</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ) : null}
          </SettingsRow>

          <SettingsRow
            title="Failed payments"
            description="What happens when a payment can't be collected."
          >
            <FormField
              control={form.control}
              name="cancel_on_failure"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center gap-3">
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <FormLabel className="m-0 font-normal">
                    Cancel the subscription if all retries fail
                  </FormLabel>
                </FormItem>
              )}
            />
          </SettingsRow>

          <SettingsRow
            title="Retry schedule"
            description="How aggressively to retry a failed payment."
          >
            <div className="flex items-center gap-2">
              <FormField
                control={form.control}
                name="retry_policy.attempts"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center gap-2">
                    <FormDescription>Retry</FormDescription>
                    <FormControl>
                      <Input
                        className="w-20"
                        type="number"
                        {...field}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormDescription>times</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="retry_policy.retry_period"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center gap-2">
                    <FormDescription>over</FormDescription>
                    <FormControl>
                      <Input
                        className="w-20"
                        type="number"
                        {...field}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormDescription>days</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </SettingsRow>

          <SettingsRow
            title="After retries"
            description="The subscription state once retries are exhausted."
          >
            <FormField
              control={form.control}
              name="retry_policy.failure_action"
              render={({ field }) => (
                <FormItem>
                  <Select onValueChange={field.onChange} defaultValue={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select what happens if all retries fail" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="cancel">Cancel the subscription</SelectItem>
                      <SelectItem value="mark_unpaid">Mark as unpaid</SelectItem>
                      <SelectItem value="past_due">Mark as past due</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsRow>
        </div>

        <div className="mt-8 flex justify-end gap-2">
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
