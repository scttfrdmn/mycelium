import { defineConfig } from 'vitepress'
import fs from 'node:fs'
import path from 'node:path'

const SITE_URL = 'https://docs.spore.host'

// Sidebar is a top-level const so the llms.txt generator (buildEnd, below) can
// walk the exact same structure the site renders — the manifest can't drift from
// the nav.
// One global sidebar (keyed on '/') applied to every page, so the whole site
// reads as a single path: Introduction → Start Here → Common Workflows → Tools →
// Automation → Administration → Reference. This is deliberate — the docs used to
// show a different sidebar per top-level section, which fragmented the mental
// model the reviewer flagged. Every link below points to a page that exists.
const sidebar = {
  '/': [
    {
      text: 'Introduction',
      collapsed: false,
      items: [
        { text: 'What is spore.host?', link: '/' },
        { text: 'How the pieces fit together', link: '/how-it-works' },
        { text: 'Security, credentials & data flow', link: '/architecture' },
        { text: 'Costs & safety guarantees', link: '/safety' },
      ]
    },
    {
      text: 'Start Here',
      collapsed: false,
      items: [
        { text: 'Quick Start', link: '/quickstart' },
        { text: 'Install', link: '/guides/installation' },
        { text: 'AWS Authentication', link: '/guides/aws-auth' },
        { text: 'Required permissions', link: '/reference/iam-permissions' },
        { text: 'Your first instance', link: '/guides/first-instance' },
        { text: 'Verify lifecycle protection', link: '/guides/first-instance#verify-lifecycle-protection' },
        { text: 'Clean up everything', link: '/guides/first-instance#clean-up' },
      ]
    },
    {
      text: 'Common Workflows',
      collapsed: false,
      items: [
        { text: 'Overview', link: '/guides/' },
        { text: 'Which execution tool?', link: '/guides/choosing-execution' },
        { text: 'Finding the right instance', link: '/guides/finding-instances' },
        { text: 'Interactive workstation', link: '/guides/jupyter' },
        { text: 'GPU training jobs', link: '/guides/gpu-training' },
        { text: 'Spot instances', link: '/guides/spot-instances' },
        { text: 'Managing instances & data', link: '/guides/managing-instances' },
        { text: 'Waiting for scarce capacity', link: '/guides/waiting-for-capacity' },
      ]
    },
    // The extension/execution-fabric layers, in the order a user climbs them:
    // customize one instance → run many → coordinate steps → hand off to an engine.
    {
      text: 'Extend an instance',
      collapsed: false,
      items: [
        { text: 'Instance plugins', link: '/guides/plugins' },
      ]
    },
    {
      text: 'Run many jobs',
      collapsed: false,
      items: [
        { text: 'Parameter sweeps', link: '/guides/parameter-sweeps' },
        { text: 'Job arrays', link: '/guides/job-arrays' },
      ]
    },
    {
      text: 'Coordinate multiple steps',
      collapsed: false,
      items: [
        { text: 'Instance queues', link: '/guides/batch-queue' },
        { text: 'Spawn pipelines', link: '/guides/pipelines' },
        { text: 'MPI clusters', link: '/guides/mpi' },
      ]
    },
    {
      text: 'Workflow adapters',
      collapsed: false,
      items: [
        { text: 'Overview & maturity', link: '/guides/workflow-engines' },
        { text: 'Nextflow (nf-spawn)', link: '/guides/nextflow' },
      ]
    },
    {
      text: 'Tools',
      collapsed: false,
      items: [
        { text: 'Overview', link: '/tools/' },
        { text: 'Truffle', link: '/tools/truffle' },
        { text: 'Spawn', link: '/tools/spawn' },
        { text: 'Spored', link: '/tools/spored' },
        { text: 'Lagotto', link: '/tools/lagotto' },
        { text: 'Spore-bot', link: '/tools/spore-bot' },
        { text: 'MCP Server', link: '/tools/mcp-server' },
        { text: 'Command reference: truffle', link: '/tools/reference/truffle' },
        { text: 'Command reference: spawn', link: '/tools/reference/spawn' },
        { text: 'Command reference: lagotto', link: '/tools/reference/lagotto' },
      ]
    },
    {
      text: 'Automation',
      collapsed: true,
      items: [
        { text: 'Python SDK', link: '/guides/python-sdk' },
        { text: 'Go libraries', link: '/guides/go-library' },
        { text: 'Events & webhooks', link: '/reference/event-schemas' },
      ]
    },
    {
      text: 'Chat & AI Control',
      collapsed: true,
      items: [
        { text: 'Slack Setup', link: '/guides/slack-setup' },
        { text: 'Teams Setup', link: '/guides/teams-setup' },
        { text: 'Discord Setup', link: '/guides/discord-setup' },
        { text: 'AI Assistant (MCP)', link: '/guides/mcp-setup' },
        { text: 'Lifecycle Notifications', link: '/guides/notifications' },
      ]
    },
    {
      text: 'Administration',
      collapsed: true,
      items: [
        { text: 'IAM Permissions', link: '/reference/iam-permissions' },
        { text: 'Self-Hosting', link: '/guides/self-hosting' },
        { text: 'Self-Hosting spore-bot', link: '/spore-bot-self-hosting' },
      ]
    },
    {
      text: 'Reference',
      collapsed: true,
      items: [
        { text: 'Configuration', link: '/reference/configuration' },
        { text: 'EC2 Tags', link: '/reference/ec2-tags' },
        { text: 'Environment Variables', link: '/reference/environment-variables' },
        { text: 'Lifecycle Events', link: '/reference/lifecycle-events' },
        { text: 'Event schemas', link: '/reference/event-schemas' },
        { text: 'Troubleshooting & common mistakes', link: '/reference/troubleshooting' },
        { text: 'Glossary', link: '/reference/glossary' },
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
  lines.push('> Ephemeral compute for researchers and data scientists: find the right EC2 instance (truffle), launch and manage it (spawn), and watch for capacity (lagotto). Instances self-terminate via TTL and idle detection. Runs on your own AWS account.')
  lines.push('')
  // Walk the single global sidebar: each top-level entry is a titled group with
  // an items[] list. One llms.txt section per group keeps the manifest in
  // lockstep with the rendered nav (only anchor-only, non-root links are skipped).
  const link = (item: any) =>
    item.link && item.link.startsWith('/') && !item.link.includes('#')
      ? `- [${item.text}](${SITE_URL}${item.link})`
      : null
  for (const group of sidebar['/']) {
    lines.push(`## ${group.text}`)
    lines.push('')
    for (const item of group.items ?? []) {
      const l = link(item)
      if (l) lines.push(l)
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
      { text: 'Start Here', link: '/quickstart' },
      { text: 'Workflows', link: '/guides/' },
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
