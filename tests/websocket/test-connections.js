const io = require('socket.io-client');
const chalk = require('chalk');
const cliProgress = require('cli-progress');

const CONFIG = {
  SERVER_URL: 'http://localhost:3000',
  TOTAL_CONNECTIONS: 100,
  CONNECTIONS_PER_BATCH: 20,
  BATCH_DELAY: 500,
  STAY_CONNECTED_TIME: 30000,
  JWT_SECRET: 'your-secret-key'
};

function generateToken(userId) {
  const jwt = require('jsonwebtoken');
  return jwt.sign({ id: `user_${userId}` }, CONFIG.JWT_SECRET);
}

let connections = [];
let successCount = 0;
let errorCount = 0;
let startTimes = new Map();

const progressBar = new cliProgress.SingleBar({
  format: '连接进度 |' + chalk.cyan('{bar}') + '| {percentage}% || {value}/{total} 连接',
  barCompleteChar: '\u2588',
  barIncompleteChar: '\u2591',
  hideCursor: true
}, cliProgress.Presets.shades_classic);

async function createConnection(userId) {
  return new Promise((resolve, reject) => {
    const startTime = Date.now();
    const token = generateToken(userId);
    
    const socket = io(CONFIG.SERVER_URL, {
      auth: { token },
      transports: ['websocket'],
      reconnection: false
    });

    startTimes.set(userId, startTime);

    socket.on('connect', () => {
      const endTime = Date.now();
      const duration = endTime - startTime;
      successCount++;
      connections.push(socket);
      socket.userId = userId;
      socket.connectDuration = duration;
      resolve({ success: true, duration, socket });
    });

    socket.on('connect_error', (error) => {
      errorCount++;
      console.error(chalk.red(`连接 ${userId} 失败: `), error.message);
      resolve({ success: false, error: error.message });
    });

    socket.on('error', (error) => {
      console.error(chalk.red(`连接 ${userId} 错误: `), error.message);
    });
  });
}

async function runConnectionBatch(startIndex, count) {
  const promises = [];
  for (let i = 0; i < count; i++) {
    const userId = startIndex + i;
    promises.push(createConnection(userId));
  }
  return Promise.all(promises);
}

async function main() {
  console.log(chalk.blue.bold('\\n🚀 HJTPX WebSocket 并发连接压力测试\\n'));
  console.log(chalk.gray('========================================\\n'));
  
  const testStartTime = Date.now();
  progressBar.start(CONFIG.TOTAL_CONNECTIONS, 0);

  for (let i = 0; i < CONFIG.TOTAL_CONNECTIONS; i += CONFIG.CONNECTIONS_PER_BATCH) {
    const batchSize = Math.min(CONFIG.CONNECTIONS_PER_BATCH, CONFIG.TOTAL_CONNECTIONS - i);
    await runConnectionBatch(i, batchSize);
    progressBar.update(i + batchSize);
    
    if (i + batchSize < CONFIG.TOTAL_CONNECTIONS) {
      await new Promise(resolve => setTimeout(resolve, CONFIG.BATCH_DELAY));
    }
  }

  progressBar.stop();
  const testEndTime = Date.now();

  const totalDuration = testEndTime - testStartTime;
  const avgConnectTime = connections.reduce((sum, s) => sum + s.connectDuration, 0) / connections.length;
  const maxConnectTime = Math.max(...connections.map(s => s.connectDuration));
  const minConnectTime = Math.min(...connections.map(s => s.connectDuration));

  console.log(chalk.green.bold('\\n📊 测试结果\\n'));
  console.log(chalk.white('总连接数: '), chalk.cyan(CONFIG.TOTAL_CONNECTIONS));
  console.log(chalk.white('成功连接: '), chalk.green(successCount));
  console.log(chalk.white('失败连接: '), chalk.red(errorCount));
  console.log(chalk.white('总耗时: '), chalk.yellow(`${totalDuration}ms'));
  console.log(chalk.white('平均连接时间: '), chalk.yellow(`${avgConnectTime.toFixed(2)}ms'));
  console.log(chalk.white('最长连接时间: '), chalk.yellow(`${maxConnectTime}ms'));
  console.log(chalk.white('最短连接时间: '), chalk.yellow(`${minConnectTime}ms'));
  console.log(chalk.white('连接成功率: '), chalk.green(((successCount / CONFIG.TOTAL_CONNECTIONS * 100).toFixed(2) + '%'));

  console.log(chalk.gray(`\\n保持连接 ${CONFIG.STAY_CONNECTED_TIME / 1000} 秒...\\n'));
  await new Promise(resolve => setTimeout(resolve, CONFIG.STAY_CONNECTED_TIME));

  console.log(chalk.blue('开始断开连接...'));
  connections.forEach(socket => {
    socket.disconnect();
  });

  console.log(chalk.green.bold('\\n✅ 测试完成!\\n'));
  process.exit(0);
}

main().catch(error => {
  console.error(chalk.red('测试出错:'), error);
  process.exit(1);
});
