"use client"

import { Skeleton, SkeletonButton } from "@/components/ui/skeleton"

interface PageSkeletonProps {
  /** Whether to show the create/add button */
  showCreateButton?: boolean
  /** Page title width */
  titleWidth?: string
  /** Children skeleton content */
  children: React.ReactNode
}

export function PageSkeleton({ 
  showCreateButton = false,
  titleWidth = "w-48",
  children 
}: PageSkeletonProps) {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <Skeleton className={`h-8 ${titleWidth}`} />
        {showCreateButton && (
          <SkeletonButton className="w-20" />
        )}
      </div>

      {/* Page Content */}
      {children}
    </div>
  )
}