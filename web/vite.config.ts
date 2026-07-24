import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'
import path from 'path'

export default defineConfig({
  base: '/panel/',
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: false,
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,jpg,webp,woff2}'],
        runtimeCaching: [
          {
            urlPattern: /\/api\//,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-cache',
              expiration: {
                maxEntries: 100,
                maxAgeSeconds: 300,
              },
            },
          },
        ],
      },
      manifest: {
        name: 'Blog API 管理面板',
        short_name: 'Blog API',
        description: 'Blog API 管理面板',
        theme_color: '#6fa67c',
        background_color: '#f7f3e9',
        display: 'standalone',
        orientation: 'portrait-primary',
        icons: [
          {
            src: '/panel/pwa-icon-512x512.webp',
            sizes: '512x512',
            type: 'image/webp',
            purpose: 'any maskable',
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    host: true,
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:10024',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: '../data/panel',
    emptyOutDir: true,
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: 'element-plus',
              test: /node_modules\/(?:\.pnpm\/)?(?:element-plus|@element-plus)/
            },
            {
              name: 'echarts',
              test: /node_modules\/(?:\.pnpm\/)?(?:echarts|zrender)/
            },
            {
              name: 'editor',
              test: /node_modules\/(?:\.pnpm\/)?(?:@codemirror|codemirror|@lezer)/
            }
          ]
        }
      }
    }
  }
})
