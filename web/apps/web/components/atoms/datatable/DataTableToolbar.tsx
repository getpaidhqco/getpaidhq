'use client'

import { X } from 'lucide-react'
import { Table } from '@tanstack/react-table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataTableViewOptions } from './DataTableViewOptions'
import { DataTableFacetedFilter } from './DataTableFacetedFilter'

interface DataTableToolbarProps<TData> {
  table: Table<TData>
  searchColumn?: string
  searchPlaceholder?: string
  filterableColumns?: {
    column: string
    title: string
    options: {
      label: string
      value: string
      icon?: React.ComponentType<{ className?: string }>
      color?: string
    }[]
  }[]
}

export function DataTableToolbar<TData>({
  table,
  searchColumn = 'name',
  searchPlaceholder = 'Search...',
  filterableColumns = [],
}: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters.length > 0

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2">
        {searchColumn && table.getColumn(searchColumn) && (
          <Input
            placeholder={searchPlaceholder}
            value={(table.getColumn(searchColumn)?.getFilterValue() as string) ?? ''}
            onChange={(event) =>
              table.getColumn(searchColumn)?.setFilterValue(event.target.value)
            }
            className="h-8 w-[150px] lg:w-[250px]"
          />
        )}

        {filterableColumns.map((filterableColumn) => {
          const column = table.getColumn(filterableColumn.column)
          if (!column) return null

          return (
            <DataTableFacetedFilter
              key={filterableColumn.column}
              column={column}
              title={filterableColumn.title}
              options={filterableColumn.options}
            />
          )
        })}

        {isFiltered && (
          <Button
            variant="ghost"
            onClick={() => table.resetColumnFilters()}
            className="h-8 px-2 lg:px-3"
          >
            Reset
            <X className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} />
    </div>
  )
}
