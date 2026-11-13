// @ts-check
import { defineConfig, fontProviders } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';
import { remarkRepositoryLinks } from './src/lib/repository-markdown.mjs';
import { unified } from '@astrojs/markdown-remark';

export default defineConfig({
  site: 'https://ghchinoy.github.io',
  base: '/drivectl',
  output: 'static',
  trailingSlash: 'never',

  integrations: [starlight({
    title: 'drivectl',
    description: 'Google Drive, Docs, and Sheets CLI & MCP server for developers and AI coding agents.',
    favicon: '/favicon.svg',
    credits: false,
    expressiveCode: {
      defaultProps: { frame: 'none' },
      styleOverrides: {
        borderRadius: '8px',
        borderColor: 'var(--md-sys-color-outline-variant)',
        codeBackground: 'var(--md-sys-color-surface-container-low)',
        frames: { shadowColor: 'transparent' },
      },
    },
    customCss: ['./src/styles/docs.css'],
    components: {
      Header: './src/components/docs/Header.astro',
      Head: './src/components/docs/Head.astro',
      PageTitle: './src/components/docs/PageTitle.astro',
      PageSidebar: './src/components/docs/PageSidebar.astro',
      ThemeSelect: './src/components/docs/ThemeSelect.astro',
      PageFrame: './src/components/docs/PageFrame.astro',
      Sidebar: './src/components/docs/Sidebar.astro',
      Pagination: './src/components/docs/Pagination.astro',
      MobileMenuToggle: './src/components/docs/MobileMenuToggle.astro',
    },
    sidebar: [
      {
        label: 'Guide',
        items: [
          { label: 'Overview', slug: 'guide' },
          { label: 'Install', slug: 'install' },
          { label: 'Getting started', slug: 'getting-started' },
          { label: 'Architecture & philosophy', slug: 'architecture' },
        ],
      },
      {
        label: 'Commands & Modes',
        items: [
          { label: 'Overview', slug: 'commands' },
          { label: 'Google Drive', slug: 'commands/drive' },
          { label: 'Google Docs', slug: 'commands/docs' },
          { label: 'Google Sheets', slug: 'commands/sheets' },
          { label: 'Dynamic Discovery & recipes', slug: 'commands/discovery' },
          { label: 'MCP server & Agent Skill', slug: 'commands/mcp' },
        ],
      },
    ],
    plugins: [starlightLlmsTxt({
      details: 'drivectl provides ergonomic native subcommands for Google Drive, Docs (Markdown AST conversion), and Sheets (A1 notation), a generic Discovery API caller (drivectl call), and a stdio/HTTP MCP server.',
    })],
  })],

  markdown: { processor: unified({ remarkPlugins: [remarkRepositoryLinks] }) },

  redirects: {
    '/docs': '/drivectl/guide',
    '/docs/getting-started': '/drivectl/getting-started',
    '/docs/architecture': '/drivectl/architecture',
    '/docs/commands': '/drivectl/commands',
    ...Object.fromEntries(['drive', 'docs', 'sheets', 'discovery', 'mcp'].map(cmd => [`/docs/commands/${cmd}`, `/drivectl/commands/${cmd}`])),
  },

  fonts: [
    {
      provider: fontProviders.local(),
      name: 'Google Sans Flex',
      cssVariable: '--font-display',
      fallbacks: ['system-ui', 'sans-serif'],
      options: {
        variants: [
          {
            weight: '100 1000',
            style: 'normal',
            display: 'swap',
            src: ['./src/assets/fonts/google-sans-flex.woff2'],
          },
        ],
      },
    },
    {
      provider: fontProviders.local(),
      name: 'Google Sans Code',
      cssVariable: '--font-mono',
      fallbacks: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      options: {
        variants: [
          {
            weight: '300 800',
            style: 'normal',
            display: 'swap',
            src: ['./src/assets/fonts/google-sans-code.woff2'],
          },
        ],
      },
    },
  ],

  vite: {
    cacheDir: new URL('./.astro/vite', import.meta.url).pathname,
    build: {
      cssTarget: 'esnext',
    },
  },
});
