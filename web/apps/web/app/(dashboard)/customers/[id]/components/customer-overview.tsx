"use client"

import { useState, useMemo, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Tabs, TabsContent, type TabItem } from "@/src/components/ui/tabs"
import {
  CreditCard,
  DollarSign,
  Mail,
  MapPin,
  Phone,
  RefreshCw,
  ShoppingCart,
} from "lucide-react"
import { formatDistanceToNow, format } from "date-fns"
import { useCustomer, useSubscriptions, useOrders, usePayments } from "@getpaidhq/react-sdk"
import { useBreadcrumb } from "@/context/breadcrumb-context"
import { CustomerLoadingSkeleton } from "./customer-loading-skeleton"
import { H1, H4 } from "@/components/ui/typography"

interface CustomerOverviewProps {
  customerId: string
}

export function CustomerOverview({ customerId }: CustomerOverviewProps) {
  const { setItems } = useBreadcrumb()
  const [activeTab, setActiveTab] = useState("overview")

  // Fetch customer data
  const { data: customer, isLoading: isLoadingCustomer } = useCustomer(customerId)

  // Fetch related data (all SDK-typed responses)
  const { data: subscriptionsResponse } = useSubscriptions({ customer_id: customerId })
  const { data: ordersResponse } = useOrders({ customer_id: customerId, limit: 100 })
  const { data: paymentsResponse } = usePayments({ customer_id: customerId, limit: 100 })

  const subscriptions = subscriptionsResponse?.data ?? []
  const orders = ordersResponse?.data ?? []
  const payments = paymentsResponse?.data ?? []

  // Calculate metrics. Per-customer invoices and MRR are no longer exposed by the API.
  const metrics = useMemo(() => {
    if (!orders.length && !payments.length && !subscriptions.length) {
      return {
        totalOrders: 0,
        activeSubscriptions: 0,
        avgOrderValue: 0,
        lastActivityDate: null as string | null,
        status: "inactive" as "active" | "recent" | "inactive",
      }
    }

    const activeSubscriptions = subscriptions.filter((sub) =>
      ["active", "trial"].includes(sub.status),
    )

    // Average order value
    const completedOrders = orders.filter((order) => order.status === "completed")
    const avgOrderValue =
      completedOrders.length > 0
        ? completedOrders.reduce((sum, order) => sum + (order.total || 0), 0) /
          completedOrders.length
        : 0

    // Last activity (most recent order or payment)
    const allDates = [...orders.map((o) => o.created_at), ...payments.map((p) => p.created_at)]
      .filter(Boolean)
      .sort((a, b) => new Date(b).getTime() - new Date(a).getTime())

    const lastActivityDate = allDates[0] || null

    const hasActiveSubscriptions = activeSubscriptions.length > 0
    const hasRecentActivity =
      lastActivityDate &&
      new Date(lastActivityDate) > new Date(Date.now() - 90 * 24 * 60 * 60 * 1000)

    const status = hasActiveSubscriptions ? "active" : hasRecentActivity ? "recent" : "inactive"

    return {
      totalOrders: orders.length,
      activeSubscriptions: activeSubscriptions.length,
      avgOrderValue: avgOrderValue / 100, // Convert from cents
      lastActivityDate,
      status,
    }
  }, [subscriptions, orders, payments])

  const tabItems: TabItem[] = [
    { id: "overview", label: "Overview" },
    { id: "subscriptions", label: "Subscriptions" },
    { id: "orders", label: "Orders" },
    { id: "payments", label: "Payments" },
  ]

  // Set breadcrumb items when customer data is available
  useEffect(() => {
    if (customer) {
      setItems([
        { label: "Customers", href: "/customers" },
        { label: customer.email || "Customer", href: `/customers/${customer.id}` },
      ])
    }
  }, [customer, setItems])

  if (isLoadingCustomer) {
    return <CustomerLoadingSkeleton />
  }

  if (!customer) {
    return <div>Customer not found</div>
  }

  const customerName =
    customer.first_name && customer.last_name
      ? `${customer.first_name} ${customer.last_name}`
      : customer.name || customer.email || "Unknown Customer"

  const initials = customerName
    .split(" ")
    .map((n) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2)

  const statusVariant =
    metrics.status === "active" ? "success" : metrics.status === "recent" ? "info" : "muted"

  return (
    <div className="space-y-6">
      {/* Header Section */}
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          <Avatar className="h-16 w-16">
            <AvatarFallback className="text-lg">{initials}</AvatarFallback>
          </Avatar>
          <div>
            <H1 className="text-2xl">{customerName}</H1>
            <div className="flex items-center gap-2 mt-1">
              <Mail className="h-4 w-4 text-muted-foreground" />
              <span className="text-muted-foreground">{customer.email}</span>
            </div>
            {customer.phone && (
              <div className="flex items-center gap-2 mt-1">
                <Phone className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">{customer.phone}</span>
              </div>
            )}
            <div className="flex items-center gap-2 mt-2">
              <Badge variant={statusVariant}>
                {metrics.status.charAt(0).toUpperCase() + metrics.status.slice(1)}
              </Badge>
              {metrics.lastActivityDate && (
                <span className="text-sm text-muted-foreground">
                  Last active{" "}
                  {formatDistanceToNow(new Date(metrics.lastActivityDate), { addSuffix: true })}
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Subscriptions</CardTitle>
            <RefreshCw className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{metrics.activeSubscriptions}</div>
            <p className="text-xs text-muted-foreground">Currently active or in trial</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Orders</CardTitle>
            <ShoppingCart className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{metrics.totalOrders}</div>
            <p className="text-xs text-muted-foreground">
              ${metrics.avgOrderValue.toFixed(2)} avg. value
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Payments</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{payments.length}</div>
            <p className="text-xs text-muted-foreground">Total payments recorded</p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content */}
      <div className="space-y-4">
        <Tabs items={tabItems} value={activeTab} onValueChange={setActiveTab} className="w-full" />

        <TabsContent value="overview" activeValue={activeTab} className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Customer Details */}
            <Card>
              <CardHeader>
                <CardTitle>Customer Details</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <div className="text-sm font-medium text-muted-foreground">First Name</div>
                    <div>{customer.first_name || "-"}</div>
                  </div>
                  <div>
                    <div className="text-sm font-medium text-muted-foreground">Last Name</div>
                    <div>{customer.last_name || "-"}</div>
                  </div>
                  <div>
                    <div className="text-sm font-medium text-muted-foreground">Email</div>
                    <div>{customer.email || "-"}</div>
                  </div>
                  <div>
                    <div className="text-sm font-medium text-muted-foreground">Phone</div>
                    <div>{customer.phone || "-"}</div>
                  </div>
                  <div>
                    <div className="text-sm font-medium text-muted-foreground">Customer ID</div>
                    <div className="font-mono text-sm">{customer.id}</div>
                  </div>
                  <div>
                    <div className="text-sm font-medium text-muted-foreground">Created</div>
                    <div>
                      {customer.created_at
                        ? format(new Date(customer.created_at), "MMM dd, yyyy")
                        : "-"}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Billing Address */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <MapPin className="h-5 w-5" />
                  Billing Address
                </CardTitle>
              </CardHeader>
              <CardContent>
                {customer.billing_address &&
                Object.keys(customer.billing_address).length > 0 ? (
                  <div className="space-y-2">
                    <div>{customer.billing_address.line1}</div>
                    {customer.billing_address.line2 && (
                      <div>{customer.billing_address.line2}</div>
                    )}
                    <div>
                      {customer.billing_address.city}, {customer.billing_address.state}{" "}
                      {customer.billing_address.postal_code}
                    </div>
                    <div>{customer.billing_address.country}</div>
                  </div>
                ) : (
                  <div className="text-muted-foreground">No billing address on file</div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="subscriptions" activeValue={activeTab} className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Subscriptions ({subscriptions.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {subscriptions.length > 0 ? (
                <div className="space-y-4">
                  {subscriptions.map((subscription) => (
                    <div
                      key={subscription.id}
                      className="flex items-center justify-between p-4 border rounded-lg"
                    >
                      <div>
                        <div className="font-medium">Subscription #{subscription.id}</div>
                        <div className="text-sm text-muted-foreground">
                          Status: <Badge variant="secondary">{subscription.status}</Badge>
                        </div>
                        {subscription.created_at && (
                          <div className="text-sm text-muted-foreground">
                            Created {format(new Date(subscription.created_at), "MMM dd, yyyy")}
                          </div>
                        )}
                      </div>
                      <div className="text-right">
                        <div className="font-medium">{subscription.status}</div>
                        <div className="text-sm text-muted-foreground">
                          {subscription.billing_interval || "N/A"}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-6">
                  <RefreshCw className="mx-auto h-12 w-12 text-muted-foreground" />
                  <H4 className="mt-2 text-sm">No subscriptions</H4>
                  <p className="mt-1 text-sm text-muted-foreground">
                    This customer has no subscriptions.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="orders" activeValue={activeTab} className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Orders ({orders.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {orders.length > 0 ? (
                <div className="space-y-4">
                  {orders.slice(0, 10).map((order) => (
                    <div
                      key={order.id}
                      className="flex items-center justify-between p-4 border rounded-lg"
                    >
                      <div>
                        <div className="font-medium">Order #{order.id}</div>
                        <div className="text-sm text-muted-foreground">
                          Status: <Badge variant="secondary">{order.status}</Badge>
                        </div>
                        {order.created_at && (
                          <div className="text-sm text-muted-foreground">
                            {format(new Date(order.created_at), "MMM dd, yyyy")}
                          </div>
                        )}
                      </div>
                      <div className="text-right">
                        <div className="font-medium">
                          ${((order.total || 0) / 100).toFixed(2)}
                        </div>
                        <div className="text-sm text-muted-foreground">{order.currency}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-6">
                  <ShoppingCart className="mx-auto h-12 w-12 text-muted-foreground" />
                  <H4 className="mt-2 text-sm">No orders</H4>
                  <p className="mt-1 text-sm text-muted-foreground">
                    This customer hasn&apos;t placed any orders yet.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="payments" activeValue={activeTab} className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Payment History ({payments.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {payments.length > 0 ? (
                <div className="space-y-4">
                  {payments.slice(0, 10).map((payment) => (
                    <div
                      key={payment.id}
                      className="flex items-center justify-between p-4 border rounded-lg"
                    >
                      <div>
                        <div className="font-medium">Payment #{payment.id}</div>
                        <div className="text-sm text-muted-foreground">
                          Status: <Badge variant="secondary">{payment.status}</Badge>
                        </div>
                        {payment.created_at && (
                          <div className="text-sm text-muted-foreground">
                            {format(new Date(payment.created_at), "MMM dd, yyyy")}
                          </div>
                        )}
                      </div>
                      <div className="text-right">
                        <div className="font-medium">
                          ${((payment.amount || 0) / 100).toFixed(2)}
                        </div>
                        <div className="text-sm text-muted-foreground">{payment.currency}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-6">
                  <CreditCard className="mx-auto h-12 w-12 text-muted-foreground" />
                  <H4 className="mt-2 text-sm">No payments</H4>
                  <p className="mt-1 text-sm text-muted-foreground">
                    No payments have been processed for this customer.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </div>
    </div>
  )
}
