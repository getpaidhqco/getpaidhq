"use client"

import {ColumnDef} from "@tanstack/react-table"
import {format} from "date-fns";

import {DataTableColumnHeader} from "./data-table-column-header"
import type {InvoiceResponse} from "@getpaidhq/sdk";
import {StatusTag, type StatusTone} from "@/components/ui/status-tag";
import {statuses} from "./data";

const STATUS_TONE: Record<string, StatusTone> = {
  paid: "success",
  pending: "warn",
  partially_paid: "info",
  open: "info",
  draft: "neutral",
  overdue: "danger",
  void: "neutral",
  cancelled: "danger",
  uncollectible: "warn",
};

export const columns: ColumnDef<InvoiceResponse>[] = [
  {
    accessorKey: "id",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Invoice"/>
    ),
    cell: ({row}) => (
      <span className="font-mono text-xs font-medium text-foreground">
        {row.original.id}
      </span>
    ),
  },
  {
    accessorKey: "customer_id",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Customer"/>
    ),
    cell: ({row}) => (
      <span className="font-mono text-xs text-muted-foreground">
        {row.original.customer_id}
      </span>
    ),
  },
  {
    accessorKey: "status",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Status"/>
    ),
    cell: ({row}) => {
      const value = row.getValue("status") as string
      const status = statuses.find((s) => s.value === value)
      const tone = STATUS_TONE[value] ?? "neutral"
      return <StatusTag tone={tone}>{status?.label ?? value}</StatusTag>
    },
    filterFn: (row, id, value) => value.includes(row.getValue(id)),
  },
  {
    accessorKey: "total",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Total" className="text-right" />
    ),
    cell: ({row}) => {
      const amount = Number(row.getValue("total"))
      const formatted = new Intl.NumberFormat("en-US", {
        style: "currency",
        currency: row.original.currency,
      }).format(amount / 100)
      return (
        <div className="flex items-baseline justify-end gap-1">
          <span className="tabular font-medium text-foreground">{formatted}</span>
          <span className="font-mono text-[10px] uppercase text-muted-foreground">
            {row.original.currency}
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: "period_start",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Period start"/>
    ),
    cell: ({row}) => {
      const date = row.getValue("period_start")
      if (!date) return <span className="text-muted-foreground">—</span>
      return (
        <span className="font-mono text-xs text-muted-foreground tabular">
          {format(new Date(date as string), "MMM d, yyyy")}
        </span>
      )
    },
  },
  {
    accessorKey: "created_at",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Created"/>
    ),
    cell: ({row}) => {
      const date = row.getValue("created_at")
      if (!date) return <span className="text-muted-foreground">—</span>
      return (
        <span className="font-mono text-xs text-muted-foreground tabular">
          {format(new Date(date as string), "MMM d, yyyy")}
        </span>
      )
    },
  },
]
