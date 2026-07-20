import type React from "react"

import { PageHeader } from "@/components/ui/page-header"
import { SettingsSidebar } from "./settings-sidebar"
import { SettingsProvider } from "./settings-context"

export default function SettingsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <SettingsProvider>
      <div className="flex flex-1 flex-col gap-8">
        <PageHeader title="Settings" />
        <div className="flex gap-10">
          <SettingsSidebar />
          <main className="min-w-0 flex-1">{children}</main>
        </div>
      </div>
    </SettingsProvider>
  )
}
