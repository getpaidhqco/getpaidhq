"use client"

import { ColumnDef } from "@tanstack/react-table"
import type { CouponResponse } from "@getpaidhq/sdk"

import { Badge } from "@/components/ui/badge"
import { DataTableColumnHeader } from "./data-table-column-header"
import { CouponRowActions } from "./coupon-row-actions"

const DISCOUNT_TYPE_LABELS: Record<string, string> = {
  percentage: "Percentage",
  fixed: "Fixed",
}

const DURATION_LABELS: Record<string, string> = {
  once: "Once",
  repeating: "Repeating",
  forever: "Forever",
}

export function createColumns(
  onEdit: (coupon: CouponResponse) => void,
): ColumnDef<CouponResponse>[] {
  return [
    {
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => <div className="flex space-x-2">{row.getValue("name")}</div>,
    },
    {
      accessorKey: "discount_type",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
      cell: ({ row }) => {
        const type = row.getValue<string>("discount_type")
        return <Badge variant="info">{DISCOUNT_TYPE_LABELS[type] ?? type}</Badge>
      },
    },
    {
      accessorKey: "duration",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Duration" />,
      cell: ({ row }) => {
        const duration = row.getValue<string>("duration")
        return <Badge variant="outline">{DURATION_LABELS[duration] ?? duration}</Badge>
      },
    },
    {
      accessorKey: "active",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
      cell: ({ row }) => {
        const active = row.getValue<boolean>("active")
        return (
          <Badge variant={active ? "success" : "muted"}>
            {active ? "Active" : "Inactive"}
          </Badge>
        )
      },
    },
    {
      id: "actions",
      cell: ({ row }) => (
        <div className="flex justify-end">
          <CouponRowActions row={row} onEdit={onEdit} />
        </div>
      ),
    },
  ]
}
