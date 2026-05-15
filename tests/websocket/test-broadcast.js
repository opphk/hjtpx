const io = require('socket.io-client');
const chalk = require('chalk');

const CONFIG = {
  SERVER_URL: 'http://localhost:3000',
  TOTAL_CLIENTS: 50,
  MESSAGES_PER_TEST: 100,
  MESSAGE_INTERVAL: 100,
  MESSAGE_SIZE: 1024,
  TEST_ROOM: 'test_room_broadcast',
  JWT_SECRET: 'your-secret-key',
  ENABLE_DELAY_MEASUREMENT: true
};

function generateToken(userId) {
  const jwt = require('jsonwebtoken');
  return jwt.sign({ id: `user_${userId}` }, CONFIG.JWT_SECRET);
}

function generateMessage(size) {
  return 'x'.repeat(size);
}

let clients = [];
let receivedMessages = new Map();
let messageTimes = [];
let messageLatencies = [];
let testStats = {
  totalSent: 0,
  totalReceived: 0,
  messagesByClient: new Map(),
  latencyHistogram: [],
  throughputSamples: []
};

async function createClient(userId) {
  return new Promise((resolve, reject) => {
    const token = generateToken(userId);
    
    const socket = io(CONFIG.SERVER_URL, {
      auth: { token },
      transports: ['websocket'],
      reconnection: true,
      timeout: 10000
    });

    socket.on('connect', () => {
      clients.push(socket);
      socket.userId = userId;
      socket.messagesReceived = 0;
      socket.receiveTimes = [];
      receivedMessages.set(socket.id, []);
      
      socket.on('broadcast', (data) => {
        const receiveTime = Date.now();
        socket.messagesReceived++;
        socket.receiveTimes.push(receiveTime);
        receivedMessages.get(socket.id).push({
          ...data,
          receiveTime,
          latency: CONFIG.ENABLE_DELAY_MEASUREMENT ? receiveTime - (data.sendTime || receiveTime) : 0
        });
        receivedMessages.total++;
      });

      socket.on('test', (data) => {
        const receiveTime = Date.now();
        socket.messagesReceived++;
        socket.receiveTimes.push(receiveTime);
        receivedMessages.get(socket.id).push({
          ...data,
          receiveTime,
          latency: CONFIG.ENABLE_DELAY_MEASUREMENT ? receiveTime - (data.sendTime || receiveTime) : 0
        });
        receivedMessages.total++;
      });

      socket.on('room:broadcast', (data) => {
        const receiveTime = Date.now();
        socket.messagesReceived++;
        socket.receiveTimes.push(receiveTime);
        receivedMessages.get(socket.id).push({
          ...data,
          receiveTime,
          latency: CONFIG.ENABLE_DELAY_MEASUREMENT ? receiveTime - (data.sendTime || receiveTime) : 0
        });
        receivedMessages.total++;
      });

      resolve(socket);
    });

    socket.on('connect_error', (error) => {
      console.error(chalk.red(`Client ${userId} connection failed: `), error.message);
      resolve(null);
    });
  });
}

async function setupClients() {
  console.log(chalk.blue('Creating clients...'));
  const promises = [];
  for (let i = 0; i < CONFIG.TOTAL_CLIENTS; i++) {
    promises.push(createClient(i));
  }
  await Promise.all(promises);
  clients = clients.filter(c => c !== null);
  console.log(chalk.green(`Successfully created ${clients.length} clients`));
}

async function joinTestRoom() {
  console.log(chalk.blue('Joining test room...'));
  const joinPromises = clients.map(client => {
    return new Promise((resolve) => {
      client.emit('join', CONFIG.TEST_ROOM, (response) => {
        if (response.success) {
          resolve(true);
        } else {
          console.warn(chalk.yellow(`Client ${client.userId} failed to join room`));
          resolve(false);
        }
      });
    });
  });
  await Promise.all(joinPromises);
  console.log(chalk.green('All clients joined test room'));
}

