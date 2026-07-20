"use client"

import * as React from "react"
import {useState, useEffect} from "react"
import {Combobox, ComboboxOption} from "@/components/ui/combobox"
import {cn} from "@/lib/utils"
import {useDebounce} from "@/hooks/use-debounce"
import {MeterCreateDialog} from "@/components/meter-create-dialog"
import {useAuth} from "@getpaidhq/auth";
import {useQuery, useQueryClient} from "@tanstack/react-query"
import {toast} from "sonner";
import type {MeterResponse as Meter} from "@getpaidhq/sdk"

interface MeterSearchProps {
  value?: string
  onValueChange?: (value: string, meter?: Meter) => void
  className?: string
  disabled?: boolean
}

export function MeterSearch({
  value,
  onValueChange,
  className,
  disabled = false,
}: MeterSearchProps) {
  const {getAuthHeaders} = useAuth()
  const queryClient = useQueryClient()
  const [meters, setMeters] = useState<Meter[]>([]);
  const [search, setSearch] = useState("")
  const [options, setOptions] = useState<ComboboxOption[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const debouncedSearch = useDebounce(search, 300)

  // Use TanStack Query to fetch meters based on search
  const metersQuery = useQuery({
    queryKey: ['meters', debouncedSearch],
    queryFn: async () => {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/meters?search=${encodeURIComponent(debouncedSearch)}`,
        {
          method: "GET",
          headers: {
            ...await getAuthHeaders(),
          },
        }
      );

      if (!response.ok) {
        throw new Error("Failed to fetch meters");
      }

      const m = await response.json();
      setMeters(m.data);
    },
    enabled: debouncedSearch.length >= 3,
    retry: 1, // Retry failed requests once
    refetchOnWindowFocus: false, // Don't refetch when window regains focus
  });

  // Use TanStack Query to fetch initial meters
  const initialMetersQuery = useQuery({
    queryKey: ['initialMeters'],
    queryFn: async () => {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/api/meters?limit=10`, // Fetch first 10 meters
        {
          method: "GET",
          headers: {
            ...await getAuthHeaders(),
          },
        }
      );

      if (!response.ok) {
        throw new Error("Failed to fetch initial meters");
      }

      const m = await response.json();
      // Only set meters if we're not already showing search results
      if (debouncedSearch.length < 3) {
        setMeters(m.data);
      }
      return m.data;
    },
    retry: 1,
    refetchOnWindowFocus: false,
  });

  const isLoading = metersQuery.isLoading || metersQuery.isFetching ||
                 initialMetersQuery.isLoading || initialMetersQuery.isFetching;
  const error = metersQuery.error || initialMetersQuery.error;

  // Show error toast if query fails
  useEffect(() => {
    if (error) {
      toast.error("Failed to fetch meters", {
        description: error instanceof Error ? error.message : "Unknown error",
        duration: 5000,
      });
      console.error("Error fetching meters:", error);
    }
  }, [metersQuery.error, initialMetersQuery.error]);

  // Update options when meters change
  useEffect(() => {
    const newOptions: ComboboxOption[] = [
      {value: "create", label: "➕ Create New Meter"},
      ...meters.map((meter) => ({
        value: meter.id,
        label: meter.name,
      })),
    ]
    setOptions(newOptions)
  }, [meters])

  // Handle value change
  const handleValueChange = (newValue: string) => {
    if (newValue === "create") {
      setDialogOpen(true)
      return
    }

    const selectedMeter = meters.find((m) => m.id === newValue)
    if (onValueChange) {
      onValueChange(newValue, selectedMeter)
    }
  }

  // Handle meter creation
  const handleMeterCreated = (newMeter: Meter) => {
    // Update the search query cache with the new meter
    queryClient.setQueryData(['meters', debouncedSearch], (oldData: any) => {
      // If there's no old data, create a new data structure
      if (!oldData) {
        return {data: [newMeter]};
      }

      // Add the new meter to the beginning of the list
      return {
        ...oldData,
        data: [newMeter, ...(oldData.data || [])]
      };
    });

    // Update the initial meters query cache with the new meter
    queryClient.setQueryData(['initialMeters'], (oldData: any) => {
      // If there's no old data, create a new data structure
      if (!oldData) {
        return {data: [newMeter]};
      }

      // Add the new meter to the beginning of the list and limit to 10
      const updatedData = [newMeter, ...(oldData.data || [])];
      return {
        ...oldData,
        data: updatedData.slice(0, 10) // Keep only the first 10 meters
      };
    });

    // Show success toast
    toast.success("Meter added to search results");

    // Select the new meter
    if (onValueChange) {
      onValueChange(newMeter.id, newMeter)
    }
  }

  return (
    <div className={cn("relative", className)}>
      <Combobox
        options={options}
        value={value}
        onValueChange={handleValueChange}
        onSearchChange={setSearch}
        placeholder="Select or search meters"
        emptyMessage={isLoading ? "Loading..." : "No meters found."}
        disabled={disabled}
      />

      <MeterCreateDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onMeterCreated={handleMeterCreated}
      />
    </div>
  )
}
