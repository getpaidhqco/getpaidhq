import { ListResponseSchema } from "@/lib/schemas";
import { AuthHeader } from "@getpaidhq/auth";

/**
 * Fetches meters data from the API
 * @param authHeaders Authentication headers
 * @param pagination Pagination parameters
 * @returns List of meters
 */
export async function fetchData(authHeaders: AuthHeader, pagination: {
  page: number
  limit: number
}) {
  const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/meters?page=${pagination?.page ?? 0}&limit=${pagination?.limit ?? 10}`, {
    headers: authHeaders
  }).then((res) =>
    res.json()
  );

  return ListResponseSchema.parse(rsp)
}
