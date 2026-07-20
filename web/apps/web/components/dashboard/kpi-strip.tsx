"use client";

import * as React from "react";
import { Area, AreaChart, ResponsiveContainer } from "recharts";
import { ArrowDown, ArrowUp } from "lucide-react";

import { cn } from "@/lib/utils";

export type KPI = {
  label: string;
  value: string;
  delta?: string;
  deltaDirection?: "up" | "down";
  sub?: string;
  spark?: number[];
  /** Override delta tone — default uses direction. */
  tone?: "success" | "danger" | "neutral";
};

/**
 * Borderless KPI row with inline sparkline. No outer card, no internal
 * vertical hairlines — typography + space carries the structure.
 */
export function KPIStrip({ items }: { items: KPI[] }) {
  return (
    <div className="@container">
      <div className="grid grid-cols-1 gap-x-10 gap-y-8 @3xl:grid-cols-2 @5xl:grid-cols-4">
        {items.map((kpi) => (
          <KPIItem key={kpi.label} kpi={kpi} />
        ))}
      </div>
    </div>
  );
}

function KPIItem({ kpi }: { kpi: KPI }) {
  const positive = kpi.tone
    ? kpi.tone === "success"
    : kpi.deltaDirection === "up";
  const Arrow = kpi.deltaDirection === "up" ? ArrowUp : ArrowDown;
  const data = (kpi.spark ?? []).map((v, i) => ({ i, v }));
  const sparkId = React.useId();

  return (
    <div className="min-w-0">
      <div className="eyebrow truncate" title={kpi.label}>
        {kpi.label}
      </div>
      <div className="mt-1 flex items-baseline justify-between gap-3">
        <div className="num-display truncate text-3xl tabular text-foreground @sm:text-4xl">
          {kpi.value}
        </div>
        {data.length > 1 ? (
          <div className="h-9 w-20 shrink-0">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={data} margin={{ top: 2, right: 0, bottom: 2, left: 0 }}>
                <defs>
                  <linearGradient id={`spark-${sparkId}`} x1="0" y1="0" x2="0" y2="1">
                    <stop
                      offset="0%"
                      stopColor={positive ? "var(--chart-1)" : "var(--destructive)"}
                      stopOpacity={0.35}
                    />
                    <stop
                      offset="100%"
                      stopColor={positive ? "var(--chart-1)" : "var(--destructive)"}
                      stopOpacity={0}
                    />
                  </linearGradient>
                </defs>
                <Area
                  type="monotone"
                  dataKey="v"
                  stroke={positive ? "var(--chart-1)" : "var(--destructive)"}
                  strokeWidth={1.5}
                  fill={`url(#spark-${sparkId})`}
                  isAnimationActive={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        ) : null}
      </div>
      {(kpi.delta || kpi.sub) && (
        <div className="mt-1 flex items-baseline gap-2 text-xs">
          {kpi.delta ? (
            <span
              className={cn(
                "inline-flex items-center gap-0.5 font-mono uppercase tracking-wider tabular",
                positive ? "text-success" : "text-destructive",
              )}
            >
              {kpi.deltaDirection ? <Arrow className="size-3" /> : null}
              {kpi.delta}
            </span>
          ) : null}
          {kpi.sub ? (
            <span className="truncate text-muted-foreground">{kpi.sub}</span>
          ) : null}
        </div>
      )}
    </div>
  );
}
