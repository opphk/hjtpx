const { QueryOptimizer, IndexRecommendation } = require('../services/queryOptimizer');

class DatabasePerformanceTest {
  constructor() {
    this.optimizer = null;
    this.testResults = [];
  }

  async setup() {
    this.optimizer = new QueryOptimizer({
      slowQueryThreshold: 100,
      maxSlowQueries: 50
    });
    console.log('Database performance test environment initialized');
  }

  async testBasicQueries() {
    console.log('\n=== Testing Basic Query Performance ===');
    
    try {
      const queries = [
        { name: 'Simple SELECT', query: 'SELECT 1 as result' },
        { name: 'User lookup by email', query: 'SELECT * FROM users WHERE email = $1', params: ['test@example.com'] },
        { name: 'Session check', query: 'SELECT * FROM sessions WHERE user_id = $1 AND is_revoked = false', params: ['123'] }
      ];

      for (const test of queries) {
        const result = await this.optimizer.execute(test.query, test.params);
        const status = result.success ? '✓' : '✗';
        console.log(`${status} ${test.name}: ${result.duration}ms`);
        
        this.testResults.push({
          test: test.name,
          duration: result.duration,
          success: result.success,
          status: result.success ? 'passed' : 'failed'
        });
      }
    } catch (error) {
      console.error('Basic query test failed:', error.message);
      this.testResults.push({
        test: 'basicQueries',
        status: 'failed',
        error: error.message
      });
    }
  }

  async testIndexPerformance() {
    console.log('\n=== Testing Index Performance ===');
    
    try {
      const indexedQuery = await this.optimizer.execute(
        'SELECT * FROM users WHERE email = $1',
        ['test@example.com']
      );
      
      const seqScanQuery = await this.optimizer.execute(
        'SELECT * FROM users WHERE name LIKE $1',
        ['%test%']
      );

      const analyzeResult = await this.optimizer.analyzeQuery(
        'SELECT * FROM users WHERE email = $1'
      );

      console.log('Index Analysis:');
      if (analyzeResult.success) {
        console.log(JSON.stringify(analyzeResult.plan, null, 2));
      }

      console.log('\n✓ Index performance test completed');

      this.testResults.push({
        test: 'indexPerformance',
        status: 'passed',
        indexedDuration: indexedQuery.duration,
        seqScanDuration: seqScanQuery.duration
      });
    } catch (error) {
      console.error('Index performance test failed:', error.message);
      this.testResults.push({
        test: 'indexPerformance',
        status: 'failed',
        error: error.message
      });
    }
  }

  async testSlowQueryDetection() {
    console.log('\n=== Testing Slow Query Detection ===');
    
    try {
      this.optimizer.stats.slowQueryThreshold = 0;

      await this.optimizer.execute('SELECT pg_sleep(0.1)');
      
      const slowQueries = this.optimizer.getSlowQueryLog();
      console.log(`✓ Detected ${slowQueries.length} slow query(ies)`);

      if (slowQueries.length > 0) {
        console.log('Sample slow query:', slowQueries[0]);
      }

      this.testResults.push({
        test: 'slowQueryDetection',
        status: 'passed',
        slowQueriesDetected: slowQueries.length
      });
    } catch (error) {
      console.error('Slow query detection test failed:', error.message);
      this.testResults.push({
        test: 'slowQueryDetection',
        status: 'failed',
        error: error.message
      });
    }
  }

  async testQueryStatistics() {
    console.log('\n=== Testing Query Statistics ===');
    
    try {
      await this.optimizer.execute('SELECT 1');
      await this.optimizer.execute('SELECT 2');
      await this.optimizer.execute('SELECT 3');

      const stats = this.optimizer.getStats();
      console.log('Query Statistics:');
      console.log(`- Total Queries: ${stats.totalQueries}`);
      console.log(`- Slow Queries: ${stats.slowQueries}`);
      console.log(`- Failed Queries: ${stats.failedQueries}`);
      console.log(`- Average Query Time: ${stats.averageQueryTime.toFixed(2)}ms`);

      if (stats.poolStats) {
        console.log('Connection Pool Stats:');
        console.log(`- Total Connections: ${stats.poolStats.total}`);
        console.log(`- Idle Connections: ${stats.poolStats.idle}`);
        console.log(`- Waiting Requests: ${stats.poolStats.waiting}`);
      }

      this.testResults.push({
        test: 'queryStatistics',
        status: 'passed',
        stats
      });
    } catch (error) {
      console.error('Query statistics test failed:', error.message);
      this.testResults.push({
        test: 'queryStatistics',
        status: 'failed',
        error: error.message
      });
    }
  }

