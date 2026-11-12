// Vite 配置：@ 指向 src；开发时把 /api 转发给后端，前后端同源，不需要处理跨域。
import path from 'node:path'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        timeout: 600_000,
        proxyTimeout: 600_000,
      },
    },
  },
  test: {
    environment: 'node',
  },
  // PDF worker 由 FilePreview 静态 import，跑在主线程。不要用 ?url 把 pdf.worker.min.mjs 打成独立 asset。
  optimizeDeps: {
    include: ['pdfjs-dist', 'pdfjs-dist/build/pdf.worker.min.mjs'],
  },
})
