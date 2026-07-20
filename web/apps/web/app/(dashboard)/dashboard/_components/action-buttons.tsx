"use client";

import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useIsMobile } from "@/hooks/use-mobile";
import DatePicker from "@/app/(dashboard)/dashboard/_components/date-picker";
import {PlusIcon} from "lucide-react";
import {DateRange} from "react-day-picker";

export function ActionButtons({onDateChange}: { onDateChange: (date: DateRange) => void }) {
  const isMobile = useIsMobile();

  return (
    <div className="flex gap-3">
      <DatePicker onDateChange={onDateChange}/>
      <TooltipProvider delayDuration={0}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="outline" className="aspect-square max-lg:p-0">
              <PlusIcon
                className="lg:-ms-1 opacity-40 size-5"
                size={20}
                aria-hidden="true"
              />
              <span className="max-lg:sr-only">Export</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent className="lg:hidden" hidden={isMobile}>
            Export
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <TooltipProvider delayDuration={0}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button className="aspect-square max-lg:p-0">
              <PlusIcon
                className="lg:-ms-1 opacity-40 size-5"
                size={20}
                aria-hidden="true"
              />
              <span className="max-lg:sr-only">Add Chart</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent className="lg:hidden" hidden={isMobile}>
            Add Chart
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}