  async testIndexRecommendations() {
    console.log('\n=== Testing Index Recommendations ===');
    
    try {
      const recommender = new IndexRecommendation(this.optimizer);
      const recommendations = await recommender.analyze();

      console.log(`✓ Generated ${recommendations.length} index recommendation(s)`);

      for (const rec of recommendations.slice(0, 5)) {
        console.log(`\n[${rec.priority.toUpperCase()}] ${rec.type}`);
        console.log(`Table: ${rec.table}`);
        console.log(`Reason: ${rec.reason}`);
        console.log(`Suggestion: ${rec.suggestion}`);
      }

      this.testResults.push({
        test: 'indexRecommendations',
        status: 'passed',
        recommendationsGenerated: recommendations.length
      });
    } catch (error) {
      console.error('Index recommendations test failed:', error.message);
      this.testResults.push({
        test: 'indexRecommendations',
        status: 'failed',
        error: error.message
      });
    }
  }

  async testQueryBatching() {
    console.log('\n=== Testing Query Batching ===');
    
    try {
      const startTime = Date.now();
      
      const promises = [];
      for (let i = 0; i < 100; i++) {
        promises.push(this.optimizer.execute('SELECT $1 as value', [i]));
      }

      await Promise.all(promises);
      const batchDuration = Date.now() - startTime;

      console.log(`✓ 100 concurrent queries completed in ${batchDuration}ms`);
      console.log(`✓ Average: ${batchDuration / 100}ms per query`);

      const stats = this.optimizer.getStats();
      console.log(`✓ Pool utilization: ${stats.poolStats.total} total connections`);

      this.testResults.push({
        test: 'queryBatching',
        status: 'passed',
        batchDuration,
        averageLatency: batchDuration / 100
      });
    } catch (error) {
      console.error('Query batching test failed:', error.message);
      this.testResults.push({
        test: 'queryBatching',
        status: 'failed',
        error: error.message
      });
    }
  }

  async testConnectionPool() {
    console.log('\n=== Testing Connection Pool ===');
    
    try {
      const initialStats = this.optimizer.getStats();
      console.log('Initial pool state:', initialStats.poolStats);

      const queries = [];
      for (let i = 0; i < 20; i++) {
        queries.push(this.optimizer.execute('SELECT $1', [i]));
      }

      await Promise.all(queries);

      const finalStats = this.optimizer.getStats();
      console.log('Final pool state:', finalStats.poolStats);

      if (finalStats.poolStats.idle > 0) {
        console.log('✓ Connection pool is healthy');
      }

      this.testResults.push({
        test: 'connectionPool',
        status: 'passed',
        poolStats: finalStats.poolStats
      });
    } catch (error) {
      console.error('Connection pool test failed:', error.message);
      this.testResults.push({
        test: 'connectionPool',
        status: 'failed',
        error: error.message
      });
    }
  }

  async runAllTests() {
    console.log('========================================');
    console.log('Database Performance & Query Optimization Tests');
    console.log('========================================');

    try {
      await this.setup();
      await this.testBasicQueries();
      await this.testIndexPerformance();
      await this.testSlowQueryDetection();
      await this.testQueryStatistics();
      await this.testIndexRecommendations();
      await this.testQueryBatching();
      await this.testConnectionPool();

      console.log('\n========================================');
      console.log('Test Summary');
      console.log('========================================');
      
      const passed = this.testResults.filter(t => t.status === 'passed').length;
      const failed = this.testResults.filter(t => t.status === 'failed').length;

      console.log(`Total Tests: ${this.testResults.length}`);
      console.log(`Passed: ${passed}`);
      console.log(`Failed: ${failed}`);

      const finalStats = this.optimizer.getStats();
      console.log('\nFinal Query Statistics:');
      console.log(`- Total Queries: ${finalStats.totalQueries}`);
      console.log(`- Slow Queries: ${finalStats.slowQueries}`);
      console.log(`- Failed Queries: ${finalStats.failedQueries}`);
      console.log(`- Average Query Time: ${finalStats.averageQueryTime.toFixed(2)}ms`);

      await this.optimizer.close();

      const hasFailures = failed > 0;
      return !hasFailures;
    } catch (error) {
      console.error('Test suite failed:', error);
      return false;
    }
  }
}

if (require.main === module) {
  const test = new DatabasePerformanceTest();
  test.runAllTests()
    .then(success => {
      process.exit(success ? 0 : 1);
    })
    .catch(error => {
      console.error('Test execution failed:', error);
      process.exit(1);
    });
}

module.exports = DatabasePerformanceTest;
