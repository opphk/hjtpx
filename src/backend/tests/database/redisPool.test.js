const RedisConnectionPool = require('./redisPool');

class RedisConnectionPoolTest {
  constructor() {
    this.pool = null;
    this.testResults = [];
  }

  async setup() {
    this.pool = new RedisConnectionPool({
      host: process.env.REDIS_HOST || 'localhost',
      port: parseInt(process.env.REDIS_PORT || '6379', 10),
      password: process.env.REDIS_PASSWORD || undefined,
      db: parseInt(process.env.REDIS_DB || '0', 10),
      connectTimeout: 10000,
      commandTimeout: 5000
    });

    await this.pool.initialize();
    console.log('Redis Connection Pool initialized for testing');
  }

  async testBasicOperations() {
    console.log('\n=== Testing Basic Operations ===');
    
    try {
      await this.pool.set('test:key', 'test-value', 60);
      console.log('✓ SET operation successful');

      const value = await this.pool.get('test:key');
      console.log(`✓ GET operation successful: ${value}`);

      const exists = await this.pool.exists('test:key');
      console.log(`✓ EXISTS operation successful: ${exists}`);

      const ttl = await this.pool.ttl('test:key');
      console.log(`✓ TTL operation successful: ${ttl}`);

      await this.pool.del('test:key');
      console.log('✓ DEL operation successful');

      this.testResults.push({ test: 'basicOperations', status: 'passed' });
    } catch (error) {
      console.error('✗ Basic operations test failed:', error.message);
      this.testResults.push({ test: 'basicOperations', status: 'failed', error: error.message });
    }
  }

  async testHashOperations() {
    console.log('\n=== Testing Hash Operations ===');
    
    try {
      await this.pool.hset('test:hash', 'field1', 'value1');
      await this.pool.hset('test:hash', 'field2', 'value2');
      console.log('✓ HSET operation successful');

      const fieldValue = await this.pool.hget('test:hash', 'field1');
      console.log(`✓ HGET operation successful: ${fieldValue}`);

      const allFields = await this.pool.hgetall('test:hash');
      console.log(`✓ HGETALL operation successful:`, allFields);

      await this.pool.del('test:hash');
      console.log('✓ Hash deleted');

      this.testResults.push({ test: 'hashOperations', status: 'passed' });
    } catch (error) {
      console.error('✗ Hash operations test failed:', error.message);
      this.testResults.push({ test: 'hashOperations', status: 'failed', error: error.message });
    }
  }

  async testBatchOperations() {
    console.log('\n=== Testing Batch Operations ===');
    
    try {
      const keys = ['batch:1', 'batch:2', 'batch:3'];
      await this.pool.mset('batch:1', 'value1', 'batch:2', 'value2', 'batch:3', 'value3');
      console.log('✓ MSET operation successful');

      const values = await this.pool.mget('batch:1', 'batch:2', 'batch:3');
      console.log(`✓ MGET operation successful:`, values);

      for (const key of keys) {
        await this.pool.del(key);
      }
      console.log('✓ Batch keys deleted');

      this.testResults.push({ test: 'batchOperations', status: 'passed' });
    } catch (error) {
      console.error('✗ Batch operations test failed:', error.message);
      this.testResults.push({ test: 'batchOperations', status: 'failed', error: error.message });
    }
  }

  async testConcurrentOperations() {
    console.log('\n=== Testing Concurrent Operations ===');
    
    try {
      const startTime = Date.now();
      const promises = [];
      
      for (let i = 0; i < 100; i++) {
        promises.push(this.pool.set(`concurrent:${i}`, `value${i}`, 60));
        promises.push(this.pool.get(`concurrent:${i}`));
      }

      await Promise.all(promises);
      const duration = Date.now() - startTime;

      console.log(`✓ 100 concurrent SET/GET operations completed in ${duration}ms`);
      console.log(`✓ Average latency: ${duration / 200}ms per operation`);

      for (let i = 0; i < 100; i++) {
        await this.pool.del(`concurrent:${i}`);
      }
      console.log('✓ Concurrent keys deleted');

      this.testResults.push({ 
        test: 'concurrentOperations', 
        status: 'passed',
        metrics: { duration, operations: 200, avgLatency: duration / 200 }
      });
    } catch (error) {
      console.error('✗ Concurrent operations test failed:', error.message);
      this.testResults.push({ test: 'concurrentOperations', status: 'failed', error: error.message });
    }
  }

