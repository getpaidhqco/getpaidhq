"use client"
import { createContext, useContext, ReactNode, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import type { PaymentResponse } from "@getpaidhq/sdk";
import { useAuth } from "@getpaidhq/auth";
import { AuthHeader } from "@getpaidhq/auth";

// Payments are read-only (the API exposes no refund or mutation endpoints), so
// the context only exposes the loaded payment and a refresh.
interface PaymentContextType {
  payment: PaymentResponse | undefined;
  isLoading: boolean;
  error: Error | null;
  refreshData: () => void;
}

const PaymentContext = createContext<PaymentContextType | undefined>(undefined);

export function PaymentProvider({
  children,
  payment
}: {
  children: ReactNode;
  payment: PaymentResponse;
}) {
  const { getAuthHeaders } = useAuth();
  const paymentId = payment.id;

  const fetchPayment = async (authHeaders: AuthHeader): Promise<PaymentResponse> => {
    const response = await fetch(
      `${process.env.NEXT_PUBLIC_API_URL}/api/payments/${paymentId}`,
      { headers: authHeaders }
    );
    if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
    return response.json();
  };

  const { data: fetchedPayment, isLoading, error, refetch } = useQuery({
    queryKey: ['payment', paymentId],
    queryFn: async () => {
      const headers = await getAuthHeaders();
      return fetchPayment(headers);
    },
    enabled: !!paymentId,
    initialData: payment,
  });

  const paymentData = fetchedPayment || payment;

  const refreshData = useCallback(() => {
    refetch();
  }, [refetch]);

  const contextValue: PaymentContextType = {
    payment: paymentData,
    isLoading,
    error,
    refreshData,
  };

  return (
    <PaymentContext.Provider value={contextValue}>
      {children}
    </PaymentContext.Provider>
  );
}

export function usePayment() {
  const context = useContext(PaymentContext);
  if (context === undefined) {
    throw new Error("usePayment must be used within a PaymentProvider");
  }
  return context;
}
