"use client"

import {X} from "lucide-react"
import {Table} from "@tanstack/react-table"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {DataTableViewOptions} from "./data-table-view-options"
import {AGGREGATION_TYPES} from "@getpaidhq/react-sdk"
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue,} from "@/components/ui/select"

// "weighted_sum" -> "Weighted sum"
const humanize = (v: string) => v.charAt(0).toUpperCase() + v.slice(1).replace(/_/g, " ")

interface DataTableToolbarProps<TData> {
  table: Table<TData>
}

export function DataTableToolbar<TData>({
  table,
}: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters.length > 0

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2">
        <Input
          placeholder="Filter meters..."
          value={(table.getColumn("name")?.getFilterValue() as string) ?? ""}
          onChange={(event) =>
            table.getColumn("name")?.setFilterValue(event.target.value)
          }
          className="h-8 w-[150px] lg:w-[250px]"
        />
        {table.getColumn("aggregation") && (
          <Select
            value={(table.getColumn("aggregation")?.getFilterValue() as string) ?? ""}
            onValueChange={(value) =>
              table.getColumn("aggregation")?.setFilterValue(value === "all" ? "" : value)
            }
          >
            <SelectTrigger className="h-8 w-[150px]">
              <SelectValue placeholder="Aggregation" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All aggregations</SelectItem>
              {AGGREGATION_TYPES.map((type) => (
                <SelectItem key={type} value={type}>
                  {humanize(type)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
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
