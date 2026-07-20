"use client"

import { toast } from "sonner"
import { useForm } from "react-hook-form"
import {
  useCreateWebhook,
  webhookResolvers,
  type CreateWebhookFormValues,
} from "@getpaidhq/react-sdk"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
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
import { Button } from "@/components/ui/button"

interface AddWebhookModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddWebhookModal({ open, onOpenChange }: AddWebhookModalProps) {
  // Validation comes from the react-sdk resolver (mirrors the server contract).
  // Events default to "*" (all) since the modal doesn't expose per-event selection.
  const form = useForm<CreateWebhookFormValues>({
    resolver: webhookResolvers.create,
    defaultValues: {
      url: "",
      events: ["*"],
      secret: "",
    },
  })

  const createWebhook = useCreateWebhook({
    onSuccess: () => {
      toast.success("Webhook created", {
        description: "The webhook has been created successfully.",
      })
      onOpenChange(false)
      form.reset()
    },
    onError: (error: Error) => {
      toast.error("Error", {
        description: error.message || "Failed to create webhook. Please try again.",
      })
    },
  })

  const onSubmit = (data: CreateWebhookFormValues) => {
    createWebhook.mutate({ ...data, secret: data.secret || undefined })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Add Webhook</DialogTitle>
          <DialogDescription>
            Enter the URL where webhook events should be sent.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="url"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Webhook URL</FormLabel>
                  <FormControl>
                    <Input placeholder="https://example.com/webhook" {...field} />
                  </FormControl>
                  <FormDescription>
                    The URL where webhook events will be sent.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="secret"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Webhook Secret (Optional)</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="whsec_..."
                      {...field}
                      value={field.value ?? ""}
                    />
                  </FormControl>
                  <FormDescription>
                    Used to verify that requests are coming from GetPaidHQ.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={createWebhook.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={createWebhook.isPending}>
                {createWebhook.isPending ? "Creating…" : "Create Webhook"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
