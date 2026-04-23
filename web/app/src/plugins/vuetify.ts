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
    defaultTheme: 'system',
    utilities: false,
    themes: {
      light: {
        dark: false,
        colors: {
          'icon-sun':           '#fbbf24', // amber-400
          'icon-moon':          '#818cf8', // indigo-400
        },
      },
      dark: {
        dark: true,
        colors: {
          'icon-sun':           '#fbbf24', // amber-400
          'icon-moon':          '#818cf8', // indigo-400
        },
      },
    },
  },
  display: {
    mobileBreakpoint: 'md',
    thresholds: {
      // repeated in tailwind.css and settings.scss
      xs: 0,
      sm: 400,
      md: 840,
      lg: 1145,
      xl: 1545,
      xxl: 2138,
    },
  },
})
