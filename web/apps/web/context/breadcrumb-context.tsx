"use client"

import React, { createContext, useContext, useState, ReactNode } from 'react';

export interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface BreadcrumbContextType {
  items: BreadcrumbItem[] | null;
  setItems: (items: BreadcrumbItem[] | null) => void;
}

const BreadcrumbContext = createContext<BreadcrumbContextType | undefined>(undefined);

interface BreadcrumbProviderProps {
  children: ReactNode;
}

export function BreadcrumbProvider({ children }: BreadcrumbProviderProps) {
  const [items, setItems] = useState<BreadcrumbItem[] | null>(null);

  return (
    <BreadcrumbContext.Provider value={{ items, setItems }}>
      {children}
    </BreadcrumbContext.Provider>
  );
}

export function useBreadcrumb() {
  const context = useContext(BreadcrumbContext);
  if (context === undefined) {
    throw new Error('useBreadcrumb must be used within a BreadcrumbProvider');
  }
  return context;
}
