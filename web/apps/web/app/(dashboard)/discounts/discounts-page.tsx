"use client"

import * as React from "react"
import { useMemo, useState } from "react"
import { PlusIcon } from "lucide-react"
import { PaginationState, Row } from "@tanstack/react-table"
import { useCoupons } from "@getpaidhq/react-sdk"
import type { CouponResponse } from "@getpaidhq/sdk"

import { Button } from "@/components/ui/button"
import { PageHeader } from "@/components/ui/page-header"
import { DataTable } from "./components/data-table"
import { createColumns } from "./components/columns"
import { CouponFormSheet } from "./components/coupon-form-sheet"

export default function DiscountsPage() {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const [sheetOpen, setSheetOpen] = useState(false)
  const [editing, setEditing] = useState<CouponResponse | null>(null)

  const openCreate = () => {
    setEditing(null)
    setSheetOpen(true)
  }

  const openEdit = (coupon: CouponResponse) => {
    setEditing(coupon)
    setSheetOpen(true)
  }

  const query = useCoupons({
    page: pagination.pageIndex,
    limit: pagination.pageSize,
  })

  const data = query.data?.data ?? []
  const total = query.data?.meta?.total ?? 0
  const pageCount = Math.ceil(total / pagination.pageSize) || 0

  const columns = useMemo(() => createColumns(openEdit), [])

  return (
    <>
      <CouponFormSheet open={sheetOpen} onOpenChange={setSheetOpen} coupon={editing} />

      <div className="flex flex-1 flex-col gap-8">
        <PageHeader
          title="Discounts"
          actions={
            <Button size="sm" onClick={openCreate}>
              <PlusIcon data-icon="inline-start" />
              New discount
            </Button>
          }
        />

        <DataTable
          columns={columns}
          data={data}
          pageCount={pageCount}
          pagination={pagination}
          onPaginationChange={setPagination}
          onRowClick={(row: Row<CouponResponse>) => openEdit(row.original)}
          isLoading={query.isLoading}
        />
      </div>
    </>
  )
}
