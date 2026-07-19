import { defineConfig } from 'vitepress'
import fs from 'node:fs'
import path from 'node:path'

const SITE_URL = 'https://docs.spore.host'

// Sidebar is a top-level const so the llms.txt generator (buildEnd, below) can
// walk the exact same structure the site renders — the manifest can't drift from
// the nav.
const sidebar = {
  '/': [
    { text: 'Quick Start', link: '/quickstart' },
    { text: 'How It Works', link: '/how-it-works' },
  ],
  '/guides/': [
    {
      text: 'Getting Started',
      collapsed: false,
      items: [
        { text: 'Installation', link: '/guides/installation' },
        { text: 'AWS Authentication', link: '/guides/aws-auth' },
        { text: 'Your First Instance', link: '/guides/first-instance' },
        { text: 'Python SDK', link: '/guides/python-sdk' },
        { text: 'Go Library', link: '/guides/go-library' },
      ]
    },
    {
      text: 'Compute',
      collapsed: false,
      items: [
        { text: 'Finding the Right Instance', link: '/guides/finding-instances' },
        { text: 'GPU Training Jobs', link: '/guides/gpu-training' },
        { text: 'Jupyter Notebooks', link: '/guides/jupyter' },
        { text: 'Spot Instances', link: '/guides/spot-instances' },
        { text: 'Managing Instances & Data', link: '/guides/managing-instances' },
      ]
    },
    {
      text: 'Automation & Control',
      collapsed: false,
      items: [
        { text: 'Slack Setup', link: '/guides/slack-setup' },
        { text: 'Teams Setup', link: '/guides/teams-setup' },
        { text: 'Discord Setup', link: '/guides/discord-setup' },
        { text: 'AI Assistant (MCP)', link: '/guides/mcp-setup' },
        { text: 'Lifecycle Notifications', link: '/guides/notifications' },
      ]
    },
    {
      text: 'Advanced',
      collapsed: false,
      items: [
        { text: 'Parameter Sweeps', link: '/guides/parameter-sweeps' },
        { text: 'Job Arrays', link: '/guides/job-arrays' },
        { text: 'Batch Queues', link: '/guides/batch-queue' },
        { text: 'MPI Clusters', link: '/guides/mpi' },
        { text: 'Pipelines', link: '/guides/pipelines' },
        { text: 'Plugins', link: '/guides/plugins' },
        { text: 'Workflow Engines', link: '/guides/workflow-engines' },
        { text: 'Nextflow (nf-spawn)', link: '/guides/nextflow' },
      ]
    },
    {
      text: 'Self-Hosting',
      collapsed: true,
      items: [
        { text: 'Overview', link: '/guides/self-hosting' },
        { text: 'Self-Hosting spore-bot', link: '/spore-bot-self-hosting' },
      ]
    },
  ],
  '/tools/': [
    {
      text: 'Tools',
      items: [
        { text: 'Overview', link: '/tools/' },
        { text: 'Truffle', link: '/tools/truffle' },
        { text: 'Spawn', link: '/tools/spawn' },
        { text: 'Spored', link: '/tools/spored' },
        { text: 'Lagotto', link: '/tools/lagotto' },
        { text: 'Spore-bot', link: '/tools/spore-bot' },
        { text: 'MCP Server', link: '/tools/mcp-server' },
      ]
    },
    {
      text: 'Command Reference',
      items: [
        { text: 'truffle', link: '/tools/reference/truffle' },
        { text: 'spawn', link: '/tools/reference/spawn' },
        { text: 'lagotto', link: '/tools/reference/lagotto' },
      ]
    },
  ],
  '/reference/': [
    {
      text: 'Reference',
      items: [
        { text: 'Configuration', link: '/reference/configuration' },
        { text: 'EC2 Tags', link: '/reference/ec2-tags' },
        { text: 'IAM Permissions', link: '/reference/iam-permissions' },
        { text: 'Lifecycle Events', link: '/reference/lifecycle-events' },
        { text: 'Environment Variables', link: '/reference/environment-variables' },
        { text: 'FAQ', link: '/reference/faq' },
        { text: 'Cheat Sheet', link: '/reference/cheatsheet' },
      ]
    },
  ],
}

