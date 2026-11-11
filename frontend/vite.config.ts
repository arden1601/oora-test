import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Port the Go service listens on locally. Matches BACKEND_HOST_PORT in the root .env.
const backendPort = process.env.BACKEND_HOST_PORT ?? '8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // The browser talks to /api on the same origin, so the session cookie stays
    // same-site. Vite forwards it to the Go service and drops the /api prefix.
    proxy: {
      '/api': {
        target: `http://localhost:${backendPort}`,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