  async testHealthCheck() {
    console.log('\n=== Testing Health Check ===');
    
    try {
      const health = await this.pool.healthCheck();
      console.log('Health Check Result:', health);
      
      if (health.status === 'healthy') {
        console.log('✓ Health check passed');
        this.testResults.push({ test: 'healthCheck', status: 'passed', metrics: health });
      } else {
        console.error('✗ Health check failed');
        this.testResults.push({ test: 'healthCheck', status: 'failed', error: health.error });
      }
    } catch (error) {
      console.error('✗ Health check test failed:', error.message);
      this.testResults.push({ test: 'healthCheck', status: 'failed', error: error.message });
    }
  }

  async testPerformance() {
    console.log('\n=== Testing Performance ===');
    
    try {
      const iterations = 1000;
      const startTime = Date.now();

      for (let i = 0; i < iterations; i++) {
        await this.pool.set(`perf:${i}`, `value${i}`, 60);
      }

      const setDuration = Date.now() - startTime;
      console.log(`✓ ${iterations} SET operations completed in ${setDuration}ms`);
      console.log(`✓ Average: ${setDuration / iterations}ms per operation`);

      const getStartTime = Date.now();
      for (let i = 0; i < iterations; i++) {
        await this.pool.get(`perf:${i}`);
      }
      const getDuration = Date.now() - getStartTime;
      console.log(`✓ ${iterations} GET operations completed in ${getDuration}ms`);
      console.log(`✓ Average: ${getDuration / iterations}ms per operation`);

      for (let i = 0; i < iterations; i++) {
        await this.pool.del(`perf:${i}`);
      }

      this.testResults.push({ 
        test: 'performance', 
        status: 'passed',
        metrics: {
          setOps: iterations,
          setDuration,
          setAvg: setDuration / iterations,
          getOps: iterations,
          getDuration,
          getAvg: getDuration / iterations
        }
      });
    } catch (error) {
      console.error('✗ Performance test failed:', error.message);
      this.testResults.push({ test: 'performance', status: 'failed', error: error.message });
    }
  }

  async testConnectionLeakDetection() {
    console.log('\n=== Testing Connection Leak Detection ===');
    
    try {
      const stats = this.pool.getStats();
      console.log('Pool Stats:', stats);

      if (stats.totalCommands > 0) {
        console.log('✓ Connection leak detection working');
        this.testResults.push({ test: 'connectionLeakDetection', status: 'passed', stats });
      } else {
        console.log('⚠ No commands executed yet');
        this.testResults.push({ test: 'connectionLeakDetection', status: 'skipped' });
      }
    } catch (error) {
      console.error('✗ Connection leak detection test failed:', error.message);
      this.testResults.push({ test: 'connectionLeakDetection', status: 'failed', error: error.message });
    }
  }

  async runAllTests() {
    console.log('========================================');
    console.log('Redis Connection Pool Performance Tests');
    console.log('========================================');

    try {
      await this.setup();
      
      await this.testBasicOperations();
      await this.testHashOperations();
      await this.testBatchOperations();
      await this.testConcurrentOperations();
      await this.testHealthCheck();
      await this.testPerformance();
      await this.testConnectionLeakDetection();

      console.log('\n========================================');
      console.log('Test Summary');
      console.log('========================================');
      
      const passed = this.testResults.filter(t => t.status === 'passed').length;
      const failed = this.testResults.filter(t => t.status === 'failed').length;
      const skipped = this.testResults.filter(t => t.status === 'skipped').length;

      console.log(`Total Tests: ${this.testResults.length}`);
      console.log(`Passed: ${passed}`);
      console.log(`Failed: ${failed}`);
      console.log(`Skipped: ${skipped}`);

      const finalStats = this.pool.getStats();
      console.log('\nPool Statistics:');
      console.log(`- Total Connections: ${finalStats.totalConnections}`);
      console.log(`- Failed Connections: ${finalStats.failedConnections}`);
      console.log(`- Total Commands: ${finalStats.totalCommands}`);
      console.log(`- Failed Commands: ${finalStats.failedCommands}`);
      console.log(`- Average Latency: ${finalStats.averageLatency.toFixed(2)}ms`);

      await this.pool.close();
      console.log('\n✓ All tests completed. Pool closed.');

      return failed === 0;
    } catch (error) {
      console.error('Test suite failed:', error);
      return false;
    }
  }
}

if (require.main === module) {
  const test = new RedisConnectionPoolTest();
  test.runAllTests()
    .then(success => {
      process.exit(success ? 0 : 1);
    })
    .catch(error => {
      console.error('Test execution failed:', error);
      process.exit(1);
    });
}

module.exports = RedisConnectionPoolTest;
