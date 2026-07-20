import {PaginationParams} from "@/lib/types";

export async function fetchData(headers: Record<string, string>, params: PaginationParams) {
  const searchParams = new URLSearchParams();
  if (params.page !== undefined) {
    searchParams.set('page', params.page.toString());
  }
  if (params.limit !== undefined) {
    searchParams.set('limit', params.limit.toString());
  }

  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/customers?${searchParams.toString()}`,
    {
      headers,
    }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch customers');
  }

  return response.json();
}
