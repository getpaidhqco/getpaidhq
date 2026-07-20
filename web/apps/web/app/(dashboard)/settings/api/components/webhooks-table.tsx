"use client"

import { ExternalLink } from "lucide-react"
import { format } from "date-fns"

import { useWebhooks } from "@getpaidhq/react-sdk"
import { type Column, DataTable } from "@/components/ui/data-table"

// The SDK exposes no WebhookResponse type or delete operation; the list
// envelope returns these fields, which the display reads.
type Webhook = {
  id: string
  url: string
  events: string[]
  created_at: string
}

export function WebhooksTable() {
  const { data, isLoading } = useWebhooks()

  const webhooks = (data?.data as Webhook[]) ?? []

  const columns: Column<Webhook>[] = [
    {
      key: "url",
      header: "Endpoint",
      render: (w) => (
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate font-mono text-xs text-foreground">{w.url}</span>
          <a
            href={w.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-muted-foreground transition hover:text-foreground"
            aria-label="Open in new tab"
          >
            <ExternalLink className="size-3.5" />
          </a>
        </div>
      ),
    },
    {
      key: "events",
      header: "Events",
      render: (w) => (
        <span className="text-sm text-muted-foreground">
          {w.events?.includes("*") ? "All events" : w.events?.join(", ")}
        </span>
      ),
    },
    {
      key: "created_at",
      header: "Created",
      render: (w) => (
        <span className="font-mono text-xs tabular text-muted-foreground">
          {format(new Date(w.created_at), "MMM d, yyyy")}
        </span>
      ),
    },
  ]

  if (isLoading) {
    return (
      <div className="py-6 text-center text-sm text-muted-foreground">
        Loading webhooks…
      </div>
    )
  }

  return (
    <DataTable
      columns={columns}
      rows={webhooks}
      empty={
        <div className="py-6 text-center text-sm text-muted-foreground">
          No webhooks configured yet. Add one to receive event notifications.
        </div>
      }
    />
  )
}
