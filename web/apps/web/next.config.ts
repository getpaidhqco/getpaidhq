import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  typescript: {
    // TODO: Fix 163 pre-existing TypeScript errors and remove this
    ignoreBuildErrors: true,
  },
  transpilePackages: ['@getpaidhq/sdk'],
  async redirects() {
    return [
      {
        source: '/',
        destination: '/dashboard',
        permanent: true, // This will make it a 308 permanent redirect
      },
    ]
  },
};

export default nextConfig;