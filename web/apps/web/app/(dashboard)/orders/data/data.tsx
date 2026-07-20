import {CheckCircle, Circle, CircleOff, Timer} from "lucide-react"
import type {ListResponse, OrderResponse} from "@getpaidhq/sdk"


export async function fetchData(options: {
  pageIndex: number
  pageSize: number
  status?: string
  customer_id?: string
  search?: string
}, authHeader: Record<string, string>): Promise<ListResponse<OrderResponse>> {
  const searchParams = new URLSearchParams();

  // Add pagination parameters
  searchParams.set('page', (options?.pageIndex ?? 0).toString());
  searchParams.set('limit', (options?.pageSize ?? 10).toString());

  // Add filter parameters if provided
  if (options.status) {
    searchParams.set('status', options.status);
  }

  if (options.customer_id) {
    searchParams.set('customer_id', options.customer_id);
  }

  // Note: search parameter is not yet supported by the API
  // but we're preparing for future API support
  if (options.search) {
    // For now, we'll handle search client-side in the component
    // but this structure allows for easy API integration later
  }

  const rsp: ListResponse<OrderResponse> = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/orders?${searchParams.toString()}`,
    {headers: authHeader},
  ).then((res) => res.json());
  return rsp
}


export const statuses = [
  {
    value: "pending",
    color: 'info',
    label: "Pending",
    icon: Circle,
  },
  {
    value: "processing",
    color: 'info',
    label: "Processing",
    icon: Timer,
  },
  {
    value: "completed",
    color: 'success',
    label: "Completed",
    icon: CheckCircle,
  },
  {
    value: "cancelled",
    color: 'destructive',
    label: "Cancelled",
    icon: CircleOff,
  },
]
