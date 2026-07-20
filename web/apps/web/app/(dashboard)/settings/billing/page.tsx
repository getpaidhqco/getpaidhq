"use client"

import { useAuth } from "@getpaidhq/auth"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { SettingsProvider } from "@/app/(dashboard)/settings/settings-context"
import InvoiceSettings from "./components/invoice-settings"
import SubscriptionSettings from "./components/subscription-settings"

export default function BillingSettings() {
  const { orgId } = useAuth()

  return (
    <SettingsProvider parentId={orgId} id="subscriptions">
      <Tabs defaultValue="subscriptions" className="space-y-6">
        <TabsList>
          <TabsTrigger value="subscriptions">Subscriptions</TabsTrigger>
          <TabsTrigger value="invoices">Invoices</TabsTrigger>
        </TabsList>
        <TabsContent value="subscriptions">
          <SubscriptionSettings />
        </TabsContent>
        <TabsContent value="invoices">
          <InvoiceSettings />
        </TabsContent>
      </Tabs>
    </SettingsProvider>
  )
}
