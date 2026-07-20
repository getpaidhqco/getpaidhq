import * as React from "react"

import { cn } from "@/lib/utils"

/**
 * Two-column settings row: label + description on the left, controls on the right.
 * Rows stack with a hairline divider between them — no need for sibling <Separator>s.
 */
export function SettingsRow({
  title,
  description,
  children,
  className,
}: {
  title: string
  description?: React.ReactNode
  children: React.ReactNode
  className?: string
}) {
  return (
    <section
      className={cn(
        "grid gap-x-8 gap-y-3 border-b border-border py-8 first:pt-0 last:border-b-0 last:pb-0 sm:grid-cols-3",
        className,
      )}
    >
      <div className="space-y-1">
        <h3 className="text-sm font-medium text-foreground">{title}</h3>
        {description ? (
          <p className="text-sm text-pretty text-muted-foreground">{description}</p>
        ) : null}
      </div>
      <div className="space-y-4 sm:col-span-2">{children}</div>
    </section>
  )
}
