"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"

import { cn } from "@/lib/utils"

const groups: { label: string; items: { title: string; url: string }[] }[] = [
  {
    label: "Workspace",
    items: [
      { title: "Payment gateways", url: "/settings/gateways" },
      { title: "Billing", url: "/settings/billing" },
      { title: "API & webhooks", url: "/settings/api" },
    ],
  },
  {
    label: "Account",
    items: [{ title: "Profile", url: "/settings/profile" }],
  },
]

export function SettingsSidebar() {
  const pathname = usePathname()

  return (
    <nav className="w-48 shrink-0 space-y-6">
      {groups.map((group) => (
        <div key={group.label} className="space-y-1">
          <p className="eyebrow px-2 pb-1">{group.label}</p>
          {group.items.map((item) => {
            const active =
              pathname === item.url || pathname?.startsWith(item.url + "/")
            return (
              <Link
                key={item.url}
                href={item.url}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex items-center rounded-md px-2 py-1.5 text-sm transition",
                  active
                    ? "bg-muted/60 font-medium text-foreground"
                    : "text-muted-foreground hover:bg-muted/40 hover:text-foreground",
                )}
              >
                {item.title}
              </Link>
            )
          })}
        </div>
      ))}
    </nav>
  )
}
