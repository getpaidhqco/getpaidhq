import "@/app/globals.css";
import { FrontendAuthProvider } from "@getpaidhq/auth";
import { loadAuthProvider } from "@getpaidhq/auth/server";

import { AppShell } from "@/components/app-shell";
import { BreadcrumbProvider } from "@/context/breadcrumb-context";
import { GetPaidHQClientProvider } from "@/components/getpaidhq-client-provider";
import { ThemeProvider } from "@/components/theme-provider";

export default async function DashboardLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const authProvider = loadAuthProvider();
  await authProvider.getToken();

  return (
    <ThemeProvider defaultTheme="system">
      <FrontendAuthProvider>
        <GetPaidHQClientProvider baseUrl={process.env.NEXT_PUBLIC_API_URL}>
          <BreadcrumbProvider>
            <AppShell>{children}</AppShell>
          </BreadcrumbProvider>
        </GetPaidHQClientProvider>
      </FrontendAuthProvider>
    </ThemeProvider>
  );
}
