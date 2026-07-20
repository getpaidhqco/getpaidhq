import {loadAuthProvider} from "@getpaidhq/auth/server";
import {redirect} from "next/navigation";
import Dashboard from "@/app/(dashboard)/dashboard/_components/dashboard";


export default async function Page() {
  const authProvider = loadAuthProvider();
  const user = await authProvider.currentUser()

  if (!user?.orgId) {
    // Handle case where user is not in an organization
    redirect("/onboarding");
  }

  // Use orgId for your logic
  return <Dashboard/>;
}
