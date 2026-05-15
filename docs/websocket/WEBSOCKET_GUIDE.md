# WebSocket 实时通信开发指南

## 概述

本文档详细介绍 HJTPX 项目中 WebSocket 实时通信系统的架构、功能和使用方法。

## 目录

1. [系统架构](#系统架构)
2. [模块说明](#模块说明)
3. [WebSocket 事件](#websocket-事件)
4. [客户端使用示例](#客户端使用示例)
5. [服务端 API](#服务端-api)
6. [最佳实践](#最佳实践)
7. [安全注意事项](#安全注意事项)
8. [性能优化](#性能优化)
9. [故障排除](#故障排除)

## 系统架构

### 架构图

```
┌─────────────────────────────────────────────────────────┐
│                    WebSocket Server                     │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │ Connection  │  │ Notification│  │  Online Status  │ │
│  │   Manager   │  │   System    │  │    Manager      │ │
│  └─────────────┘  └─────────────┘  └─────────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │   Message   │  │  Heartbeat  │  │  Socket.IO      │ │
│  │ Broadcaster │  │   System    │  │    Server       │ │
│  └─────────────┘  └─────────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────┤
│              Express HTTP Server + Apollo GraphQL       │
└─────────────────────────────────────────────────────────┘
```

### 技术栈

- **Socket.IO**: 实时通信库
- **Redis**: 在线状态存储和缓存
- **Express.js**: HTTP 服务层
- **JWT**: 认证令牌

## 模块说明

### 1. ConnectionManager (连接管理器)

负责管理所有 WebSocket 连接。

**主要功能:**
- 连接池管理
- 连接状态追踪
- 用户-连接映射
- 房间-连接映射
- 重连次数限制

**关键方法:**

```javascript
// 添加连接
connectionManager.addConnection(socket);

// 移除连接
connectionManager.removeConnection(socketId);

// 获取用户的所有连接
connectionManager.getUserConnections(userId);

// 检查用户是否在线
connectionManager.isUserOnline(userId);

// 获取在线用户列表
connectionManager.getOnlineUsers();

// 添加房间连接
connectionManager.addRoomConnection(room, socketId, userId);

// 获取统计数据
connectionManager.getStats();
```

### 2. NotificationSystem (通知系统)

负责实时通知的发送和管理。

**主要功能:**
- 单用户通知
- 批量通知
- 广播通知
- 通知持久化
- 通知队列
- 通知历史

**关键方法:**

```javascript
// 发送通知给单个用户
await notificationSystem.sendToUser(userId, {
  type: 'info',
  title: '新消息',
  message: '您有一条新消息',
  priority: 'normal',
  data: { messageId: '123' }
});

// 批量发送通知
await notificationSystem.sendToUsers(userIds, notification);

// 广播通知
notificationSystem.broadcast(notification, 'notifications');

// 标记通知为已读
await notificationSystem.markAsRead(notificationId, userId);

// 获取通知历史
await notificationSystem.getNotificationHistory(userId, { limit: 50 });
```

### 3. OnlineStatusManager (在线状态管理器)

管理用户在线状态。

**主要功能:**
- 用户在线/离线状态
- 状态更新（online, away, busy, offline）
- 最后活跃时间追踪
- 状态历史记录
- 自动状态变更（如空闲检测）

**关键方法:**

```javascript
// 设置用户在线
onlineStatusManager.setUserOnline(userId, socketId, metadata);

// 设置用户离线
onlineStatusManager.setUserOffline(userId, reason, metadata);

// 更新用户状态
onlineStatusManager.updateUserStatus(userId, 'away', metadata);

// 获取用户状态
onlineStatusManager.getUserStatus(userId);

// 获取所有在线用户
onlineStatusManager.getOnlineUsers();

// 检查用户是否在线
onlineStatusManager.isUserOnline(userId);

// 获取按状态分组的用户
onlineStatusManager.getUsersByStatus('online');
```

### 4. MessageBroadcaster (消息广播器)

处理消息的发送和历史记录。

**主要功能:**
- 私聊消息
- 群组消息
- 系统消息广播
- 频道消息
- 消息历史
- 消息队列

**关键方法:**

```javascript
// 发送私聊消息
messageBroadcaster.sendPrivateMessage(fromUserId, toUserId, {
  content: '你好',
  type: 'text',
  metadata: {}
});

// 发送群组消息
messageBroadcaster.sendGroupMessage(fromUserId, room, {
  content: '大家好',
  type: 'text'
});

// 广播系统消息
messageBroadcaster.broadcastSystemMessage({
  content: '系统维护通知',
  priority: 'high'
});

// 发送到频道
messageBroadcaster.broadcastToChannel(channel, fromUserId, {
  content: '频道消息'
});

// 获取用户消息历史
messageBroadcaster.getUserHistory(userId, { limit: 50 });

// 获取房间消息历史
messageBroadcaster.getRoomHistory(room, { limit: 100 });
```

### 5. HeartbeatSystem (心跳系统)

维护连接的健康状态。

**主要功能:**
- 心跳检测
- 超时检测
- 自动重连
- 重连次数限制
- 连接健康监控

**关键方法:**

```javascript
// 注册心跳
heartbeatSystem.registerSocketHeartbeat(socket);

// 处理心跳响应
heartbeatSystem.handleHeartbeatResponse(socketId);

// 处理心跳超时
heartbeatSystem.handleHeartbeatTimeout(socketId);

// 获取连接心跳状态
heartbeatSystem.getHeartbeatStatus(socketId);

// 获取所有心跳统计
heartbeatSystem.getAllHeartbeatStats();

// 获取心跳系统统计
heartbeatSystem.getStats();
```

## WebSocket 事件

### 客户端可发送的事件

| 事件名 | 参数 | 说明 |
|--------|------|------|
| `join` | room, callback | 加入房间 |
| `leave` | room, callback | 离开房间 |
| `subscribe` | channel, callback | 订阅频道 |
| `unsubscribe` | channel, callback | 取消订阅 |
| `message` | data, callback | 发送消息 |
| `broadcast` | data, callback | 广播消息 |
| `private_message` | {to, content, type}, callback | 发送私信 |
| `group_message` | {room, content, type}, callback | 发送群组消息 |
| `heartbeat` | - | 发送心跳 |
| `status_update` | {status}, callback | 更新状态 |
| `get:online_users` | callback | 获取在线用户 |
| `get:history` | {type, room, limit}, callback | 获取历史 |
| `get:metrics` | callback | 获取指标 |

### 服务器发送的事件

| 事件名 | 参数 | 说明 |
|--------|------|------|
| `connected` | {socketId, message} | 连接成功 |
| `notification` | notificationData | 收到通知 |
| `message` | messageData | 收到消息 |
| `online_status` | {status, timestamp} | 在线状态更新 |
| `presence:update` | {userId, isOnline, timestamp} | 用户在线状态变更 |
| `user:joined` | {userId, room, timestamp} | 用户加入房间 |
| `user:left` | {userId, room, timestamp} | 用户离开房间 |
| `heartbeat:ack` | {timestamp, serverTime} | 心跳响应 |
| `message:ack` | {messageId, status, timestamp} | 消息确认 |
| `data:update` | {entityType, entityId, action, data} | 数据更新 |

## 客户端使用示例

### 基础连接

```javascript
import { io } from 'socket.io-client';

const socket = io('http://localhost:3000', {
  auth: {
    token: 'your-jwt-token'
  },
  transports: ['websocket', 'polling']
});

socket.on('connect', () => {
  console.log('Connected:', socket.id);
});

socket.on('connected', (data) => {
  console.log('Server confirmed:', data.message);
  console.log('My socket ID:', data.socketId);
});

socket.on('disconnect', (reason) => {
  console.log('Disconnected:', reason);
});

socket.on('connect_error', (error) => {
  console.error('Connection error:', error.message);
});
```

### 心跳机制

```javascript
let heartbeatInterval;

function startHeartbeat() {
  heartbeatInterval = setInterval(() => {
    socket.emit('heartbeat');
  }, 20000);
}

socket.on('heartbeat:ack', (data) => {
  console.log('Heartbeat acknowledged:', data.serverTime);
});

socket.on('disconnect', () => {
  clearInterval(heartbeatInterval);
});
```

### 接收通知

```javascript
socket.on('notification', (notification) => {
  console.log('New notification:', notification);

  switch (notification.type) {
    case 'info':
      showInfoToast(notification);
      break;
    case 'warning':
      showWarningToast(notification);
      break;
    case 'error':
      showErrorToast(notification);
      break;
    default:
      showGenericNotification(notification);
  }
});

socket.on('notification:read', (data) => {
  console.log('Notification marked as read:', data.notificationId);
});
```

### 加入房间和订阅

```javascript
// 加入房间
socket.emit('join', 'room-123', (response) => {
  if (response.success) {
    console.log('Joined room:', response.room);
  }
});

// 加入群组消息
socket.emit('group_message', {
  room: 'chat-room-1',
  content: 'Hello everyone!',
  type: 'text'
}, (response) => {
  if (response.success) {
    console.log('Message sent:', response.messageId);
  }
});

// 监听房间消息
socket.on('message', (message) => {
  if (message.type === 'group') {
    console.log('Room message from', message.from, ':', message.content);
  }
});
```

### 私聊消息

```javascript
// 发送私信
socket.emit('private_message', {
  to: 'user-id-456',
  content: 'Hi, this is a private message',
  type: 'text',
  metadata: { replyTo: 'msg-123' }
}, (response) => {
  if (response.success) {
    console.log('Private message sent:', response.messageId);
  }
});

// 接收私信
socket.on('message', (message) => {
  if (message.type === 'private') {
    console.log('Private message from', message.from, ':', message.content);
  }
});
```

### 在线状态管理

```javascript
// 更新自己的状态
socket.emit('status_update', { status: 'away' }, (response) => {
  if (response.success) {
    console.log('Status updated to away');
  }
});

// 获取在线用户
socket.emit('get:online_users', (response) => {
  if (response.success) {
    console.log('Online users:', response.users);
  }
});

// 监听他人状态变化
socket.on('online_status', (data) => {
  console.log('User status changed:', data.userId, '->', data.status);
});

socket.on('presence:update', (data) => {
  if (data.isOnline) {
    console.log('User came online:', data.userId);
  } else {
    console.log('User went offline:', data.userId);
  }
});
```

### 错误处理

```javascript
socket.on('error', (error) => {
  console.error('Socket error:', error);
});

socket.on('connection:timeout', (data) => {
  console.warn('Connection timeout:', data);
  // 提示用户重新连接
});
```

## 服务端 API

### WebSocketServer 类

```javascript
const WebSocketServer = require('./backend/websocket');

// 创建服务器
const wss = new WebSocketServer(httpServer);

// 发送通知给用户
await wss.sendNotification(userId, notification);

// 批量发送通知
await wss.sendBulkNotifications(userIds, notification);

// 广播通知
wss.broadcastNotification(notification, 'notifications');

// 发送消息给特定用户
wss.sendToUser(userId, 'custom_event', data);

// 广播消息
wss.broadcast('custom_event', data, room);

// 获取连接统计
wss.getConnectionStats();

// 获取详细指标
wss.getDetailedMetrics();

// 关闭服务器
wss.close();
```

### 使用 WebSocketService

```javascript
const websocketService = require('./backend/services/websocketService');

// 发送通知
await websocketService.sendNotification(userId, {
  type: 'info',
  title: 'Test',
  message: 'Test notification'
});

// 发送批量通知
await websocketService.sendBulkNotifications(userIds, notification);

// 广播通知
websocketService.broadcastNotification(notification, 'notifications');

// 推送数据更新
await websocketService.pushDataUpdate(userId, 'document', docId, 'update', data);

// 广播数据更新
await websocketService.broadcastDataUpdate('document', docId, 'update', data);

// 检查用户是否在线
websocketService.isUserOnline(userId);

// 获取在线用户
websocketService.getOnlineUsers();

// 获取连接统计
websocketService.getConnectionStats();
```

## 最佳实践

### 1. 连接管理

```javascript
// ✅ 正确：在组件卸载时断开连接
useEffect(() => {
  socket.connect();

  return () => {
    socket.disconnect();
  };
}, []);

// ❌ 错误：未清理连接
useEffect(() => {
  socket.connect();
}, []);
```

### 2. 心跳机制

```javascript
// ✅ 正确：定期发送心跳
const HEARTBEAT_INTERVAL = 20000;

socket.on('connect', () => {
  startHeartbeat();
});

function startHeartbeat() {
  const interval = setInterval(() => {
    if (socket.connected) {
      socket.emit('heartbeat');
    }
  }, HEARTBEAT_INTERVAL);

  socket.on('disconnect', () => clearInterval(interval));
}
```

### 3. 错误重试

```javascript
// ✅ 正确：实现重连逻辑
socket.on('connect_error', (error) => {
  console.error('Connection error:', error);

  setTimeout(() => {
    socket.connect();
  }, 5000);
});

socket.on('disconnect', (reason) => {
  if (reason === 'io server disconnect') {
    // 服务器断开，需要手动重连
    socket.connect();
  }
});
```

### 4. 消息确认

```javascript
// ✅ 正确：等待消息确认
socket.emit('private_message', data, (ack) => {
  if (ack.success) {
    console.log('Message confirmed:', ack.messageId);
  } else {
    console.error('Message failed:', ack.error);
    // 重试逻辑
  }
});
```

### 5. 状态管理

```javascript
// ✅ 正确：离开时更新状态
useEffect(() => {
  return () => {
    socket.emit('status_update', { status: 'offline' });
  };
}, []);

// ✅ 正确：页面可见性变化时更新状态
document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    socket.emit('status_update', { status: 'away' });
  } else {
    socket.emit('status_update', { status: 'online' });
  }
});
```

## 安全注意事项

### 1. 认证

```javascript
// ✅ 正确：使用有效的 JWT 令牌
const token = getAuthToken();
const socket = io('http://localhost:3000', {
  auth: { token }
});

// ❌ 错误：暴露敏感信息
socket.auth = {
  token: 'Bearer xxx',
  userId: '123'
};
```

### 2. 输入验证

```javascript
// ✅ 正确：验证所有输入
socket.emit('message', {
  content: sanitize(input),
  type: validateType(type)
});

// ❌ 错误：未验证输入
socket.emit('message', {
  content: userInput // 未处理
});
```

### 3. 速率限制

```javascript
// ✅ 正确：实现客户端速率限制
class RateLimiter {
  constructor(maxMessages, timeWindow) {
    this.messages = [];
    this.maxMessages = maxMessages;
    this.timeWindow = timeWindow;
  }

  canSend() {
    const now = Date.now();
    this.messages = this.messages.filter(
      t => now - t < this.timeWindow
    );
    return this.messages.length < this.maxMessages;
  }

  recordSend() {
    this.messages.push(Date.now());
  }
}
```

### 4. 敏感数据保护

```javascript
// ✅ 正确：不要在事件中发送敏感信息
socket.emit('update_profile', {
  name: user.name,
  publicId: user.publicId
});

// ❌ 错误：发送敏感信息
socket.emit('update_profile', {
  password: user.password,
  email: user.email
});
```

## 性能优化

### 1. 连接池管理

```javascript
// ✅ 正确：复用连接
const socket = io('http://localhost:3000', {
  reconnection: true,
  reconnectionAttempts: 5,
  reconnectionDelay: 1000
});
```

### 2. 消息压缩

```javascript
// ✅ 正确：启用压缩
const io = new Server(httpServer, {
  perMessageDeflate: {
    threshold: 1024
  }
});
```

### 3. 心跳优化

```javascript
// ✅ 正确：动态调整心跳间隔
const adaptiveHeartbeat = () => {
  const interval = socket.connected ? 20000 : 60000;
  setInterval(sendHeartbeat, interval);
};
```

### 4. 批量操作

```javascript
// ✅ 正确：批量发送消息
const messageQueue = [];
let flushTimeout;

function queueMessage(message) {
  messageQueue.push(message);

  if (!flushTimeout) {
    flushTimeout = setTimeout(() => {
      socket.emit('batch_messages', messageQueue);
      messageQueue.length = 0;
      flushTimeout = null;
    }, 100);
  }
}
```

## 故障排除

### 常见问题

#### 1. 连接失败

**问题**: WebSocket 连接失败

**解决方案**:
```javascript
// 检查网络
// 确认服务器运行
// 验证 JWT 令牌
// 检查 CORS 配置

socket.on('connect_error', (error) => {
  console.error('Connection error:', error.message);
  // error.message 可能包含:
  // - 'Authentication token required'
  // - 'Authentication failed'
  // - 'websocket error'
});
```

#### 2. 心跳超时

**问题**: 连接被判定为超时

**解决方案**:
```javascript
// 降低心跳间隔
const HEARTBEAT_INTERVAL = 15000;

// 增加超时时间
socket.io.engine.pingTimeout = 60000;
socket.io.engine.pingInterval = 25000;
```

#### 3. 消息丢失

**问题**: 发送的消息没有送达

**解决方案**:
```javascript
// 使用确认机制
socket.emit('message', data, (ack) => {
  if (!ack.success) {
    // 重试
    setTimeout(() => queueMessage(data), 1000);
  }
});

// 实现消息队列
const messageQueue = [];

socket.on('connect', () => {
  // 重新发送队列中的消息
  messageQueue.forEach(msg => socket.emit('message', msg));
  messageQueue.length = 0;
});
```

#### 4. 性能问题

**问题**: 连接数过多导致性能下降

**解决方案**:
```javascript
// 使用 Redis Adapter 进行水平扩展
const { createAdapter } = require('@socket.io/redis-adapter');
const { createClient } = require('redis');

const pubClient = createClient({ url: 'redis://localhost:6379' });
const subClient = pubClient.duplicate();

await Promise.all([pubClient.connect(), subClient.connect()]);

io.adapter(createAdapter(pubClient, subClient));
```

### 日志分析

```javascript
// 服务器端日志
logger.info('Client connected', {
  socketId: socket.id,
  userId: socket.userId
});

logger.error('Socket error', {
  socketId: socket.id,
  error: error.message
});

// 客户端日志
socket.on('connect', () => {
  console.log('Socket connected:', socket.id);
});

socket.on('disconnect', (reason) => {
  console.log('Socket disconnected:', reason);
});
```

## 测试

### 单元测试

```javascript
describe('WebSocket Modules', () => {
  test('ConnectionManager should track connections', () => {
    const manager = new ConnectionManager();
    const mockSocket = { id: 'test', userId: 'user1' };

    manager.addConnection(mockSocket);

    expect(manager.connections.size).toBe(1);
    expect(manager.isUserOnline('user1')).toBe(true);
  });

  test('NotificationSystem should send notifications', async () => {
    const system = new NotificationSystem();
    system.initialize(mockIO, mockConnectionManager);

    const result = await system.sendToUser('user1', {
      title: 'Test',
      message: 'Test message'
    });

    expect(result).toBeDefined();
  });
});
```

### 集成测试

```javascript
describe('WebSocket Integration', () => {
  test('should handle full connection lifecycle', async () => {
    // 1. 创建客户端
    const client = io('http://localhost:3000', {
      auth: { token: validToken }
    });

    // 2. 等待连接
    await new Promise(resolve => client.on('connected', resolve));

    // 3. 加入房间
    client.emit('join', 'test-room');
    await waitForEvent(client, 'user:joined');

    // 4. 发送消息
    client.emit('message', { content: 'Hello' });
    await waitForEvent(client, 'message:ack');

    // 5. 断开连接
    client.disconnect();
  });
});
```

## 参考资料

- [Socket.IO 官方文档](https://socket.io/docs/v4/)
- [WebSocket MDN](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
- [Socket.IO Redis Adapter](https://github.com/socketio/socket.io-redis-adapter)

## 更新日志

- **v1.0.0** (2026-05-15): 初始版本，包含所有核心模块
