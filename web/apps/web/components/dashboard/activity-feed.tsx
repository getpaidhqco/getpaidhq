"use client";

import {
  ArrowRight,
  CreditCard,
  FileText,
  RefreshCw,
  TriangleAlert,
  UserPlus,
} from "lucide-react";

import { SectionHead } from "@/components/ui/section-head";
import { cn } from "@/lib/utils";

type ActivityKind = "subscription" | "invoice" | "payment" | "alert" | "user";
type ActivityTone = "success" | "warn" | "danger" | "info";

type ActivityItem = {
  kind: ActivityKind;
  title: string;
  detail: string;
  time: string;
  tone: ActivityTone;
};

const ICONS: Record<ActivityKind, React.ComponentType<{ className?: string }>> = {
  subscription: RefreshCw,
  invoice: FileText,
  payment: CreditCard,
  alert: TriangleAlert,
  user: UserPlus,
};

const TONE: Record<ActivityTone, string> = {
  success: "text-success",
  warn: "text-warning",
  danger: "text-destructive",
  info: "text-info",
};

// Placeholder until an /api/activity endpoint exists.
const ACTIVITY: ActivityItem[] = [
  {
    kind: "subscription",
    title: "New subscription upgrade",
    detail: "Customer moved to a higher tier · annual",
    time: "2m ago",
    tone: "success",
  },
  {
    kind: "payment",
    title: "Payment captured",
    detail: "Recent charge succeeded",
    time: "5m ago",
    tone: "success",
  },
  {
    kind: "alert",
    title: "Dunning recovered",
    detail: "Retry succeeded after 2 attempts",
    time: "14m ago",
    tone: "info",
  },
  {
    kind: "user",
    title: "New trial started",
    detail: "Source: marketing-site",
    time: "32m ago",
    tone: "info",
  },
  {
    kind: "alert",
    title: "Card declined",
    detail: "Insufficient funds · retry scheduled",
    time: "1h ago",
    tone: "warn",
  },
  {
    kind: "invoice",
    title: "Invoice issued",
    detail: "Net 30 terms",
    time: "1h ago",
    tone: "info",
  },
];

export function ActivityFeed() {
  return (
    <section className="flex h-full flex-col gap-3">
      <SectionHead
        title="Activity"
        subtitle="Live event stream"
        action={
          <a
            href="#"
            className="inline-flex items-center gap-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground hover:text-foreground"
          >
            View all <ArrowRight className="size-3" />
          </a>
        }
      />

      <ul role="list" className="flex-1 overflow-y-auto">
        {ACTIVITY.map((a, i) => {
          const Icon = ICONS[a.kind];
          return (
            <li
              key={i}
              className={cn(
                "flex items-start gap-3 py-2.5",
                i !== 0 && "border-t border-border",
              )}
            >
              <span className={cn("mt-0.5 shrink-0", TONE[a.tone])}>
                <Icon className="size-3.5" />
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-baseline justify-between gap-2">
                  <p className="truncate text-sm font-medium text-foreground">
                    {a.title}
                  </p>
                  <span className="shrink-0 font-mono text-[10px] tabular text-muted-foreground">
                    {a.time}
                  </span>
                </div>
                <p className="truncate text-xs text-muted-foreground">{a.detail}</p>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
