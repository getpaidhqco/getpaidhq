"use client";

import {Button} from "@/components/ui/button";
import {
  endOfMonth,
  endOfYear,
  startOfMonth,
  startOfYear,
  subDays,
  subMonths,
  subYears,
} from "date-fns";
import {Calendar} from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {format} from "date-fns";
import {CalendarIcon} from "lucide-react";
import {useState} from "react";
import {DateRange} from "react-day-picker";
import {cn} from "@/lib/utils";

export default function DatePicker({onDateChange}: { onDateChange: (date: DateRange) => void }) {
  const today = new Date();
  const yesterday = {
    from: subDays(today, 1),
    to: subDays(today, 1),
  };
  const last7Days = {
    from: subDays(today, 6),
    to: today,
  };
  const last30Days = {
    from: subDays(today, 29),
    to: today,
  };
  const monthToDate = {
    from: startOfMonth(today),
    to: today,
  };
  const lastMonth = {
    from: startOfMonth(subMonths(today, 1)),
    to: endOfMonth(subMonths(today, 1)),
  };
  const yearToDate = {
    from: startOfYear(today),
    to: today,
  };
  const lastYear = {
    from: startOfYear(subYears(today, 1)),
    to: endOfYear(subYears(today, 1)),
  };
  const [month, setMonth] = useState(today);
  const [date, setDate] = useState<DateRange | undefined>(yearToDate);

  const handleRangeChange = (range: DateRange) => {
    setDate(range);
    onDateChange(range);
  };

  return (
    <div className="*:not-first:mt-2">
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline" className="justify-start min-w-62">
            <CalendarIcon
              size={16}
              className="opacity-40 -ms-1 group-hover:text-foreground shrink-0 transition-colors"
              aria-hidden="true"
            />
            <span className={cn("truncate", !date && "text-muted-foreground")}>
              {date?.from ? (
                date.to ? (
                  <>
                    {format(date.from, "LLL dd, y")} -{" "}
                    {format(date.to, "LLL dd, y")}
                  </>
                ) : (
                  format(date.from, "LLL dd, y")
                )
              ) : (
                "Pick a date range"
              )}
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0">
          <div className="flex max-sm:flex-col">
            <div className="relative py-4 max-sm:order-1 max-sm:border-t sm:w-32">
              <div className="h-full sm:border-e">
                <div className="flex flex-col px-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(today);
                      handleRangeChange({ from: today, to: today });
                    }}
                  >
                    Today
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(yesterday.to);
                      handleRangeChange(yesterday);
                    }}
                  >
                    Yesterday
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(last7Days.to);
                      handleRangeChange(last7Days);
                    }}
                  >
                    Last 7 days
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(last30Days.to);
                      handleRangeChange(last30Days);
                    }}
                  >
                    Last 30 days
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(monthToDate.to);
                      handleRangeChange(monthToDate);
                    }}
                  >
                    Month to date
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(lastMonth.to);
                      handleRangeChange(lastMonth);
                    }}
                  >
                    Last month
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(yearToDate.to);
                      handleRangeChange(yearToDate);
                    }}
                  >
                    Year to date
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start"
                    onClick={() => {
                      setMonth(lastYear.to);
                      handleRangeChange(lastYear);
                    }}
                  >
                    Last year
                  </Button>
                </div>
              </div>
            </div>
            <Calendar
              mode="range"
              selectedRange={date?.from ? { from: date.from, to: date.to } : undefined}
              onSelectRange={(newRange) => {
                if (newRange) {
                  handleRangeChange(newRange);
                }
              }}
              className="p-2"
              maxDate={today}
            />
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
