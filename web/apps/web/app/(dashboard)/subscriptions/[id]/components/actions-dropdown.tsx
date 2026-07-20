'use client'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import {Button} from "@/components/ui/button";
import {ChevronDown} from "lucide-react";
import type {SubscriptionResponse} from "@getpaidhq/sdk";


export default function ActionsDropdown({onSelect, subscription}: {
  onSelect: (t: string) => void
  subscription: SubscriptionResponse
}) {

  return <>
    <DropdownMenu >
      <DropdownMenuTrigger asChild>
        <Button variant="default">
          Actions
          <ChevronDown className="h-4 w-4"/>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem onClick={() => onSelect('update')}>Update subscription</DropdownMenuItem>
        {subscription.status === 'paused' &&
            <DropdownMenuItem onClick={() => onSelect('resume')}>Resume payment collection</DropdownMenuItem>}
        {subscription.status !== 'paused' &&
            <DropdownMenuItem onClick={() => onSelect('pause')}>Pause payment collection</DropdownMenuItem>}
        <DropdownMenuSeparator/>
        <DropdownMenuItem onClick={() => onSelect('cancel')}>Cancel Subscription</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </>
}
