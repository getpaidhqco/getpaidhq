"use client"

import * as React from "react"
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Label } from "@/components/ui/label"

export interface ComboboxOption {
  value: string
  label: string
}

export interface ComboboxProps {
  options: ComboboxOption[]
  value?: string
  onValueChange?: (value: string) => void
  onSearchChange?: (search: string) => void
  placeholder?: string
  emptyMessage?: string
  disabled?: boolean
  className?: string
  label?: string
}

export function Combobox({
  options,
  value,
  onValueChange,
  onSearchChange,
  placeholder = "Select option...",
  emptyMessage = "No options found.",
  disabled = false,
  className,
  label
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false)
  const [localValue, setLocalValue] = React.useState(value || "")

  // Update localValue when value prop changes
  React.useEffect(() => {
    if (value !== undefined) {
      setLocalValue(value)
    }
  }, [value])

  // Handle value change
  const handleValueChange = React.useCallback((newValue: string) => {
    setLocalValue(newValue)
    if (onValueChange) {
      onValueChange(newValue)
    }
  }, [onValueChange])

  // Handle search change
  const handleSearchChange = React.useCallback((search: string) => {
    if (onSearchChange) {
      onSearchChange(search)
    }
  }, [onSearchChange])

  return (
    <div className={cn("space-y-2", className)}>
      {label && <Label htmlFor={`combobox-${label}`}>{label}</Label>}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            id={label ? `combobox-${label}` : undefined}
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className={cn("w-full justify-between", className)}
            disabled={disabled}
          >
           <span className="truncate max-w-sm"> {localValue
              ? options.find((option) => option.value === localValue)?.label
              : placeholder}</span>
            <ChevronsUpDownIcon className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-full p-0">
          <Command>
            <CommandInput
              placeholder={placeholder}
              onValueChange={handleSearchChange}
            />
            <CommandList>
              <CommandEmpty>{emptyMessage}</CommandEmpty>
              <CommandGroup>
                {options.map((option) => (
                  <CommandItem
                    key={option.value}
                    value={option.value}
                    onSelect={(currentValue) => {
                      handleValueChange(currentValue === localValue ? "" : currentValue)
                      setOpen(false)
                    }}
                  >
                    <CheckIcon
                      className={cn(
                        "mr-2 h-4 w-4",
                        localValue === option.value ? "opacity-100" : "opacity-0"
                      )}
                    />
                    {option.label}
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  )
}
