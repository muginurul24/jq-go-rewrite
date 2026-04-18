import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return undefined
          }

          if (
            id.includes('/react/') ||
            id.includes('/react-dom/') ||
            id.includes('@tanstack') ||
            id.includes('react-hook-form') ||
            id.includes('/zod/')
          ) {
            return 'vendor-react'
          }

          if (
            id.includes('framer-motion') ||
            id.includes('lucide-react') ||
            id.includes('next-themes') ||
            id.includes('recharts') ||
            id.includes('sonner') ||
            id.includes('radix-ui')
          ) {
            return 'vendor-ui'
          }

          return 'vendor-misc'
        },
      },
    },
  },
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/backoffice/api/': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
