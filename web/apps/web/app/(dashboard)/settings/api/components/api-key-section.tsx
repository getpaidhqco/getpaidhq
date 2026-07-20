"use client"

import { useState } from "react"
import { Plus } from "lucide-react"

import { Button } from "@/components/ui/button"
import { SectionHead } from "@/components/ui/section-head"
import { ApiKeysTable } from "./api-keys-table"
import { CreateApiKeyModal } from "./create-api-key-modal"

export function ApiKeySection() {
  const [open, setOpen] = useState(false)

  return (
    <section>
      <SectionHead
        title="API keys"
        subtitle="Authenticate API requests. Keys are shown once at creation — keep them secret."
        action={
          <Button size="sm" onClick={() => setOpen(true)}>
            <Plus data-icon="inline-start" />
            Create API key
          </Button>
        }
      />
      <div className="pt-4">
        <ApiKeysTable />
      </div>
      <CreateApiKeyModal open={open} onOpenChange={setOpen} />
    </section>
  )
}
