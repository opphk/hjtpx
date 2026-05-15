import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import React from 'react';
import { usePerformanceMetrics, useNetworkStatus, useLazyLoad } from '../hooks/usePerformance';

describe('Performance Metrics Hooks', () => {
  describe('usePerformanceMetrics', () => {
    it('should initialize with null metrics', () => {
      const { result } = renderHook(() => usePerformanceMetrics());
      
      expect(result.current.metrics).toEqual({
        lcp: null,
        fid: null,
        cls: null,
        fcp: null,
        ttfb: null,
        inp: null
      });
    });

    it('should track performance metrics after mount', async () => {
      const { result } = renderHook(() => usePerformanceMetrics());
      
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 100));
      });
      
      expect(result.current.webVitalsSupported).toBe(true);
      expect(result.current.isLoading).toBe(false);
    });
  });

  describe('useNetworkStatus', () => {
    it('should initialize with network status', () => {
      const { result } = renderHook(() => useNetworkStatus());
      
      expect(typeof result.current.isOnline).toBe('boolean');
      expect(result.current.effectiveType).toBeTruthy();
    });
  });

  describe('useLazyLoad', () => {
    it('should initialize with visibility false', () => {
      const { result } = renderHook(() => useLazyLoad());
      
      expect(result.current.isVisible).toBe(false);
      expect(result.current.isLoaded).toBe(false);
    });
  });
});

describe('Bundle Size Validation', () => {
  const MAX_BUNDLE_SIZE_KB = 500;
  const MAX_INITIAL_BUNDLE_SIZE_KB = 150;

  it('should validate chunk size limits', async () => {
    const fs = await import('fs');
    const path = await import('path');
    
    const frontendRoot = path.join(process.cwd(), 'src', 'frontend');
    const distPath = path.join(frontendRoot, 'dist');
    
    if (!fs.existsSync(distPath)) {
      console.warn('⚠️  dist folder not found. Run build first.');
      return;
    }

    const assetsPath = path.join(distPath, 'assets');
    if (!fs.existsSync(assetsPath)) {
      console.warn('⚠️  assets folder not found.');
      return;
    }

    const jsFiles = fs.readdirSync(assetsPath).filter(f => f.endsWith('.js'));
    let maxSize = 0;
    let oversizedChunks = [];

    jsFiles.forEach(file => {
      const filePath = path.join(assetsPath, file);
      const stats = fs.statSync(filePath);
      const sizeKB = stats.size / 1024;
      
      if (sizeKB > maxSize) {
        maxSize = sizeKB;
      }
      
      if (sizeKB > MAX_BUNDLE_SIZE_KB) {
        oversizedChunks.push({ file, sizeKB });
      }
    });

    console.log(`📦 Max chunk size: ${maxSize.toFixed(2)} KB`);
    
    if (oversizedChunks.length > 0) {
      console.warn('⚠️  Oversized chunks:');
      oversizedChunks.forEach(chunk => {
        console.warn(`  - ${chunk.file}: ${chunk.sizeKB.toFixed(2)} KB`);
      });
    }

    expect(maxSize).toBeLessThan(MAX_BUNDLE_SIZE_KB);
  });

  it('should validate initial bundle size', async () => {
    const fs = await import('fs');
    const path = await import('path');
    
    const frontendRoot = path.join(process.cwd(), 'src', 'frontend');
    const indexPath = path.join(frontendRoot, 'dist', 'index.html');
    
    if (!fs.existsSync(indexPath)) {
      console.warn('⚠️  index.html not found. Run build first.');
      return;
    }

    const content = fs.readFileSync(indexPath, 'utf-8');
    
    const scriptMatches = content.match(/<script[^>]*src="([^"]*)"[^>]*>/g) || [];
    let totalInitialSize = 0;

    for (const match of scriptMatches) {
      const srcMatch = match.match(/src="([^"]*)"/);
      if (srcMatch) {
        const scriptPath = path.join(frontendRoot, 'dist', srcMatch[1]);
        if (fs.existsSync(scriptPath)) {
          const stats = fs.statSync(scriptPath);
          totalInitialSize += stats.size / 1024;
        }
      }
    }

    console.log(`📦 Total initial bundle size: ${totalInitialSize.toFixed(2)} KB`);
    
    expect(totalInitialSize).toBeLessThan(MAX_INITIAL_BUNDLE_SIZE_KB);
  });
});

