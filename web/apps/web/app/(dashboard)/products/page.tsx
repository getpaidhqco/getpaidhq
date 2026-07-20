import {Metadata} from "next"
import {fetchData} from "./components/data";
import {loadAuthProvider} from "@getpaidhq/auth/server";
import ProductsPage from "./products-page";

export const metadata: Metadata = {
  title: "Products",
}


export default async function Page() {
  const authProvider = loadAuthProvider();
  const products = await fetchData(await authProvider.getAuthHeader(), {page: 0, limit: 10});

  return (
    <>
      <ProductsPage products={products.data}/>
    </>)
}
