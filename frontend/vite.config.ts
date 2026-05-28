import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "url";
import { dirname, resolve } from "path";

const __dirname = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@heroui/toast": resolve(__dirname, "node_modules/@heroui/toast/dist/index.js"),
    },
  },
  build: {
    outDir: resolve(__dirname, "../backend/internal/httpserver/frontend_dist"),
    emptyOutDir: true,
  },
  server: {
    port: 3000,
    proxy: {
      "/api": "http://localhost:8080",
      "/channel": "http://localhost:8080",
      "/v1": "http://localhost:8080",
    },
  },
});
