"use client"

import React, { createContext, useContext, useState, useEffect } from "react";
import type { MeterResponse } from "@getpaidhq/sdk";
import { fetchMeter } from "./data";
import { useAuth } from "@getpaidhq/auth";

// Meters are read-only beyond creation (no update/delete API), so the context
// only exposes the loaded meter and a refresh.
interface MeterContextType {
  meter: MeterResponse | null;
  isLoading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
}

const MeterContext = createContext<MeterContextType | undefined>(undefined);

export function useMeter() {
  const context = useContext(MeterContext);
  if (!context) {
    throw new Error("useMeter must be used within a MeterProvider");
  }
  return context;
}

interface MeterProviderProps {
  id: string;
  initialData?: MeterResponse;
  children: React.ReactNode;
}

export function MeterProvider({ id, initialData, children }: MeterProviderProps) {
  const [meter, setMeter] = useState<MeterResponse | null>(initialData || null);
  const [isLoading, setIsLoading] = useState<boolean>(!initialData);
  const [error, setError] = useState<Error | null>(null);
  const { getAuthHeaders } = useAuth();

  const fetchData = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const authHeaders = await getAuthHeaders();
      const data = await fetchMeter(id, authHeaders);
      setMeter(data);
    } catch (err) {
      setError(err instanceof Error ? err : new Error("An unknown error occurred"));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (!initialData) {
      fetchData();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, initialData]);

  const refresh = async () => {
    await fetchData();
  };

  return (
    <MeterContext.Provider value={{ meter, isLoading, error, refresh }}>
      {children}
    </MeterContext.Provider>
  );
}
