"use client"
import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { toast } from "sonner"
import { format } from "date-fns"
import { useMeter } from "../meter-context"
import { Badge } from "@/components/ui/badge"
import { Subheading } from "@/components/atoms/heading"
import { H1 } from "@/components/ui/typography"

// "weighted_sum" -> "Weighted sum"
const humanize = (v?: string) =>
  v ? v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ") : ""

export default function ViewMeter() {
  const router = useRouter()
  const { meter, isLoading, error } = useMeter()

  // Surface fetch errors as a toast (in an effect — never during render).
  useEffect(() => {
    if (error) {
      toast.error("Failed to fetch meter", {
        description: error instanceof Error ? error.message : "Unknown error",
        duration: 5000,
      })
      console.error("Error fetching meter:", error)
    }
  }, [error])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-foreground mx-auto"></div>
          <p className="mt-4 text-muted-foreground">Loading meter...</p>
        </div>
      </div>
    )
  }

  if (error || !meter) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <p className="text-destructive text-lg">Failed to load meter</p>
          <Button variant="outline" className="mt-4" onClick={() => router.push("/meters")}>
            Back to Meters
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div>
      {/* Header */}
      <div className="flex justify-between items-center p-6">
        <div className="flex items-center space-x-4">
          <H1 className="text-2xl">{meter.name}</H1>
          <Badge variant="info">{humanize(meter.aggregation)}</Badge>
          {meter.carry_over ? <Badge variant="success">Carry over</Badge> : null}
        </div>
      </div>

      <div className="p-6">
        <Subheading className="text-lg!">Details</Subheading>
        <Separator className="my-2 mb-4" />

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <Label>Code</Label>
            <p className="text-sm text-muted-foreground font-mono">{meter.code}</p>
          </div>

          <div>
            <Label>Aggregation</Label>
            <p className="text-sm text-muted-foreground">{humanize(meter.aggregation)}</p>
          </div>

          <div>
            <Label>Field Name</Label>
            <p className="text-sm text-muted-foreground">{meter.field_name || "—"}</p>
          </div>

          <div>
            <Label>Carry over</Label>
            <p className="text-sm text-muted-foreground">{meter.carry_over ? "Yes" : "No"}</p>
          </div>

          <div>
            <Label>Rounding Mode</Label>
            <p className="text-sm text-muted-foreground">{humanize(meter.rounding_mode) || "None"}</p>
          </div>

          <div>
            <Label>Rounding Scale</Label>
            <p className="text-sm text-muted-foreground">
              {meter.rounding_scale ?? "—"}
            </p>
          </div>

          {meter.group_by?.length ? (
            <div>
              <Label>Group By</Label>
              <p className="text-sm text-muted-foreground">{meter.group_by.join(", ")}</p>
            </div>
          ) : null}
        </div>

        {/* Metadata */}
        <div className="mt-12">
          <Subheading className="text-lg!">Metadata</Subheading>
          <Separator className="my-2" />
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-4">
            <div>
              <Label>Created At</Label>
              <p className="text-sm text-muted-foreground">
                {format(new Date(meter.created_at), "PPP p")}
              </p>
            </div>
            <div>
              <Label>Last Updated</Label>
              <p className="text-sm text-muted-foreground">
                {format(new Date(meter.updated_at), "PPP p")}
              </p>
            </div>
            <div>
              <Label>Meter ID</Label>
              <p className="text-sm text-muted-foreground font-mono">{meter.id}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
