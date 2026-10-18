/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  base: "/static/assets/",
  build: {
    outDir: "dist/client",
    emptyOutDir: true,
    assetsDir: ".",
    cssCodeSplit: false,
  },
  server: {
    fs: { allow: [".."] },
  },
  test: {
    environment: "jsdom",
    include: ["test/**/*.test.{ts,tsx}"],
  },
});