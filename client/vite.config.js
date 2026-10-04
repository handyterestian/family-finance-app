import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    // Di dev Docker: gateway diakses via nama service Docker network
    // Di luar Docker (npm run dev lokal): ubah VITE_API_URL=http://localhost:8000
    proxy: {
      '/api': {
        target: process.env.VITE_GATEWAY_URL || 'http://localhost:8000',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
    // HMR melalui Docker — gunakan host mesin lokal agar WebSocket terhubung
    hmr: {
      host: 'localhost',
      port: 5173,
    },
    watch: {
      // Polling diperlukan di Docker (inotify tidak selalu tersedia di volume mount)
      usePolling: true,
      interval: 300,
    },
  },
})
