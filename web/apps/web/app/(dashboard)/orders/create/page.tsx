import {Metadata} from "next"
import CreateOrderPage from "@/app/(dashboard)/orders/create/components/create-order"
import {Heading} from "@/components/atoms/heading"

export const metadata: Metadata = {
  title: "Create Order",
}

export default function Page() {
  return (
    <div className="space-y-6">
      <div>
        <Heading>Create Order</Heading>
      </div>
      <CreateOrderPage />
    </div>
  )
}
