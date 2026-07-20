"use client"

import {Card, CardContent, CardHeader} from "@/components/ui/card"
import {Skeleton, SkeletonAvatar, SkeletonButton} from "@/components/ui/skeleton"

export function CustomerLoadingSkeleton() {
  return (
    <div className="space-y-6">
      {/* Header Section Skeleton */}
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          {/* Avatar Skeleton */}
          <SkeletonAvatar size="xl" className="w-16 h-16" />

          <div className="space-y-3">
            {/* Customer Name */}
            <Skeleton className="h-8 w-48" />

            {/* Email */}
            <div className="flex items-center gap-2">
              <Skeleton className="h-4 w-4" />
              <Skeleton className="h-4 w-64" />
            </div>

            {/* Phone (optional) */}
            <div className="flex items-center gap-2">
              <Skeleton className="h-4 w-4" />
              <Skeleton className="h-4 w-40" />
            </div>

            {/* Status and Last Active */}
            <div className="flex items-center gap-2">
              <Skeleton className="h-6 w-16 rounded-full" />
              <Skeleton className="h-4 w-32" />
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex gap-2">
          <SkeletonButton size="sm" className="w-16" />
          <SkeletonButton size="sm" className="w-32" />
        </div>
      </div>

      {/* Metrics Cards Skeleton */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
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

      {/* Tabs Skeleton */}
      <div className="space-y-4">
        <div className="flex space-x-1 border-b">
          {['Overview', 'Subscriptions', 'Orders', 'Invoices', 'Payments'].map((tab) => (
            <Skeleton key={tab} className="h-10 w-24 rounded-b-none" />
          ))}
        </div>

        {/* Tab Content Skeleton */}
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Customer Details Card */}
            <Card>
              <CardHeader>
                <Skeleton className="h-6 w-32" />
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  {Array.from({ length: 6 }).map((_, index) => (
                    <div key={index}>
                      <Skeleton className="h-4 w-20 mb-1" />
                      <Skeleton className="h-5 w-32" />
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>

            {/* Billing Address Card */}
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Skeleton className="h-5 w-5" />
                  <Skeleton className="h-6 w-32" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-4 w-24" />
                </div>
              </CardContent>
            </Card>
          </div>

          {/* MRR Breakdown Card Skeleton (conditional) */}
          <Card>
            <CardHeader>
              <Skeleton className="h-6 w-64" />
              <Skeleton className="h-4 w-96" />
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {Array.from({ length: 2 }).map((_, index) => (
                  <div key={index} className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="space-y-2">
                      <Skeleton className="h-5 w-32" />
                      <Skeleton className="h-4 w-48" />
                      <Skeleton className="h-3 w-40" />
                    </div>
                    <div className="text-right space-y-1">
                      <Skeleton className="h-5 w-20" />
                      <Skeleton className="h-4 w-24" />
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Recent Activity Card */}
          <Card>
            <CardHeader>
              <Skeleton className="h-6 w-32" />
              <Skeleton className="h-4 w-64" />
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {Array.from({ length: 3 }).map((_, index) => (
                  <div key={index} className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="space-y-2">
                      <Skeleton className="h-5 w-40" />
                      <Skeleton className="h-4 w-32" />
                      <Skeleton className="h-3 w-24" />
                    </div>
                    <div className="text-right space-y-1">
                      <Skeleton className="h-5 w-16" />
                      <Skeleton className="h-4 w-12" />
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
