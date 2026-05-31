import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { docsSidebar } from './src/navigation/sidebar';

export default defineConfig({
  site: 'https://trebuchetdynamics.github.io',
  base: '/goncho',
  integrations: [
    starlight({
      title: 'Goncho',
      description: 'Trust-preserving context for Go agents.',
      sidebar: docsSidebar,
    }),
  ],
});
