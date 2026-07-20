"use client"

import * as React from "react"
import { useMemo, useState } from "react"
import { ChevronDown, ChevronRight } from "lucide-react"
import { format } from "date-fns"

import { useEditProduct } from "@/app/(dashboard)/products/[id]/context/product-context"
import {
  Surface,
  SurfaceContent,
  SurfaceHeader,
  SurfaceTitle,
} from "@/components/ui/surface"

function FieldRow({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-0 py-2 text-sm">
      <dt className="text-muted-foreground">{k}</dt>
      <dd className="min-w-0 truncate text-right text-foreground">{v}</dd>
    </div>
  )
}

export default function MetadataCard() {
  const { product } = useEditProduct()
  const [rawOpen, setRawOpen] = useState(false)

  const meta = useMemo(
    () => Object.entries(product.metadata ?? {}),
    [product.metadata],
  )

  return (
    <div className="space-y-6">
      <Surface>
        <SurfaceContent>
          <dl className="divide-y divide-border">
            <FieldRow
              k="ID"
              v={
                <code className="font-mono text-xs text-muted-foreground">
                  {product.id}
                </code>
              }
            />
            {product.created_at ? (
              <FieldRow
                k="Created"
                v={
                  <span className="font-mono text-xs text-muted-foreground tabular">
                    {format(new Date(product.created_at), "MMM d, yyyy 'at' HH:mm")}
                  </span>
                }
              />
            ) : null}
            {product.updated_at ? (
              <FieldRow
                k="Updated"
                v={
                  <span className="font-mono text-xs text-muted-foreground tabular">
                    {format(new Date(product.updated_at), "MMM d, yyyy 'at' HH:mm")}
                  </span>
                }
              />
            ) : null}
            <FieldRow k="Variants" v={product.variants?.length ?? 0} />
          </dl>
        </SurfaceContent>
      </Surface>

      <Surface>
        <SurfaceHeader>
          <SurfaceTitle>Metadata</SurfaceTitle>
        </SurfaceHeader>
        <SurfaceContent>
          {meta.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No metadata. Attach custom key-value pairs via the API.
            </p>
          ) : (
            <dl className="divide-y divide-border">
              {meta.map(([k, v]) => (
                <FieldRow key={k} k={k} v={String(v)} />
              ))}
            </dl>
          )}

          <button
            type="button"
            onClick={() => setRawOpen((o) => !o)}
            className="mt-4 inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground transition hover:text-foreground"
          >
            {rawOpen ? (
              <ChevronDown className="size-3" />
            ) : (
              <ChevronRight className="size-3" />
            )}
            Raw JSON
          </button>
          {rawOpen ? (
            <pre className="mt-2 overflow-x-auto rounded-md border border-border bg-muted/40 p-3 font-mono text-[11px] leading-relaxed text-foreground">
              <code>{JSON.stringify(product, null, 2)}</code>
            </pre>
          ) : null}
        </SurfaceContent>
      </Surface>
    </div>
  )
}
