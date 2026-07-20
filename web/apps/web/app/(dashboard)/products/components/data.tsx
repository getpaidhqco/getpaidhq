import {Check, CheckCircle, Circle, CircleOff, Timer,} from "lucide-react"
import type {ListResponse, ProductResponse} from "@getpaidhq/sdk";
import {AuthHeader} from "@getpaidhq/auth";


export async function fetchData(authHeaders: AuthHeader, pagination: {
  page: number
  limit: number
}): Promise<ListResponse<ProductResponse>> {
  const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/products?page=${pagination?.page ?? 0}&limit=${pagination?.limit ?? 10}`, {
    headers: authHeaders
  }).then((res) =>
    res.json()
  );

  return rsp as ListResponse<ProductResponse>
}


export const labels = [
  {
    value: "bug",
    label: "Bug",
  },
  {
    value: "feature",
    label: "Feature",
  },
  {
    value: "documentation",
    label: "Documentation",
  },
]

export const statuses = [
  {
    value: "active",
    label: "Active",
    icon: Check,
  },
  {
    value: "pending",
    label: "Pending",
    icon: Circle,
  },
  {
    value: "trial",
    label: "In Trial",
    icon: Timer,
  },
  {
    value: "completed",
    label: "Completed",
    icon: CheckCircle,
  },
  {
    value: "canceled",
    label: "Canceled",
    icon: CircleOff,
  },
]
