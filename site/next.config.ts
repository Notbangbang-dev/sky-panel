import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Pin the workspace root to this project — the parent home directory has
  // an unrelated package-lock.json that would otherwise confuse detection.
  turbopack: {
    root: __dirname,
  },
};

export default nextConfig;
