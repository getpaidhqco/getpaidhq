"use client"

import * as React from "react"
import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import {
  useUpdateProduct,
  productResolvers,
  type UpdateProductFormValues,
} from "@getpaidhq/react-sdk"

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
import { SectionHead } from "@/components/ui/section-head"
import { Textarea } from "@/components/ui/textarea"
import { useEditProduct } from "@/app/(dashboard)/products/[id]/context/product-context"

export function DetailsCard() {
  const { product, refreshProduct } = useEditProduct()

  const form = useForm<UpdateProductFormValues>({
    resolver: productResolvers.update,
    defaultValues: {
      name: product.name,
      description: product.description || "",
    },
  })

  useEffect(() => {
    form.reset({
      name: product.name,
      description: product.description || "",
    })
  }, [product, form])

  const updateProduct = useUpdateProduct(product.id, {
    onSuccess: () => {
      toast.success("Product updated")
      refreshProduct()
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to update product", { duration: 8000 })
    },
  })

  const isDirty = form.formState.isDirty
  const isPending = updateProduct.isPending

  const onSubmit = (values: UpdateProductFormValues) => updateProduct.mutate(values)

  return (
    <section className="space-y-4">
      <SectionHead
        title="Details"
        action={
          isDirty ? (
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => form.reset()}
                disabled={isPending}
              >
                Discard
              </Button>
              <Button
                type="button"
                size="sm"
                onClick={form.handleSubmit(onSubmit)}
                disabled={isPending}
              >
                {isPending ? "Saving…" : "Save changes"}
              </Button>
            </div>
          ) : null
        }
      />

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Name</FormLabel>
                <FormControl>
                  <Input placeholder="Pro plan" {...field} value={field.value ?? ""} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Description</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder="What customers get when they buy this."
                    rows={4}
                    {...field}
                    value={field.value ?? ""}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </form>
      </Form>
    </section>
  )
}
