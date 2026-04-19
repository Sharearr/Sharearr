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
  },
  display: {
    mobileBreakpoint: 'md',
    thresholds: {
      // keep in sync with tailwind.css and settings.scss
      xs: 0, sm: 600, md: 960, lg: 1280, xl: 1920, xxl: 2560,
    },
  },
})
