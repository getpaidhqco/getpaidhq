import * as React from "react";

import { cn } from "@/lib/utils";

export type KpiRowItem = {
  label: string;
  value: React.ReactNode;
  delta?: string;
  /** "+" delta is success-toned, "-" is destructive. Force override here. */
  deltaTone?: "success" | "danger" | "muted";
  sub?: string;
};

/**
 * Borderless KPI row. Cells separated by space + typography only —
 * no outer card, no internal vertical hairlines. Defaults to 4-up on md.
 */
export function KpiRow({
  items,
  className,
  cols = 4,
}: {
  items: KpiRowItem[];
  className?: string;
  cols?: 3 | 4 | 5;
}) {
  const colsCls =
    cols === 3
      ? "grid-cols-2 md:grid-cols-3"
      : cols === 5
        ? "grid-cols-2 md:grid-cols-3 lg:grid-cols-5"
        : "grid-cols-2 md:grid-cols-4";

  return (
    <div className={cn("grid gap-x-10 gap-y-6", colsCls, className)}>
      {items.map((k) => (
        <Kpi key={k.label} item={k} />
      ))}
    </div>
  );
}

function Kpi({ item }: { item: KpiRowItem }) {
  const tone =
    item.deltaTone ??
    (item.delta?.trim().startsWith("-") ? "danger" : item.delta ? "success" : "muted");

  return (
    <div className="min-w-0">
      <div className="eyebrow truncate" title={item.label}>
        {item.label}
      </div>
      <div className="num-display mt-1 truncate text-3xl tabular text-foreground">
        {item.value}
      </div>
      {(item.delta || item.sub) && (
        <div className="mt-1 flex items-baseline gap-2 text-xs">
          {item.delta ? (
            <span
              className={cn(
                "font-mono uppercase tracking-wider",
                tone === "success" && "text-success",
                tone === "danger" && "text-destructive",
                tone === "muted" && "text-muted-foreground",
              )}
            >
              {item.delta}
            </span>
          ) : null}
          {item.sub ? (
            <span className="truncate text-muted-foreground">{item.sub}</span>
          ) : null}
        </div>
      )}
    </div>
  );
}
