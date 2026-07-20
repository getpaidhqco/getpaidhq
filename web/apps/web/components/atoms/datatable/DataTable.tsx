'use client'

import {
  Cell,
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  PaginationState,
  Row,
  RowSelectionState,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import React from 'react'
import { twMerge } from 'tailwind-merge'
import { DataTablePagination } from './DataTablePagination'
import { useRouter } from 'next/navigation'

export interface ReactQueryLoading {
  isFetching: boolean
  isFetched: boolean
  isLoading: boolean
  status: string
  fetchStatus: string
}

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  pageCount?: number
  pagination?: PaginationState
  onPaginationChange?: OnChangeFn<PaginationState>
  sorting?: SortingState
  onSortingChange?: OnChangeFn<SortingState>
  columnFilters?: ColumnFiltersState
  onColumnFiltersChange?: OnChangeFn<ColumnFiltersState>
  columnVisibility?: VisibilityState
  onColumnVisibilityChange?: OnChangeFn<VisibilityState>
  getSubRows?: (row: TData) => TData[] | undefined
  className?: string
  wrapperClassName?: string
  headerClassName?: string
  isLoading: boolean | ReactQueryLoading
  getCellColSpan?: (cell: Cell<TData, unknown>) => number
  getRowId?: (originalRow: TData, index: number, parent?: Row<TData>) => string
  rowSelection?: RowSelectionState
  enableRowSelection?: boolean
  onRowSelectionChange?: OnChangeFn<RowSelectionState>
  onRowClick?: (row: Row<TData>) => void
  resourceName?: string
  resourcePath?: string
  filterComponent?: React.ReactNode
  toolbarComponent?: React.ReactNode | ((props: { table: Table<TData> }) => React.ReactNode)
}

export type DataTableColumnDef<TData, TValue = unknown> = ColumnDef<
  TData,
  TValue
>

export type DataTablePaginationState = PaginationState
export type DataTableSortingState = SortingState

const queryIsDisabled = (s: ReactQueryLoading): boolean => {
  if (s.status === 'pending' && s.fetchStatus === 'idle') {
    return true
  }
  return false
}

export function DataTable<TData, TValue>({
  columns,
  data,
  pageCount,
  pagination,
  onPaginationChange,
  sorting,
  onSortingChange,
  columnFilters,
  onColumnFiltersChange,
  columnVisibility,
  onColumnVisibilityChange,
  getSubRows,
  className,
  wrapperClassName,
  headerClassName,
  isLoading,
  getCellColSpan,
  getRowId,
  rowSelection,
  enableRowSelection,
  onRowSelectionChange,
  onRowClick,
  resourceName,
  resourcePath,
  filterComponent,
  toolbarComponent,
}: DataTableProps<TData, TValue>) {
  const router = useRouter()

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    manualPagination: true,
    manualSorting: true,
    pageCount,
    onPaginationChange,
    onSortingChange,
    onColumnFiltersChange,
    onColumnVisibilityChange,
    getSubRows,
    getExpandedRowModel: getExpandedRowModel(),
    getRowId,
    enableRowSelection,
    onRowSelectionChange,
    enableMultiRowSelection: false,
    state: {
      pagination,
      sorting,
      rowSelection,
      columnFilters: columnFilters || [],
      columnVisibility: columnVisibility || {},
    },
  })

  const calcLoading =
    typeof isLoading === 'boolean'
      ? isLoading
      : (!isLoading.isFetched || isLoading.isLoading) &&
        !queryIsDisabled(isLoading)

  return (
    <div className={twMerge('flex flex-col gap-6', className)}>
      {/* Toolbar */}
      {toolbarComponent ? (
        typeof toolbarComponent === 'function'
          ? toolbarComponent({ table })
          : toolbarComponent
      ) : (
        filterComponent && (
          <div className="flex items-center justify-between">
            {filterComponent}
          </div>
        )
      )}

      {/* Table */}
      <div
        className={twMerge(
          'dark:border-polar-700 overflow-hidden rounded-2xl border border-gray-200',
          wrapperClassName,
        )}
      >
        <Table className="table-fixed">
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow
                key={headerGroup.id}
                className={twMerge(
                  'dark:bg-gphq-800 bg-gray-50',
                  headerClassName,
                )}
              >
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )}
                    </TableHead>
                  )
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {calcLoading ? (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  Loading...
                </TableCell>
              </TableRow>
            ) : (
              <>
                {table.getRowModel().rows?.length ? (
                  table.getRowModel().rows.map((row) => (
                    <TableRow
                      key={row.id}
                      className={
                        enableRowSelection || onRowClick || resourcePath
                          ? row.getCanSelect()
                            ? 'cursor-pointer'
                            : 'cursor-pointer'
                          : undefined
                      }
                      data-state={
                        enableRowSelection
                          ? row.getIsSelected()
                            ? 'selected'
                            : undefined
                          : undefined
                      }
                      onClick={
                        onRowClick
                          ? () => onRowClick(row)
                          : resourcePath
                            ? () => router.push(`${resourcePath}/${(row.original as any).id}`)
                            : enableRowSelection
                              ? row.getToggleSelectedHandler()
                              : undefined
                      }
                    >
                      {row.getVisibleCells().map((cell) => {
                        const colSpan = getCellColSpan
                          ? getCellColSpan(cell)
                          : 1

                        return (
                          <React.Fragment key={cell.id}>
                            {colSpan ? (
                              <TableCell colSpan={colSpan}>
                                {flexRender(
                                  cell.column.columnDef.cell,
                                  cell.getContext(),
                                )}
                              </TableCell>
                            ) : null}
                          </React.Fragment>
                        )
                      })}
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell
                      colSpan={columns.length}
                      className="h-24 text-center"
                    >
                      No Results
                    </TableCell>
                  </TableRow>
                )}
              </>
            )}
          </TableBody>
        </Table>
      </div>
      {pagination ? <DataTablePagination table={table} /> : null}
    </div>
  )
}
