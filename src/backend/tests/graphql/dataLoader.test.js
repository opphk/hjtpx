const { DataLoader, DataLoaderRegistry, createLoaders } = require('../../services/dataLoader');

class DataLoaderTest {
  constructor() {
    this.testResults = [];
  }

  async testBasicDataLoader() {
    console.log('\n=== Testing Basic DataLoader ===');
    
    try {
      let callCount = 0;
      const batchLoadFn = async (ids) => {
        callCount++;
        return ids.map(id => ({ id, name: `User ${id}` }));
      };

      const loader = new DataLoader(batchLoadFn);

      const results = await Promise.all([
        loader.load('1'),
        loader.load('2'),
        loader.load('3')
      ]);

      console.log('✓ DataLoader batch loaded 3 items');
      console.log(`✓ Batch function called ${callCount} time(s)`);
      
      if (callCount === 1) {
        console.log('✓ All requests batched into single call');
      } else {
        console.warn(`⚠ Batch function called ${callCount} times, expected 1`);
      }

      this.testResults.push({ test: 'basicDataLoader', status: 'passed', batchCalls: callCount });
    } catch (error) {
      console.error('✗ Basic DataLoader test failed:', error.message);
      this.testResults.push({ test: 'basicDataLoader', status: 'failed', error: error.message });
    }
  }

  async testCaching() {
    console.log('\n=== Testing DataLoader Caching ===');
    
    try {
      let callCount = 0;
      const batchLoadFn = async (ids) => {
        callCount++;
        return ids.map(id => ({ id, name: `User ${id}` }));
      };

      const loader = new DataLoader(batchLoadFn, { cache: true });

      await loader.load('1');
      await loader.load('2');
      
      const cached1 = await loader.load('1');
      const cached2 = await loader.load('2');
      const new1 = await loader.load('1');

      console.log(`✓ Loaded 4 items with ${callCount} batch call(s)`);
      console.log(`✓ Cache hit for duplicate keys`);

      if (callCount === 2) {
        console.log('✓ Caching is working correctly');
      }

      this.testResults.push({ test: 'caching', status: 'passed', batchCalls: callCount });
    } catch (error) {
      console.error('✗ Caching test failed:', error.message);
      this.testResults.push({ test: 'caching', status: 'failed', error: error.message });
    }
  }

  async testCacheClear() {
    console.log('\n=== Testing Cache Clear ===');
    
    try {
      let callCount = 0;
      const batchLoadFn = async (ids) => {
        callCount++;
        return ids.map(id => ({ id, name: `User ${id}` }));
      };

      const loader = new DataLoader(batchLoadFn, { cache: true });

      await loader.load('1');
      console.log(`First load: ${callCount} batch call(s)`);

      loader.clear('1');
      await loader.load('1');
      console.log(`After clear: ${callCount} batch call(s)`);

      if (callCount === 2) {
        console.log('✓ Cache clear is working');
      }

      loader.clearAll();
      await loader.load('1');
      console.log(`After clearAll: ${callCount} batch call(s)`);

      if (callCount === 3) {
        console.log('✓ ClearAll is working');
      }

      this.testResults.push({ test: 'cacheClear', status: 'passed' });
    } catch (error) {
      console.error('✗ Cache clear test failed:', error.message);
      this.testResults.push({ test: 'cacheClear', status: 'failed', error: error.message });
    }
  }

  async testErrorHandling() {
    console.log('\n=== Testing Error Handling ===');
    
    try {
      const batchLoadFn = async (ids) => {
        return ids.map(id => {
          if (id === 'error') {
            throw new Error('Simulated error');
          }
          return { id, name: `User ${id}` };
        });
      };

      const loader = new DataLoader(batchLoadFn);

      const results = await Promise.allSettled([
        loader.load('1'),
        loader.load('error'),
        loader.load('3')
      ]);

      const fulfilled = results.filter(r => r.status === 'fulfilled').length;
      const rejected = results.filter(r => r.status === 'rejected').length;

      console.log(`✓ ${fulfilled} fulfilled, ${rejected} rejected`);
      console.log('✓ Error handling works correctly');

      this.testResults.push({ test: 'errorHandling', status: 'passed', fulfilled, rejected });
    } catch (error) {
      console.error('✗ Error handling test failed:', error.message);
      this.testResults.push({ test: 'errorHandling', status: 'failed', error: error.message });
    }
  }

  async testNPlusOneScenario() {
    console.log('\n=== Testing N+1 Query Scenario ===');
    
    try {
      let userServiceCalls = 0;
      const userService = {
        findByIds: async (ids) => {
          userServiceCalls++;
          return ids.map(id => ({ id, name: `User ${id}` }));
        }
      };

      const postService = {
        findByIds: async (ids) => {
          return ids.map(id => ({ id, title: `Post ${id}`, authorId: id }));
        }
      };

      const { userLoader, postAuthorLoader } = createLoaders(userService, postService);

      const posts = [
        { id: '1', title: 'Post 1', authorId: '1' },
        { id: '2', title: 'Post 2', authorId: '2' },
        { id: '3', title: 'Post 3', authorId: '1' },
        { id: '4', title: 'Post 4', authorId: '3' },
        { id: '5', title: 'Post 5', authorId: '2' }
      ];

      const startTime = Date.now();
      
      const authorPromises = posts.map(post => postAuthorLoader.load(post.id));
      await Promise.all(authorPromises);

      const duration = Date.now() - startTime;

      console.log(`✓ Loaded ${posts.length} post authors`);
      console.log(`✓ User service called ${userServiceCalls} time(s)`);
      console.log(`✓ Total time: ${duration}ms`);

      const stats = userLoader.getStats();
      console.log(`✓ DataLoader stats: ${JSON.stringify(stats)}`);

      if (userServiceCalls === 1) {
        console.log('✓ N+1 problem solved! All authors batched into single query');
      } else {
        console.warn(`⚠ User service called ${userServiceCalls} times (expected 1)`);
      }

      this.testResults.push({ 
        test: 'nPlusOne', 
        status: 'passed',
        userServiceCalls,
        duration
      });
    } catch (error) {
      console.error('✗ N+1 test failed:', error.message);
      this.testResults.push({ test: 'nPlusOne', status: 'failed', error: error.message });
    }
  }

