import React, {useMemo} from "react";

import {cn} from "@/lib/utils";

// data
import {currencies} from "country-data-list";

// shadcn
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// radix-ui
import {SelectProps} from "@radix-ui/react-select";

// types
export interface Currency {
  code: string;
  decimals: number;
  name: string;
  number: string;
  symbol?: string;
}

interface CurrencySelectProps extends Omit<SelectProps, "onValueChange"> {
  onValueChange?: (value: string) => void;
  placeholder?: string;
  variant?: "default" | "small";
  valid?: boolean;
}

const CurrencySelectComponent = React.forwardRef<HTMLButtonElement, CurrencySelectProps>(
  (
    {
      value,
      onValueChange,
      placeholder = "Select currency",
      variant = "default",
      valid = true,
      ...props
    },
    ref
  ) => {
    const handleValueChange = (newValue: string) => {
      if (onValueChange) {
        onValueChange(newValue);
      }
    };

    // Memoize the currency items to prevent recreating them on every render
    const currencyItems = useMemo(() => {
      return currencies.all.map((currency) => (
        <SelectItem key={currency?.code} value={currency?.code || ""}>
          <div className="flex items-center w-full gap-2">
            <span className="text-sm text-muted-foreground w-8 text-left">
              {currency?.code}
            </span>
            <span>{currency?.name}</span>
          </div>
        </SelectItem>
      ));
    }, []);

    return (
      <Select
        value={value}
        onValueChange={handleValueChange}
        {...props}
        data-valid={valid}
      >
        <SelectTrigger
          className={cn("w-full", variant === "small" && "w-fit gap-2")}
          data-valid={valid}
          ref={ref}
        >
          {value && variant === "small" ? (
            <SelectValue placeholder={placeholder}>
              <span>{value}</span>
            </SelectValue>
          ) : (
            <SelectValue placeholder={placeholder}/>
          )}
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {currencyItems}
          </SelectGroup>
        </SelectContent>
      </Select>
    );
  }
);

CurrencySelectComponent.displayName = "CurrencySelect";

// Wrap with React.memo to prevent unnecessary rerenders
const CurrencySelect = React.memo(CurrencySelectComponent);

export {CurrencySelect};
