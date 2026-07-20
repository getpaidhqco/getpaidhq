"use client"

import * as React from "react"
import {useState, useEffect} from "react"
import {Combobox, ComboboxOption} from "@/components/ui/combobox"
import {cn} from "@/lib/utils"
import {useDebounce} from "@/hooks/use-debounce"
import {ProductCreateDialog} from "@/components/product-create-dialog"
import {useAuth} from "@getpaidhq/auth";
import {useQuery, useQueryClient} from "@tanstack/react-query"
import {useProducts, useProduct} from "@getpaidhq/react-sdk"
import type {ProductResponse as Product} from "@getpaidhq/sdk"
import {toast} from "sonner";

interface ProductSearchProps {
  value?: string
  onValueChange?: (value: string, product?: Product) => void
  className?: string
  disabled?: boolean
}

export function ProductSearch({
  value,
  onValueChange,
  className,
  disabled = false,
}: ProductSearchProps) {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState("")
  const [options, setOptions] = useState<ComboboxOption[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const debouncedSearch = useDebounce(search, 300)

  // Use SDK to fetch products
  const searchParams = debouncedSearch.length >= 3 ? { search: debouncedSearch, limit: 20 } : { limit: 10 }
  const {
    data: productsResponse,
    isLoading,
    error
  } = useProducts(searchParams)

  const products = productsResponse?.data || []
  console.log("Products from SDK:", products)

  // Show error toast if query fails
  useEffect(() => {
    if (error) {
      toast.error("Failed to fetch products", {
        description: error instanceof Error ? error.message : "Unknown error",
        duration: 5000,
      });
      console.error("Error fetching products:", error);
    }
  }, [error]);

  // Update options when products change
  useEffect(() => {
    const newOptions: ComboboxOption[] = [
      {value: "create", label: "➕ Create New Product"},
      ...products.map((product) => ({
        value: product.id,
        label: product.name,
      })),
    ]
    setOptions(newOptions)
  }, [products])

  // State to track if we're fetching product details
  const [selectedProductId, setSelectedProductId] = useState<string | null>(null)

  // Fetch full product details when a product is selected
  const {
    data: selectedProductDetails,
    isLoading: isLoadingProductDetails,
  } = useProduct(selectedProductId || "", {
    enabled: !!selectedProductId
  })

  // Handle value change
  const handleValueChange = async (newValue: string) => {
    if (newValue === "create") {
      setDialogOpen(true)
      return
    }

    const selectedProduct = products.find((p) => p.id === newValue)
    
    // If the selected product doesn't have variants/prices, fetch full details
    if (selectedProduct && (!selectedProduct.variants || selectedProduct.variants.length === 0)) {
      console.log("Product lacks variant/price data, fetching full details...")
      setSelectedProductId(newValue)
      return
    }

    if (onValueChange) {
      onValueChange(newValue, selectedProduct)
    }
  }

  // When product details are loaded, call the callback
  useEffect(() => {
    if (selectedProductDetails && selectedProductId && onValueChange) {
      console.log("Full product details loaded:", selectedProductDetails)
      onValueChange(selectedProductId, selectedProductDetails)
      setSelectedProductId(null) // Reset
    }
  }, [selectedProductDetails, selectedProductId, onValueChange])

  // Handle product creation
  const handleProductCreated = (newProduct: Product) => {
    // Update the search query cache with the new product
    queryClient.setQueryData(['products', debouncedSearch], (oldData: any) => {
      // If there's no old data, create a new data structure
      if (!oldData) {
        return {data: [newProduct]};
      }

      // Add the new product to the beginning of the list
      return {
        ...oldData,
        data: [newProduct, ...(oldData.data || [])]
      };
    });

    // Update the initial products query cache with the new product
    queryClient.setQueryData(['initialProducts'], (oldData: any) => {
      // If there's no old data, create a new data structure
      if (!oldData) {
        return {data: [newProduct]};
      }

      // Add the new product to the beginning of the list and limit to 10
      const updatedData = [newProduct, ...(oldData.data || [])];
      return {
        ...oldData,
        data: updatedData.slice(0, 10) // Keep only the first 10 products
      };
    });

    // Show success toast
    toast.success("Product added to search results");

    // Select the new product
    if (onValueChange) {
      onValueChange(newProduct.id, newProduct)
    }
  }

  return (
    <div className={cn("relative", className)}>
      <Combobox
        options={options}
        value={value}
        onValueChange={handleValueChange}
        onSearchChange={setSearch}
        placeholder="Select or search products"
        emptyMessage={isLoading || isLoadingProductDetails ? "Loading..." : "No products found."}
        disabled={disabled || isLoadingProductDetails}
      />

      <ProductCreateDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onProductCreated={handleProductCreated}
      />
    </div>
  )
}
