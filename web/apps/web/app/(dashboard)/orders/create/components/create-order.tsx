"use client"

import {useState, useMemo, useCallback} from "react"
import {Plus, Trash2, Loader2} from "lucide-react"
import {useForm, useFieldArray} from "react-hook-form"
import {useRouter} from "next/navigation"
import {toast} from "sonner"
import {useCreateOrder, orderResolvers, type CreateOrderFormValues} from "@getpaidhq/react-sdk"
import type {ProductResponse, PriceResponse, CustomerResponse} from "@getpaidhq/sdk"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"
import {Separator} from "@/components/ui/separator"
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from "@/components/ui/form"
import {FormLayout, FormSection, FormActions} from "@/components/ui/form-section"
import {CustomerSearch} from "@/components/customer-search"
import {ProductSearch} from "@/components/atoms/product-search"

// Local line-item view model: tracks the chosen product/price so we can render the
// price dropdown and compute totals. The validated form only carries ids + quantity.
interface OrderItem {
  product_id: string
  price_id: string
  quantity: number
  selectedProduct?: ProductResponse
  selectedPrice?: PriceResponse
}

interface CreateOrderProps {
  onClose?: () => void
}

export default function CreateOrderPage({onClose}: CreateOrderProps = {}) {
  const router = useRouter()

  const createOrderMutation = useCreateOrder({
    onSuccess: (data) => {
      toast.success("Order created successfully")
      onClose?.()
      router.push(`/orders/${data.order.id}`)
    },
    onError: (error: Error) => {
      toast.error(`Failed to create order: ${error.message || "Unknown error"}`)
    },
  })

  const [selectedCustomer, setSelectedCustomer] = useState<CustomerResponse | null>(null)
  const [orderItems, setOrderItems] = useState<OrderItem[]>([])

  // Validation comes from the react-sdk resolver (mirrors CreateOrderRequest). The nested
  // customer/cart/options shapes are built up by the field handlers below.
  const form = useForm<CreateOrderFormValues>({
    resolver: orderResolvers.create,
    defaultValues: {
      psp_id: "",
      customer: {email: "", first_name: ""},
      cart: {currency: "USD", items: []},
      metadata: {},
    },
  })

  const {fields, append, remove} = useFieldArray({
    control: form.control,
    // cart is modelled permissively (z.record), so the field-array path isn't statically typed.
    name: "cart.items" as never,
  })

  const onSubmit = useCallback(
    (data: CreateOrderFormValues) => {
      const items = (data.cart?.items as OrderItem[] | undefined) ?? []
      if (items.length === 0) {
        toast.error("Please add at least one item to the order")
        return
      }
      if (items.some((item) => !item.product_id || !item.price_id)) {
        toast.error("Please select a product and price for all items")
        return
      }
      createOrderMutation.mutate(data)
    },
    [createOrderMutation],
  )

  // Order totals are derived from the selected prices (unit_price is in cents).
  const orderTotals = useMemo(() => {
    let subtotal = 0
    orderItems.forEach((item) => {
      if (item.selectedPrice) {
        subtotal += (item.selectedPrice.unit_price * item.quantity) / 100
      }
    })
    const tax = subtotal * 0.15
    return {subtotal, tax, total: subtotal + tax}
  }, [orderItems])

  const addItem = useCallback(() => {
    append({product_id: "", price_id: "", quantity: 1} as never)
    setOrderItems((prev) => [...prev, {product_id: "", price_id: "", quantity: 1}])
  }, [append])

  const removeItem = useCallback(
    (index: number) => {
      remove(index)
      setOrderItems((prev) => prev.filter((_, i) => i !== index))
    },
    [remove],
  )

  const handleCustomerSelect = useCallback(
    (_customerId: string, customer?: {id: string}) => {
      const selected = (customer as CustomerResponse | undefined) ?? null
      setSelectedCustomer(selected)
      form.setValue("customer", selected ? {id: selected.id} : {email: "", first_name: ""})
    },
    [form],
  )

  const handleProductSelect = useCallback(
    (index: number, product?: ProductResponse) => {
      if (!product) return
      setOrderItems((prev) => {
        const next = [...prev]
        next[index] = {
          ...next[index],
          product_id: product.id,
          selectedProduct: product,
          price_id: "",
          selectedPrice: undefined,
        }
        return next
      })
      form.setValue(`cart.items.${index}.product_id` as never, product.id as never)
      form.setValue(`cart.items.${index}.price_id` as never, "" as never)
    },
    [form],
  )

  const handlePriceSelect = useCallback(
    (index: number, priceId: string) => {
      const item = orderItems[index]
      if (!item?.selectedProduct) return

      let selectedPrice: PriceResponse | undefined
      for (const variant of item.selectedProduct.variants ?? []) {
        selectedPrice = variant.prices?.find((p) => p.id === priceId)
        if (selectedPrice) break
      }
      if (!selectedPrice) return

      setOrderItems((prev) => {
        const next = [...prev]
        next[index] = {...next[index], price_id: priceId, selectedPrice}
        return next
      })
      form.setValue(`cart.items.${index}.price_id` as never, priceId as never)
    },
    [orderItems, form],
  )

  const handleQuantityChange = useCallback(
    (index: number, quantity: number) => {
      if (quantity < 1) return
      setOrderItems((prev) => {
        const next = [...prev]
        next[index] = {...next[index], quantity}
        return next
      })
      form.setValue(`cart.items.${index}.quantity` as never, quantity as never)
    },
    [form],
  )

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <FormLayout>
          <FormSection
            title="Customer"
            description="Who this order is for, and the currency it's billed in."
          >
            <div className="space-y-2">
              <Label htmlFor="customer">Customer</Label>
              <CustomerSearch
                value={selectedCustomer?.id || ""}
                onValueChange={handleCustomerSelect}
              />
            </div>

            <FormField
              control={form.control}
              name={"cart.currency" as never}
              render={({field}) => (
                <FormItem>
                  <FormLabel>Currency</FormLabel>
                  <Select onValueChange={field.onChange} value={(field.value as string) || "USD"}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select currency" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="USD">USD - US Dollar</SelectItem>
                      <SelectItem value="EUR">EUR - Euro</SelectItem>
                      <SelectItem value="GBP">GBP - British Pound</SelectItem>
                      <SelectItem value="ZAR">ZAR - South African Rand</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </FormSection>

          <FormSection
            title="Order items"
            description="Add the products and prices to include in this order."
          >
            <div className="flex justify-end">
              <Button type="button" variant="outline" size="sm" onClick={addItem}>
                <Plus className="mr-2 h-4 w-4" />
                Add Item
              </Button>
            </div>

            {fields.length === 0 ? (
              <div className="rounded-lg border border-dashed py-6 text-center text-muted-foreground">
                No items added yet. Click &quot;Add Item&quot; to get started.
              </div>
            ) : (
              <div className="space-y-4">
                {fields.map((field, index) => (
                  <div key={field.id} className="flex items-start gap-4 rounded-lg border p-4">
                    <div className="grid flex-1 grid-cols-1 gap-4 md:grid-cols-4">
                      <div className="space-y-2 md:col-span-2">
                        <Label htmlFor={`product-${index}`}>Product</Label>
                        <ProductSearch
                          value={orderItems[index]?.selectedProduct?.id || ""}
                          onValueChange={(product) =>
                            handleProductSelect(index, product as ProductResponse | undefined)
                          }
                        />
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor={`price-${index}`}>Price</Label>
                        <Select
                          value={orderItems[index]?.price_id || ""}
                          onValueChange={(value) => handlePriceSelect(index, value)}
                          disabled={!orderItems[index]?.selectedProduct}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="Select price" />
                          </SelectTrigger>
                          <SelectContent>
                            {orderItems[index]?.selectedProduct?.variants?.flatMap((variant) =>
                              (variant.prices ?? []).map((price) => (
                                <SelectItem key={price.id} value={price.id}>
                                  ${(price.unit_price / 100).toFixed(2)}
                                  {price.billing_interval &&
                                    price.billing_interval !== "none" &&
                                    ` / ${price.billing_interval}`}
                                  {price.label && ` (${price.label})`}
                                </SelectItem>
                              )),
                            )}
                            {orderItems[index]?.selectedProduct &&
                              !orderItems[index]?.selectedProduct?.variants?.some(
                                (v) => (v.prices?.length ?? 0) > 0,
                              ) && (
                                <SelectItem value="no-pricing" disabled>
                                  No pricing options available for this product
                                </SelectItem>
                              )}
                          </SelectContent>
                        </Select>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor={`quantity-${index}`}>Quantity</Label>
                        <Input
                          type="number"
                          min={1}
                          value={orderItems[index]?.quantity || 1}
                          onChange={(e) =>
                            handleQuantityChange(index, parseInt(e.target.value) || 1)
                          }
                        />
                      </div>
                    </div>

                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeItem(index)}
                      className="mt-7"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </FormSection>

          {orderItems.length > 0 && (
            <FormSection title="Summary" description="Estimated totals for this order.">
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Subtotal</span>
                  <span>${orderTotals.subtotal.toFixed(2)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Tax (15%)</span>
                  <span>${orderTotals.tax.toFixed(2)}</span>
                </div>
                <Separator />
                <div className="flex justify-between text-lg font-medium">
                  <span>Total</span>
                  <span>${orderTotals.total.toFixed(2)}</span>
                </div>
              </div>
            </FormSection>
          )}

          <FormActions>
            <Button
              type="button"
              variant="outline"
              onClick={onClose ?? (() => router.push("/orders"))}
              disabled={createOrderMutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createOrderMutation.isPending || fields.length === 0}>
              {createOrderMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {createOrderMutation.isPending ? "Creating Order…" : "Create Order"}
            </Button>
          </FormActions>
        </FormLayout>
      </form>
    </Form>
  )
}
