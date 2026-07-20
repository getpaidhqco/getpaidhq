'use client'

import React, { useState } from 'react'
import { DataTable } from './DataTable'
import {
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  PaginationState,
  SortingState,
  Table,
  VisibilityState,
} from '@tanstack/react-table'

export type ResourceType = 'customers' | 'orders' | 'payments'

interface ResourceDataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  pageCount: number
  isLoading: boolean | { isFetching: boolean; isFetched: boolean; isLoading: boolean; status: string; fetchStatus: string }
  pagination?: PaginationState
  onPaginationChange?: OnChangeFn<PaginationState>
  sorting?: SortingState
  onSortingChange?: OnChangeFn<SortingState>
  columnFilters?: ColumnFiltersState
  onColumnFiltersChange?: OnChangeFn<ColumnFiltersState>
  columnVisibility?: VisibilityState
  onColumnVisibilityChange?: OnChangeFn<VisibilityState>
  resourcePath?: string
  filterComponent?: React.ReactNode
  toolbarComponent?: React.ReactNode | ((props: { table: Table<TData> }) => React.ReactNode)
  className?: string
  wrapperClassName?: string
  headerClassName?: string
}

export function ResourceDataTable<TData, TValue>({
  columns,
  data,
  pageCount,
  isLoading,
  pagination: externalPagination,
  onPaginationChange: externalPaginationChange,
  sorting: externalSorting,
  onSortingChange: externalSortingChange,
  columnFilters: externalColumnFilters,
  onColumnFiltersChange: externalColumnFiltersChange,
  columnVisibility: externalColumnVisibility,
  onColumnVisibilityChange: externalColumnVisibilityChange,
  resourcePath,
  filterComponent,
  toolbarComponent,
  className,
  wrapperClassName,
  headerClassName,
}: ResourceDataTableProps<TData, TValue>) {
  // Internal state for table (used if external state is not provided)
  const [rowSelection, setRowSelection] = useState({})
  const [internalColumnVisibility, setInternalColumnVisibility] = useState<VisibilityState>({})
  const [internalColumnFilters, setInternalColumnFilters] = useState<ColumnFiltersState>([])
  const [internalSorting, setInternalSorting] = useState<SortingState>([])
  const [internalPagination, setInternalPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  // Use external state if provided, otherwise use internal state
  const pagination = externalPagination || internalPagination
  const setPagination = externalPaginationChange || setInternalPagination
  const sorting = externalSorting || internalSorting
  const setSorting = externalSortingChange || setInternalSorting
  const columnFilters = externalColumnFilters || internalColumnFilters
  const setColumnFilters = externalColumnFiltersChange || setInternalColumnFilters
  const columnVisibility = externalColumnVisibility || internalColumnVisibility
  const setColumnVisibility = externalColumnVisibilityChange || setInternalColumnVisibility

  return (
    <DataTable
      columns={columns}
      data={data}
      pageCount={pageCount}
      pagination={pagination}
      onPaginationChange={setPagination}
      sorting={sorting}
      onSortingChange={setSorting}
      columnFilters={columnFilters}
      onColumnFiltersChange={setColumnFilters}
      columnVisibility={columnVisibility}
      onColumnVisibilityChange={setColumnVisibility}
      rowSelection={rowSelection}
      onRowSelectionChange={setRowSelection}
      isLoading={isLoading}
      resourcePath={resourcePath}
      filterComponent={filterComponent}
      toolbarComponent={toolbarComponent}
      className={className}
      wrapperClassName={wrapperClassName}
      headerClassName={headerClassName}
    />
  )
}
