"use client"
import { createContext, useContext, ReactNode, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@getpaidhq/auth";
import type { InvoiceResponse } from "@getpaidhq/sdk";
import { fetchInvoice } from "./data";

// Invoices are read-only (no update/action API), so the context only exposes
// the loaded invoice and a refresh.
interface InvoiceContextType {
  invoice: InvoiceResponse | undefined;
  isLoading: boolean;
  error: Error | null;
  refresh: () => void;
}

const InvoiceContext = createContext<InvoiceContextType | undefined>(undefined);

export function InvoiceProvider({
  children,
  id,
  initialData,
}: {
  children: ReactNode;
  id: string;
  initialData?: InvoiceResponse;
}) {
  const { getAuthHeaders } = useAuth();

  const {
    data: invoice,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['invoice', id],
    queryFn: async () => {
      const headers = await getAuthHeaders();
      return fetchInvoice(id, headers);
    },
    initialData,
    enabled: !!process.env.NEXT_PUBLIC_API_URL && !!id,
  });

  const refresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const contextValue: InvoiceContextType = {
    invoice,
    isLoading,
    error,
    refresh,
  };

  return (
    <InvoiceContext.Provider value={contextValue}>
      {children}
    </InvoiceContext.Provider>
  );
}

export function useInvoice() {
  const context = useContext(InvoiceContext);
  if (context === undefined) {
    throw new Error("useInvoice must be used within an InvoiceProvider");
  }
  return context;
}
