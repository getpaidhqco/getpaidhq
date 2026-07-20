"use client";

import * as React from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { format, subMonths } from "date-fns";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { useAuth } from "@getpaidhq/auth";
import { fetchMrrArrData } from "@/app/(dashboard)/dashboard/data";
import { SectionHead } from "@/components/ui/section-head";
import type { RecurringRevenue } from "@/lib/schemas/recurring-revenue";

const RANGES = [
  { label: "1M", months: 1 },
  { label: "3M", months: 3 },
  { label: "12M", months: 12 },
] as const;

type RangeLabel = (typeof RANGES)[number]["label"];

export function RevenueChart() {
  const { getAuthHeaders } = useAuth();
  const [rangeLabel, setRangeLabel] = React.useState<RangeLabel>("12M");
  const range = RANGES.find((r) => r.label === rangeLabel)!;

  const endDate = React.useMemo(() => new Date(), []);
  const startDate = React.useMemo(
    () => subMonths(endDate, range.months),
    [endDate, range.months],
  );

  const query = useQuery<RecurringRevenue[] | undefined>({
    queryKey: ["mrr", startDate.toISOString(), endDate.toISOString()],
    queryFn: async () =>
      fetchMrrArrData({
        type: "mrr",
        startDate: startDate.toISOString().split("T")[0]!,
        endDate: endDate.toISOString().split("T")[0]!,
        authHeaders: await getAuthHeaders(),
      }),
    placeholderData: keepPreviousData,
  });

  const data = (query.data ?? []).map((d) => ({
    month: (() => {
      try {
        return format(new Date(d.period), "MMM");
      } catch {
        return d.period;
      }
    })(),
    mrr: typeof d.total === "number" ? d.total : Number(d.total ?? 0),
  }));

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        title="MRR over time"
        subtitle="Recurring revenue across all customers and currencies"
        action={
          <div role="tablist" className="inline-flex items-center gap-0.5">
            {RANGES.map((r) => (
              <button
                key={r.label}
                type="button"
                role="tab"
                aria-selected={rangeLabel === r.label}
                onClick={() => setRangeLabel(r.label)}
                className={
                  rangeLabel === r.label
                    ? "rounded-sm bg-muted px-2 py-1 font-mono text-[10px] uppercase tracking-wider text-foreground"
                    : "rounded-sm px-2 py-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground hover:text-foreground"
                }
              >
                {r.label}
              </button>
            ))}
          </div>
        }
      />

      <div className="h-72">
        {query.isLoading || data.length === 0 ? (
          <div className="grid h-full place-items-center text-xs text-muted-foreground">
            {query.isLoading ? "Loading…" : "No revenue data for this range."}
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 10, right: 12, bottom: 0, left: -8 }}>
              <defs>
                <linearGradient id="grad-mrr" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--chart-1)" stopOpacity={0.3} />
                  <stop offset="100%" stopColor="var(--chart-1)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid vertical={false} stroke="var(--border)" strokeDasharray="3 3" />
              <XAxis
                dataKey="month"
                stroke="var(--muted-foreground)"
                tick={{ fontSize: 11 }}
                tickLine={false}
                axisLine={false}
              />
              <YAxis
                stroke="var(--muted-foreground)"
                tick={{ fontSize: 11 }}
                tickLine={false}
                axisLine={false}
                tickFormatter={(v) =>
                  v === 0 ? "$0" : `$${((v as number) / 1000).toFixed(0)}k`
                }
                width={40}
              />
              <Tooltip
                cursor={{ stroke: "var(--border-strong)" }}
                contentStyle={{
                  background: "var(--popover)",
                  border: "1px solid var(--border)",
                  borderRadius: "calc(var(--radius) * 1)",
                  fontSize: 12,
                  color: "var(--popover-foreground)",
                  boxShadow: "none",
                }}
                labelStyle={{
                  color: "var(--muted-foreground)",
                  fontSize: 10,
                  textTransform: "uppercase",
                  letterSpacing: "0.08em",
                  marginBottom: 4,
                }}
                formatter={(value) => {
                  const num = typeof value === "number" ? value : Number(value);
                  return [`$${num.toLocaleString()}`, "MRR"];
                }}
              />
              <Area
                type="monotone"
                dataKey="mrr"
                stroke="var(--chart-1)"
                strokeWidth={1.75}
                fill="url(#grad-mrr)"
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>

      <div className="flex items-center gap-4 text-xs text-muted-foreground">
        <span className="inline-flex items-center gap-1.5">
          <span className="size-1.5 rounded-full bg-chart-1" />
          MRR
        </span>
      </div>
    </section>
  );
}
