"use client"

import * as React from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { FormSection, FormActions } from "@/components/ui/form-section"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import { useCreateCustomer } from "@getpaidhq/react-sdk"
import type { CustomerResponse } from "@getpaidhq/sdk"

// The customers API only supports creation (no update), so this dialog is
// create-only. Validation uses React Hook Form native `rules` — there is no
// customer resolver in the react-sdk validation layer.
interface CustomerFormValues {
  name: string
  email: string
  phone: string
  billing_address: {
    line1: string
    line2: string
    city: string
    state: string
    postal_code: string
    country: string
  }
}

const emptyValues: CustomerFormValues = {
  name: "",
  email: "",
  phone: "",
  billing_address: {
    line1: "",
    line2: "",
    city: "",
    state: "",
    postal_code: "",
    country: "US",
  },
}

interface CustomerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCustomerSaved: (customer: CustomerResponse) => void
}

export function CustomerDialog({ open, onOpenChange, onCustomerSaved }: CustomerDialogProps) {
  const form = useForm<CustomerFormValues>({ defaultValues: emptyValues })

  // Reset form when dialog opens.
  React.useEffect(() => {
    if (open) {
      form.reset(emptyValues)
    }
  }, [open, form])

  const createCustomer = useCreateCustomer({
    onSuccess: (newCustomer) => {
      toast.success("Customer created successfully")
      onCustomerSaved(newCustomer as CustomerResponse)
      onOpenChange(false)
      form.reset(emptyValues)
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to create customer", {
        duration: 8000,
      })
      console.error("Error creating customer:", error)
    },
  })

  const onSubmit = (data: CustomerFormValues) => {
    // CreateCustomerInput: email (required) + optional first_name/last_name/phone/
    // billing_address. Split the single name field into first/last.
    const [firstName, ...rest] = data.name.trim().split(/\s+/)
    const a = data.billing_address
    const hasAddress = !!(a.line1 || a.line2 || a.city || a.state || a.postal_code)
    createCustomer.mutate({
      email: data.email,
      first_name: firstName,
      last_name: rest.join(" "),
      ...(data.phone ? { phone: data.phone } : {}),
      ...(hasAddress ? { billing_address: a } : {}),
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Create New Customer</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-10">
            <FormSection title="Details">
              <FormField
                control={form.control}
                name="name"
                rules={{
                  required: "Customer name is required",
                  minLength: { value: 3, message: "Customer name must be at least 3 characters" },
                }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Customer Name</FormLabel>
                    <FormControl>
                      <Input placeholder="Customer Name" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="email"
                rules={{
                  required: "Email is required",
                  pattern: {
                    value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
                    message: "Please enter a valid email address",
                  },
                }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input placeholder="Email" type="email" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="phone"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Phone</FormLabel>
                    <FormControl>
                      <Input placeholder="Phone" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </FormSection>

            <FormSection title="Billing Address" description="Optional — add it now or later.">
              <FormField
                control={form.control}
                name="billing_address.line1"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Address Line 1</FormLabel>
                    <FormControl>
                      <Input placeholder="Address Line 1" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="billing_address.line2"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Address Line 2</FormLabel>
                    <FormControl>
                      <Input placeholder="Address Line 2" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="billing_address.city"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>City</FormLabel>
                    <FormControl>
                      <Input placeholder="City" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="billing_address.state"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>State</FormLabel>
                    <FormControl>
                      <Input placeholder="State" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="billing_address.postal_code"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Postal Code</FormLabel>
                    <FormControl>
                      <Input placeholder="Postal Code" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="billing_address.country"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Country</FormLabel>
                    <FormControl>
                      <Input placeholder="Country" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </FormSection>

            <FormActions>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createCustomer.isPending}>
                {createCustomer.isPending ? "Creating..." : "Create Customer"}
              </Button>
            </FormActions>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
