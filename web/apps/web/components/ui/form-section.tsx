import * as React from "react"

import { cn } from "@/lib/utils"
import { H3, Muted } from "@/components/ui/typography"

/**
 * Single-column form layout primitives. Forms read top-to-bottom in one column,
 * grouped into labelled sections — easier to follow than multi-column grids.
 *
 * Usage:
 *   <FormLayout>
 *     <FormSection title="Details" description="…">
 *       <FormField … />
 *       <FormField … />
 *     </FormSection>
 *     <FormActions>…buttons…</FormActions>
 *   </FormLayout>
 */

function FormLayout({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      data-slot="form-layout"
      className={cn("mx-auto w-full max-w-2xl space-y-10", className)}
      {...props}
    />
  )
}

interface FormSectionProps extends Omit<React.HTMLAttributes<HTMLElement>, "title"> {
  title: React.ReactNode
  description?: React.ReactNode
}

function FormSection({ title, description, className, children, ...props }: FormSectionProps) {
  return (
    <section data-slot="form-section" className={cn("space-y-5", className)} {...props}>
      <div className="space-y-1">
        <H3 className="text-base">{title}</H3>
        {description ? <Muted className="text-sm">{description}</Muted> : null}
      </div>
      <div className="space-y-5">{children}</div>
    </section>
  )
}

function FormActions({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      data-slot="form-actions"
      className={cn("flex justify-end gap-3 border-t border-border pt-6", className)}
      {...props}
    />
  )
}

export { FormLayout, FormSection, FormActions }
