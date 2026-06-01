import type { StarlightUserConfig } from '@astrojs/starlight/types';

export const docsSidebar = [
  {
    label: 'Start',
    items: [
      { label: 'Quick Start', slug: 'start/quick-start' },
      { label: 'Current Capabilities', slug: 'start/current-capabilities' },
    ],
  },
  {
    label: 'Concepts',
    items: [
      { label: 'Trust-Preserving Context', slug: 'concepts/trust-preserving-context' },
      { label: 'Evidence, Claims, and Beliefs', slug: 'concepts/evidence-claims-beliefs' },
      { label: 'Local-First Memory', slug: 'concepts/local-first-memory' },
      { label: 'Design Boundaries', slug: 'concepts/design-boundaries' },
      { label: 'Session Lifecycle', slug: 'concepts/session-lifecycle' },
      { label: 'Orientation Packs', slug: 'concepts/orientation-packs' },
      { label: 'Negative Memory', slug: 'concepts/negative-memory' },
      { label: 'Glossary', slug: 'concepts/glossary' },
    ],
  },
  {
    label: 'Reference',
    items: [
      { label: 'Core API', slug: 'reference/core-api' },
      { label: 'Memory Tools', slug: 'reference/memory-tools' },
      { label: 'Local Markdown Memory', slug: 'reference/local-markdown-memory' },
      { label: 'Retrieval Benchmarks', slug: 'reference/retrieval-benchmarks' },
      { label: 'Honcho Compatibility', slug: 'reference/honcho-compatibility' },
    ],
  },
  {
    label: 'Integrations',
    items: [{ label: 'Gormes Agent Integration', slug: 'integrations/gormes-agent' }],
  },
  {
    label: 'Operators',
    items: [{ label: 'Operator Runbook', slug: 'operators/runbook' }],
  },
  {
    label: 'Roadmap',
    items: [
      { label: 'Architecture Direction', slug: 'roadmap/architecture-direction' },
      { label: 'Benchmark Roadmap', slug: 'roadmap/benchmark-roadmap' },
    ],
  },
] satisfies StarlightUserConfig['sidebar'];
