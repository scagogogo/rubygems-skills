import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

// https://vitepress.dev/reference/site-config
export default withMermaid(
  defineConfig({
  lang: 'en-US',
  title: 'rubygems-skills',
  description: 'A production-ready Go SDK for the RubyGems.org API — built for AI agents.',

  // GitHub Pages project site: https://<user>.github.io/rubygems-skills/
  base: '/rubygems-skills/',

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['meta', { name: 'theme-color', content: '#cc342d' }],
    ['meta', { property: 'og:title', content: 'rubygems-skills' }],
    ['meta', { property: 'og:description', content: 'A production-ready Go SDK for the RubyGems.org API — built for AI agents.' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
  ],

  lastUpdated: true,
  cleanUrls: true,

  themeConfig: {
    // Social links in the navbar
    socialLinks: [
      { icon: 'github', link: 'https://github.com/scagogogo/rubygems-skills' },
      { icon: 'x', link: 'https://x.com/' },
    ],

    // Search
    search: {
      provider: 'local',
      options: {
        translations: {
          button: {
            buttonText: 'Search docs',
            buttonAriaLabel: 'Search',
          },
        },
      },
    },

    // Top navigation
    nav: [
      { text: 'Guide', link: '/guide/what-is-rubygems-skills', activeMatch: '/guide/' },
      { text: 'AI Agents', link: '/ai-agents/claude-code', activeMatch: '/ai-agents/' },
      { text: 'API', link: '/api/repository', activeMatch: '/api/' },
      { text: 'CLI', link: '/cli/install', activeMatch: '/cli/' },
      { text: 'Auto-Install', link: '/auto-install/overview', activeMatch: '/auto-install/' },
      {
        text: 'GitHub',
        link: 'https://github.com/scagogogo/rubygems-skills',
      },
    ],

    // Sidebar
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

    // Page footer
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024-present rubygems-skills',
    },

    // Edit link
    editLink: {
      pattern: 'https://github.com/scagogogo/rubygems-skills/edit/main/website/docs/:path',
      text: 'Edit this page on GitHub',
    },

    // Last updated text
    lastUpdated: {
      text: 'Last updated',
    },

    // Prev/next page links
    docFooter: {
      prev: 'Previous',
      next: 'Next',
    },

    // Outline (right-side table of contents)
    outline: {
      level: [2, 3],
      label: 'On this page',
    },

    // Dark mode toggle
    darkModeSwitchLabel: 'Theme',
    sidebarMenuLabel: 'Menu',
    returnToTopLabel: 'Back to top',
  },

  // Mermaid diagram support — ```mermaid blocks render as SVG.
  // Theme follows the site's dark/light mode automatically.
  mermaid: {
    // see vitepress-plugin-mermaid for options
  },
}),
)
