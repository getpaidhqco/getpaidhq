import {LoginComponent} from "@getpaidhq/auth/client";

export default function LoginPage() {
  return <LoginComponent afterSignInUrl={"/onboarding"}/>;
}
