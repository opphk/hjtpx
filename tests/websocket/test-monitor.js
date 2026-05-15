const io = require('socket.io-client');
const chalk = require('chalk');

const CONFIG = {
  SERVER_URL: 'http://localhost:3000',
  TEST_DURATION: 60000,
  UPDATE_INTERVAL: 5000,
  METRICS_INTERVAL: 10000,
  JWT_SECRET: 'your-secret-key'
};

function generateToken(userId) {
  const jwt = require('jsonwebtoken');
  return jwt.sign({ id: `user_${userId}` }, CONFIG.JWT_SECRET);
}

let stats = {
  startTime: null,
  connections: 0,
  messagesReceived: 0,
  errors: 0,
  memoryUsage: [],
  networkStats: [],
  serverMetrics: [],
  eventsReceived: 0
};

let socket = null;

async function createMonitorClient() {
  return new Promise((resolve, reject) => {
    const token = generateToken('monitor');
    socket = io(CONFIG.SERVER_URL, {
      auth: { token },
      transports: ['websocket'],
      reconnection: true
    });

    socket.on('connect', () => {
      console.log(chalk.green('Monitor client connected'));
      socket.emit('get:metrics', (response) => {
        if (response.success) {
          console.log(chalk.cyan('Initial server metrics received'));
        }
      });
      resolve(socket);
    });

    socket.on('connected', (data) => {
      console.log(chalk.green('Server acknowledged connection:'), data.message);
    });

    socket.on('presence:update', (data) => {
      stats.eventsReceived++;
      stats.serverMetrics.push({
        timestamp: new Date().toISOString(),
        event: 'presence:update',
        userId: data.userId,
        isOnline: data.isOnline
      });
    });

    socket.on('notification', (data) => {
      stats.eventsReceived++;
      stats.messagesReceived++;
    });

    socket.on('reconnect', () => {
      stats.eventsReceived++;
      console.log(chalk.yellow('Reconnected to server'));
    });

    socket.on('disconnect', (reason) => {
      console.log(chalk.red('Disconnected from server:'), reason);
    });

    socket.on('connect_error', (error) => {
      stats.errors++;
      console.error(chalk.red('Monitor client connection failed: '), error.message);
      reject(error);
    });
  });
}

function collectMemoryUsage() {
  const usage = process.memoryUsage();
  stats.memoryUsage.push({
    timestamp: new Date().toISOString(),
    rss: usage.rss,
    heapTotal: usage.heapTotal,
    heapUsed: usage.heapUsed,
    external: usage.external
  });
}

function collectNetworkStats() {
  if (socket && socket.io) {
    const engine = socket.io.engine;
    stats.networkStats.push({
      timestamp: new Date().toISOString(),
      connected: socket.connected,
      transport: engine.transport.name,
      packetsSent: engine.packetsSent,
      packetsReceived: engine.packetsReceived,
      bytesSent: engine.bytesSent,
      bytesReceived: engine.bytesReceived
    });
  }
}

async function requestServerMetrics() {
  if (socket && socket.connected) {
    return new Promise((resolve) => {
      socket.emit('get:metrics', (response) => {
        if (response.success) {
          stats.serverMetrics.push({
            timestamp: new Date().toISOString(),
            ...response.metrics
          });
        }
        resolve();
      });
    });
  }
}

function calculateStats() {
  const memUsage = stats.memoryUsage;
  const netStats = stats.networkStats;
  
  const avgHeapUsed = memUsage.reduce((sum, m) => sum + m.heapUsed, 0) / memUsage.length;
  const maxHeapUsed = Math.max(...memUsage.map(m => m.heapUsed));
  const minHeapUsed = Math.min(...memUsage.map(m => m.heapUsed));
  
  const heapGrowth = memUsage.length > 1 
    ? memUsage[memUsage.length - 1].heapUsed - memUsage[0].heapUsed 
    : 0;
  
  return {
    avgHeapUsed,
    maxHeapUsed,
    minHeapUsed,
    heapGrowth,
    totalEventsReceived: stats.eventsReceived,
    totalMessagesReceived: stats.messagesReceived,
    totalErrors: stats.errors,
    uptime: stats.startTime ? Date.now() - stats.startTime.getTime() : 0
  };
}

