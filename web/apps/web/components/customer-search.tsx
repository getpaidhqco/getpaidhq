"use client"

import * as React from "react"
import { useEffect, useState } from "react"
import { Combobox, ComboboxOption } from "@/components/ui/combobox"
import { cn } from "@/lib/utils"
import { useDebounce } from "@/hooks/use-debounce"
import { CustomerDialog } from "@/components/customer-dialog"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { useCustomers } from "@getpaidhq/react-sdk"
import type { CustomerResponse } from "@getpaidhq/sdk"

interface CustomerSearchProps {
  value?: string
  onValueChange?: (value: string, customer?: CustomerResponse) => void
  className?: string
  disabled?: boolean
}

export function CustomerSearch({
  value,
  onValueChange,
  className,
  disabled = false,
}: CustomerSearchProps) {
  const queryClient = useQueryClient()
  const [customers, setCustomers] = useState<CustomerResponse[]>([])
  const [search, setSearch] = useState("")
  const [options, setOptions] = useState<ComboboxOption[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const debouncedSearch = useDebounce(search, 300)
  const customersQuery = useCustomers({}, { refetchOnWindowFocus: false })

  useEffect(() => {
    if (customersQuery.data) {
      setCustomers(customersQuery.data.data ?? [])
    }
  }, [customersQuery.data])

  const isLoading = customersQuery.isLoading || customersQuery.isFetching
  const error = customersQuery.error

  // Surface fetch errors as a toast (in an effect — never during render).
  useEffect(() => {
    if (error) {
      toast.error("Failed to fetch customers", {
        description: error instanceof Error ? error.message : "Unknown error",
        duration: 5000,
      })
      console.error("Error fetching customers:", error)
    }
  }, [error])

  // Update options when customers change
  useEffect(() => {
    const newOptions: ComboboxOption[] = [
      { value: "create", label: "➕ Create New Customer" },
      ...customers.map((customer) => ({
        value: customer.id,
        label: `${customer.first_name} ${customer.last_name} (${customer.email})`,
      })),
    ]
    setOptions(newOptions)
  }, [customers])

  // Handle value change
  const handleValueChange = (newValue: string) => {
    if (newValue === "create") {
      setDialogOpen(true)
      return
    }

    const selectedCustomer = customers.find((c) => c.id === newValue)
    if (onValueChange) {
      onValueChange(newValue, selectedCustomer)
    }
  }

  // Handle customer creation
  const handleCustomerSaved = (newCustomer: CustomerResponse) => {
    // Update the search query cache with the new customer
    queryClient.setQueryData(["customers", debouncedSearch], (oldData: any) => {
      if (!oldData) {
        return { data: [newCustomer] }
      }
      return {
        ...oldData,
        data: [newCustomer, ...(oldData.data || [])],
      }
    })

    // Update the initial customers query cache with the new customer
    queryClient.setQueryData(["initialCustomers"], (oldData: any) => {
      if (!oldData) {
        return { data: [newCustomer] }
      }
      const updatedData = [newCustomer, ...(oldData.data || [])]
      return {
        ...oldData,
        data: updatedData.slice(0, 10), // Keep only the first 10 customers
      }
    })

    toast.success("Customer added to search results")

    if (onValueChange) {
      onValueChange(newCustomer.id, newCustomer)
    }
  }

  return (
    <div className={cn("relative", className)}>
      <Combobox
        options={options}
        value={value}
        onValueChange={handleValueChange}
        onSearchChange={setSearch}
        placeholder="Select or search customers"
        emptyMessage={isLoading ? "Loading..." : "No customers found."}
        disabled={disabled}
      />

      <CustomerDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onCustomerSaved={handleCustomerSaved}
      />
    </div>
  )
}
