require('dotenv').config({
  path: process.env.NODE_ENV === 'production' ? '.env.production' :
        process.env.NODE_ENV === 'staging' ? '.env.staging' : '.env'
});

const http = require('http');
const express = require('express');

const app = express();

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

const server = http.createServer(app);

const WebSocketServer = require('./src/backend/websocket');
const websocketServer = new WebSocketServer(server);

const PORT = process.env.WS_TEST_PORT || 3002;

server.listen(PORT, () => {
  console.log(`WebSocket Test Server running on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`WebSocket endpoint: ws://localhost:${PORT}`);
  console.log('\nYou can now run the load test with:');
  console.log(`WS_TEST_URL=http://localhost:${PORT} node tests/websocket/load-test.js\n`);
});

process.on('SIGTERM', () => {
  console.log('SIGTERM received, closing server...');
  websocketServer.close();
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('SIGINT received, closing server...');
  websocketServer.close();
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

module.exports = { server, websocketServer };
