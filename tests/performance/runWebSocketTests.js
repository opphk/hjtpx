const WebSocketLoadTest = require('./websocketLoadTest');

const CONFIG = {
  baseUrl: process.env.WS_TEST_URL || 'http://localhost:3000',
  concurrentConnections: parseInt(process.env.WS_CONCURRENT_CONNECTIONS) || 50,
  testDuration: parseInt(process.env.WS_TEST_DURATION) || 30000,
  messageSize: parseInt(process.env.WS_MESSAGE_SIZE) || 1024,
  useAuth: process.env.WS_USE_AUTH !== 'false'
};

async function runTests() {
  console.log('='.repeat(60));
  console.log('WebSocket Performance Test Suite');
  console.log('='.repeat(60));
  console.log(`Configuration:`, CONFIG);
  console.log('='.repeat(60));

  const loadTest = new WebSocketLoadTest(CONFIG);

  const testType = process.argv[2] || 'all';

  try {
    switch (testType) {
      case 'load':
        console.log('\n🚀 Running Load Test...\n');
        const loadResult = await loadTest.runLoadTest();
        console.log('\n📊 Load Test Results:');
        console.log(JSON.stringify(loadResult, null, 2));
        break;

      case 'burst':
        console.log('\n🚀 Running Connection Burst Test...\n');
        const burstResult = await loadTest.runConnectionBurstTest(5, 50);
        console.log('\n📊 Burst Test Results:');
        console.log(JSON.stringify(burstResult, null, 2));
        break;

      case 'broadcast':
        console.log('\n🚀 Running Message Broadcast Test...\n');
        const broadcastResult = await loadTest.runMessageBroadcastTest(100, 500);
        console.log('\n📊 Broadcast Test Results:');
        console.log(JSON.stringify(broadcastResult, null, 2));
        break;

      case 'heartbeat':
        console.log('\n🚀 Running Heartbeat Test...\n');
        const heartbeatResult = await loadTest.runHeartbeatTest(50, 30000);
        console.log('\n📊 Heartbeat Test Results:');
        console.log(JSON.stringify(heartbeatResult, null, 2));
        break;

      case 'memory':
        console.log('\n🚀 Running Memory Leak Test...\n');
        const memoryResult = await loadTest.runMemoryLeakTest(100, 60000);
        console.log('\n📊 Memory Test Results:');
        console.log(JSON.stringify(memoryResult, null, 2));
        break;

      case 'all':
      default:
        console.log('\n🚀 Running Full Test Suite...\n');

        console.log('Test 1: Connection Burst Test');
        await loadTest.runConnectionBurstTest(3, 20);

        console.log('\nTest 2: Message Broadcast Test');
        await loadTest.runMessageBroadcastTest(30, 100);

        console.log('\nTest 3: Heartbeat Test');
        await loadTest.runHeartbeatTest(20, 20000);

        console.log('\nTest 4: Full Load Test');
        const fullResult = await loadTest.runLoadTest();

        console.log('\n📊 Final Results:');
        console.log(JSON.stringify(fullResult, null, 2));
        break;
    }

    console.log('\n✅ All tests completed successfully!');
    process.exit(0);
  } catch (error) {
    console.error('\n❌ Test failed:', error);
    process.exit(1);
  }
}

runTests();
