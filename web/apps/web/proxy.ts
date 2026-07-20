import { NextRequest, NextResponse } from "next/server";
import { loadAuthProvider } from "@getpaidhq/auth/server";

const authProvider = loadAuthProvider();

async function defaultProxy(req: NextRequest) {
  // Fallback for providers without their own middleware (eg. apikey):
  // resolve the user so request-time auth runs, then let the request through.
  await authProvider.auth(req);
  return NextResponse.next();
}

export const proxy =
  (authProvider.getMiddleware as ((req: NextRequest) => Response | Promise<Response>) | undefined) ??
  defaultProxy;

export default proxy;

export const config = {
  matcher: [
    // Skip Next.js internals and all static files, unless found in search params
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    // Always run for API routes
    "/(api|trpc)(.*)",
  ],
};
