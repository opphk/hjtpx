const cacheWarmer = require('./src/backend/services/cache_warming');
const cacheMonitor = require('./src/backend/services/cacheMonitor');
const cacheConsistency = require('./src/backend/services/cache_consistency');

async function testComponents() {
  console.log('Testing Cache Components...\n');
  
  console.log('1. Testing Cache Warmer:');
  const warmerStats = cacheWarmer.getStats();
  console.log('   ✓ Warmer initialized');
  console.log('   Startup warmings:', warmerStats.startupWarmings);
  console.log('   Scheduled warmings:', warmerStats.scheduledWarmings);
  console.log('   Hot data warmings:', warmerStats.hotDataWarmings);
  console.log('   Total items warmed:', warmerStats.totalItemsWarmed);
  
  console.log('\n2. Testing Cache Warmer custom warming:');
  const items = [
    { key: 'custom:test1', value: { data: 1 }, ttl: 300 },
    { key: 'custom:test2', value: { data: 2 }, ttl: 300 }
  ];
  const warmed = await cacheWarmer.warmCustomCache(items);
  console.log('   ✓ Custom items warmed:', warmed);
  
  console.log('\n3. Testing Cache Monitor:');
  const metrics = cacheMonitor.collectMetrics();
  console.log('   ✓ Metrics collected');
  console.log('   Hit Rate:', metrics.hitRate?.toFixed(2) + '%' || 'N/A');
  console.log('   Memory Used:', metrics.memory?.used || 0, 'bytes');
  
  console.log('\n4. Testing Cache Monitor report generation:');
  const report = cacheMonitor.generateReport();
  console.log('   ✓ Report generated');
  console.log('   Status:', report.summary.status);
  console.log('   Total alerts:', report.summary.totalAlerts);
  console.log('   Critical alerts:', report.summary.criticalAlerts);
  console.log('   Recommendations:', report.recommendations.length);
  
  console.log('\n5. Testing Cache Monitor health check:');
  const health = cacheMonitor.getHealthStatus();
  console.log('   ✓ Health status retrieved');
  console.log('   Overall healthy:', health.healthy);
  console.log('   L1 enabled:', health.l1.enabled);
  console.log('   L2 connected:', health.l2.connected);
  
  console.log('\n6. Testing Cache Consistency:');
  const consistencyStats = cacheConsistency.getConsistencyStats();
  console.log('   ✓ Consistency module ready');
  console.log('   Connected:', consistencyStats.connected);
  console.log('   Transaction log size:', consistencyStats.transactionLogSize);
  
  console.log('\n7. Testing transaction creation:');
  const tx = await cacheConsistency.startTransaction();
  console.log('   ✓ Transaction created');
  console.log('   Transaction ID:', tx.id);
  
  console.log('\n8. Testing alert system:');
  const alerts = cacheMonitor.getAlerts();
  console.log('   ✓ Alerts retrieved');
  console.log('   Active alerts:', alerts.length);
  
  cacheMonitor.stopMonitoring();
  
  console.log('\n✅ All cache components test completed successfully!');
}

testComponents().catch(console.error);
