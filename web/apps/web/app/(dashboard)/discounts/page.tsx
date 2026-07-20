import { Metadata } from "next"
import DiscountsPage from "./discounts-page"

export const metadata: Metadata = {
  title: "Discounts",
}

export default function Page() {
  return <DiscountsPage />
}
