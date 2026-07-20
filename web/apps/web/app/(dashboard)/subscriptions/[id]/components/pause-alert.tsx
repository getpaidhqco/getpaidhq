"use client"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";


export default function PauseAlert({isOpen, onClose, onSubmit}: {isOpen: boolean, onClose: () => void, onSubmit: () => void}) {

  return (
    <AlertDialog open={isOpen} onOpenChange={(o) => { if (!o) onClose() }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you sure you want to pause this subscription?</AlertDialogTitle>
          <AlertDialogDescription>
            No more payments will be processed until you resume.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onClose}>Go back</AlertDialogCancel>
          <AlertDialogAction onClick={() => onSubmit()}>Pause</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
