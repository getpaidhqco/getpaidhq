"use client"

import React, { createContext, useContext, useState, ReactNode } from 'react';
import type { ProductResponse } from '@getpaidhq/sdk';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@getpaidhq/auth';

interface ProductContextType {
  product: ProductResponse;
  refreshProduct: () => Promise<void>;
  isSubscription: () => boolean;
}

const ProductContext = createContext<ProductContextType | undefined>(undefined);

interface ProductProviderProps {
  product: ProductResponse;
  children: ReactNode;
}

export function ProductProvider({ product: initialProduct, children }: ProductProviderProps) {
  const [product, setProduct] = useState<ProductResponse>(initialProduct);
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();

  // Lazy query to refresh product data
  const refreshProductMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/products/${product.id}`,
        { headers: await getAuthHeaders() }
      );
      if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
      return response.json();
    },
    onSuccess: (data) => {
      setProduct(data);
      // Invalidate and refetch any queries that depend on this product
      queryClient.invalidateQueries({ queryKey: ['product', product.id] });
    },
  });

  const refreshProduct = async () => {
    await refreshProductMutation.mutateAsync();
  };

  const isSubscription = () => {
    if (!product.variants || product.variants.length === 0) {
      return false;
    }

    return product.variants.some(variant =>
      (variant.prices ?? []).some(price => price.category === 'subscription')
    );
  };

  return (
    <ProductContext.Provider value={{ product, refreshProduct, isSubscription }}>
      {children}
    </ProductContext.Provider>
  );
}

export function useEditProduct() {
  const context = useContext(ProductContext);
  if (context === undefined) {
    throw new Error('useEditProduct must be used within a ProductProvider');
  }
  return context;
}
