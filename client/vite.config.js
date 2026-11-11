import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    // Di dev Podman: gateway diakses via nama service Podman network
    // Di luar Podman (npm run dev lokal): ubah VITE_API_URL=http://localhost:8000
    proxy: {
      '/api': {
        target: process.env.VITE_GATEWAY_URL || 'http://localhost:8000',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
    // HMR melalui Podman — gunakan host mesin lokal agar WebSocket terhubung
    hmr: {
      host: 'localhost',
      port: 5173,
    },
    watch: {
      // Polling diperlukan di Podman (inotify tidak selalu tersedia di volume mount)
      usePolling: true,
      interval: 300,
    },
  },
})
