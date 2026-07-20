import { defineConfig } from 'tsup';

export default defineConfig({
  entry: [
    'src/index.ts',
    'src/client.ts',
    'src/resources/index.ts',
    'src/types/index.ts',
    'src/errors/errors.ts',
  ],
  format: ['cjs', 'esm'],
  dts: true,
  splitting: true, // Enable code splitting for tree-shaking
  sourcemap: true,
  clean: true,
  minify: true,
  treeshake: true,
  target: 'es2022',
  outDir: 'dist',
  shims: true,
});