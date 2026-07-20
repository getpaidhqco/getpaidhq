import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGetPaidHQClient } from './use-getpaidhq-client.js';
import { QueryOptions } from '../types/index.js';

// Query keys for coupons (the discount definitions). Coupon codes hang off a
// coupon, so their key is nested under the coupon detail.
export const couponKeys = {
  all: ['coupons'] as const,
  lists: () => [...couponKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...couponKeys.lists(), filters] as const,
  details: () => [...couponKeys.all, 'detail'] as const,
  detail: (id: string) => [...couponKeys.details(), id] as const,
  codes: (couponId: string) => [...couponKeys.detail(couponId), 'codes'] as const,
};

// Discounts are the applied-coupon records (read-only on the API).
export const discountKeys = {
  all: ['discounts'] as const,
  detail: (id: string) => [...discountKeys.all, 'detail', id] as const,
};

/**
 * Hook to fetch a paginated list of coupons
 */
export function useCoupons(params?: Record<string, any>, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: couponKeys.list(params || {}),
    queryFn: () => client.coupons.list(params),
    ...options,
  });
}

/**
 * Hook to fetch a single coupon by ID
 */
export function useCoupon(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: couponKeys.detail(id),
    queryFn: () => client.coupons.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a new coupon
 */
export function useCreateCoupon(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.coupons.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: couponKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update an existing coupon (name / active / metadata only).
 */
export function useUpdateCoupon(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.coupons.update(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: couponKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: couponKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to delete a coupon
 */
export function useDeleteCoupon(options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => client.coupons.delete(id),
    onSuccess: (data, id) => {
      queryClient.invalidateQueries({ queryKey: couponKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: couponKeys.lists() });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to fetch the codes attached to a coupon
 */
export function useCouponCodes(couponId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: couponKeys.codes(couponId),
    queryFn: () => client.coupons.listCodes(couponId),
    enabled: !!couponId && (options?.enabled !== false),
    ...options,
  });
}

/**
 * Hook to create a code for a coupon
 */
export function useCreateCouponCode(couponId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: any) => client.coupons.createCode(couponId, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: couponKeys.codes(couponId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to update a coupon code
 */
export function useUpdateCouponCode(couponId: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ codeId, data }: { codeId: string; data: any }) =>
      client.coupons.updateCode(codeId, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: couponKeys.codes(couponId) });
      options?.onSuccess?.(data);
    },
    onError: (error) => {
      options?.onError?.(error);
    },
  });
}

/**
 * Hook to fetch a single applied discount by ID (read-only).
 */
export function useDiscount(id: string, options?: QueryOptions) {
  const client = useGetPaidHQClient();

  return useQuery({
    queryKey: discountKeys.detail(id),
    queryFn: () => client.discounts.get(id),
    enabled: !!id && (options?.enabled !== false),
    ...options,
  });
}
