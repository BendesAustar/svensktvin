// vitest.config.ts
import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
  plugins: [sveltekit()],
  test: {
    environment: 'node',
    include: ['src/tests/**/*.test.ts']
  },
  resolve: {
    alias: {
      $lib: '/src/lib'
    }
  }
});
