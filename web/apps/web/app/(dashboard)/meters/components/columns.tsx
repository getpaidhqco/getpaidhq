"use client"

import { ColumnDef } from "@tanstack/react-table"
import { DataTableColumnHeader } from "./data-table-column-header"
import { DataTableRowActions } from "./data-table-row-actions"
import { format } from "date-fns"
import type { MeterResponse } from "@getpaidhq/sdk"
import { Badge } from "@/components/ui/badge"

// "weighted_sum" -> "Weighted sum"
const humanize = (v: string) => (v ? v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ") : "")

export const columns: ColumnDef<MeterResponse>[] = [
  {
    accessorKey: "code",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Code" />,
    cell: ({ row }) => (
      <div className="font-mono text-xs font-medium truncate">{row.original.code}</div>
    ),
  },
  {
    accessorKey: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    cell: ({ row }) => <div className="font-medium">{row.original.name}</div>,
  },
  {
    accessorKey: "aggregation",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Aggregation" />,
    cell: ({ row }) => (
      <Badge variant="info">{humanize(row.original.aggregation)}</Badge>
    ),
    filterFn: (row, id, value) => value.includes(row.getValue(id)),
  },
  {
    accessorKey: "field_name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Field" />,
    cell: ({ row }) => (
      <div className="text-muted-foreground">{row.original.field_name || "—"}</div>
    ),
  },
  {
    accessorKey: "carry_over",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Carry over" />,
    cell: ({ row }) => (
      <Badge variant={row.original.carry_over ? "success" : "muted"}>
        {row.original.carry_over ? "Yes" : "No"}
      </Badge>
    ),
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => {
      const date = row.original.created_at
      if (!date) return null
      return <div>{format(new Date(date), "PPP")}</div>
    },
  },
  {
    id: "actions",
    cell: ({ row }) => <DataTableRowActions row={row} />,
  },
]
