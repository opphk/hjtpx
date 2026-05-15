import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { visualizer } from 'rollup-plugin-visualizer';
import { gzipSize } from 'gzip-size';
import { readFileSync } from 'fs';
import path from 'path';

const ANALYZE_MODE = process.env.ANALYZE === 'true';
const GENERATE_STATS = process.env.GENERATE_STATS === 'true';

function getGzipSize(filePath) {
  try {
    const code = readFileSync(filePath);
    return gzipSize.sync(code);
  } catch {
    return 0;
  }
}

export default defineConfig({
  plugins: [
    react(),
    ANALYZE_MODE && visualizer({
      filename: 'dist/stats.html',
      open: true,
      gzipSize: true,
      brotliSize: true,
      template: 'treemap',
      projectRoot: path.resolve(__dirname, 'src'),
      output: {
        filename: 'dist/stats.html',
        format: 'html'
      }
    }),
    GENERATE_STATS && visualizer({
      filename: 'dist/stats.json',
      json: true,
      gzipSize: true
    })
  ].filter(Boolean),
  
  build: {
    target: 'esnext',
    sourcemap: process.env.NODE_ENV === 'development',
    
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: {
            test: /[\\/]node_modules[\\/]/,
            name: 'vendor',
            chunks: 'all',
            priority: 10
          },
          react: {
            test: /[\\/]node_modules[\\/](react|react-dom|react-router-dom)[\\/]/,
            name: 'react-vendor',
            chunks: 'all',
            priority: 20
          },
          charts: {
            test: /[\\/]node_modules[\\/](recharts|chart\.js|echarts|d3)[\\/]/,
            name: 'charts-vendor',
            chunks: 'all',
            priority: 15
          },
          ui: {
            test: /[\\/]node_modules[\\/](@mui|antd|element-ui|chakra-ui)[\\/]/,
            name: 'ui-vendor',
            chunks: 'all',
            priority: 15
          }
        },
        chunkFileNames: 'assets/js/[name]-[hash].js',
        entryFileNames: 'assets/js/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          const name = assetInfo.name || '';
          const ext = path.extname(name);
          
          if (/\.(png|jpe?g|gif|svg|webp|ico)$/.test(ext)) {
            return 'assets/images/[name]-[hash][ext]';
          }
          if (/\.(woff2?|eot|ttf|otf)$/.test(ext)) {
            return 'assets/fonts/[name]-[hash][ext]';
          }
          if (/\.css$/.test(ext)) {
            return 'assets/css/[name]-[hash][ext]';
          }
          return 'assets/[name]-[hash][ext]';
        }
      }
    },
    
    chunkSizeWarningLimit: 500,
    
    reportCompressedSize: true,
    
    minify: 'terser',
    
    terserOptions: {
      compress: {
        drop_console: process.env.NODE_ENV === 'production',
        drop_debugger: true,
        pure_funcs: ['console.log', 'console.info', 'console.debug'],
        passes: 2,
        unsafe: {
          prototype: false,
          constructor: false
        }
      },
      mangle: {
        safari10: true
      },
      format: {
        comments: false,
        ecma: 2020
      }
    }
  },
  
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      'react-router-dom',
      'axios',
      'dayjs',
      'lodash'
    ],
    exclude: [
      '@vitejs/plugin-react'
    ]
  },
  
  esbuild: {
    jsxFactory: 'React.createElement',
    jsxFragment: 'React.Fragment',
    target: 'esnext',
    supported: {
      'top-level-await': true,
      'dynamic-import': true
    }
  },
  
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@components': path.resolve(__dirname, 'src/components'),
      '@pages': path.resolve(__dirname, 'src/pages'),
      '@hooks': path.resolve(__dirname, 'src/hooks'),
      '@utils': path.resolve(__dirname, 'src/utils'),
      '@services': path.resolve(__dirname, 'src/services'),
      '@store': path.resolve(__dirname, 'src/store'),
      '@assets': path.resolve(__dirname, 'src/assets'),
      '@styles': path.resolve(__dirname, 'src/styles'),
      '@config': path.resolve(__dirname, 'src/config')
    },
    extensions: ['.mjs', '.js', '.jsx', '.ts', '.tsx', '.json']
  },
  
  server: {
    port: 3000,
    host: true,
    open: false,
    cors: true,
    proxy: {
      '/api': {
        target: process.env.API_URL || 'http://localhost:8080',
        changeOrigin: true,
        secure: false
      },
      '/graphql': {
        target: process.env.API_URL || 'http://localhost:8080',
        changeOrigin: true,
        secure: false
      }
    }
  },
  
  preview: {
    port: 4173,
    host: true
  },
  
  css: {
    modules: {
      localsConvention: 'camelCase',
      generateScopedName: '[name]__[local]___[hash:base64:5]'
    },
    preprocessorOptions: {
      scss: {
        additionalData: `@import "@styles/variables.scss";`
      }
    }
  },
  
  json: {
    stringify: false
  },
  
  define: {
    'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
    'process.env.API_URL': JSON.stringify(process.env.API_URL || 'http://localhost:8080'),
    'process.env.CDN_URL': JSON.stringify(process.env.CDN_URL || ''),
    '__DEV__': process.env.NODE_ENV !== 'production',
    '__VERSION__': JSON.stringify(process.env.npm_package_version || '1.0.0')
  }
});
