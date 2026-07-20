import {Suspense} from "react";
import ViewInvoice from "@/app/(dashboard)/invoices/[id]/components/view-invoice";
import {InvoiceProvider} from "./invoice-context";
import {fetchInvoice} from "./data";
import {loadAuthProvider} from "@getpaidhq/auth/server";
import { ResourceDetailSkeleton } from "@/components/skeletons";

export default async function Page({params}: { params: Promise<{ id: string }> }) {
  const authProvider = loadAuthProvider();
  const authHeaders = await authProvider.getAuthHeader()
  const { id } = await params
  // Fetch invoice data on the server
  let initialData;
  try {
    initialData = await fetchInvoice(id, authHeaders);
  } catch (error) {
    console.error("Error fetching invoice:", error);
    // We'll let the client-side handle the error display
  }

  return (
    <Suspense fallback={<ResourceDetailSkeleton metricsCount={4} showTabs={true} detailSections={2} />}>
      <InvoiceProvider id={id} initialData={initialData}>
        <ViewInvoice/>
      </InvoiceProvider>
    </Suspense>
  );
}