// Build an llms.txt manifest (https://llmstxt.org) from the sidebar so AI
// assistants get a curated, always-current map of the docs. Written to the build
// output dir at build time; regenerated on every build, so it can't go stale.
function writeLlmsTxt(outDir: string) {
  const lines: string[] = []
  lines.push('# spore.host documentation')
  lines.push('')
  lines.push('> Ephemeral compute for researchers and data scientists: find the right EC2 instance (truffle), launch and manage it (spawn), and watch for capacity (lagotto). Instances self-terminate via TTL and idle detection.')
  lines.push('')
  const sections: Array<[string, any[]]> = [
    ['Start here', sidebar['/']],
    ['Guides', sidebar['/guides/']],
    ['Tools & command reference', sidebar['/tools/']],
    ['Reference', sidebar['/reference/']],
  ]
  const link = (item: any) =>
    item.link && item.link.startsWith('/')
      ? `- [${item.text}](${SITE_URL}${item.link})`
      : null
  for (const [heading, groups] of sections) {
    lines.push(`## ${heading}`)
    lines.push('')
    for (const entry of groups) {
      if (entry.items) {
        for (const item of entry.items) {
          const l = link(item)
          if (l) lines.push(l)
        }
      } else {
        const l = link(entry)
        if (l) lines.push(l)
      }
    }
    lines.push('')
  }
  fs.writeFileSync(path.join(outDir, 'llms.txt'), lines.join('\n'))
}

export default defineConfig({
  title: 'spore.host',
  description: 'Ephemeral compute for researchers and data scientists.',
  lang: 'en-US',

  // Emit sitemap.xml for search + AI indexers (VitePress only generates it when
  // a hostname is set). docs.spore.host is the CloudFront-fronted docs domain.
  sitemap: {
    hostname: 'https://docs.spore.host',
  },

  srcExclude: [
    'research/**',
    'DNSSEC_CONFIGURATION.md',
    'gen/**',
  ],

  head: [
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    ['link', { href: 'https://fonts.googleapis.com/css2?family=Atkinson+Hyperlegible:ital,wght@0,400;0,700;1,400;1,700&family=Atkinson+Hyperlegible+Mono:ital,wght@0,400;0,700;1,400;1,700&display=swap', rel: 'stylesheet' }],
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    // Default OpenGraph/Twitter card metadata (per-page title/description still
    // override via frontmatter). Helps link unfurls and AI/social indexers.
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'spore.host docs' }],
    ['meta', { property: 'og:title', content: 'spore.host documentation' }],
    ['meta', { property: 'og:description', content: 'Ephemeral compute for researchers and data scientists.' }],
    ['meta', { property: 'og:url', content: 'https://docs.spore.host/' }],
    ['meta', { name: 'twitter:card', content: 'summary' }],
  ],

  themeConfig: {
    siteTitle: 'spore.host',
    logo: null,

    nav: [
      { text: 'Quick Start', link: '/quickstart' },
      { text: 'Guides', link: '/guides/' },
      { text: 'Tools', link: '/tools/' },
      { text: 'Reference', link: '/reference/' },
      { text: 'spore.host', link: 'https://spore.host', target: '_blank' },
    ],

    sidebar,

    socialLinks: [
      { icon: 'github', link: 'https://github.com/spore-host/spore-host' },
    ],

    editLink: {
      pattern: 'https://github.com/spore-host/spore-host/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },

    footer: {
      message: 'Released under the <a href="https://github.com/spore-host/spore-host/blob/main/LICENSE">Apache 2.0 License</a>.',
      copyright: '© 2026 Scott Friedman',
    },

    search: {
      provider: 'local',
    },
  },

  // Generate llms.txt (AI-assistant manifest) into the build output on every
  // build, derived from the sidebar above so it can't drift from the nav.
  buildEnd(siteConfig) {
    writeLlmsTxt(siteConfig.outDir)
  },
})
