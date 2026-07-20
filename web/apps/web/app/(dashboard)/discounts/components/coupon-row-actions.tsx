"use client"

import { useState } from "react"
import { Row } from "@tanstack/react-table"
import { MoreHorizontal } from "lucide-react"
import { useDeleteCoupon } from "@getpaidhq/react-sdk"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import type { CouponResponse } from "@getpaidhq/sdk"

interface CouponRowActionsProps {
  row: Row<CouponResponse>
  onEdit: (coupon: CouponResponse) => void
}

export function CouponRowActions({ row, onEdit }: CouponRowActionsProps) {
  const coupon = row.original
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)

  const deleteCoupon = useDeleteCoupon({
    onSuccess: () => {
      toast.success(`Discount "${coupon.name}" deleted`)
      setShowDeleteDialog(false)
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Please try again."
      toast.error("Failed to delete discount", { description: message })
    },
  })

  return (
    <div onClick={(event) => event.stopPropagation()}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
          >
            <MoreHorizontal />
            <span className="sr-only">Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-[160px]">
          <DropdownMenuItem onSelect={() => onEdit(coupon)}>Edit</DropdownMenuItem>
          <DropdownMenuItem onSelect={() => navigator.clipboard.writeText(coupon.id)}>
            Copy discount ID
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            variant="destructive"
            onSelect={(event) => {
              event.preventDefault()
              setShowDeleteDialog(true)
            }}
          >
            Delete
            <DropdownMenuShortcut>⌘⌫</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete discount?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete{" "}
              <span className="font-medium text-foreground">{coupon.name}</span>. This
              action cannot be undone. If the discount is in use, deactivate it instead.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteCoupon.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={deleteCoupon.isPending}
              onClick={(event) => {
                event.preventDefault()
                deleteCoupon.mutate(coupon.id)
              }}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              {deleteCoupon.isPending ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
