"use client"

import { Skeleton, SkeletonAvatar, SkeletonButton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader } from "@/components/ui/card"

interface ResourceDetailSkeletonProps {
  /** Whether to show avatar in header */
  showAvatar?: boolean
  /** Number of metric cards to show */
  metricsCount?: number
  /** Whether to show tabs navigation */
  showTabs?: boolean
  /** Number of detail sections to show */
  detailSections?: number
}

export function ResourceDetailSkeleton({ 
  showAvatar = false,
  metricsCount = 4,
  showTabs = true,
  detailSections = 2
}: ResourceDetailSkeletonProps) {
  return (
    <div className="space-y-6">
      {/* Header Section */}
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          {showAvatar && <SkeletonAvatar size="xl" className="w-16 h-16" />}
          
          <div className="space-y-3">
            <Skeleton className="h-8 w-64" />
            <div className="flex items-center gap-2">
              <Skeleton className="h-4 w-4" />
              <Skeleton className="h-4 w-48" />
            </div>
            <div className="flex items-center gap-2">
              <Skeleton className="h-6 w-20 rounded-full" />
              <Skeleton className="h-4 w-32" />
            </div>
          </div>
        </div>
        
        <div className="flex gap-2">
          <SkeletonButton size="sm" className="w-16" />
          <SkeletonButton size="sm" className="w-24" />
        </div>
      </div>

      {/* Metrics Cards */}
      {metricsCount > 0 && (
        <div className={`grid gap-4 md:grid-cols-2 lg:grid-cols-${metricsCount}`}>
          {Array.from({ length: metricsCount }).map((_, index) => (
            <Card key={index}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-4" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-20 mb-1" />
                <Skeleton className="h-3 w-32" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Main Content Area */}
      <div className="space-y-4">
        {/* Tabs */}
        {showTabs && (
          <div className="flex space-x-1 border-b">
            {['Overview', 'Details', 'History', 'Settings'].map((tab, index) => (
              <Skeleton key={tab} className="h-10 w-24 rounded-b-none" />
            ))}
          </div>
        )}

        {/* Content Sections */}
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {Array.from({ length: detailSections }).map((_, index) => (
              <Card key={index}>
                <CardHeader>
                  <Skeleton className="h-6 w-32" />
                  <Skeleton className="h-4 w-48" />
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    {Array.from({ length: 4 }).map((_, fieldIndex) => (
                      <div key={fieldIndex}>
                        <Skeleton className="h-4 w-20 mb-1" />
                        <Skeleton className="h-5 w-32" />
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {/* Additional Content Card */}
          <Card>
            <CardHeader>
              <Skeleton className="h-6 w-40" />
              <Skeleton className="h-4 w-64" />
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {Array.from({ length: 3 }).map((_, index) => (
                  <div key={index} className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="space-y-2">
                      <Skeleton className="h-5 w-40" />
                      <Skeleton className="h-4 w-48" />
                      <Skeleton className="h-3 w-24" />
                    </div>
                    <div className="text-right space-y-1">
                      <Skeleton className="h-5 w-20" />
                      <Skeleton className="h-4 w-16" />
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}