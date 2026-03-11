import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import HeroTerminal from './components/HeroTerminal.vue'
import './custom.css'

export default {
  ...DefaultTheme,
  enhanceApp({ app }) {
    app.component('HeroTerminal', HeroTerminal)
  }
} satisfies Theme
