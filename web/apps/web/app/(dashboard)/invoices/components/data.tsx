import {Check, CheckCircle, Circle, CircleOff, Timer,} from "lucide-react"
import {ListResponseSchema} from "@/lib/schemas";
import {AuthHeader} from "@getpaidhq/auth";

export async function fetchData(authHeaders: AuthHeader, pagination: {
  page: number
  limit: number
}) {
  const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/invoices?page=${pagination?.page ?? 0}&limit=${pagination?.limit ?? 10}`, {
    headers: authHeaders
  }).then((res) =>
    res.json()
  );

  return ListResponseSchema.parse(rsp)
}

export const statuses = [
  {
    value: "paid",
    label: "Paid",
    icon: CheckCircle,
    color: "success",
  },
  {
    value: "pending",
    label: "Pending",
    icon: Timer,
    color: "warning",
  },
  {
    value: "partially_paid",
    label: "Partially Paid",
    icon: Check,
    color: "info",
  },
  {
    value: "overdue",
    label: "Overdue",
    icon: Circle,
    color: "destructive",
  },
  {
    value: "draft",
    label: "Draft",
    icon: Circle,
    color: "muted",
  },
  {
    value: "uncollectible",
    label: "Uncollectible",
    icon: CircleOff,
    color: "warning",
  },
  {
    value: "void",
    label: "Void",
    icon: CircleOff,
    color: "destructive",
  },
  {
    value: "cancelled",
    label: "Cancelled",
    icon: CircleOff,
    color: "destructive",
  },
]
