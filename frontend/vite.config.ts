import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // Split large vendor dependencies into separate chunks for better caching
          'vendor-antd': ['antd', '@ant-design/icons', '@ant-design/pro-components'],
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          'vendor-charts': ['recharts'],
          'vendor-utils': ['axios', 'dayjs', 'zustand', 'i18next', 'react-i18next'],
        },
      },
    },
  },
})
