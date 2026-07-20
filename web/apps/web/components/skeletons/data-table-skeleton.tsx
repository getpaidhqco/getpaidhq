"use client"

import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

interface DataTableSkeletonProps {
  /** Number of columns to show skeleton for */
  columnCount?: number
  /** Number of rows to show skeleton for */
  rowCount?: number
  /** Whether to show the search/filter toolbar */
  showToolbar?: boolean
  /** Whether to show pagination controls */
  showPagination?: boolean
}

// Deterministic per-column widths so the skeleton lines up with the real table
// columns and never triggers a hydration mismatch.
const cellWidth = (colIndex: number, columnCount: number) => {
  if (colIndex === 0) return "w-[120px]"
  if (colIndex === columnCount - 1) return "w-8"
  return ["w-20", "w-24", "w-16", "w-28"][colIndex % 4]
}

export function DataTableSkeleton({
  columnCount = 5,
  rowCount = 8,
  showToolbar = true,
  showPagination = true,
}: DataTableSkeletonProps) {
  return (
    <div className="space-y-4">
      {/* Toolbar skeleton */}
      {showToolbar && (
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Skeleton className="h-8 w-[150px] lg:w-[250px]" />
          </div>
          <Skeleton className="ml-auto hidden h-8 w-[70px] lg:flex" />
        </div>
      )}

      {/* Table skeleton — mirrors the real DataTable layout (no card / rounded corners) */}
      <div>
        <Table>
          <TableHeader>
            <TableRow>
              {Array.from({ length: columnCount }).map((_, index) => (
                <TableHead key={index}>
                  <Skeleton className="h-4 w-16" />
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {Array.from({ length: rowCount }).map((_, rowIndex) => (
              <TableRow key={rowIndex}>
                {Array.from({ length: columnCount }).map((_, colIndex) => (
                  <TableCell key={colIndex}>
                    <Skeleton
                      className={`h-4 ${cellWidth(colIndex, columnCount)}`}
                    />
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* Pagination skeleton */}
      {showPagination && (
        <div className="flex items-center justify-between px-2">
          <div className="flex-1" />
          <div className="flex items-center space-x-6 lg:space-x-8">
            <Skeleton className="h-8 w-[120px]" />
            <Skeleton className="h-8 w-[100px]" />
            <div className="flex items-center space-x-2">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className="h-8 w-8" />
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
