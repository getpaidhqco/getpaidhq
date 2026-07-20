"use client"

import { ColumnDef } from "@tanstack/react-table"
import { DataTableColumnHeader } from "./data-table-column-header"
import { format } from "date-fns"
import { formatCurrency } from "@/lib/currency"
import { Badge } from "@/components/ui/badge"
import type { PaymentResponse } from "@getpaidhq/sdk"
import { statuses } from "@/app/(dashboard)/subscriptions/[id]/data"

export const columns: ColumnDef<PaymentResponse>[] = [
  {
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Date" />,
    cell: ({ row }) => {
      const value = row.getValue("created_at") as string
      if (!value) return null
      return <span className="tabular">{format(new Date(value), "MMM d, yyyy HH:mm")}</span>
    },
  },
  {
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => {
      const status = statuses.find((status) => status.value === row.getValue("status"))
      if (!status) return null
      return (
        <Badge variant={status.color as "success" | "warning" | "destructive" | "info" | "muted"}>
          {status.label}
        </Badge>
      )
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
  },
  {
    accessorKey: "reference",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Reference" />,
    cell: ({ row }) => (
      <span className="block max-w-[18rem] truncate font-mono text-xs text-muted-foreground">
        {row.original.reference}
      </span>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Amount" className="justify-end" />
    ),
    cell: ({ row }) => (
      <div className="text-right font-medium tabular">
        {formatCurrency(row.original.currency, row.getValue("amount"))}
      </div>
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
    enableSorting: false,
  },
]
