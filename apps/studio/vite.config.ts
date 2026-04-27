import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiProxyTarget = env.VITE_MANMU_API_PROXY_TARGET ?? 'http://127.0.0.1:8080'

  return {
    plugins: [react(), tailwindcss()],
    server: {
      proxy: {
        '/api': apiProxyTarget,
        '/healthz': apiProxyTarget,
        '/readyz': apiProxyTarget,
      },
    },
  }
})
