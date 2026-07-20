import OnboardingPage from "@/app/(auth)/onboarding/_components/onboarding";
import {loadAuthProvider} from "@getpaidhq/auth/server";
import {redirect} from "next/navigation";


export default async function Onboarding() {
  const authProvider = loadAuthProvider();
  const user = await authProvider.currentUser()

  console.log('onboarding', user.orgId)
  if (user?.orgId) {
    // Handle case where user is not in an organization
    redirect("/dashboard");
  }

  // Use orgId for your logic
  return <OnboardingPage/>;
}
