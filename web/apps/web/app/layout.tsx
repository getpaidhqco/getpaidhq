import type {Metadata} from "next";
import {GeistSans, GeistMono} from "geist/font";
import "./globals.css";
import QueryProvider from "@/app/query-provider";

import {Toaster} from "@/components/ui/sonner";
import {FrontendAuthProvider} from "@getpaidhq/auth";
import {ThemeScript} from "@/app/theme-script";


export const metadata: Metadata = {
  title: "GetPaidHQ",
  description: "GetPaidHQ gives South African SaaS and subscription businesses the payment infrastructure they deserve. Built for local processors, designed for global scale.",
};

export default function RootLayout({
                                     children,
                                   }: Readonly<{
  children: React.ReactNode;
}>) {



  return (
    <html lang="en" suppressHydrationWarning>
    <body className={`${GeistSans.variable} ${GeistMono.variable} bg-background text-foreground antialiased`}>
    <ThemeScript/>
    <FrontendAuthProvider>
      <QueryProvider>
        {children}
      </QueryProvider>
      <Toaster/>
    </FrontendAuthProvider>
    </body>
    </html>

  );
}
