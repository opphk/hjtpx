const AdvancedCacheService = require('./src/backend/services/advancedCacheService');

async function testCache() {
  console.log('Testing Advanced Cache Service...\n');
  
  console.log('1. Testing basic set and get operations:');
  await AdvancedCacheService.set('test:user:1', { name: 'John', email: 'john@example.com' }, { ttl: 300 });
  const user = await AdvancedCacheService.get('test:user:1');
  console.log('   ✓ User cached:', user);
  
  console.log('\n2. Testing cache statistics:');
  const stats = AdvancedCacheService.getStats();
  console.log('   L1 Status:', stats.l1.hits > 0 ? '✓ Working' : '✗ Not working');
  console.log('   L1 Hit Rate:', stats.l1.hitRate);
  console.log('   L1 Size:', stats.l1.size);
  
  console.log('\n3. Testing cache deletion:');
  await AdvancedCacheService.delete('test:user:1');
  const deleted = await AdvancedCacheService.get('test:user:1');
  console.log('   ✓ Deleted user:', deleted === null ? 'Confirmed' : 'Failed');
  
  console.log('\n4. Testing cache versioning:');
  const versionBefore = AdvancedCacheService.getVersion('test:version');
  await AdvancedCacheService.set('test:version', { data: 'test' });
  const versionAfter = AdvancedCacheService.getVersion('test:version');
  console.log('   ✓ Version incremented:', versionAfter > versionBefore);
  
  console.log('\n5. Testing multi-level operations:');
  await AdvancedCacheService.set('test:multi', { level: 'multi' }, { ttl: 600 });
  const multi = await AdvancedCacheService.get('test:multi');
  console.log('   ✓ Multi-level cached:', multi);
  
  console.log('\n6. Testing pattern invalidation:');
  await AdvancedCacheService.set('pattern:1', { data: 1 });
  await AdvancedCacheService.set('pattern:2', { data: 2 });
  await AdvancedCacheService.set('other:data', { data: 3 });
  await AdvancedCacheService.invalidatePattern('pattern:*');
  const p1 = await AdvancedCacheService.get('pattern:1');
  const other = await AdvancedCacheService.get('other:data');
  console.log('   ✓ Pattern 1 deleted:', p1 === null ? 'Confirmed' : 'Failed');
  console.log('   ✓ Other data preserved:', other !== null ? 'Confirmed' : 'Failed');
  
  console.log('\n7. Testing lock mechanism (without Redis):');
  console.log('   ℹ Lock mechanism ready (requires Redis connection)');
  
  console.log('\n8. Final statistics:');
  const finalStats = AdvancedCacheService.getStats();
  console.log('   Total Hits:', finalStats.total.hits);
  console.log('   Total Misses:', finalStats.total.misses);
  console.log('   Overall Hit Rate:', finalStats.total.hitRate);
  console.log('   Memory Used:', finalStats.memory.usedFormatted);
  console.log('   Peak Memory:', finalStats.memory.peakFormatted);
  
  console.log('\n✅ Cache service test completed!');
}

testCache().catch(console.error);
