"use client"

import * as React from "react"
import {
  ColumnDef,
  ColumnFiltersState,
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  PaginationState,
  SortingState,
  useReactTable,
  VisibilityState,
} from "@tanstack/react-table"

import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow,} from "@/components/ui/table"
import {DataTablePagination} from "./data-table-pagination"
import {DataTableToolbar} from "./data-table-toolbar"
import {keepPreviousData, useQuery} from "@tanstack/react-query";
import {fetchData} from "../data/data";
import {useRouter} from "next/navigation";
import {useAuth} from "@getpaidhq/auth";
import { DataTableSkeleton } from "@/components/skeletons";

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
}

export function DataTable<TData, TValue>({
                                           columns,
                                           data,
                                         }: DataTableProps<TData, TValue>) {
  const router = useRouter()
  const {getAuthHeaders} = useAuth()
  const [rowSelection, setRowSelection] = React.useState({})
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>({
      globalFilter: false, // Hide the global filter column
    })
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    []
  )
  const [sorting, setSorting] = React.useState<SortingState>([])
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })


  // Extract filters from columnFilters
  const searchFilter = columnFilters.find(filter => filter.id === 'globalFilter')?.value as string || '';
  const statusFilter = columnFilters.find(filter => filter.id === 'status')?.value as string[] || [];

  const dataQuery = useQuery({
    queryKey: ['orders', 'list', pagination, searchFilter, statusFilter],
    queryFn: async () => fetchData({
      pageIndex: searchFilter ? 0 : pagination.pageIndex, // Reset to first page when searching
      pageSize: searchFilter ? 100 : pagination.pageSize, // Fetch more data when searching for better client-side filtering
      search: searchFilter,
      status: statusFilter.length > 0 ? statusFilter[0] : undefined, // API only supports single status
    }, await getAuthHeaders()),
    placeholderData: keepPreviousData, // don't have 0 rows flash while changing pages/loading next page
  })

  const table = useReactTable({
    data: (dataQuery.data?.data ?? data) as TData[],
    columns,
    rowCount: searchFilter ? undefined : (dataQuery.data?.meta?.total ?? 0), // Use undefined when searching to enable client-side pagination
    state: {
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
      pagination,
    },
    autoResetPageIndex: false,
    manualPagination: !searchFilter, // Use client-side pagination when searching
    manualFiltering: false, // Enable client-side filtering for search
    enableRowSelection: false,
    onRowSelectionChange: setRowSelection,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  })

  if (dataQuery.isLoading && !data.length) {
    return <DataTableSkeleton columnCount={7} rowCount={8} />;
  }

  return (
    <div className="space-y-4">
      <DataTableToolbar table={table}/>
      <div className="">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id}
                               colSpan={header.colSpan}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                    </TableHead>
                  )
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  className="cursor-pointer"
                  onClick={() => router.push(`/orders/${(row.original as { id: string }).id}`)}
                  data-state={row.getIsSelected() && "selected"}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <DataTablePagination table={table}/>
    </div>
  )
}
