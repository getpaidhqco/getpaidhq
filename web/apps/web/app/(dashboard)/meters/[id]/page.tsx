import { Suspense } from "react";
import ViewMeter from "@/app/(dashboard)/meters/[id]/components/view-meter";
import { MeterProvider } from "./meter-context";
import { fetchMeter } from "./data";
import { loadAuthProvider } from "@getpaidhq/auth/server";
import { ResourceDetailSkeleton } from "@/components/skeletons";

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const authProvider = loadAuthProvider();
  const authHeaders = await authProvider.getAuthHeader()

  // Fetch meter data on the server
  let initialData;
  try {
    initialData = await fetchMeter(id, authHeaders);
  } catch (error) {
    console.error("Error fetching meter:", error);
    // We'll let the client-side handle the error display
  }

  return (
    <Suspense fallback={<ResourceDetailSkeleton metricsCount={3} showTabs={true} detailSections={2} />}>
      <MeterProvider id={id} initialData={initialData}>
        <ViewMeter />
      </MeterProvider>
    </Suspense>
  );
}