async function runBroadcastTest() {
  console.log(chalk.blue('Starting broadcast test...'));
  receivedMessages.total = 0;
  messageTimes = [];
  messageLatencies = [];

  const sender = clients[0];
  const testStartTime = Date.now();
  
  for (let i = 0; i < CONFIG.MESSAGES_PER_TEST; i++) {
    const sendTime = Date.now();
    sender.emit('broadcast', {
      message: generateMessage(CONFIG.MESSAGE_SIZE),
      type: 'test',
      sendTime: sendTime
    });
    messageTimes.push(sendTime);
    await new Promise(resolve => setTimeout(resolve, CONFIG.MESSAGE_INTERVAL));
  }

  const testEndTime = Date.now();
  
  await new Promise(resolve => setTimeout(resolve, 2000));

  const totalTestTime = testEndTime - testStartTime;
  const expectedMessages = CONFIG.MESSAGES_PER_TEST * (clients.length - 1);
  const successRate = (receivedMessages.total / expectedMessages) * 100;
  const messagesPerSecond = receivedMessages.total / (totalTestTime / 1000);

  return {
    totalTime: totalTestTime,
    expected: expectedMessages,
    received: receivedMessages.total,
    successRate,
    throughput: messagesPerSecond
  };
}

async function runRoomBroadcastTest() {
  console.log(chalk.blue('Starting room broadcast test...'));
  receivedMessages.total = 0;
  messageTimes = [];
  messageLatencies = [];

  const sender = clients[0];
  const testStartTime = Date.now();
  const roomMessageCount = Math.min(50, CONFIG.MESSAGES_PER_TEST);
  
  for (let i = 0; i < roomMessageCount; i++) {
    const sendTime = Date.now();
    sender.emit('broadcast', {
      room: CONFIG.TEST_ROOM,
      message: generateMessage(CONFIG.MESSAGE_SIZE),
      type: 'room:broadcast',
      sendTime: sendTime
    });
    messageTimes.push(sendTime);
    await new Promise(resolve => setTimeout(resolve, CONFIG.MESSAGE_INTERVAL));
  }

  const testEndTime = Date.now();
  
  await new Promise(resolve => setTimeout(resolve, 2000));

  const totalTestTime = testEndTime - testStartTime;
  const expectedMessages = roomMessageCount * (clients.length - 1);
  const successRate = (receivedMessages.total / expectedMessages) * 100;
  const messagesPerSecond = receivedMessages.total / (totalTestTime / 1000);

  return {
    totalTime: totalTestTime,
    expected: expectedMessages,
    received: receivedMessages.total,
    successRate,
    throughput: messagesPerSecond
  };
}

function calculateLatencyMetrics() {
  const allLatencies = [];
  receivedMessages.forEach((messages) => {
    messages.forEach(msg => {
      if (msg.latency && msg.latency > 0) {
        allLatencies.push(msg.latency);
      }
    });
  });

  if (allLatencies.length === 0) {
    return { avg: 0, min: 0, max: 0, p50: 0, p95: 0, p99: 0 };
  }

  allLatencies.sort((a, b) => a - b);
  return {
    avg: allLatencies.reduce((sum, l) => sum + l, 0) / allLatencies.length,
    min: allLatencies[0],
    max: allLatencies[allLatencies.length - 1],
    p50: allLatencies[Math.floor(allLatencies.length * 0.5)],
    p95: allLatencies[Math.floor(allLatencies.length * 0.95)],
    p99: allLatencies[Math.floor(allLatencies.length * 0.99)]
  };
}

