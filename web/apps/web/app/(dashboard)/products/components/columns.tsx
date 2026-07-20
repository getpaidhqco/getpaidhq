"use client"

import {ColumnDef} from "@tanstack/react-table"

import {DataTableColumnHeader} from "./data-table-column-header"
import {DataTableRowActions} from "./data-table-row-actions"
import {format} from "date-fns";
import type {ProductResponse} from "@getpaidhq/sdk";


export const columns: ColumnDef<ProductResponse>[] = [
  {
    accessorKey: "name",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Name"/>
    ),
    cell: ({row}) => {
      return (
        <div className="flex space-x-2">
          {row.getValue("name")}
        </div>
      )
    },
  },
  {
    accessorKey: "created_at",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Created At"/>
    ),
    cell: ({row}) => {
      return (
        <div className="flex space-x-2">
          {format(new Date(row.getValue("created_at")), 'PPpp')}
        </div>
      )
    },
  },

  {
    id: "actions",
    cell: ({row}) => <div className="flex justify-end">
      <DataTableRowActions row={row}/>
    </div>,
  },
]
