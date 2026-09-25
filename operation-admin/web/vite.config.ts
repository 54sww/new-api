import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_')
  const serverUrl = env.VITE_API_PROXY_TARGET || 'http://localhost:3100'

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
      },
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `@use "@/styles/variables.scss" as *;`,
        },
      },
    },
    server: {
      host: '0.0.0.0',
      port: 3101,
      strictPort: true,
      proxy: {
        '/api': {
          target: serverUrl,
          changeOrigin: true,
        },
      },
    },
  }
})
