import {Metadata} from "next"
import {notFound} from 'next/navigation'
import {loadAuthProvider} from "@getpaidhq/auth/server";
import {AuthHeader} from "@getpaidhq/auth";
import type {PaymentMethodResponse, SubscriptionResponse} from "@getpaidhq/sdk";
import SubscriptionPage from "@/app/(dashboard)/subscriptions/[id]/components/subscription-page";
import {SubscriptionProvider} from "@/app/(dashboard)/subscriptions/[id]/subscription-context";

export const metadata: Metadata = {
    title: "Subscriptions",
}

async function fetchData(id: string, authHeaders: AuthHeader): Promise<SubscriptionResponse | undefined> {
    try {
        const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/subscriptions/${id}`, {
            headers: authHeaders
        })
        if (!rsp.ok) throw new Error(`${rsp.status} ${rsp.statusText}`)
        return (await rsp.json()) as SubscriptionResponse
    } catch (e: unknown) {
        console.log(e)
    }
}

async function fetchPaymentMethod(id: string, authHeaders: AuthHeader): Promise<PaymentMethodResponse | undefined> {
    try {
        const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/payment-methods/${id}`, {
            headers: authHeaders
        })
        if (!rsp.ok) throw new Error(`${rsp.status} ${rsp.statusText}`)
        return (await rsp.json()) as PaymentMethodResponse
    } catch (e: unknown) {
        console.log(e)
    }
}


export default async function Page({params}: { params: Promise<{ id: string }> }) {
    const authProvider = loadAuthProvider();
    const {id} = await params
    const authHeaders = await authProvider.getAuthHeader()
    const subscription = await fetchData(id, authHeaders);

    if (!subscription) {
        notFound()
    }
    const paymentMethod = await fetchPaymentMethod(subscription.payment_method_id, authHeaders);

    return (
        <SubscriptionProvider subscription={subscription}>
            <SubscriptionPage paymentMethod={paymentMethod} />
        </SubscriptionProvider>
    )
}
