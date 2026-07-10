import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The app talks to the Go gateway (default http://localhost:8090). The gateway
// sends CORS headers, so no dev proxy is required; override the base URLs via
// VITE_API_BASE / VITE_WS_BASE (see .env.example).
export default defineConfig({
  plugins: [react()],
  server: { port: 5173 },
});
