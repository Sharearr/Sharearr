import '../styles/layers.css'
import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi-svg'
import { mdiMagnify, mdiHeart, mdiAccountCircle, mdiWeatherNight, mdiWeatherSunny } from '@mdi/js'

export default createVuetify({
  icons: {
    defaultSet: 'mdi',
    aliases: {
      ...aliases,
      search:  mdiMagnify,
      donate:  mdiHeart,
      user:    mdiAccountCircle,
      moon:    mdiWeatherNight,
      sun:     mdiWeatherSunny,
    },
    sets: {
      mdi,
    },
  },
  theme: {
    defaultTheme: window.matchMedia?.('(prefers-color-scheme: dark)')?.matches ? 'dark' : 'light',
    utilities: false,
    themes: {
      light: {
        dark: false,
        colors: {
          'icon-sun':           '#fbbf24', // amber-400
          'icon-moon':          '#818cf8', // indigo-400
          background:           '#E5E7EB', // gray-200
          surface:              '#FFFFFF', // white — contrast vs background: ~1.25:1 + elevation shadow
          'surface-bright':     '#FFFFFF',
          'surface-light':      '#F3F4F6', // gray-100
          'surface-variant':    '#F3F4F6', // gray-100
          'on-surface':         '#111827', // gray-900
          'on-surface-variant': '#4B5563', // gray-600
        },
      },
      dark: {
        dark: true,
        colors: {
          'icon-sun':           '#fbbf24', // amber-400
          'icon-moon':          '#818cf8', // indigo-400
          background:           '#030712', // gray-950
          surface:              '#111827', // gray-900
          'surface-bright':     '#374151', // gray-700
          'surface-light':      '#1F2937', // gray-800
          'surface-variant':    '#1F2937', // gray-800
          'on-surface':         '#F9FAFB', // gray-50
          'on-surface-variant': '#D1D5DB', // gray-300
        },
      },
    },
  },
  display: {
    mobileBreakpoint: 'md',
    thresholds: {
      // keep in sync with tailwind.css and settings.scss
      xs: 0, sm: 600, md: 960, lg: 1280, xl: 1920, xxl: 2560,
    },
  },
})
