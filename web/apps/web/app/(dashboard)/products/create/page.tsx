import {Metadata} from "next"
import {Heading} from '@/components/atoms/heading'
import ProductForm from "@/app/(dashboard)/products/create/components/product-form";

export const metadata: Metadata = {
  title: "Create Product",
}

export default async function ProductPage() {

  return (
    <>
      <div className="mt-4 lg:mt-8">
        <div className="flex items-center gap-4">
          <Heading>Create Product</Heading>
        </div>

        <ProductForm/>
      </div>
    </>
  )
}
