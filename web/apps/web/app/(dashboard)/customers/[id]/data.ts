export async function fetchData(id: string, headers: Record<string, string>) {
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/customers/${id}`,
    {
      headers,
    }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch customer');
  }

  return response.json();
}
