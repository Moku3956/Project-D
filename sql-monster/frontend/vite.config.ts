import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 開発中は /api へのリクエストをGoのサーバー(既定 :8081)へプロキシする。
// これでフロント側は常に相対パスで呼べるようになり、CORSも避けられる。
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: process.env.API_TARGET ?? 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
})
