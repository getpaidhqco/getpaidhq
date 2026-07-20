"use client"

import { useEffect, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useForm } from "react-hook-form"
import { TriangleAlert } from "lucide-react"
import { useGetPaidHQClient } from "@getpaidhq/react-sdk"
import type { ApiKeyCreateResponse, CreateApiKeyInput } from "@getpaidhq/sdk"

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
import { ApiKeyCopyButton } from "./api-key-copy-button"
import { toast } from "sonner"

// No react-sdk zod schema exists for api-keys, so this form uses RHF-native
// `rules` validation typed by the SDK CreateApiKeyInput request shape.
interface CreateApiKeyModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateApiKeyModal({ open, onOpenChange }: CreateApiKeyModalProps) {
  const client = useGetPaidHQClient()
  const queryClient = useQueryClient()
  const [createdKey, setCreatedKey] = useState<string | null>(null)

  const form = useForm<CreateApiKeyInput>({
    defaultValues: { name: "" },
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateApiKeyInput): Promise<ApiKeyCreateResponse> =>
      client.apiKeys.create({ name: data.name || undefined }),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] })
      setCreatedKey(result.key)
    },
  })

  // Never toast during render — surface mutation errors from an effect.
  useEffect(() => {
    if (createMutation.error) {
      toast.error("Error", {
        description:
          createMutation.error.message ||
          "Failed to create API key. Please try again.",
      })
    }
  }, [createMutation.error])

  // Reset everything when the dialog fully closes so reopening starts fresh.
  const handleOpenChange = (next: boolean) => {
    onOpenChange(next)
    if (!next) {
      setCreatedKey(null)
      form.reset()
      createMutation.reset()
    }
  }

  const onSubmit = (data: CreateApiKeyInput) => createMutation.mutate(data)

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        {createdKey ? (
          <>
            <DialogHeader>
              <DialogTitle>API key created</DialogTitle>
              <DialogDescription>
                Copy your key now — for security, you won't be able to see it
                again.
              </DialogDescription>
            </DialogHeader>

            <div className="flex min-w-0 items-start gap-2 py-2">
              <code className="min-w-0 flex-1 rounded-md bg-muted/60 px-3 py-2 font-mono text-xs break-all text-foreground">
                {createdKey}
              </code>
              <ApiKeyCopyButton apiKey={createdKey} />
            </div>

            <div className="flex items-start gap-2 rounded-md bg-warning/10 px-3 py-2 text-xs text-warning-foreground">
              <TriangleAlert className="mt-0.5 size-3.5 shrink-0 text-warning" />
              <span className="text-muted-foreground">
                Store this somewhere safe. Anyone with this key can act on your
                account.
              </span>
            </div>

            <DialogFooter>
              <Button type="button" onClick={() => handleOpenChange(false)}>
                Done
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>Create API key</DialogTitle>
              <DialogDescription>
                Give your key an optional name to help you recognize it later.
              </DialogDescription>
            </DialogHeader>

            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                  control={form.control}
                  name="name"
                  rules={{
                    maxLength: {
                      value: 64,
                      message: "Keep it under 64 characters",
                    },
                  }}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Name (optional)</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="e.g. ci-deploy"
                          {...field}
                          value={field.value ?? ""}
                        />
                      </FormControl>
                      <FormDescription>
                        A label to identify where this key is used.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <DialogFooter>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => handleOpenChange(false)}
                    disabled={createMutation.isPending}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" disabled={createMutation.isPending}>
                    {createMutation.isPending ? "Creating…" : "Create key"}
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
