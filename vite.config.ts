import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import path from "path";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    watch: {
      ignored: ["**/src-tauri/**", "**/core/**"],
    },
  },
  build: {
    // Performance: target modern browsers only (Tauri uses latest Chromium)
    target: "esnext",
    // Smaller chunks
    chunkSizeWarningLimit: 400,
    rollupOptions: {
      output: {
        manualChunks: {
          // Split heavy deps into separate chunks
          "vendor-vue": ["vue", "vue-router", "pinia"],
          "vendor-xterm": ["@xterm/xterm", "@xterm/addon-fit", "@xterm/addon-search", "@xterm/addon-web-links"],
          "vendor-ui": ["class-variance-authority", "clsx", "tailwind-merge"],
        },
      },
    },
    // Minify for smaller bundle
    minify: "esbuild",
    // Source maps off in production
    sourcemap: false,
  },
  // Preload terminal theme data
  optimizeDeps: {
    include: ["@xterm/xterm", "@xterm/addon-fit", "@xterm/addon-search", "@xterm/addon-web-links"],
  },
});
