import * as React from "react"

import { cn } from "@/lib/utils"

type HeadingProps = React.HTMLAttributes<HTMLHeadingElement>
type ParagraphProps = React.HTMLAttributes<HTMLParagraphElement>
type SpanProps = React.HTMLAttributes<HTMLSpanElement>
type CodeProps = React.HTMLAttributes<HTMLElement>

function H1({ className, ...props }: HeadingProps) {
  return (
    <h1
      data-slot="h1"
      className={cn(
        "scroll-m-20 text-3xl font-semibold tracking-tight text-balance text-foreground",
        className,
      )}
      {...props}
    />
  )
}

function H2({ className, ...props }: HeadingProps) {
  return (
    <h2
      data-slot="h2"
      className={cn(
        "scroll-m-20 text-2xl font-semibold tracking-tight text-balance text-foreground",
        className,
      )}
      {...props}
    />
  )
}

function H3({ className, ...props }: HeadingProps) {
  return (
    <h3
      data-slot="h3"
      className={cn(
        "scroll-m-20 text-xl font-semibold tracking-tight text-balance text-foreground",
        className,
      )}
      {...props}
    />
  )
}

function H4({ className, ...props }: HeadingProps) {
  return (
    <h4
      data-slot="h4"
      className={cn(
        "scroll-m-20 text-base font-semibold text-balance text-foreground",
        className,
      )}
      {...props}
    />
  )
}

function P({ className, ...props }: ParagraphProps) {
  return (
    <p
      data-slot="p"
      className={cn("text-sm text-pretty text-foreground", className)}
      {...props}
    />
  )
}

function Lead({ className, ...props }: ParagraphProps) {
  return (
    <p
      data-slot="lead"
      className={cn("text-lg text-pretty text-muted-foreground", className)}
      {...props}
    />
  )
}

function Muted({ className, ...props }: ParagraphProps) {
  return (
    <p
      data-slot="muted"
      className={cn("text-sm text-pretty text-muted-foreground", className)}
      {...props}
    />
  )
}

function Small({ className, ...props }: SpanProps) {
  return (
    <span
      data-slot="small"
      className={cn("text-xs font-medium text-muted-foreground", className)}
      {...props}
    />
  )
}

function InlineCode({ className, ...props }: CodeProps) {
  return (
    <code
      data-slot="inline-code"
      className={cn(
        "rounded bg-muted px-[0.3rem] py-[0.2rem] font-mono text-sm font-medium",
        className,
      )}
      {...props}
    />
  )
}

function Blockquote({ className, ...props }: React.HTMLAttributes<HTMLQuoteElement>) {
  return (
    <blockquote
      data-slot="blockquote"
      className={cn(
        "border-l-2 pl-6 text-sm text-pretty text-muted-foreground italic",
        className,
      )}
      {...props}
    />
  )
}

export { H1, H2, H3, H4, P, Lead, Muted, Small, InlineCode, Blockquote }
