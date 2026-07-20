import type { ListResponse, SubscriptionResponse } from "@getpaidhq/sdk"

export async function fetchData(
  options: {
    pageIndex: number
    pageSize: number
  },
  authHeader: Record<string, string>
): Promise<ListResponse<SubscriptionResponse>> {
  const rsp = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/subscriptions?page=${options?.pageIndex ?? 0}&limit=${options?.pageSize ?? 10}`,
    {
      headers: authHeader,
    }
  ).then((res) => res.json())

  return rsp as ListResponse<SubscriptionResponse>
}

export const statuses = [
  { value: "active", label: "Active" },
  { value: "paused", label: "Paused" },
  { value: "past_due", label: "Past Due" },
  { value: "pending", label: "Pending" },
  { value: "trial", label: "In Trial" },
  { value: "completed", label: "Completed" },
  { value: "cancelled", label: "Cancelled" },
]
