"use client"

import {useState} from "react"
import {Row} from "@tanstack/react-table"
import {MoreHorizontal} from "lucide-react"
import {useArchiveProduct, useDeleteProduct, useUnarchiveProduct} from "@getpaidhq/react-sdk"
import {toast} from "sonner"

import {Button} from "@/components/ui/button"
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

import type {ProductResponse} from "@getpaidhq/sdk";

interface DataTableRowActionsProps<TData> {
  row: Row<TData>
}

export function DataTableRowActions<TData>({
                                             row,
                                           }: DataTableRowActionsProps<TData>) {
  const data = row.original as ProductResponse
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [showArchiveDialog, setShowArchiveDialog] = useState(false)
  const isArchived = data.status === "archived"

  const deleteProduct = useDeleteProduct({
    onSuccess: () => {
      toast.success(`Product "${data.name}" deleted`)
      setShowDeleteDialog(false)
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Please try again."
      toast.error("Failed to delete product", {description: message})
    },
  })

  const archiveProduct = useArchiveProduct({
    onSuccess: () => {
      toast.success(`Product "${data.name}" archived`)
      setShowArchiveDialog(false)
    },
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Please try again."
      toast.error("Failed to archive product", {description: message})
    },
  })

  const unarchiveProduct = useUnarchiveProduct({
    onSuccess: () => toast.success(`Product "${data.name}" restored`),
    onError: (error: unknown) => {
      const message = error instanceof Error ? error.message : "Please try again."
      toast.error("Failed to restore product", {description: message})
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
            <MoreHorizontal/>
            <span className="sr-only">Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-[160px]">
          <DropdownMenuItem>Edit</DropdownMenuItem>
          <DropdownMenuItem>Make a copy</DropdownMenuItem>
          <DropdownMenuItem onSelect={() => navigator.clipboard.writeText(data.id)}>
            Copy Product ID
          </DropdownMenuItem>
          <DropdownMenuSeparator/>
          {isArchived ? (
            <DropdownMenuItem
              disabled={unarchiveProduct.isPending}
              onSelect={(event) => {
                event.preventDefault()
                unarchiveProduct.mutate(data.id)
              }}
            >
              Unarchive
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault()
                setShowArchiveDialog(true)
              }}
            >
              Archive
            </DropdownMenuItem>
          )}
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
            <AlertDialogTitle>Delete product?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete <span className="font-medium text-foreground">{data.name}</span>.
              This action cannot be undone. Products that have ever been sold can&apos;t be deleted —
              archive them instead to keep order history intact.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteProduct.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={deleteProduct.isPending}
              onClick={(event) => {
                event.preventDefault()
                deleteProduct.mutate(data.id)
              }}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              {deleteProduct.isPending ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={showArchiveDialog} onOpenChange={setShowArchiveDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Archive product?</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="font-medium text-foreground">{data.name}</span> will be hidden
              from listings and can no longer be sold. Existing orders, subscriptions and history
              are kept, and active subscriptions keep billing. You can unarchive it at any time.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={archiveProduct.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={archiveProduct.isPending}
              onClick={(event) => {
                event.preventDefault()
                archiveProduct.mutate(data.id)
              }}
            >
              {archiveProduct.isPending ? "Archiving…" : "Archive"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
