import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite'
import vuetify from 'vite-plugin-vuetify'
import Fonts from 'unplugin-fonts/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    tailwindcss(),
    vue(),
    vuetify({ styles: { configFile: 'src/styles/settings.scss' } }),
    Fonts({
      custom: {
        families: [
          {
            name: 'Roboto',
            src: 'node_modules/@fontsource/roboto/files/roboto-latin-700-normal.woff2',
          },
          {
            name: 'Space Grotesk',
            src: 'node_modules/@fontsource/space-grotesk/files/space-grotesk-latin-*-normal.woff2',
          },
          {
            name: 'Hack',
            src: 'node_modules/hack-font/build/web/fonts/hack-*-subset.woff2',
          },
        ],
      },
    }),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
})
