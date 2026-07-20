import type { InvoiceResponse } from "@getpaidhq/sdk";
import { Ban, Banknote, Check, Circle, CircleOff } from "lucide-react";

/**
 * Payment status definitions with colors and icons
 */
export const paymentStatuses = [
  {
    value: "succeeded",
    label: "Succeeded",
    color: "success",
    icon: Check,
  }, {
    value: "failed",
    color: "destructive",
    label: "Failed",
    icon: Ban,
  },
  {
    value: "pending",
    color: "muted",
    label: "Pending",
    icon: Circle,
  },
  {
    value: "refunded",
    color: "info",
    label: "Refunded",
    icon: Banknote,
  },
  {
    value: "cancelled",
    color: "destructive",
    label: "Cancelled",
    icon: CircleOff,
  },
];

/**
 * Fetches invoice data from the API. Invoices are read-only (the API exposes no
 * invoice mutations or line-item endpoints), so this is the only data operation.
 * @param id Invoice ID
 * @param authHeader Authentication headers
 * @returns Invoice data
 */
export async function fetchInvoice(
  id: string,
  authHeader: Record<string, string>
): Promise<InvoiceResponse> {
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/invoices/${id}`,
    {
      method: "GET",
      headers: authHeader,
    }
  );

  if (!response.ok) {
    throw new Error("Failed to fetch invoice");
  }

  return (await response.json()) as InvoiceResponse;
}
