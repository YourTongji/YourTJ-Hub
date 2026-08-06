import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      // 开发时同源代理到 Go 后端，避免 CORS
      "/api": "http://localhost:8080",
    },
  },
  build: {
    // 产物输出到 server/webdist，由 go:embed 打包（见 ADR-0001）
    outDir: "../server/webdist",
    emptyOutDir: true,
  },
});
