"use client"

import {ColumnDef} from "@tanstack/react-table"

import {statuses} from "../data/data"
import {DataTableColumnHeader} from "./data-table-column-header"
import {format} from "date-fns";
import {formatCurrency} from "@/lib/currency";
import {StatusTag, type StatusTone} from "@/components/ui/status-tag";
import type {OrderResponse} from "@getpaidhq/sdk";


export const columns: ColumnDef<OrderResponse>[] = [
  {
    id: "globalFilter",
    accessorFn: (row) => {
      // Combine searchable fields for global search
      const searchableText = [
        row.reference,
        row.customer?.email || '',
        format(new Date(row.created_at), 'MMM d, yyyy'),
        format(new Date(row.created_at), 'MMM d, h:mm a'),
        format(new Date(row.created_at), 'yyyy-MM-dd'),
      ].join(' ').toLowerCase();
      return searchableText;
    },
    header: () => null,
    cell: () => null,
    enableSorting: false,
    enableHiding: true,
    filterFn: (row, columnId, filterValue) => {
      if (!filterValue) return true;
      const searchValue = filterValue.toLowerCase();
      const rowValue = row.getValue(columnId) as string;
      return rowValue.includes(searchValue);
    },
  },
  {
    accessorKey: "reference",
    enableSorting: true,
    enableHiding: true,
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Ref"/>
    ),
    cell: ({row}) => <div className="text-xs font-semibold truncate">{row.original.reference}</div>,
  },



  {
    accessorKey: "created_at",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Date"/>
    ),
    cell: ({row}) => {
      return (
        <div className="flex space-x-2">
          {format(new Date(row.getValue("created_at")), 'MMM d, h:mm a')}
        </div>
      )
    },
  },

  {
    accessorKey: "status",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Status"/>
    ),
    cell: ({row}) => {
      const value = row.getValue("status") as string
      const status = statuses.find((s) => s.value === value)
      const TONE: Record<string, StatusTone> = {
        fulfilled: "success",
        completed: "success",
        paid: "success",
        processing: "info",
        pending: "info",
        failed: "danger",
        cancelled: "neutral",
        refunded: "neutral",
      }
      return <StatusTag tone={TONE[value] ?? "neutral"}>{status?.label ?? value}</StatusTag>
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
  },



  {
    accessorKey: "customer",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Customer"/>
    ),
    cell: ({row}) => <div className="truncate">{row.original.customer.email}</div>,
    enableSorting: false,
    enableHiding: false,

  },


  {
    accessorKey: "total",
    header: ({column}) => (
      <DataTableColumnHeader column={column} title="Total" className="text-right" />
    ),
    cell: ({row}) => {
      return (
        <div className="flex space-x-2 justify-end">
          {formatCurrency(row.original.currency, row.getValue('total'))}
        </div>
      )
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id))
    },
    enableSorting: false,
  },

  // {
  //   id: "actions",
  //   cell: ({row}) => <DataTableRowActions row={row}/>,
  // },
]
