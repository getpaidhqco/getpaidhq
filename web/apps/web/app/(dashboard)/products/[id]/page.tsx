import {Metadata} from "next"
import type {ProductResponse} from "@getpaidhq/sdk";
import {notFound} from 'next/navigation'
import {fetchData} from "./data";
import {loadAuthProvider} from "@getpaidhq/auth/server";
import {ProductEditPage} from "@/app/(dashboard)/products/[id]/components/product-edit-page";
import {ProductProvider} from "@/app/(dashboard)/products/[id]/context/product-context";

export const metadata: Metadata = {
  title: "Product",
}

export default async function ProductPage({params}: { params: Promise<{ id: string }> }) {
  const {id} = await params
  const authProvider = loadAuthProvider();
  const product: ProductResponse = await fetchData(id, await authProvider.getAuthHeader());

  if (!product) {
    notFound()
  }

  return (
    <ProductProvider product={product}>
      <ProductEditPage/>
    </ProductProvider>
  )
}
