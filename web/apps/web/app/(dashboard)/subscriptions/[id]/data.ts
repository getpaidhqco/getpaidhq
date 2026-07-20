import {Ban, Banknote, Check, Circle, CircleOff} from "lucide-react";
import type {ListResponse, PaymentResponse} from "@getpaidhq/sdk";


export async function fetchSubscriptionPayments(id: string, options: {
    pageIndex: number
    pageSize: number
}, authHeader: Record<string, string>): Promise<ListResponse<PaymentResponse>> {

    const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/subscriptions/${id}/payments?page=${options?.pageIndex ?? 0}&limit=${options?.pageSize ?? 10}&sort_order=desc&sort_by=created_at`, {
        headers: authHeader
    })
        .then((res) => res.json());
    return rsp as ListResponse<PaymentResponse>
}


export const statuses = [
    {
        value: "succeeded",
        label: "Succeeded",
        color: 'success',
        icon: Check,
    }, {
        value: "failed",
        color: 'destructive',
        label: "Failed",
        icon: Ban,
    },
    {
        value: "pending",
        color: 'muted',
        label: "Pending",
        icon: Circle,
    },
    {
        value: "refunded",
        color: 'info',
        label: "Refunded",
        icon: Banknote,
    },
    {
        value: "cancelled",
        color: 'destructive',
        label: "Cancelled",
        icon: CircleOff,
    },
]