function printTestResults(broadcastResults, roomBroadcastResults) {
  console.log(chalk.green.bold('\n📊 Broadcast Performance Test Results\n'));
  
  console.log(chalk.white('Global Broadcast Test:'));
  console.log(chalk.cyan('  - Clients:              '), clients.length);
  console.log(chalk.cyan('  - Messages Sent:       '), CONFIG.MESSAGES_PER_TEST);
  console.log(chalk.cyan('  - Expected Received:   '), broadcastResults.expected);
  console.log(chalk.green('  - Actual Received:     '), broadcastResults.received);
  console.log(chalk.green('  - Success Rate:        '), broadcastResults.successRate.toFixed(2) + '%');
  console.log(chalk.cyan('  - Total Time:          '), broadcastResults.totalTime + 'ms');
  console.log(chalk.yellow('  - Throughput:          '), broadcastResults.throughput.toFixed(2) + ' msg/s');
  
  console.log(chalk.white('\nRoom Broadcast Test:'));
  console.log(chalk.cyan('  - Room:                '), CONFIG.TEST_ROOM);
  console.log(chalk.cyan('  - Clients in Room:     '), clients.length);
  console.log(chalk.green('  - Success Rate:        '), roomBroadcastResults.successRate.toFixed(2) + '%');
  console.log(chalk.yellow('  - Throughput:          '), roomBroadcastResults.throughput.toFixed(2) + ' msg/s');
  
  const latencyMetrics = calculateLatencyMetrics();
  console.log(chalk.white('\nLatency Metrics (ms):'));
  console.log(chalk.cyan('  - Average:             '), latencyMetrics.avg.toFixed(2));
  console.log(chalk.cyan('  - Minimum:             '), latencyMetrics.min.toFixed(2));
  console.log(chalk.cyan('  - Maximum:             '), latencyMetrics.max.toFixed(2));
  console.log(chalk.cyan('  - P50 (Median):        '), latencyMetrics.p50.toFixed(2));
  console.log(chalk.cyan('  - P95:                 '), latencyMetrics.p95.toFixed(2));
  console.log(chalk.cyan('  - P99:                 '), latencyMetrics.p99.toFixed(2));
  
  console.log(chalk.white('\nPer-Client Statistics:'));
  const clientStats = clients.map(c => ({
    userId: c.userId,
    received: c.messagesReceived
  })).sort((a, b) => b.received - a.received);
  
  clientStats.slice(0, 5).forEach((stat, idx) => {
    console.log(chalk.gray(`  ${idx + 1}. Client ${stat.userId}: ${stat.received} messages`));
  });
  
  const avgPerClient = clientStats.reduce((sum, c) => sum + c.received, 0) / clientStats.length;
  const variance = clientStats.reduce((sum, c) => sum + Math.pow(c.received - avgPerClient, 2), 0) / clientStats.length;
  console.log(chalk.cyan('  - Average per client:  '), avgPerClient.toFixed(2));
  console.log(chalk.cyan('  - Std Deviation:        '), Math.sqrt(variance).toFixed(2));
}

async function main() {
  console.log(chalk.blue.bold('\n🚀 HJTPX WebSocket Broadcast Performance Test\n'));
  console.log(chalk.gray('========================================\n'));
  console.log(chalk.cyan('Configuration:'));
  console.log(chalk.white('  - Server URL:         '), CONFIG.SERVER_URL);
  console.log(chalk.white('  - Total Clients:       '), CONFIG.TOTAL_CLIENTS);
  console.log(chalk.white('  - Messages per Test:   '), CONFIG.MESSAGES_PER_TEST);
  console.log(chalk.white('  - Message Interval:    '), CONFIG.MESSAGE_INTERVAL + 'ms');
  console.log(chalk.white('  - Message Size:        '), CONFIG.MESSAGE_SIZE + ' bytes');
  console.log(chalk.gray('\n========================================\n'));

  await setupClients();
  
  if (clients.length < 2) {
    console.log(chalk.red('Insufficient clients, test cannot proceed'));
    process.exit(1);
  }

  const broadcastResults = await runBroadcastTest();
  
  await new Promise(resolve => setTimeout(resolve, 1000));
  
  await joinTestRoom();
  const roomBroadcastResults = await runRoomBroadcastTest();
  
  printTestResults(broadcastResults, roomBroadcastResults);

  console.log(chalk.blue('\nCleaning up clients...'));
  clients.forEach(client => client.disconnect());

  console.log(chalk.green.bold('\n✅ Test completed!\n'));
  process.exit(0);
}

main().catch(error => {
  console.error(chalk.red('Test error:'), error);
  process.exit(1);
});
