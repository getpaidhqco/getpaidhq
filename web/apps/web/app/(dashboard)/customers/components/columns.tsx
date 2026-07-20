"use client"

import { ColumnDef } from "@tanstack/react-table"
import { DataTableColumnHeader } from "./data-table-column-header"
import { DataTableRowActions } from "./data-table-row-actions"
import { format } from "date-fns"
import type { CustomerResponse } from "@getpaidhq/sdk"

export const columns: ColumnDef<CustomerResponse>[] = [
  {
    accessorKey: "id",
    enableSorting: false,
    enableHiding: true,
    size: 80,
    header: ({ column }) => <DataTableColumnHeader column={column} title="ID" />,
    cell: ({ row }) => (
      <div className="w-[80px] text-xs font-semibold truncate">{row.original.id}</div>
    ),
  },
  {
    id: "name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    cell: ({ row }) => {
      const fullName = `${row.original.first_name || ""} ${row.original.last_name || ""}`.trim()
      return <div className="flex space-x-2">{fullName || "—"}</div>
    },
  },
  {
    accessorKey: "email",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Email" />,
    cell: ({ row }) => <div className="flex space-x-2">{row.original.email || "—"}</div>,
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created At" />,
    cell: ({ row }) => {
      const date = row.original.created_at
      if (!date) return null
      return <div className="flex space-x-2">{format(new Date(date), "PPpp")}</div>
    },
  },
  {
    id: "actions",
    cell: ({ row }) => <DataTableRowActions row={row} />,
  },
]
