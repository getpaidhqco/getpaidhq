import { PaginationState } from "@tanstack/react-table";
import { AuthHeader } from "@getpaidhq/auth";
import type { PaymentResponse } from "@getpaidhq/sdk";

export const statuses = [
  {
    value: "succeeded",
    label: "Succeeded",
    color: "success",
  },
  {
    value: "pending",
    label: "Pending",
    color: "warning",
  },
  {
    value: "failed",
    label: "Failed",
    color: "destructive",
  },
  {
    value: "refunded",
    label: "Refunded",
    color: "info",
  },
];

export async function fetchData(
  pagination: PaginationState,
  authHeaders: AuthHeader
) {
  try {
    const { pageIndex, pageSize } = pagination;
    const response = await fetch(
      `${process.env.NEXT_PUBLIC_API_URL}/api/payments?page=${pageIndex}&limit=${pageSize}`,
      { headers: authHeaders }
    );
    if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
    const body = await response.json();

    const payments = (body.data ?? []) as PaymentResponse[];

    return {
      data: payments,
      meta: body.meta,
    };
  } catch (error) {
    console.error("Error fetching payments:", error);
    return {
      data: [] as PaymentResponse[],
      meta: {
        total: 0,
        page: 1,
        limit: 10,
      },
    };
  }
}
