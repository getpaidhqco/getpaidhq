import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "~/lib/utils"

const badgeVariants = cva(
  "inline-flex items-center justify-center rounded-full px-3 py-1 text-xs font-medium w-fit whitespace-nowrap shrink-0 [&>svg]:size-3 gap-1.5 [&>svg]:pointer-events-none transition-all duration-200 overflow-hidden",
  {
    variants: {
      variant: {
        default:
          "bg-primary/10 text-primary border border-primary/20 [a&]:hover:bg-primary/15",
        secondary:
          "bg-secondary text-secondary-foreground border border-border/50 [a&]:hover:bg-secondary/80",
        destructive:
          "bg-destructive/10 text-destructive border border-destructive/20 [a&]:hover:bg-destructive/15",
        outline:
          "text-foreground border border-border/60 [a&]:hover:bg-accent [a&]:hover:text-accent-foreground",
        success:
          "bg-green-50 text-green-700 border border-green-200/50 [a&]:hover:bg-green-100",
        warning:
          "bg-amber-50 text-amber-700 border border-amber-200/50 [a&]:hover:bg-amber-100",
        muted:
          "bg-muted text-muted-foreground border border-border/50 [a&]:hover:bg-muted/80",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "span"

  return (
    <Comp
      data-slot="badge"
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
