import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Hamburger',
  description: '高性能多协议网关与代理编排平台',
  lang: 'zh-CN',
  cleanUrls: true,
  lastUpdated: true,
  themeConfig: {
    logo: '🍔',
    siteTitle: 'Hamburger Docs',
    nav: [
      { text: '首页', link: '/' },
      { text: '功能总览', link: '/guide/overview' },
      { text: '配置指南', link: '/guide/configuration' },
      { text: '快速开始', link: '/guide/quick-start' }
    ],
    sidebar: [
      {
        text: '文档导航',
        items: [
          { text: '功能总览', link: '/guide/overview' },
          { text: '核心特性', link: '/guide/features' },
          { text: '配置指南', link: '/guide/configuration' },
          { text: '快速开始', link: '/guide/quick-start' }
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/' }
    ],
    footer: {
      message: 'Built with VitePress',
      copyright: 'Hamburger Project'
    },
    search: {
      provider: 'local'
    },
    outline: {
      level: [2, 3],
      label: '目录'
    },
    docFooter: {
      prev: '上一页',
      next: '下一页'
    }
  },
  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    }
  }
})
