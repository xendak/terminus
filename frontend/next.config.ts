import type { NextConfig } from "next";

// The Go server owns /api/*. Proxying it through Next keeps the session
// cookie (HttpOnly, SameSite=Lax) first-party for the browser.
const backend = process.env.BACKEND_URL ?? "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  devIndicators: false,
  // The dev server only serves its JS to localhost by default; without this,
  // opening http://127.0.0.1:3210 left forms without their handlers.
  allowedDevOrigins: ["127.0.0.1"],
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${backend}/api/:path*` }];
  },
};

export default nextConfig;
