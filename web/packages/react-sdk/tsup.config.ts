import { defineConfig } from 'tsup';

export default defineConfig({
  entry: ['src/index.ts'],
  format: ['esm'],
  dts: true,
  splitting: false,
  sourcemap: true,
  clean: true,
  external: [
    'react',
    'react-dom',
    '@tanstack/react-query',
    '@getpaidhq/sdk',
    'zod',
    'react-hook-form',
    '@hookform/resolvers',
  ],
  banner: {
    js: '"use client";',
  },
});