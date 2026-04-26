import 'unfonts.css'
import './styles/tailwind.css'
import './styles/main.scss'
import { createApp } from 'vue'
import vuetify from './plugins/vuetify'
import pinia from './stores'
import App from './App.vue'

createApp(App).use(vuetify).use(pinia).mount('#app')
