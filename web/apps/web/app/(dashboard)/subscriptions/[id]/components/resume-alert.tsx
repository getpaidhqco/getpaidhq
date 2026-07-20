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
import {RadioGroup, RadioGroupItem} from "@/components/ui/radio-group";
import {useState} from "react";
import {Label} from "@/components/ui/label";

export default function ResumeAlert({isOpen, onClose, onSubmit}: {
  isOpen: boolean,
  onClose: () => void,
  onSubmit: (resumeBehavior: "continue_existing_billing_period" | "start_new_billing_period") => Promise<void>
}) {
  const [resumeBehavior, setResumeBehavior] = useState<"continue_existing_billing_period" | "start_new_billing_period">("start_new_billing_period");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async () => {
    setIsLoading(true);
    try {
      await onSubmit(resumeBehavior);
      // Dialog will be closed by the parent component on successful completion
    } catch (error) {
      // If there's an error, we stop loading but keep the dialog open
      setIsLoading(false);
      console.error("Error resuming subscription:", error);
    }
  };

  return (
    <AlertDialog open={isOpen} onOpenChange={(o) => { if (!o && !isLoading) onClose() }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Resume payment collection?</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to resume collecting payments? Any future invoices for this subscription will resume
            payment collection.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="mt-4 space-y-4">
          <RadioGroup
            value={resumeBehavior}
            onValueChange={(value) => setResumeBehavior(value as "continue_existing_billing_period" | "start_new_billing_period")}
            className="space-y-3"
          >
            <div className="flex items-start gap-2">
              <RadioGroupItem id="continue_existing" value="continue_existing_billing_period" />
              <div className="grid gap-1.5">
                <Label htmlFor="continue_existing" className="font-medium">Continue existing billing period</Label>
                <p className="text-sm text-muted-foreground">Resume the subscription using the existing billing period.</p>
              </div>
            </div>

            <div className="flex items-start gap-2">
              <RadioGroupItem id="start_new" value="start_new_billing_period" />
              <div className="grid gap-1.5">
                <Label htmlFor="start_new" className="font-medium">Start new billing period</Label>
                <p className="text-sm text-muted-foreground">Resume the subscription by starting a new billing period.</p>
              </div>
            </div>
          </RadioGroup>
        </div>

        <AlertDialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isLoading}>
            Go back
          </Button>
          <Button onClick={handleSubmit} disabled={isLoading}>
            {isLoading ? "Resuming..." : "Resume"}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
