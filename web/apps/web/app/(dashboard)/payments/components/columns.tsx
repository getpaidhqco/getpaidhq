"use client"

import { ColumnDef } from "@tanstack/react-table"
import { format } from "date-fns"

import { statuses } from "../data/data"
import { DataTableColumnHeader } from "./data-table-column-header"
import type { PaymentResponse } from "@getpaidhq/sdk"
import { formatCurrency } from "@/lib/currency"
import { StatusTag, type StatusTone } from "@/components/ui/status-tag"

const STATUS_TONE: Record<string, StatusTone> = {
  succeeded: "success",
  pending: "info",
  processing: "info",
  failed: "danger",
  refunded: "neutral",
}

export const columns: ColumnDef<PaymentResponse>[] = [
  {
    accessorKey: "reference",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Reference" />
    ),
    cell: ({ row }) => (
      <span className="font-mono text-xs text-foreground">{row.getValue("reference")}</span>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      const value = row.getValue("status") as string
      const status = statuses.find((s) => s.value === value)
      const tone = STATUS_TONE[value] ?? "neutral"
      return (
        <StatusTag tone={tone}>{status?.label ?? value}</StatusTag>
      )
    },
    filterFn: (row, id, value) => value.includes(row.getValue(id)),
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Date" />
    ),
    cell: ({ row }) => (
      <span className="font-mono text-xs text-muted-foreground tabular">
        {format(new Date(row.getValue("created_at")), "MMM d · HH:mm")}
      </span>
    ),
  },
  {
    accessorKey: "amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Amount" className="text-right" />
    ),
    cell: ({ row }) => (
      <div className="flex items-baseline justify-end gap-1">
        <span className="font-medium tabular text-foreground">
          {formatCurrency(row.original.currency, row.getValue("amount"))}
        </span>
        <span className="font-mono text-[10px] uppercase text-muted-foreground">
          {row.original.currency}
        </span>
      </div>
    ),
    filterFn: (row, id, value) => value.includes(row.getValue(id)),
    enableSorting: false,
  },
  {
    accessorKey: "net_amount",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Net" className="text-right" />
    ),
    cell: ({ row }) => {
      const net = row.getValue("net_amount") as number | null | undefined
      if (net == null) return <div className="text-right text-muted-foreground">—</div>
      return (
        <div className="text-right tabular text-foreground">
          {formatCurrency(row.original.currency, net)}
        </div>
      )
    },
    filterFn: (row, id, value) => value.includes(row.getValue(id)),
    enableSorting: false,
  },
  {
    accessorKey: "order_id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Order" />
    ),
    cell: ({ row }) => {
      const v = row.getValue("order_id") as string | null | undefined
      if (!v) return <span className="text-muted-foreground">—</span>
      return (
        <span className="block max-w-[160px] truncate font-mono text-xs text-muted-foreground">
          {v}
        </span>
      )
    },
    enableSorting: false,
  },
  {
    accessorKey: "subscription_id",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Subscription" />
    ),
    cell: ({ row }) => {
      const v = row.getValue("subscription_id") as string | null | undefined
      if (!v) return <span className="text-muted-foreground">—</span>
      return (
        <span className="block max-w-[160px] truncate font-mono text-xs text-muted-foreground">
          {v}
        </span>
      )
    },
    enableSorting: false,
  },
]
