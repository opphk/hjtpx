import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import viteCompression from 'vite-plugin-compression';

export default defineConfig({
  plugins: [
    react(),
    viteCompression({
      algorithm: 'gzip',
      ext: '.gz',
      threshold: 1024,
      deleteOriginFile: false,
      compressionOptions: {
        level: 9
      }
    }),
    viteCompression({
      algorithm: 'brotliCompress',
      ext: '.br',
      threshold: 1024,
      deleteOriginFile: false,
      compressionOptions: {
        params: {
          [2]: 11
        }
      }
    }),
    visualizer({
      filename: 'dist/stats.html',
      open: false,
      gzipSize: true,
      brotliSize: true
    })
  ],
  build: {
    target: 'es2020',
    minify: 'terser',
    sourcemap: false,
    chunkSizeWarningLimit: 500,
    rollupOptions: {
      output: {
        manualChunks: id => {
          if (id.includes('node_modules')) {
            if (id.includes('react')) {
              return 'vendor-react';
            }
            if (id.includes('socket.io')) {
              return 'vendor-socket';
            }
            if (id.includes('recharts') || id.includes('d3-')) {
              return 'vendor-charts';
            }
            if (id.includes('i18next') || id.includes('react-i18next')) {
              return 'vendor-i18n';
            }
            if (id.includes('date-fns')) {
              return 'vendor-date';
            }
            if (id.includes('papaparse')) {
              return 'vendor-csv';
            }
            if (id.includes('prop-types') || id.includes('warning')) {
              return 'vendor-polyfills';
            }
            if (id.includes('@vitejs') || id.includes('vite')) {
              return 'vendor-build';
            }
            return 'vendor-misc';
          }
          if (id.includes('src/pages')) {
            const pageName = id.split('pages/')[1]?.replace('.jsx', '') || 'unknown';
            return `page-${pageName}`;
          }
          if (id.includes('src/components')) {
            const componentName = id.split('components/')[1]?.split('/')[0] || 'common';
            return `component-${componentName}`;
          }
        },
        chunkFileNames: 'assets/js/[name]-[hash].js',
        entryFileNames: 'assets/js/[name]-[hash].js',
        assetFileNames: 'assets/[ext]/[name]-[hash].[ext]'
      }
    },
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true,
        pure_funcs: ['console.log', 'console.info', 'console.debug', 'console.warn', 'console.assert'],
        passes: 3,
        unsafe_arrows: true,
        unsafe_methods: true,
        unsafe_comps: true,
        unsafe_proto: true,
        unsafe_regexp: true,
        ecma: 2020,
        module: true,
        toplevel: true,
        arguments: true,
        dead_code: true,
        drop_labels: true,
        ecma: 2020
      },
      mangle: {
        safari10: true,
        properties: false,
        toplevel: true,
        reserved: ['React', 'ReactDOM']
      },
      format: {
        comments: false,
        ecma: 2020,
        ascii_only: true,
        inline_script: true
      }
    },
    reportCompressedSize: true,
    cssCodeSplit: true,
    assetsInlineLimit: 4096
  },
  optimizeDeps: {
    include: ['react', 'react-dom', 'react-router-dom', 'socket.io-client', 'i18next', 'react-i18next', 'date-fns', 'recharts', 'papaparse'],
    exclude: [],
    esbuildOptions: {
      target: 'es2020',
      treeShaking: true,
      keepNames: true
    },
    buildCompression: 'gzip',
    maxConcurrency: 8
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test-setup.js'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      exclude: [
        'node_modules/',
        'test/',
        'test-results/',
        'dist/',
        '**/*.config.*'
      ],
      thresholds: {
        branches: 70,
        functions: 70,
        lines: 70,
        statements: 70
      }
    }
  }
});
