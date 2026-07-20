"use client"
import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useUpdateSubscription,
  subscriptionResolvers,
  type UpdateSubscriptionFormValues,
} from "@getpaidhq/react-sdk"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useSubscription } from "@/app/(dashboard)/subscriptions/[id]/subscription-context"

// Statuses a merchant can move a subscription to from this dialog.
const STATUS_OPTIONS = [
  { value: "active", label: "Active" },
  { value: "paused", label: "Paused" },
  { value: "past_due", label: "Past due" },
  { value: "cancelled", label: "Cancelled" },
] as const

export default function UpdateSubscriptionDialog({
  isOpen,
  onClose,
  subscriptionId,
  onSuccess,
}: {
  isOpen: boolean
  onClose: () => void
  subscriptionId: string
  onSuccess?: () => void
}) {
  const { subscription } = useSubscription()

  // Validation comes from the react-sdk resolver (mirrors the server's
  // UpdateSubscriptionRequest contract) — no local schema, no inline rules.
  const form = useForm<UpdateSubscriptionFormValues>({
    resolver: subscriptionResolvers.update,
    defaultValues: {
      status: subscription?.status ?? "active",
    },
  })

  const updateSubscription = useUpdateSubscription(subscriptionId, {
    onSuccess: () => {
      toast.success("Subscription updated successfully")
      form.reset()
      onClose()
      onSuccess?.()
    },
    onError: (error: Error) => {
      toast.error("Failed to update subscription", {
        description: error instanceof Error ? error.message : "Unknown error",
      })
    },
  })

  const onSubmit = (values: UpdateSubscriptionFormValues) => {
    updateSubscription.mutate(values)
  }

  return (
    <AlertDialog
      open={isOpen}
      onOpenChange={(o) => {
        if (!o && !updateSubscription.isPending) onClose()
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Update Subscription</AlertDialogTitle>
          <AlertDialogDescription>
            Change the status of this subscription.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="mt-4 space-y-4">
            <FormField
              control={form.control}
              name="status"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Status</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value ?? ""}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select status" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {STATUS_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    The lifecycle status of this subscription.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <AlertDialogFooter>
              <Button
                variant="outline"
                type="button"
                onClick={onClose}
                disabled={updateSubscription.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={updateSubscription.isPending}>
                {updateSubscription.isPending ? "Updating…" : "Update Subscription"}
              </Button>
            </AlertDialogFooter>
          </form>
        </Form>
      </AlertDialogContent>
    </AlertDialog>
  )
}
