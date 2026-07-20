"use client"

import { useState } from "react"
import { Plus } from "lucide-react"

import { Button } from "@/components/ui/button"
import { SectionHead } from "@/components/ui/section-head"
import { AddWebhookModal } from "./add-webhook-modal"
import { WebhooksTable } from "./webhooks-table"

export function WebhooksSection() {
  const [open, setOpen] = useState(false)

  return (
    <section>
      <SectionHead
        title="Webhooks"
        action={
          <Button size="sm" onClick={() => setOpen(true)}>
            <Plus data-icon="inline-start" />
            Add webhook
          </Button>
        }
      />
      <WebhooksTable />
      <AddWebhookModal open={open} onOpenChange={setOpen} />
    </section>
  )
}
