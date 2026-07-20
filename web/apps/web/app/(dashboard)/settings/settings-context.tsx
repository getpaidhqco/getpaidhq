"use client"
import { createContext, useContext, ReactNode, useCallback, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@getpaidhq/auth";
import { AuthHeader } from "@getpaidhq/auth";
import { z } from "zod";
import { useForm, UseFormReturn } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";

// Define the settings type
interface Settings {
  [key: string]: any;
}

// Export the FailureActionEnum
export const FailureActionEnum = z.enum(["cancel", "mark_unpaid", "past_due"]);
export type FailureAction = z.infer<typeof FailureActionEnum>;

// Retry Policy Schema
const RetryPolicySchema = z.object({
  attempts: z.number().int().min(0),
  interval: z.string().optional().nullable(),
  retry_period: z.number().int().min(0),
  failure_action: FailureActionEnum,
});

// Combined Schema for all settings
export const SettingsFormSchema = z.object({
  // Invoice Settings
  invoice_prefix: z.string(),
  enable_invoice_pdfs: z.boolean(),

  // Subscription Settings
  retry_policy: RetryPolicySchema,
  reminder_days: z.number().min(0, "Reminder days must be a positive number"),
  email_reminders: z.boolean(),
  cancel_on_failure: z.boolean(),
}).refine(
  (data) => !data.email_reminders || data.reminder_days > 0,
  {
    message: "Reminder days must be greater than 0 if email reminders are enabled",
    path: ["reminder_days"],
  }
);

export type SettingsFormValues = z.infer<typeof SettingsFormSchema>;

// Define the combined context type
interface SettingsContextType {
  // Settings data management
  settings: Settings | undefined;
  isLoading: boolean;
  error: Error | null;
  refresh: () => void;
  updateSettings: (data: Partial<Settings>) => Promise<any>;

  // Form state management
  form: UseFormReturn<SettingsFormValues>;
  submitForm: () => Promise<void>;
}

// Create the context
const SettingsContext = createContext<SettingsContextType | undefined>(undefined);

// Create a provider component
export function SettingsProvider({
  children,
  parentId,
  id,
  initialData
}: {
  children: ReactNode;
  parentId?: string;
  id?: string;
  initialData?: Settings;
}) {
  const { getAuthHeaders, orgId } = useAuth();
  const queryClient = useQueryClient();

  // Build the API URL based on the provided params
  const getApiUrl = () => {
    let url = `${process.env.NEXT_PUBLIC_API_URL}/api/settings`;

    if (parentId) {
      url += `/${parentId}`;
    }

    if (id) {
      url += `/${id}`;
    }

    return url;
  };

  // Function to fetch settings data
  const fetchSettings = async (authHeaders: AuthHeader): Promise<Settings> => {
    const response = await fetch(getApiUrl(), {
      headers: {
        ...authHeaders,
        'Content-Type': 'application/json'
      }
    });

    if (!response.ok) {
      throw new Error(`Error fetching settings: ${response.statusText}`);
    }

    return response.json();
  };

  // Query for settings data
  const {
    data: settings,
    isLoading,
    error,
    refetch
  } = useQuery({
    queryKey: ['settings', parentId, id],
    queryFn: async () => {
      const headers = await getAuthHeaders();
      return fetchSettings(headers);
    },
    initialData,
    enabled: !!process.env.NEXT_PUBLIC_API_URL
  });

  // Function to refresh data
  const refresh = useCallback(() => {
    refetch();
  }, [refetch]);

  // Update settings mutation
  const updateMutation = useMutation({
    mutationFn: async (data: Partial<Settings>) => {
      const headers = await getAuthHeaders();

      const response = await fetch(getApiUrl(), {
        method: 'PUT',
        headers: {
          ...headers,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
      });

      if (!response.ok) {
        throw new Error(`Error updating settings: ${response.statusText}`);
      }

      return response.json();
    },
    onSuccess: () => {
      // Invalidate the settings query to trigger a refetch
      queryClient.invalidateQueries({ queryKey: ['settings', parentId, id] });
    }
  });

  // Wrapper function for update mutation
  const updateSettings = useCallback((data: Partial<Settings>) => {
    return new Promise((resolve, reject) => {
      updateMutation.mutate(data, {
        onSuccess: (data) => {
          resolve(data);
        },
        onError: (error: Error) => {
          reject(error);
        }
      });
    });
  }, [updateMutation]);

  // Create form with combined schema
  const form = useForm<SettingsFormValues>({
    resolver: zodResolver(SettingsFormSchema),
    defaultValues: {
      // Invoice Settings defaults
      invoice_prefix: settings?.invoice_prefix ?? 'INV',
      enable_invoice_pdfs: settings?.enable_invoice_pdfs ?? false,

      // Subscription Settings defaults
      reminder_days: settings?.reminder_days ?? 0,
      email_reminders: settings?.email_reminders ?? false,
      cancel_on_failure: settings?.cancel_on_failure ?? false,
      retry_policy: settings?.retry_policy ?? {
        attempts: 3,
        retry_period: 14,
        failure_action: "cancel",
      },
    },
  });

  // Update form values when settings are loaded
  useEffect(() => {
    if (settings) {
      form.reset({
        // Invoice Settings
        invoice_prefix: settings.invoice_prefix ?? 'INV',
        enable_invoice_pdfs: settings.enable_invoice_pdfs ?? false,

        // Subscription Settings
        reminder_days: settings.reminder_days || 0,
        email_reminders: settings.email_reminders || false,
        cancel_on_failure: settings.cancel_on_failure || false,
        retry_policy: {
          attempts: settings.retry_policy?.attempts || 3,
          retry_period: settings.retry_policy?.retry_period || 14,
          failure_action: settings.retry_policy?.failure_action || "cancel",
        },
      });
    }
  }, [settings, form]);

  // Submit function that updates all settings
  const submitForm = async () => {
    try {
      const values = form.getValues();
      await updateSettings(values);
    } catch (error) {
      console.error("Failed to update settings:", error);
      throw error;
    }
  };

  // Create the combined context value
  const contextValue: SettingsContextType = {
    // Settings data management
    settings,
    isLoading,
    error,
    refresh,
    updateSettings,

    // Form state management
    form,
    submitForm,
  };

  return (
    <SettingsContext.Provider value={contextValue}>
      {children}
    </SettingsContext.Provider>
  );
}

// Custom hook to use the settings context
export function useSettings() {
  const context = useContext(SettingsContext);
  if (context === undefined) {
    throw new Error("useSettings must be used within a SettingsProvider");
  }
  return context;
}

// For backward compatibility with existing code that uses useSettingsForm
export function useSettingsForm() {
  const context = useContext(SettingsContext);
  if (context === undefined) {
    throw new Error("useSettingsForm must be used within a SettingsProvider");
  }
  return {
    form: context.form,
    isLoading: context.isLoading,
    submitForm: context.submitForm,
  };
}
