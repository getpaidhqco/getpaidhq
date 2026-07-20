import * as React from "react";

import { cn } from "@/lib/utils";

function Surface({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="surface"
      className={cn(
        "flex flex-col gap-4 rounded-md border border-border bg-card text-card-foreground",
        className,
      )}
      {...props}
    />
  );
}

function SurfaceHeader({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="surface-header"
      className={cn(
        "flex items-start justify-between gap-3 px-5 pt-4",
        className,
      )}
      {...props}
    />
  );
}

function SurfaceTitle({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="surface-title"
      className={cn("text-sm font-semibold tracking-tight", className)}
      {...props}
    />
  );
}

function SurfaceDescription({
  className,
  ...props
}: React.ComponentProps<"p">) {
  return (
    <p
      data-slot="surface-description"
      className={cn("text-xs text-muted-foreground", className)}
      {...props}
    />
  );
}

function SurfaceContent({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="surface-content"
      className={cn("px-5 pb-4", className)}
      {...props}
    />
  );
}

function SurfaceFooter({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="surface-footer"
      className={cn(
        "mt-auto flex items-center justify-between gap-3 border-t border-border px-5 py-3 text-xs text-muted-foreground",
        className,
      )}
      {...props}
    />
  );
}

export {
  Surface,
  SurfaceHeader,
  SurfaceTitle,
  SurfaceDescription,
  SurfaceContent,
  SurfaceFooter,
};
