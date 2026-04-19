import '../styles/layers.css'
import 'vuetify/styles'
import { h, type Component } from 'vue'
import { createVuetify } from 'vuetify'
import type { IconSet, IconProps } from 'vuetify'
import SunIconColored from '../components/icons/SunIconColored.vue'
import MoonIconColored from '../components/icons/MoonIconColored.vue'

const heroicons: IconSet = {
  component: (props: IconProps) => h(props.icon as Component, { 'aria-hidden': 'true' }),
}

export default createVuetify({
  icons: {
    defaultSet: 'heroicons',
    sets: { heroicons },
    aliases: {
      sun: SunIconColored,
      moon: MoonIconColored,
    },
  },
  theme: {
    defaultTheme: window.matchMedia?.('(prefers-color-scheme: dark)')?.matches ? 'dark' : 'light',
    utilities: false,
    themes: {
      light: {
        dark: false,
        colors: {
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
