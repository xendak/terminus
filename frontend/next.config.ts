import type { NextConfig } from "next";

// The Go server owns /api/*. Proxying it through Next keeps the session
// cookie (HttpOnly, SameSite=Lax) first-party for the browser.
const backend = process.env.BACKEND_URL ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  devIndicators: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${backend}/api/:path*` }];
  },
};

export default nextConfig;
