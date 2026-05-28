import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "url";
import { dirname, resolve, join } from "path";
import { existsSync, readdirSync, statSync } from "fs";

const __dirname = dirname(fileURLToPath(import.meta.url));

// Fix missing .mjs files in react-aria package
function reactAriaMjsFix(): Plugin {
  const exportsDir = join(__dirname, "node_modules/react-aria/dist/exports");
  const aliases: Record<string, string> = {};

  if (existsSync(exportsDir)) {
    for (const file of readdirSync(exportsDir)) {
      if (file.endsWith(".js") && !file.endsWith(".cjs") && !file.endsWith(".js.map")) {
        const base = file.replace(/\.js$/, "");
        if (!existsSync(join(exportsDir, `${base}.mjs`))) {
          aliases[`react-aria/${base}`] = join(exportsDir, file);
        }
      }
    }
  }

  const privateDir = join(exportsDir, "private");
  if (existsSync(privateDir)) {
    for (const dir of readdirSync(privateDir)) {
      const dirPath = join(privateDir, dir);
      if (!statSync(dirPath).isDirectory()) continue;
      for (const file of readdirSync(dirPath)) {
        if (file.endsWith(".js") && !file.endsWith(".cjs") && !file.endsWith(".js.map")) {
          const base = file.replace(/\.js$/, "");
          const key = `react-aria/private/${dir}/${base}`;
          if (!existsSync(join(dirPath, `${base}.mjs`))) {
            aliases[key] = join(dirPath, file);
          }
        }
      }
    }
  }

  return {
    name: "react-aria-mjs-fix",
    config() {
      return { resolve: { alias: aliases } };
    },
  };
}

export default defineConfig({
  plugins: [react(), tailwindcss(), reactAriaMjsFix()],
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
