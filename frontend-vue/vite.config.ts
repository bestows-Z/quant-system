import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://127.0.0.1:8888',
        changeOrigin: true
      },
      '/ws': {
        target: process.env.VITE_WS_BASE_URL || 'ws://127.0.0.1:8081',
        changeOrigin: true,
        ws: true
      }
    }
  }
})
