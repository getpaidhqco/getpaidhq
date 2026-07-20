"use client"

import {Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle,} from "@/components/ui/sheet"
import {Button} from "@/components/ui/button"
import CreateOrder from "@/app/(dashboard)/orders/create/components/create-order"
import {useState} from "react"

type CreateOrderDrawerProps = {
  open: boolean
  onClose: () => void
}

export function CreateOrderDrawer({ open, onClose }: CreateOrderDrawerProps) {
  return (
    <Sheet
      open={open} onOpenChange={onClose}>
      <SheetContent className="w-[600px] sm:max-w-[600px] overflow-y-auto m-2 rounded-2xl">
        <SheetHeader>
          <SheetTitle>Create Order</SheetTitle>
          <SheetDescription>
            Create a new order for a customer.
          </SheetDescription>
        </SheetHeader>
        <div className="flex-1 px-4">
          <CreateOrder onClose={onClose} />
        </div>
      </SheetContent>
    </Sheet>
  )
}

export function CreateOrderButton() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <Button onClick={() => setOpen(true)}>Create Order</Button>
      <CreateOrderDrawer open={open} onClose={() => setOpen(false)} />
    </>
  )
}
