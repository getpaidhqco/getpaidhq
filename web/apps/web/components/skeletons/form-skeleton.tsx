"use client"

import { Skeleton, SkeletonButton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader } from "@/components/ui/card"

interface FormSkeletonProps {
  /** Number of form sections/cards */
  sections?: number
  /** Number of fields per section */
  fieldsPerSection?: number
  /** Whether to show action buttons */
  showActions?: boolean
  /** Form title */
  showTitle?: boolean
}

export function FormSkeleton({ 
  sections = 2, 
  fieldsPerSection = 4,
  showActions = true,
  showTitle = true
}: FormSkeletonProps) {
  return (
    <div className="space-y-6">
      {/* Form Title */}
      {showTitle && (
        <div className="space-y-2">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-96" />
        </div>
      )}

      {/* Form Sections */}
      {Array.from({ length: sections }).map((_, sectionIndex) => (
        <Card key={sectionIndex}>
          <CardHeader>
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-4 w-64" />
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid gap-6 md:grid-cols-2">
              {Array.from({ length: fieldsPerSection }).map((_, fieldIndex) => (
                <div key={fieldIndex} className="space-y-2">
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="h-10 w-full" />
                  {fieldIndex % 3 === 0 && (
                    <Skeleton className="h-3 w-48" />
                  )}
                </div>
              ))}
            </div>

            {/* Additional form elements */}
            {sectionIndex === 0 && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-24 w-full" />
                </div>
                
                <div className="flex items-center gap-2">
                  <Skeleton className="h-4 w-4" />
                  <Skeleton className="h-4 w-32" />
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      ))}

      {/* Form Actions */}
      {showActions && (
        <div className="flex items-center justify-end gap-2">
          <SkeletonButton className="w-20" />
          <SkeletonButton className="w-16" />
        </div>
      )}
    </div>
  )
}