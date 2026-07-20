"use client"

import * as React from "react"
import {useGetPaidHQClient} from "@getpaidhq/react-sdk"
import {AsyncSelect} from "@/components/atoms/async-select";
import {ProductResponse as SdkProduct} from "@getpaidhq/sdk";

interface ProductSearchProps {
  value?: string
  onValueChange?: (product?: SdkProduct) => void
  className?: string
  disabled?: boolean
}

export function ProductSearch({
                                value,
                                onValueChange,
                                className,
                                disabled = false
                              }: ProductSearchProps) {
  const client = useGetPaidHQClient();
  const fetcher = React.useCallback(async () => {
    // Fetch products - they should already include variants and prices according to the API spec
    const productsResponse = await client.products.list({})
    return productsResponse.data
  }, [client])


  // Handle value change
  const handleValueChange = React.useCallback((product?: SdkProduct) => {
    if (onValueChange) {
      onValueChange(product)
    }
  }, [onValueChange])

  return (
    <div className={className}>
      <AsyncSelect<SdkProduct>
        value={value || ""}
        fetcher={fetcher}
        preload
        disabled={disabled}
        filterFn={(product, query) => product.name.toLowerCase().includes(query.toLowerCase())}
        renderOption={(product) => (
          <div className="flex flex-col gap-1">
            <span className="font-medium">{product.name}</span>
            {product.description && (
              <span className="text-sm text-muted-foreground">{product.description}</span>
            )}
          </div>
        )}
        getOptionValue={(product) => product.id}
        getDisplayValue={(product) => (
          <div className="flex items-center gap-2 text-left">
            {product.name}
          </div>
        )}
        notFound={<div className="py-6 text-center text-sm">No products found</div>}
        label="Products"
        placeholder="Select a product"
        onChange={handleValueChange}
        width="100%"
      />
    </div>
  )
}
