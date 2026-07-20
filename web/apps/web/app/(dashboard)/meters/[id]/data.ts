import type { MeterResponse } from "@getpaidhq/sdk";

/**
 * Fetches a meter from the API. The meters API is read-only beyond creation
 * (no update or delete), so this is the only data operation for a single meter.
 */
export async function fetchMeter(
  id: string,
  authHeader: Record<string, string>
): Promise<MeterResponse> {
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_URL}/api/meters/${id}`,
    {
      method: "GET",
      headers: authHeader,
    }
  );

  if (!response.ok) {
    throw new Error("Failed to fetch meter");
  }

  return (await response.json()) as MeterResponse;
}