describe('Code Splitting Validation', () => {
  it('should create separate chunks for routes', async () => {
    const fs = await import('fs');
    const path = await import('path');
    
    const frontendRoot = path.join(process.cwd(), 'src', 'frontend');
    const distPath = path.join(frontendRoot, 'dist');
    
    if (!fs.existsSync(distPath)) {
      console.warn('⚠️  dist folder not found. Run build first.');
      return;
    }

    const assetsPath = path.join(distPath, 'assets');
    if (!fs.existsSync(assetsPath)) {
      return;
    }

    const jsFiles = fs.readdirSync(assetsPath).filter(f => f.endsWith('.js'));
    
    const pageChunks = jsFiles.filter(f => f.startsWith('page-'));
    const vendorChunks = jsFiles.filter(f => f.startsWith('vendor-'));
    
    console.log(`📦 Page chunks: ${pageChunks.length}`);
    console.log(`📦 Vendor chunks: ${vendorChunks.length}`);
    
    expect(pageChunks.length).toBeGreaterThan(0);
    expect(vendorChunks.length).toBeGreaterThan(0);
  });
});

describe('Compression Validation', () => {
  it('should generate gzip compressed files', async () => {
    const fs = await import('fs');
    const path = await import('path');
    
    const frontendRoot = path.join(process.cwd(), 'src', 'frontend');
    const distPath = path.join(frontendRoot, 'dist');
    
    if (!fs.existsSync(distPath)) {
      console.warn('⚠️  dist folder not found. Run build first.');
      return;
    }

    const assetsPath = path.join(distPath, 'assets');
    if (!fs.existsSync(assetsPath)) {
      return;
    }

    const gzFiles = fs.readdirSync(assetsPath).filter(f => f.endsWith('.gz'));
    const brFiles = fs.readdirSync(assetsPath).filter(f => f.endsWith('.br'));
    
    console.log(`📦 Gzip files: ${gzFiles.length}`);
    console.log(`📦 Brotli files: ${brFiles.length}`);
    
    expect(gzFiles.length).toBeGreaterThan(0);
    expect(brFiles.length).toBeGreaterThan(0);
  });

  it('should validate compression ratio', async () => {
    const fs = await import('fs');
    const path = await import('path');
    
    const frontendRoot = path.join(process.cwd(), 'src', 'frontend');
    const distPath = path.join(frontendRoot, 'dist');
    
    if (!fs.existsSync(distPath)) {
      console.warn('⚠️  dist folder not found. Run build first.');
      return;
    }

    const assetsPath = path.join(distPath, 'assets');
    if (!fs.existsSync(assetsPath)) {
      return;
    }

    const jsFiles = fs.readdirSync(assetsPath).filter(f => f.endsWith('.js'));
    const gzFiles = fs.readdirSync(assetsPath).filter(f => f.endsWith('.gz'));
    
    for (const jsFile of jsFiles.slice(0, 3)) {
      const jsPath = path.join(assetsPath, jsFile);
      const gzPath = path.join(assetsPath, `${jsFile}.gz`);
      
      if (fs.existsSync(jsPath) && fs.existsSync(gzPath)) {
        const jsSize = fs.statSync(jsPath).size;
        const gzSize = fs.statSync(gzPath).size;
        const ratio = (1 - gzSize / jsSize) * 100;
        
        console.log(`📦 ${jsFile}: ${ratio.toFixed(1)}% compression`);
        
        expect(ratio).toBeGreaterThan(0);
      }
    }
  });
});
