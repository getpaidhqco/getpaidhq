import * as React from "react";

import { cn } from "@/lib/utils";

export type StatusTone = "success" | "warn" | "danger" | "info" | "neutral";

const TONE: Record<StatusTone, string> = {
  success: "bg-success/10 text-success border-success/30",
  warn: "bg-warning/10 text-warning border-warning/30",
  danger: "bg-destructive/10 text-destructive border-destructive/30",
  info: "bg-info/10 text-info border-info/30",
  neutral: "bg-muted text-muted-foreground border-border",
};

export function StatusTag({
  tone = "neutral",
  children,
  className,
  withDot = true,
}: {
  tone?: StatusTone;
  children: React.ReactNode;
  className?: string;
  withDot?: boolean;
}) {
  return (
    <span
      data-status={tone}
      className={cn(
        "inline-flex h-5 items-center gap-1 rounded-sm border px-1.5 font-mono text-[10px] uppercase tracking-wider",
        TONE[tone],
        className,
      )}
    >
      {withDot ? <span className="size-1.5 rounded-full bg-current" /> : null}
      {children}
    </span>
  );
}
