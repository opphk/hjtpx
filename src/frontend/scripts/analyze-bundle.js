import { visualizer } from 'rollup-plugin-visualizer';
import { writeFileSync, readFileSync, existsSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

const BUNDLE_LIMITS = {
  initial: {
    warn: 150 * 1024,
    error: 250 * 1024
  },
  vendor: {
    warn: 200 * 1024,
    error: 300 * 1024
  },
  page: {
    warn: 100 * 1024,
    error: 150 * 1024
  },
  component: {
    warn: 50 * 1024,
    error: 100 * 1024
  }
};

export function analyzeBundle() {
  const statsPath = join(process.cwd(), 'dist', 'stats.html');
  
  if (!existsSync(statsPath)) {
    console.warn('⚠️  Bundle analysis stats not found.');
    console.info('💡 Run "npm run build" first to generate bundle analysis.');
    return;
  }

  const stats = JSON.parse(readFileSync(statsPath.replace('.html', '.json'), 'utf-8'));
  
  console.log('\n📊 Bundle Analysis Report\n');
  console.log('═'.repeat(80));
  
  const modules = stats.modules || [];
  const chunks = stats.chunks || [];
  
  const moduleSizes = modules.map(m => ({
    name: m.name || m.id,
    size: m.size,
    percentage: ((m.size / getTotalSize(modules)) * 100).toFixed(2)
  }));
  
  moduleSizes.sort((a, b) => b.size - a.size);
  
  console.log('\n🔍 Top 10 Largest Modules:\n');
  moduleSizes.slice(0, 10).forEach((mod, i) => {
    const sizeKB = (mod.size / 1024).toFixed(2);
    const bar = '█'.repeat(Math.ceil(mod.percentage / 2)).padEnd(50);
    console.log(`${i + 1}. ${mod.name}`);
    console.log(`   Size: ${sizeKB} KB (${mod.percentage}%)`);
    console.log(`   ${bar}\n`);
  });
  
  const chunkSizes = chunks.map(c => ({
    name: c.name || c.fileName,
    size: c.size,
    gzipSize: c.gzipSize || 0,
    entry: c.isEntry,
    initial: c.isInitial
  }));
  
  chunkSizes.sort((a, b) => b.size - a.size);
  
  console.log('\n📦 All Chunks:\n');
  chunkSizes.forEach((chunk, i) => {
    const sizeKB = (chunk.size / 1024).toFixed(2);
    const gzipKB = (chunk.gzipSize / 1024).toFixed(2);
    const tags = [];
    if (chunk.entry) tags.push('entry');
    if (chunk.initial) tags.push('initial');
    
    console.log(`${i + 1}. ${chunk.name}`);
    console.log(`   Size: ${sizeKB} KB | Gzip: ${gzipKB} KB`);
    if (tags.length) console.log(`   Tags: ${tags.join(', ')}`);
    console.log();
  });
  
  console.log('\n🎯 Optimization Recommendations:\n');
  
  const warnings = [];
  
  chunkSizes.forEach(chunk => {
    const limit = chunk.initial ? BUNDLE_LIMITS.initial : 
                  chunk.name.includes('vendor') ? BUNDLE_LIMITS.vendor :
                  chunk.name.includes('page') ? BUNDLE_LIMITS.page :
                  BUNDLE_LIMITS.component;
    
    if (chunk.size > limit.error) {
      warnings.push({
        level: 'error',
        chunk: chunk.name,
        size: (chunk.size / 1024).toFixed(2),
        limit: (limit.error / 1024).toFixed(2)
      });
    } else if (chunk.size > limit.warn) {
      warnings.push({
        level: 'warning',
        chunk: chunk.name,
        size: (chunk.size / 1024).toFixed(2),
        limit: (limit.warn / 1024).toFixed(2)
      });
    }
  });
  
  if (warnings.length === 0) {
    console.log('✅ All chunks are within recommended size limits!\n');
  } else {
    warnings.forEach(w => {
      const icon = w.level === 'error' ? '❌' : '⚠️';
      console.log(`${icon} ${w.level.toUpperCase()}: ${w.chunk}`);
      console.log(`   Current: ${w.size} KB | Limit: ${w.limit} KB\n`);
    });
  }
  
  const totalSize = getTotalSize(chunks);
  const totalGzipSize = chunks.reduce((sum, c) => sum + (c.gzipSize || 0), 0);
  
  console.log('\n📈 Summary:\n');
  console.log(`   Total Bundle Size: ${(totalSize / 1024).toFixed(2)} KB`);
  console.log(`   Total Gzip Size: ${(totalGzipSize / 1024).toFixed(2)} KB`);
  console.log(`   Compression Ratio: ${((1 - totalGzipSize / totalSize) * 100).toFixed(1)}%`);
  console.log(`   Number of Chunks: ${chunks.length}`);
  console.log('\n' + '═'.repeat(80) + '\n');
  
  return {
    totalSize,
    totalGzipSize,
    compressionRatio: (1 - totalGzipSize / totalSize) * 100,
    warnings,
    chunks: chunkSizes
  };
}

function getTotalSize(items) {
  return items.reduce((sum, item) => sum + (item.size || 0), 0);
}

analyzeBundle();
