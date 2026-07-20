// Add types specific to the React SDK
export interface QueryOptions {
  enabled?: boolean;
  staleTime?: number;
  refetchOnWindowFocus?: boolean;
  retry?: boolean | number;
  retryDelay?: number;
  onSuccess?: (data: any) => void;
  onError?: (error: Error) => void;
}
