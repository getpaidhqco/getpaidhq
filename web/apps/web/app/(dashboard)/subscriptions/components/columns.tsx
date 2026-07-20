"use client"

import { ColumnDef } from "@tanstack/react-table"

import { statuses } from "../data/data"
import { DataTableColumnHeader } from "./data-table-column-header"
import type { SubscriptionResponse } from "@getpaidhq/sdk"
import { format } from "date-fns"
import { formatCurrency } from "@/lib/currency"
import { StatusTag, type StatusTone } from "@/components/ui/status-tag"

export const columns: ColumnDef<SubscriptionResponse>[] = [
  {
    accessorKey: "customer",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Customer" />,
    cell: ({ row }) => <div className="truncate">{row.original.customer?.email}</div>,
    enableSorting: false,
    enableHiding: false,
  },

  {
    accessorKey: "status",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => {
      const value = row.getValue("status") as string
      const status = statuses.find((s) => s.value === value)
      const TONE: Record<string, StatusTone> = {
        active: "success",
        trialing: "info",
        past_due: "warn",
        paused: "neutral",
        canceled: "neutral",
        cancelled: "neutral",
        unpaid: "danger",
        incomplete: "warn",
      }
      return <StatusTag tone={TONE[value] ?? "neutral"}>{status?.label ?? value}</StatusTag>
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
  },

  {
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created At" />,
    cell: ({ row }) => {
      const value = row.getValue("created_at") as string
      if (!value) return null
      return <div className="flex space-x-2">{format(new Date(value), "MMM d, HH:mm a")}</div>
    },
  },

  {
    accessorKey: "renews_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Renews At" />,
    cell: ({ row }) => {
      const value = row.getValue("renews_at") as string
      if (!value) {
        return null
      }
      return <div className="flex space-x-2">{format(new Date(value), "MMM d")}</div>
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
  },

  {
    accessorKey: "cycles_processed",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Cycles Processed" className="text-right" />
    ),
    cell: ({ row }) => (
      <div className="flex justify-end space-x-2">{row.getValue("cycles_processed")}</div>
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
    enableSorting: false,
  },

  {
    accessorKey: "total_revenue",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Total Revenue" className="text-right" />
    ),
    cell: ({ row }) => (
      <div className="flex justify-end space-x-2">
        {formatCurrency(row.original.currency, row.getValue("total_revenue"))}
      </div>
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
    enableSorting: false,
  },
]
