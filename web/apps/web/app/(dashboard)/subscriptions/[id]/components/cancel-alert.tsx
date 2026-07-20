"use client"
import {Button} from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {useState} from "react";

export default function CancelAlert({isOpen, onClose, onSubmit}: {
  isOpen: boolean,
  onClose: () => void,
  onSubmit: (reason: string) => Promise<void>
}) {
  const [reason] = useState<string>("Customer requested");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async () => {
    setIsLoading(true);
    try {
      await onSubmit(reason);
      // Dialog will be closed by the parent component on successful completion
    } catch (error) {
      // If there's an error, we stop loading but keep the dialog open
      setIsLoading(false);
      console.error("Error canceling subscription:", error);
    }
  };

  return (
    <AlertDialog open={isOpen} onOpenChange={(o) => { if (!o && !isLoading) onClose() }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Cancel subscription?</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to cancel this subscription? This action cannot be undone and the customer will no longer be charged for this subscription.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <AlertDialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isLoading}>
            Go back
          </Button>
          <Button
            variant="destructive"
            onClick={handleSubmit}
            disabled={isLoading}
          >
            {isLoading ? "Canceling..." : "Cancel Subscription"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
