"use client";

import * as React from "react";
import { Download } from "lucide-react";
import { startOfYear } from "date-fns";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import type { DateRange } from "react-day-picker";

import { useAuth } from "@getpaidhq/auth";
import {
  fetchActiveSubscribers,
  fetchCustomerChurnRates,
  fetchMrrArrData,
  fetchRefunds,
} from "@/app/(dashboard)/dashboard/data";
import { ActivityFeed } from "@/components/dashboard/activity-feed";
import { KPIStrip, type KPI } from "@/components/dashboard/kpi-strip";
import { RevenueChart } from "@/components/dashboard/revenue-chart";
import { TransactionsTable } from "@/components/dashboard/transactions-table";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";
import DatePicker from "@/app/(dashboard)/dashboard/_components/date-picker";
import { formatCurrency } from "@/lib/currency";
import type { RecurringRevenue } from "@/lib/schemas/recurring-revenue";

type Series = RecurringRevenue[] | undefined;

function useDashboardKpis(startDate: Date, endDate: Date) {
  const { getAuthHeaders } = useAuth();
  const start = startDate.toISOString().split("T")[0]!;
  const end = endDate.toISOString().split("T")[0]!;

  const mrr = useQuery<Series>({
    queryKey: ["mrr", start, end],
    queryFn: async () =>
      fetchMrrArrData({ type: "mrr", startDate: start, endDate: end, authHeaders: await getAuthHeaders() }),
    placeholderData: keepPreviousData,
  });

  const subscribers = useQuery<Series>({
    queryKey: ["active-subscribers", start, end],
    queryFn: async () =>
      fetchActiveSubscribers({ startDate: start, endDate: end, authHeaders: await getAuthHeaders() }),
    placeholderData: keepPreviousData,
  });

  const churn = useQuery<Series>({
    queryKey: ["churn", start, end],
    queryFn: async () =>
      fetchCustomerChurnRates({ startDate: start, endDate: end, authHeaders: await getAuthHeaders() }),
    placeholderData: keepPreviousData,
  });

  const refunds = useQuery<Series>({
    queryKey: ["refunds", start, end],
    queryFn: async () =>
      fetchRefunds({ startDate: start, endDate: end, authHeaders: await getAuthHeaders() }),
    placeholderData: keepPreviousData,
  });

  return { mrr, subscribers, churn, refunds };
}

function lastValue(series: Series): { total: number; growth: number; spark: number[] } {
  const arr = series ?? [];
  const last = arr[arr.length - 1];
  return {
    total: last?.total ?? 0,
    growth: last?.growth_mom ?? 0,
    spark: arr.map((p) => p.total ?? 0),
  };
}

export default function Dashboard() {
  const [range, setRange] = React.useState<{ from: Date; to: Date }>({
    from: startOfYear(new Date()),
    to: new Date(),
  });

  const { mrr, subscribers, churn, refunds } = useDashboardKpis(range.from, range.to);

  const mrrV = lastValue(mrr.data);
  const subV = lastValue(subscribers.data);
  const churnV = lastValue(churn.data);
  const refundsV = lastValue(refunds.data);

  const fmtPct = (v: number) => `${v >= 0 ? "+" : ""}${v.toFixed(1)}%`;

  const kpis: KPI[] = [
    {
      label: "Monthly recurring revenue",
      value: formatCurrency("USD", mrrV.total),
      delta: mrrV.growth ? fmtPct(mrrV.growth) : undefined,
      deltaDirection: mrrV.growth >= 0 ? "up" : "down",
      sub: "vs. last period",
      spark: mrrV.spark,
    },
    {
      label: "Active subscribers",
      value: subV.total.toLocaleString(),
      delta: subV.growth ? fmtPct(subV.growth) : undefined,
      deltaDirection: subV.growth >= 0 ? "up" : "down",
      sub: "rolling period",
      spark: subV.spark,
    },
    {
      label: "Churn rate",
      value: `${(churnV.total).toFixed(1)}%`,
      delta: churnV.growth ? fmtPct(churnV.growth) : undefined,
      deltaDirection: churnV.growth <= 0 ? "up" : "down",
      tone: churnV.growth <= 0 ? "success" : "danger",
      sub: "lower is better",
      spark: churnV.spark,
    },
    {
      label: "Refunds",
      value: formatCurrency("USD", refundsV.total),
      delta: refundsV.growth ? fmtPct(refundsV.growth) : undefined,
      deltaDirection: refundsV.growth <= 0 ? "up" : "down",
      tone: refundsV.growth <= 0 ? "success" : "danger",
      sub: "in selected range",
      spark: refundsV.spark,
    },
  ];

  const handleDateChange = (next: DateRange) => {
    if (next.from && next.to) setRange({ from: next.from, to: next.to });
  };

  return (
    <div className="flex flex-col gap-10">
      <PageHeader
        title="Dashboard"
        actions={
          <>
            <DatePicker onDateChange={handleDateChange} />
            <Button variant="outline" size="sm">
              <Download className="size-3.5" data-icon="inline-start" />
              Export
            </Button>
          </>
        }
      />

      <KPIStrip items={kpis} />

      <div className="grid grid-cols-1 gap-10 xl:grid-cols-3">
        <div className="xl:col-span-2">
          <RevenueChart />
        </div>
        <ActivityFeed />
      </div>

      <TransactionsTable />
    </div>
  );
}
