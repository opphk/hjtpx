# WebSocket 文档

## 概述

本系统实现了完整的WebSocket连接管理和重试机制，包括心跳检测、自动重连和连接状态管理。

## 后端 WebSocket 管理器

### 主要功能

1. **心跳检测**
   - 定期发送ping消息
   - 检测客户端存活状态
   - 自动终止不活跃的连接

2. **连接管理**
   - 跟踪所有连接
   - 管理用户认证状态
   - 支持房间订阅

3. **消息处理**
   - 支持多种消息类型
   - JSON格式消息解析
   - 错误处理

### API

```javascript
const websocketManager = require('./services/websocketManager');

// 初始化
websocketManager.initialize(httpServer);

// 广播消息
websocketManager.broadcast('event_name', data);
websocketManager.broadcast('event_name', data, 'room_name');

// 发送消息给用户
websocketManager.sendToUser(userId, 'event_name', data);

// 获取统计信息
websocketManager.getConnectionStats();
websocketManager.getDetailedMetrics();

// 获取在线用户
websocketManager.getOnlineUsers();

// 获取连接客户端
websocketManager.getConnectedClients();

// 关闭
websocketManager.close();
```

## 前端 Socket 服务

### 主要功能

1. **自动重连**
   - 指数退避策略
   - 最大重连次数限制
   - 重连状态通知

2. **心跳机制**
   - 定期ping消息
   - 超时检测
   - 自动重连

3. **消息队列**
   - 离线消息缓冲
   - 自动重发
   - 队列管理

4. **连接状态**
   - 实时状态监听
   - 状态变化通知
   - 连接统计

### API

```javascript
import { createSocketService } from './services/socketService';

// 创建服务
const socket = createSocketService({
  url: 'ws://localhost:3000',
  maxReconnectAttempts: 5,
  reconnectDelay: 1000,
  heartbeatIntervalTime: 30000,
  heartbeatTimeoutTime: 5000,
  autoConnect: true
});

// 连接
socket.connect(userId);

// 断开连接
socket.disconnect();

// 发送消息
socket.send({ type: 'event', data: 'value' });

// 监听事件
const unsubscribe = socket.on('notification', (data) => {
  console.log('Notification:', data);
});

// 监听连接状态变化
socket.onConnectionChange((state) => {
  console.log('Connection state:', state);
});

// 获取状态
socket.getConnectionState();
socket.isConnected();
socket.getStats();
```

### 连接状态

- `disconnected`: 未连接
- `connecting`: 连接中
- `connected`: 已连接
- `reconnecting`: 重连中
- `error`: 连接错误
- `failed`: 连接失败

## 消息类型

### 客户端到服务端

| 类型 | 描述 |
|------|------|
| `ping` | 心跳检测 |
| `auth` | 认证 |
| `subscribe` | 订阅房间 |
| `unsubscribe` | 取消订阅 |
| `notification_read` | 标记通知已读 |

### 服务端到客户端

| 类型 | 描述 |
|------|------|
| `connected` | 连接成功 |
| `auth_success` | 认证成功 |
| `auth_error` | 认证失败 |
| `subscribed` | 订阅成功 |
| `unsubscribed` | 取消订阅成功 |
| `pong` | 心跳响应 |
| `notification` | 通知 |
| `data:update` | 数据更新 |
| `reconnected` | 重连成功 |
| `error` | 错误 |

## 测试覆盖

- 连接管理测试
- 消息处理测试
- 心跳机制测试
- 重连逻辑测试
- 事件监听测试
- 错误处理测试

## 最佳实践

1. 始终监听连接状态变化
2. 实现消息确认机制
3. 使用消息队列处理离线场景
4. 合理设置心跳间隔
5. 设置合理的重连策略
6. 清理资源时调用disconnect()
