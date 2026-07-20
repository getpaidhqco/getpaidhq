import { Suspense } from "react"

import { ApiKeySection } from "./components/api-key-section"
import { WebhooksSection } from "./components/webhooks-section"

export default async function ApiSettings() {
  return (
    <div className="space-y-10">
      <ApiKeySection />

      <Suspense
        fallback={
          <div className="text-sm text-muted-foreground">Loading webhooks…</div>
        }
      >
        <WebhooksSection />
      </Suspense>
    </div>
  )
}
