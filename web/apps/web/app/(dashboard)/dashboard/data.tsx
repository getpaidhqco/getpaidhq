import { RecurringRevenueSchema } from "@/lib/schemas/recurring-revenue";

type Options = DateOptions & {
  type: string | "mrr" | "arr";
};
type DateOptions = {
  startDate: string;
  endDate: string;
  authHeaders: Record<string, string>;
};

async function fetchReport(url: string, headers: Record<string, string>) {
  const rsp = await fetch(url, { headers });
  if (!rsp.ok) throw new Error(`${rsp.status} ${rsp.statusText}`);
  return RecurringRevenueSchema.array().parse(await rsp.json());
}

export async function fetchMrrArrData(options: Options) {
  try {
    return await fetchReport(
      `${process.env.NEXT_PUBLIC_API_URL}/api/reports/revenue/${options.type}?start_date=${options.startDate}&end_date=${options.endDate}`,
      options.authHeaders,
    );
  } catch (e: unknown) {
    console.error("Error fetching MRR/ARR data:", e);
    return [];
  }
}

export async function fetchActiveSubscribers(options: DateOptions) {
  try {
    return await fetchReport(
      `${process.env.NEXT_PUBLIC_API_URL}/api/reports/active-subscribers?start_date=${options.startDate}&end_date=${options.endDate}`,
      options.authHeaders,
    );
  } catch (e: unknown) {
    console.error("Error fetching active subscribers:", e);
    return [];
  }
}

export async function fetchCustomerChurnRates(options: DateOptions) {
  try {
    return await fetchReport(
      `${process.env.NEXT_PUBLIC_API_URL}/api/reports/churn/rates?start_date=${options.startDate}&end_date=${options.endDate}`,
      options.authHeaders,
    );
  } catch (e: unknown) {
    console.error("Error fetching churn rates:", e);
    return [];
  }
}

export async function fetchRefunds(options: DateOptions) {
  try {
    return await fetchReport(
      `${process.env.NEXT_PUBLIC_API_URL}/api/reports/refunds?start_date=${options.startDate}&end_date=${options.endDate}`,
      options.authHeaders,
    );
  } catch (e: unknown) {
    console.error("Error fetching refunds:", e);
    return [];
  }
}
