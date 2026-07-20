import * as React from "react";

import { cn } from "@/lib/utils";

export function SectionHead({
  eyebrow,
  title,
  subtitle,
  action,
  className,
  size = "default",
}: {
  eyebrow?: string;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  size?: "default" | "sm";
}) {
  return (
    <header
      className={cn(
        "flex items-end justify-between gap-3 border-b border-border",
        size === "default" ? "pb-3" : "pb-2",
        className,
      )}
    >
      <div className="min-w-0">
        {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
        <h2
          className={cn(
            "tracking-tight text-foreground",
            size === "default" ? "text-base font-semibold" : "text-sm font-semibold",
            eyebrow && "mt-0.5",
          )}
        >
          {title}
        </h2>
        {subtitle ? (
          <p className="mt-0.5 text-xs text-muted-foreground">{subtitle}</p>
        ) : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </header>
  );
}
