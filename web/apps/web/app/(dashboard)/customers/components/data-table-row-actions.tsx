"use client"

import { Row } from "@tanstack/react-table"
import { MoreHorizontal } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useRouter } from "next/navigation"
import type { CustomerResponse } from "@getpaidhq/sdk"

interface DataTableRowActionsProps {
  row: Row<CustomerResponse>
}

// The customers API is read-only beyond creation (no update or delete), so the
// only row action is View.
export function DataTableRowActions({ row }: DataTableRowActionsProps) {
  const router = useRouter()
  const customer = row.original

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
          onClick={(e) => e.stopPropagation()}
        >
          <MoreHorizontal />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[160px]">
        <DropdownMenuItem
          onClick={(e) => {
            e.stopPropagation()
            router.push(`/customers/${customer.id}`)
          }}
        >
          View
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
