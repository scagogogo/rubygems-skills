import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

// https://vitepress.dev/reference/site-config
export default withMermaid(
  defineConfig({
  title: 'rubygems-skills',
  lastUpdated: true,
  cleanUrls: true,

  // Source directory: docs/ (so index.md becomes the home page)
  srcDir: 'docs',

  // GitHub Pages project site: https://<user>.github.io/rubygems-skills/
  base: '/rubygems-skills/',

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['meta', { name: 'theme-color', content: '#cc342d' }],
    ['meta', { property: 'og:title', content: 'rubygems-skills' }],
    ['meta', { property: 'og:description', content: 'A production-ready Go SDK for the RubyGems.org API — built for AI agents.' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
  ],

  // Internationalization
  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      description: 'A production-ready Go SDK for the RubyGems.org API — built for AI agents.',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/what-is-rubygems-skills', activeMatch: '/guide/' },
          { text: 'AI Agents', link: '/ai-agents/claude-code', activeMatch: '/ai-agents/' },
          { text: 'API', link: '/api/repository', activeMatch: '/api/' },
          { text: 'CLI', link: '/cli/install', activeMatch: '/cli/' },
          { text: 'Auto-Install', link: '/auto-install/overview', activeMatch: '/auto-install/' },
          { text: 'GitHub', link: 'https://github.com/scagogogo/rubygems-skills' },
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Introduction',
              items: [
                { text: 'What is rubygems-skills?', link: '/guide/what-is-rubygems-skills' },
                { text: 'Why use it?', link: '/guide/why' },
                { text: 'How it works', link: '/guide/how-it-works' },
              ],
            },
            {
              text: 'Getting Started',
              items: [
                { text: 'Installation', link: '/guide/installation' },
                { text: 'Quick Start', link: '/guide/quick-start' },
                { text: 'Configuration', link: '/guide/configuration' },
              ],
            },
            {
              text: 'Features',
              items: [
                { text: 'Mirrors', link: '/guide/mirrors' },
                { text: 'Caching', link: '/guide/caching' },
                { text: 'Retry & Backoff', link: '/guide/retry' },
                { text: 'Bulk Operations', link: '/guide/bulk-operations' },
                { text: 'Error Handling', link: '/guide/error-handling' },
              ],
            },
          ],
          '/ai-agents/': [
            {
              text: 'AI Agent Integration',
              items: [
                { text: 'Overview', link: '/ai-agents/overview' },
                { text: 'Claude Code', link: '/ai-agents/claude-code' },
                { text: 'Codex (OpenAI)', link: '/ai-agents/codex' },
                { text: 'Copy-Paste Prompts', link: '/ai-agents/prompts' },
              ],
            },
          ],
          '/api/': [
            {
              text: 'API Reference',
              items: [
                { text: 'Repository (Read)', link: '/api/repository' },
                { text: 'WriteRepository (Write)', link: '/api/write-repository' },
                { text: 'Models', link: '/api/models' },
                { text: 'Options', link: '/api/options' },
                { text: 'Errors', link: '/api/errors' },
              ],
            },
          ],
          '/cli/': [
            {
              text: 'CLI Tool',
              items: [
                { text: 'Installation', link: '/cli/install' },
                { text: 'Commands', link: '/cli/commands' },
                { text: 'Examples', link: '/cli/examples' },
              ],
            },
          ],
          '/auto-install/': [
            {
              text: 'Auto-Install',
              items: [
                { text: 'Overview', link: '/auto-install/overview' },
                { text: 'Supported Platforms', link: '/auto-install/platforms' },
                { text: 'Usage', link: '/auto-install/usage' },
              ],
            },
          ],
        },
        footer: {
          message: 'Released under the MIT License.',
          copyright: 'Copyright © 2024-present rubygems-skills',
        },
        editLink: {
          pattern: 'https://github.com/scagogogo/rubygems-skills/edit/main/website/docs/:path',
          text: 'Edit this page on GitHub',
        },
        lastUpdated: {
          text: 'Last updated',
        },
        docFooter: {
          prev: 'Previous',
          next: 'Next',
        },
        outline: {
          level: [2, 3],
          label: 'On this page',
        },
        darkModeSwitchLabel: 'Theme',
        sidebarMenuLabel: 'Menu',
        returnToTopLabel: 'Back to top',
      },
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      description: '面向 RubyGems.org API 的生产级 Go SDK —— 为 AI Agent 而构建。',
      link: '/zh/',
      themeConfig: {
        nav: [
          { text: '指南', link: '/zh/guide/what-is-rubygems-skills', activeMatch: '/zh/guide/' },
          { text: 'AI Agent', link: '/zh/ai-agents/claude-code', activeMatch: '/zh/ai-agents/' },
          { text: 'API', link: '/zh/api/repository', activeMatch: '/zh/api/' },
          { text: 'CLI', link: '/zh/cli/install', activeMatch: '/zh/cli/' },
          { text: '自动安装', link: '/zh/auto-install/overview', activeMatch: '/zh/auto-install/' },
          { text: 'GitHub', link: 'https://github.com/scagogogo/rubygems-skills' },
        ],
        sidebar: {
          '/zh/guide/': [
            {
              text: '介绍',
              items: [
                { text: '什么是 rubygems-skills？', link: '/zh/guide/what-is-rubygems-skills' },
                { text: '为什么选择它？', link: '/zh/guide/why' },
                { text: '工作原理', link: '/zh/guide/how-it-works' },
              ],
            },
            {
              text: '快速开始',
              items: [
                { text: '安装', link: '/zh/guide/installation' },
                { text: '快速入门', link: '/zh/guide/quick-start' },
                { text: '配置', link: '/zh/guide/configuration' },
              ],
            },
            {
              text: '功能特性',
              items: [
                { text: '镜像源', link: '/zh/guide/mirrors' },
                { text: '缓存', link: '/zh/guide/caching' },
                { text: '重试与退避', link: '/zh/guide/retry' },
                { text: '批量操作', link: '/zh/guide/bulk-operations' },
                { text: '错误处理', link: '/zh/guide/error-handling' },
              ],
            },
          ],
          '/zh/ai-agents/': [
            {
              text: 'AI Agent 集成',
              items: [
                { text: '概览', link: '/zh/ai-agents/overview' },
                { text: 'Claude Code', link: '/zh/ai-agents/claude-code' },
                { text: 'Codex (OpenAI)', link: '/zh/ai-agents/codex' },
                { text: '复制粘贴提示词', link: '/zh/ai-agents/prompts' },
              ],
            },
          ],
          '/zh/api/': [
            {
              text: 'API 参考',
              items: [
                { text: 'Repository (读操作)', link: '/zh/api/repository' },
                { text: 'WriteRepository (写操作)', link: '/zh/api/write-repository' },
                { text: '数据模型', link: '/zh/api/models' },
                { text: '配置选项', link: '/zh/api/options' },
                { text: '错误类型', link: '/zh/api/errors' },
              ],
            },
          ],
          '/zh/cli/': [
            {
              text: '命令行工具',
              items: [
                { text: '安装', link: '/zh/cli/install' },
                { text: '命令参考', link: '/zh/cli/commands' },
                { text: '使用示例', link: '/zh/cli/examples' },
              ],
            },
          ],
          '/zh/auto-install/': [
            {
              text: '自动安装',
              items: [
                { text: '概览', link: '/zh/auto-install/overview' },
                { text: '支持的平台', link: '/zh/auto-install/platforms' },
                { text: '使用方法', link: '/zh/auto-install/usage' },
              ],
            },
          ],
        },
        footer: {
          message: '基于 MIT 许可证发布。',
          copyright: '版权所有 © 2024 至今 rubygems-skills',
        },
        editLink: {
          pattern: 'https://github.com/scagogogo/rubygems-skills/edit/main/website/docs/:path',
          text: '在 GitHub 上编辑此页',
        },
        lastUpdated: {
          text: '最后更新',
        },
        docFooter: {
          prev: '上一页',
          next: '下一页',
        },
        outline: {
          level: [2, 3],
          label: '本页目录',
        },
        darkModeSwitchLabel: '主题',
        sidebarMenuLabel: '菜单',
        returnToTopLabel: '返回顶部',
      },
    },
  },

  // Shared theme config
  themeConfig: {
    socialLinks: [
      { icon: 'github', link: 'https://github.com/scagogogo/rubygems-skills' },
      { icon: 'x', link: 'https://x.com/' },
    ],
    search: {
      provider: 'local',
    },
  },

  // Mermaid diagram support — ```mermaid blocks render as SVG.
  // Theme follows the site's dark/light mode automatically.
  mermaid: {
    // see vitepress-plugin-mermaid for options
  },
}),
)
