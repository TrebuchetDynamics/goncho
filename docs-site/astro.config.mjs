import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://trebuchetdynamics.github.io',
  base: '/goncho',
  integrations: [
    starlight({
      title: 'Goncho',
      description: 'Trust-preserving context for Go agents.',
      sidebar: [
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
            { label: 'Local-First Memory', slug: 'concepts/local-first-memory' },
            { label: 'Evidence, Claims, and Beliefs', slug: 'concepts/evidence-claims-beliefs' },
            { label: 'Session Lifecycle', slug: 'concepts/session-lifecycle' },
            { label: 'Orientation Packs', slug: 'concepts/orientation-packs' },
            { label: 'Negative Memory', slug: 'concepts/negative-memory' },
            { label: 'Design Boundaries', slug: 'concepts/design-boundaries' },
            { label: 'Glossary', slug: 'concepts/glossary' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Core API', slug: 'reference/core-api' },
            { label: 'Local Markdown Memory', slug: 'reference/local-markdown-memory' },
            { label: 'Memory Tools', slug: 'reference/memory-tools' },
            { label: 'Retrieval Benchmarks', slug: 'reference/retrieval-benchmarks' },
            { label: 'Honcho Compatibility', slug: 'reference/honcho-compatibility' },
          ],
        },
        {
          label: 'Operators',
          items: [{ label: 'Operator Runbook', slug: 'operators/runbook' }],
        },
        {
          label: 'Integrations',
          items: [{ label: 'Gormes Agent', slug: 'integrations/gormes-agent' }],
        },
        {
          label: 'Roadmap',
          items: [
            { label: 'Architecture Direction', slug: 'roadmap/architecture-direction' },
            { label: 'Benchmark Roadmap', slug: 'roadmap/benchmark-roadmap' },
          ],
        },
      ],
    }),
  ],
});
