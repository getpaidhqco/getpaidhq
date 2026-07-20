import * as React from "react";

import { cn } from "@/lib/utils";

function PageHeader({
  eyebrow,
  title,
  description,
  actions,
  className,
}: {
  eyebrow?: string;
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}) {
  // Underline anchors a multi-line header block (title + description/eyebrow).
  // For a single-line title + actions, whitespace + type contrast is enough.
  const bordered = Boolean(description || eyebrow);

  return (
    <header
      data-slot="page-header"
      className={cn(
        "flex flex-wrap items-end justify-between gap-4",
        bordered && "border-b border-border pb-5",
        className,
      )}
    >
      <div className="min-w-0 flex-1">
        {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">{title}</h1>
        {description ? (
          <p className="mt-1 max-w-2xl text-pretty text-sm text-muted-foreground">
            {description}
          </p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex flex-wrap items-center gap-2">{actions}</div>
      ) : null}
    </header>
  );
}

export { PageHeader };
