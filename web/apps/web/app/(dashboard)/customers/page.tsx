import {Metadata} from "next"
import {fetchData} from "./components/data";
import {loadAuthProvider} from "@getpaidhq/auth/server";
import CustomersPage from "./customers-page";

export const metadata: Metadata = {
  title: "Customers",
}

export default async function Page() {
  const authProvider = loadAuthProvider();
  const customers = await fetchData(await authProvider.getAuthHeader(), {page: 0, limit: 10});

  return (
    <>
      <CustomersPage customers={customers.data}/>
    </>)
}
