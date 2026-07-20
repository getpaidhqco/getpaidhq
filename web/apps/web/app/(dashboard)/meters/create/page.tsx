import { Metadata } from "next"
import { MeterForm } from "./components/meter-form"
import { PageHeader } from "@/components/ui/page-header"

export const metadata: Metadata = {
  title: "Create Meter",
}

export default function CreateMeterPage() {
  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHeader
        title="New meter"
        description="Define how usage events are aggregated into a billable quantity."
      />
      <MeterForm />
    </div>
  )
}
