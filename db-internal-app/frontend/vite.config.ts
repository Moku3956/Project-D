import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 開発中は /api へのリクエストをGoのサーバー(既定 :8082)へプロキシする。
// Cookieベースのセッションを使うため、同一オリジン扱いにできるプロキシが必須。
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: process.env.API_TARGET ?? 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
})
