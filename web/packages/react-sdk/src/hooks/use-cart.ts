import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for carts
export const cartKeys = {
  all: ['carts'] as const,
  lists: () => [...cartKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...cartKeys.lists(), filters] as const,
  details: () => [...cartKeys.all, 'detail'] as const,
  detail: (id: string) => [...cartKeys.details(), id] as const,
};

/**
 * Hook to add item to cart
 * POST /api/carts/{id}/add
 */
export function useAddCartItem(cartId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { product_id: string; price_id: string; quantity: number }) => 
      (client as any).carts?.addItem?.(cartId, data) || Promise.reject(new Error('Cart functionality not available')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: cartKeys.detail(cartId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to remove item from cart
 * POST /api/carts/{id}/remove
 */
export function useRemoveCartItem(cartId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { product_id: string }) => 
      (client as any).carts?.removeItem?.(cartId, data) || Promise.reject(new Error('Cart functionality not available')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: cartKeys.detail(cartId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to validate coupon code for cart
 * POST /api/carts/{id}/validate-coupon
 */
export function useValidateCartCoupon(cartId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { coupon_code: string }) => 
      (client as any).carts?.validateCoupon?.(cartId, data) || Promise.reject(new Error('Cart coupon validation not available')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: cartKeys.detail(cartId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}



/**
 * Hook to fetch a single cart by ID
 */
export function useCart(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: cartKeys.detail(id),
    queryFn: () => (client as any).carts?.get?.(id) || Promise.reject(new Error('Cart functionality not available')),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}







/**
 * Hook to convert cart to payment link (checkout)
 */
export function useCheckoutCart(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => (client as any).carts?.checkout?.(id, data) || Promise.reject(new Error('Cart checkout not available')),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: cartKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: cartKeys.lists() });
      // Also invalidate payment links since we created one
      queryClient.invalidateQueries({ queryKey: ['payment-links'] });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}