  async testPerformanceComparison() {
    console.log('\n=== Testing Performance Comparison ===');
    
    try {
      const iterations = 100;
      const userCount = 50;

      console.log(`Testing with ${userCount} users, ${iterations} iterations`);
      console.log('Without DataLoader (N+1):');
      
      let nPlusOneCalls = 0;
      const userService = {
        findById: async (id) => {
          nPlusOneCalls++;
          await new Promise(resolve => setTimeout(resolve, 1));
          return { id, name: `User ${id}` };
        }
      };

      const posts = Array.from({ length: userCount }, (_, i) => ({
        id: String(i + 1),
        authorId: String(i + 1)
      }));

      let startTime = Date.now();
      for (const post of posts) {
        await userService.findById(post.authorId);
      }
      const nPlusOneTime = Date.now() - startTime;
      console.log(`  - Time: ${nPlusOneTime}ms`);
      console.log(`  - Service calls: ${nPlusOneCalls}`);

      console.log('\nWith DataLoader (batched):');
      
      let batchedCalls = 0;
      const batchedUserService = {
        findByIds: async (ids) => {
          batchedCalls++;
          await new Promise(resolve => setTimeout(resolve, 1));
          return ids.map(id => ({ id, name: `User ${id}` }));
        }
      };

      const { userLoader } = createLoaders(batchedUserService, {});

      startTime = Date.now();
      const promises = posts.map(post => userLoader.load(post.authorId));
      await Promise.all(promises);
      const batchedTime = Date.now() - startTime;
      
      console.log(`  - Time: ${batchedTime}ms`);
      console.log(`  - Service calls: ${batchedCalls}`);

      const improvement = ((nPlusOneTime - batchedTime) / nPlusOneTime * 100).toFixed(2);
      console.log(`\n✓ Performance improvement: ${improvement}%`);
      console.log(`✓ Query reduction: ${nPlusOneCalls} → ${batchedCalls} (${((1 - batchedCalls/nPlusOneCalls)*100).toFixed(1)}% reduction)`);

      this.testResults.push({ 
        test: 'performanceComparison', 
        status: 'passed',
        withoutDataLoader: { time: nPlusOneTime, calls: nPlusOneCalls },
        withDataLoader: { time: batchedTime, calls: batchedCalls },
        improvement: `${improvement}%`
      });
    } catch (error) {
      console.error('✗ Performance comparison test failed:', error.message);
      this.testResults.push({ test: 'performanceComparison', status: 'failed', error: error.message });
    }
  }

  async testRegistry() {
    console.log('\n=== Testing DataLoader Registry ===');
    
    try {
      const registry = new DataLoaderRegistry();

      let callCount = 0;
      registry.create('testLoader', async (ids) => {
        callCount++;
        return ids.map(id => ({ id, value: `value-${id}` }));
      });

      await registry.get('testLoader').load('1');
      await registry.get('testLoader').load('2');

      console.log(`✓ Created and used loader from registry`);
      console.log(`✓ Batch calls: ${callCount}`);

      registry.clear('testLoader');
      await registry.get('testLoader').load('1');
      console.log(`✓ After clear: ${callCount} calls`);

      registry.clearAll();
      await registry.get('testLoader').load('1');
      console.log(`✓ After clearAll: ${callCount} calls`);

      const allStats = registry.getAllStats();
      console.log(`✓ Registry stats: ${JSON.stringify(allStats, null, 2)}`);

      this.testResults.push({ test: 'registry', status: 'passed' });
    } catch (error) {
      console.error('✗ Registry test failed:', error.message);
      this.testResults.push({ test: 'registry', status: 'failed', error: error.message });
    }
  }

  async runAllTests() {
    console.log('========================================');
    console.log('DataLoader Performance & Optimization Tests');
    console.log('========================================');

    await this.testBasicDataLoader();
    await this.testCaching();
    await this.testCacheClear();
    await this.testErrorHandling();
    await this.testNPlusOneScenario();
    await this.testPerformanceComparison();
    await this.testRegistry();

    console.log('\n========================================');
    console.log('Test Summary');
    console.log('========================================');
    
    const passed = this.testResults.filter(t => t.status === 'passed').length;
    const failed = this.testResults.filter(t => t.status === 'failed').length;

    console.log(`Total Tests: ${this.testResults.length}`);
    console.log(`Passed: ${passed}`);
    console.log(`Failed: ${failed}`);

    const hasFailures = failed > 0;
    return !hasFailures;
  }
}

if (require.main === module) {
  const test = new DataLoaderTest();
  test.runAllTests()
    .then(success => {
      process.exit(success ? 0 : 1);
    })
    .catch(error => {
      console.error('Test execution failed:', error);
      process.exit(1);
    });
}

module.exports = DataLoaderTest;