function printMonitorResults() {
  console.log(chalk.green.bold('\n📊 WebSocket Monitoring Results\n'));
  
  console.log(chalk.white('Test Duration:'), chalk.cyan(`${(CONFIG.TEST_DURATION / 1000).toFixed(1)} seconds`));
  console.log(chalk.white('Monitor Uptime:'), chalk.cyan(`${(calculateStats().uptime / 1000).toFixed(1)} seconds`));
  
  console.log(chalk.white('\nClient Statistics:'));
  console.log(chalk.cyan('  - Total Events Received:  '), stats.eventsReceived);
  console.log(chalk.cyan('  - Total Messages Received:'), stats.messagesReceived);
  console.log(chalk.cyan('  - Total Errors:          '), stats.errors);
  
  console.log(chalk.white('\nMemory Usage (MB):'));
  console.log(chalk.cyan('  - Average Heap Used:  '), (calculateStats().avgHeapUsed / 1024 / 1024).toFixed(2));
  console.log(chalk.cyan('  - Maximum Heap Used:  '), (calculateStats().maxHeapUsed / 1024 / 1024).toFixed(2));
  console.log(chalk.cyan('  - Minimum Heap Used:  '), (calculateStats().minHeapUsed / 1024 / 1024).toFixed(2));
  console.log(chalk.cyan('  - Heap Growth:        '), (calculateStats().heapGrowth / 1024 / 1024).toFixed(2));
  
  if (stats.networkStats.length > 0) {
    const latestNet = stats.networkStats[stats.networkStats.length - 1];
    console.log(chalk.white('\nNetwork Statistics:'));
    console.log(chalk.cyan('  - Connection Status:    '), latestNet.connected ? chalk.green('Connected') : chalk.red('Disconnected'));
    console.log(chalk.cyan('  - Transport:            '), latestNet.transport);
    console.log(chalk.cyan('  - Packets Sent:         '), latestNet.packetsSent);
    console.log(chalk.cyan('  - Packets Received:     '), latestNet.packetsReceived);
    console.log(chalk.cyan('  - Bytes Sent:           '), (latestNet.bytesSent / 1024).toFixed(2) + ' KB');
    console.log(chalk.cyan('  - Bytes Received:       '), (latestNet.bytesReceived / 1024).toFixed(2) + ' KB');
  }
  
  if (stats.serverMetrics.length > 0) {
    const latestMetrics = stats.serverMetrics[stats.serverMetrics.length - 1];
    if (latestMetrics.currentConnections !== undefined) {
      console.log(chalk.white('\nServer Metrics (Latest):'));
      console.log(chalk.cyan('  - Current Connections:   '), latestMetrics.currentConnections);
      console.log(chalk.cyan('  - Online Users:          '), latestMetrics.onlineUsers);
      console.log(chalk.cyan('  - Total Connections:     '), latestMetrics.totalConnections);
      console.log(chalk.cyan('  - Messages Sent:        '), latestMetrics.messagesSent);
      console.log(chalk.cyan('  - Messages Received:    '), latestMetrics.messagesReceived);
      console.log(chalk.cyan('  - Server Errors:        '), latestMetrics.errors);
      
      if (latestMetrics.avgConnectionTime) {
        console.log(chalk.cyan('  - Avg Connection Time:  '), (latestMetrics.avgConnectionTime / 1000).toFixed(2) + 's');
      }
    }
  }
  
  const eventsPerSecond = stats.eventsReceived / (CONFIG.TEST_DURATION / 1000);
  console.log(chalk.white('\nPerformance Metrics:'));
  console.log(chalk.cyan('  - Events per Second:     '), eventsPerSecond.toFixed(2));
  console.log(chalk.cyan('  - Memory Samples:       '), stats.memoryUsage.length);
  console.log(chalk.cyan('  - Network Samples:      '), stats.networkStats.length);
}

async function main() {
  console.log(chalk.blue.bold('\n🚀 HJTPX WebSocket Monitoring Test\n'));
  console.log(chalk.gray('========================================\n'));
  console.log(chalk.cyan('Configuration:'));
  console.log(chalk.white('  - Server URL:     '), CONFIG.SERVER_URL);
  console.log(chalk.white('  - Test Duration:   '), (CONFIG.TEST_DURATION / 1000) + 's');
  console.log(chalk.white('  - Update Interval: '), CONFIG.UPDATE_INTERVAL + 'ms');
  console.log(chalk.gray('\n========================================\n'));

  try {
    await createMonitorClient();
    stats.startTime = new Date();

    console.log(chalk.blue(`Monitoring will run for ${CONFIG.TEST_DURATION / 1000} seconds...\n`));

    const memoryInterval = setInterval(collectMemoryUsage, CONFIG.UPDATE_INTERVAL);
    const networkInterval = setInterval(collectNetworkStats, CONFIG.UPDATE_INTERVAL);
    const metricsInterval = setInterval(requestServerMetrics, CONFIG.METRICS_INTERVAL);

    const startTime = Date.now();
    let lastPrintTime = startTime;

    const updateLoop = setInterval(async () => {
      const now = Date.now();
      const elapsed = now - startTime;
      const percent = Math.min((elapsed / CONFIG.TEST_DURATION) * 100, 100);
      
      if (now - lastPrintTime >= 5000) {
        process.stdout.clearLine(0);
        process.stdout.cursorTo(0);
        console.log(chalk.cyan(`Progress: ${percent.toFixed(1)}% | Events: ${stats.eventsReceived} | Errors: ${stats.errors}`));
        lastPrintTime = now;
      }
    }, 1000);

    await new Promise(resolve => setTimeout(resolve, CONFIG.TEST_DURATION));

    clearInterval(memoryInterval);
    clearInterval(networkInterval);
    clearInterval(metricsInterval);
    clearInterval(updateLoop);

    if (socket) {
      socket.disconnect();
    }

    console.log(chalk.green.bold('\n\n✅ Monitoring completed successfully!\n'));
    printMonitorResults();
  } catch (error) {
    console.error(chalk.red('Monitoring error:'), error);
  }

  console.log(chalk.green.bold('\n✅ Test completed!\n'));
  process.exit(0);
}

main();
