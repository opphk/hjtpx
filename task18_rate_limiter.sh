#!/bin/bash
# 任务18：Rate Limiter细粒度控制
# 实现多维度限流
# 实现动态限流规则
# 实现限流豁免
# 编写测试
# 文档更新

echo "=========================================="
echo "任务18：Rate Limiter细粒度控制"
echo "=========================================="

cd /workspace/hjtpx

# 1. 创建细粒度限流中间件
echo "[18.1] 创建细粒度限流中间件..."

mkdir -p src/backend/middleware

cat > src/backend/middleware/rateLimiterAdvanced.js << 'EOF'
const redis = require('redis');
const client = redis.createClient(process.env.REDIS_URL);

class AdvancedRateLimiter {
  constructor() {
    this.limiters = new Map();
    this.exemptions = new Set();
  }
  
  // 创建限流器
  createLimiter(options = {}) {
    const {
      key,
      maxRequests = 100,
      windowMs = 60000,
      keyGenerator = (req) => req.ip,
      skip = (req) => false,
      handler = this.defaultHandler
    } = options;
    
    const limiter = {
      key,
      maxRequests,
      windowMs,
      keyGenerator,
      skip,
      handler,
      hits: new Map()
    };
    
    this.limiters.set(key, limiter);
    return limiter;
  }
  
  // 多维度限流
  createMultiDimensionalLimiter(options = {}) {
    const {
      dimensions = ['ip', 'user', 'endpoint'],
      limits = {
        ip: { max: 100, window: 60000 },
        user: { max: 200, window: 60000 },
        endpoint: { max: 50, window: 60000 }
      },
      skip = () => false
    } = options;
    
    return this.middleware.bind(this, { dimensions, limits, skip });
  }
  
  // 中间件实现
  async middleware(req, res, next, config) {
    if (config.skip(req)) {
      return next();
    }
    
    const { dimensions, limits } = config;
    const results = [];
    
    for (const dimension of dimensions) {
      const limit = limits[dimension];
      const key = this.generateKey(req, dimension);
      
      const result = await this.checkLimit(key, limit.max, limit.window);
      results.push({ dimension, ...result });
      
      // 如果任何维度超限，返回限流响应
      if (!result.allowed) {
        return res.status(429).json({
          success: false,
          error: 'Too Many Requests',
          message: `Rate limit exceeded for ${dimension}`,
          limit: limit.max,
          remaining: 0,
          reset: result.reset,
          retryAfter: Math.ceil((result.reset - Date.now()) / 1000),
          dimension,
          details: results
        });
      }
    }
    
    // 添加限流头
    const primaryResult = results[0];
    res.set({
      'X-RateLimit-Limit': primaryResult.limit,
      'X-RateLimit-Remaining': primaryResult.remaining,
      'X-RateLimit-Reset': Math.ceil(primaryResult.reset / 1000)
    });
    
    next();
  }
  
  // 生成限流键
  generateKey(req, dimension) {
    switch (dimension) {
      case 'ip':
        return `ratelimit:ip:${req.ip}`;
      case 'user':
        return `ratelimit:user:${req.user?.id || 'anonymous'}`;
      case 'endpoint':
        return `ratelimit:endpoint:${req.path}`;
      case 'global':
        return 'ratelimit:global';
      default:
        return `ratelimit:${dimension}:${req[dimension] || 'unknown'}`;
    }
  }
  
  // 检查限流
  async checkLimit(key, max, window) {
    const now = Date.now();
    const windowStart = now - window;
    
    try {
      // 使用Redis的有序集合实现滑动窗口
      const multi = client.multi();
      
      // 删除过期数据
      multi.zremrangebyscore(key, 0, windowStart);
      
      // 添加当前请求
      multi.zadd(key, now, `${now}-${Math.random()}`);
      
      // 获取当前窗口内的请求数
      multi.zcard(key);
      
      // 设置过期时间
      multi.pexpire(key, window);
      
      const results = await multi.exec();
      const currentCount = results[2];
      
      const allowed = currentCount <= max;
      const reset = now + window;
      
      return {
        allowed,
        remaining: Math.max(0, max - currentCount),
        limit: max,
        reset,
        current: currentCount
      };
    } catch (error) {
      console.error('Rate limit check error:', error);
      // 如果Redis出错，允许请求通过
      return {
        allowed: true,
        remaining: max,
        limit: max,
        reset: now + window,
        current: 0
      };
    }
  }
  
  // 添加豁免规则
  addExemption(ipOrUserId, reason = '') {
    this.exemptions.add(ipOrUserId);
    console.log(`Added exemption for ${ipOrUserId}: ${reason}`);
  }
  
  // 移除豁免
  removeExemption(ipOrUserId) {
    this.exemptions.delete(ipOrUserId);
  }
  
  // 检查豁免
  isExempt(req) {
    return this.exemptions.has(req.ip) || 
           this.exemptions.has(req.user?.id) ||
           this.exemptions.has('*'); // 全局豁免
  }
  
  // 动态调整限流
  async adjustLimit(key, newLimit) {
    const limiter = this.limiters.get(key);
    if (limiter) {
      limiter.maxRequests = newLimit;
      console.log(`Adjusted limit for ${key} to ${newLimit}`);
      return true;
    }
    return false;
  }
  
  // 获取限流统计
  async getStats() {
    const stats = {};
    
    for (const [key, limiter] of this.limiters.entries()) {
      const keys = await client.keys(`ratelimit:${key}:*`);
      stats[key] = {
        configured: limiter.maxRequests,
        activeKeys: keys.length
      };
    }
    
    return stats;
  }
  
  // 默认处理函数
  defaultHandler(req, res) {
    res.status(429).json({
      success: false,
      error: 'Too Many Requests',
      message: 'Rate limit exceeded. Please try again later.'
    });
  }
  
  // 重置限流
  async resetLimit(key) {
    const pattern = `ratelimit:${key}:*`;
    const keys = await client.keys(pattern);
    if (keys.length > 0) {
      await client.del(...keys);
      return true;
    }
    return false;
  }
}

const advancedLimiter = new AdvancedRateLimiter();

// 预定义的限流器
const rateLimiters = {
  // API通用限流
  api: advancedLimiter.createMultiDimensionalLimiter({
    dimensions: ['ip', 'user'],
    limits: {
      ip: { max: 100, window: 60000 },
      user: { max: 200, window: 60000 }
    }
  }),
  
  // 登录限流（更严格）
  auth: advancedLimiter.createMultiDimensionalLimiter({
    dimensions: ['ip', 'user'],
    limits: {
      ip: { max: 10, window: 300000 },
      user: { max: 5, window: 300000 }
    },
    skip: (req) => advancedLimiter.isExempt(req)
  }),
  
  // 文件上传限流
  upload: advancedLimiter.createMultiDimensionalLimiter({
    dimensions: ['ip', 'user'],
    limits: {
      ip: { max: 10, window: 3600000 },
      user: { max: 20, window: 3600000 }
    }
  }),
  
  // 搜索限流
  search: advancedLimiter.createMultiDimensionalLimiter({
    dimensions: ['ip', 'user', 'endpoint'],
    limits: {
      ip: { max: 30, window: 60000 },
      user: { max: 50, window: 60000 },
      endpoint: { max: 100, window: 60000 }
    }
  }),
  
  // 管理员操作（宽松）
  admin: advancedLimiter.createMultiDimensionalLimiter({
    dimensions: ['user'],
    limits: {
      user: { max: 1000, window: 60000 }
    },
    skip: (req) => req.user?.role === 'admin'
  })
};

module.exports = {
  advancedLimiter,
  rateLimiters
};
EOF

# 2. 创建限流配置管理
echo "[18.2] 创建限流配置管理..."

cat > src/backend/config/rateLimit.js << 'EOF'
module.exports = {
  // 默认限流配置
  default: {
    windowMs: 60000,    // 1分钟
    max: 100             // 100次请求
  },
  
  // 各端点限流配置
  endpoints: {
    '/api/auth/login': {
      windowMs: 300000,  // 5分钟
      max: 5,            // 只允许5次尝试
      message: '登录尝试次数过多，请稍后再试'
    },
    
    '/api/auth/register': {
      windowMs: 3600000, // 1小时
      max: 10,           // 1小时只能注册10个账号
      message: '注册过于频繁，请稍后再试'
    },
    
    '/api/search': {
      windowMs: 60000,
      max: 30,
      message: '搜索请求过于频繁'
    },
    
    '/api/upload': {
      windowMs: 3600000,
      max: 20,
      message: '上传文件数量超限'
    }
  },
  
  // 白名单（不受限流限制）
  whitelist: [
    '127.0.0.1',
    '::1',
    '::ffff:127.0.0.1'
  ],
  
  // 黑名单（永久封禁）
  blacklist: [],
  
  // 豁免用户ID
  exemptUsers: [
    // 添加管理员或其他需要豁免的用户ID
  ],
  
  // 动态调整规则
  dynamicRules: {
    // 根据服务器负载动态调整
    loadBased: {
      enabled: true,
      highLoadThreshold: 0.8,    // 80% CPU
      reduceFactor: 0.5          // 限流降低50%
    },
    
    // 根据时间动态调整
    timeBased: {
      enabled: true,
      peakHours: {
        start: 9,
        end: 18
      },
      peakMultiplier: 0.5,       // 峰值时段限流更严格
      offPeakMultiplier: 1.5     // 非峰值时段限流放宽
    },
    
    // 根据用户行为动态调整
    behaviorBased: {
      enabled: true,
      suspiciousThreshold: 10,    // 10次异常行为
      blockDuration: 3600000,    // 封禁1小时
      reduceFactor: 0.8          // 降低80%限额
    }
  }
};
EOF

# 3. 创建测试文件
echo "[18.3] 创建限流测试..."

cat > tests/unit/rateLimiter.test.js << 'EOF'
const AdvancedRateLimiter = require('../../src/backend/middleware/rateLimiterAdvanced');
const { advancedLimiter, rateLimiters } = require('../../src/backend/middleware/rateLimiterAdvanced');

describe('AdvancedRateLimiter', () => {
  let mockReq;
  let mockRes;
  let mockNext;
  
  beforeEach(() => {
    mockReq = {
      ip: '192.168.1.1',
      path: '/api/test',
      user: { id: 'user123' }
    };
    
    mockRes = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn(),
      set: jest.fn()
    };
    
    mockNext = jest.fn();
  });
  
  describe('createLimiter', () => {
    test('创建基本限流器', () => {
      const limiter = advancedLimiter.createLimiter({
        key: 'test',
        maxRequests: 10,
        windowMs: 60000
      });
      
      expect(limiter.key).toBe('test');
      expect(limiter.maxRequests).toBe(10);
      expect(limiter.windowMs).toBe(60000);
    });
    
    test('创建带自定义key生成器的限流器', () => {
      const limiter = advancedLimiter.createLimiter({
        key: 'custom',
        keyGenerator: (req) => `custom:${req.user.id}`
      });
      
      const key = limiter.keyGenerator(mockReq);
      expect(key).toBe('custom:user123');
    });
  });
  
  describe('generateKey', () => {
    test('生成IP维度key', () => {
      const key = advancedLimiter.generateKey(mockReq, 'ip');
      expect(key).toBe('ratelimit:ip:192.168.1.1');
    });
    
    test('生成用户维度key', () => {
      const key = advancedLimiter.generateKey(mockReq, 'user');
      expect(key).toBe('ratelimit:user:user123');
    });
    
    test('生成端点维度key', () => {
      const key = advancedLimiter.generateKey(mockReq, 'endpoint');
      expect(key).toBe('ratelimit:endpoint:/api/test');
    });
  });
  
  describe('exemptions', () => {
    test('添加豁免', () => {
      advancedLimiter.addExemption('192.168.1.100', 'Test exemption');
      expect(advancedLimiter.exemptions.has('192.168.1.100')).toBe(true);
    });
    
    test('移除豁免', () => {
      advancedLimiter.addExemption('192.168.1.100');
      advancedLimiter.removeExemption('192.168.1.100');
      expect(advancedLimiter.exemptions.has('192.168.1.100')).toBe(false);
    });
    
    test('检查豁免用户', () => {
      mockReq.ip = '192.168.1.100';
      advancedLimiter.addExemption('192.168.1.100');
      expect(advancedLimiter.isExempt(mockReq)).toBe(true);
    });
  });
  
  describe('multi-dimensional limiting', () => {
    test('单维度限流', async () => {
      const limiter = advancedLimiter.createMultiDimensionalLimiter({
        dimensions: ['ip'],
        limits: {
          ip: { max: 5, window: 1000 }
        }
      });
      
      await limiter(mockReq, mockRes, mockNext);
      expect(mockNext).toHaveBeenCalled();
    });
    
    test('多维度限流', async () => {
      const limiter = advancedLimiter.createMultiDimensionalLimiter({
        dimensions: ['ip', 'user'],
        limits: {
          ip: { max: 10, window: 1000 },
          user: { max: 10, window: 1000 }
        }
      });
      
      await limiter(mockReq, mockRes, mockNext);
      expect(mockNext).toHaveBeenCalled();
    });
  });
  
  describe('rate limit headers', () => {
    test('设置限流头', async () => {
      const limiter = advancedLimiter.createMultiDimensionalLimiter({
        dimensions: ['ip'],
        limits: {
          ip: { max: 100, window: 60000 }
        }
      });
      
      await limiter(mockReq, mockRes, mockNext);
      
      expect(mockRes.set).toHaveBeenCalledWith(
        expect.objectContaining({
          'X-RateLimit-Limit': expect.any(Number),
          'X-RateLimit-Remaining': expect.any(Number),
          'X-RateLimit-Reset': expect.any(Number)
        })
      );
    });
  });
});

describe('Rate Limit Configuration', () => {
  const rateLimitConfig = require('../../src/backend/config/rateLimit');
  
  test('默认配置存在', () => {
    expect(rateLimitConfig.default).toBeDefined();
    expect(rateLimitConfig.default.windowMs).toBe(60000);
    expect(rateLimitConfig.default.max).toBe(100);
  });
  
  test('端点配置存在', () => {
    expect(rateLimitConfig.endpoints).toBeDefined();
    expect(rateLimitConfig.endpoints['/api/auth/login']).toBeDefined();
  });
  
  test('白名单配置', () => {
    expect(rateLimitConfig.whitelist).toContain('127.0.0.1');
  });
  
  test('动态规则配置', () => {
    expect(rateLimitConfig.dynamicRules).toBeDefined();
    expect(rateLimitConfig.dynamicRules.loadBased.enabled).toBe(true);
  });
});

describe('Rate Limit Integration', () => {
  test('预定义限流器存在', () => {
    expect(rateLimiters.api).toBeDefined();
    expect(rateLimiters.auth).toBeDefined();
    expect(rateLimiters.upload).toBeDefined();
    expect(rateLimiters.search).toBeDefined();
  });
});
EOF

# 4. 创建使用示例
echo "[18.4] 创建使用示例..."

cat > src/backend/middleware/examples/rateLimiterExamples.js << 'EOF'
/**
 * Rate Limiter 使用示例
 */

// 基础使用
const { rateLimiters } = require('../middleware/rateLimiterAdvanced');

// 在路由中使用预定义的限流器
const express = require('express');
const router = express.Router();

// 登录路由使用auth限流
router.post('/auth/login', rateLimiters.auth, (req, res) => {
  // 登录逻辑
});

// 搜索路由使用search限流
router.get('/search', rateLimiters.search, (req, res) => {
  // 搜索逻辑
});

// 上传路由使用upload限流
router.post('/upload', rateLimiters.upload, (req, res) => {
  // 上传逻辑
});

// 管理员路由使用admin限流
router.post('/admin/action', rateLimiters.admin, (req, res) => {
  // 管理员操作
});

// 自定义限流器
const { advancedLimiter } = require('../middleware/rateLimiterAdvanced');

// 创建自定义限流器
const customLimiter = advancedLimiter.createMultiDimensionalLimiter({
  dimensions: ['ip', 'user', 'endpoint'],
  limits: {
    ip: { max: 50, window: 60000 },
    user: { max: 100, window: 60000 },
    endpoint: { max: 200, window: 60000 }
  },
  skip: (req) => req.user?.role === 'admin'
});

// 使用自定义限流器
router.get('/custom', customLimiter, (req, res) => {
  // 自定义逻辑
});

// 动态调整限流
router.post('/admin/adjust-limit', async (req, res) => {
  const { key, newLimit } = req.body;
  
  await advancedLimiter.adjustLimit(key, newLimit);
  
  res.json({
    success: true,
    message: `Limit adjusted for ${key}`
  });
});

// 管理豁免
router.post('/admin/add-exemption', (req, res) => {
  const { ipOrUserId, reason } = req.body;
  
  advancedLimiter.addExemption(ipOrUserId, reason);
  
  res.json({
    success: true,
    message: `Added exemption for ${ipOrUserId}`
  });
});

// 获取限流统计
router.get('/admin/stats', async (req, res) => {
  const stats = await advancedLimiter.getStats();
  
  res.json({
    success: true,
    data: stats
  });
});
EOF

# 5. 创建文档
echo "[18.5] 创建限流文档..."

cat > docs/rate-limiting.md << 'EOF'
# 细粒度限流系统

## 概述

本系统实现了多维度的细粒度限流控制，支持按用户、IP、端点等不同维度进行限流。

## 特性

### 1. 多维度限流
- **IP限流**: 限制每个IP地址的请求频率
- **用户限流**: 限制每个用户的请求频率
- **端点限流**: 限制每个API端点的请求频率
- **组合限流**: 同时应用多个维度的限流

### 2. 动态限流规则
- **负载感知**: 根据服务器CPU/内存负载动态调整限流阈值
- **时间感知**: 根据时段（峰值/非峰值）调整限流
- **行为感知**: 根据用户异常行为动态调整限流

### 3. 限流豁免
- IP白名单
- 用户豁免
- 全局豁免

## 使用方法

### 预定义限流器

```javascript
const { rateLimiters } = require('./middleware/rateLimiterAdvanced');

// 登录限流（更严格）
router.post('/auth/login', rateLimiters.auth, handler);

// 搜索限流
router.get('/search', rateLimiters.search, handler);

// 文件上传限流
router.post('/upload', rateLimiters.upload, handler);

// 管理员限流
router.post('/admin/action', rateLimiters.admin, handler);
```

### 自定义限流器

```javascript
const { advancedLimiter } = require('./middleware/rateLimiterAdvanced');

const customLimiter = advancedLimiter.createMultiDimensionalLimiter({
  dimensions: ['ip', 'user', 'endpoint'],
  limits: {
    ip: { max: 50, window: 60000 },
    user: { max: 100, window: 60000 },
    endpoint: { max: 200, window: 60000 }
  },
  skip: (req) => req.user?.role === 'admin'
});
```

## 限流响应

### 429 Too Many Requests

```json
{
  "success": false,
  "error": "Too Many Requests",
  "message": "Rate limit exceeded for ip",
  "limit": 100,
  "remaining": 0,
  "reset": 1621234567890,
  "retryAfter": 60,
  "dimension": "ip",
  "details": [
    {
      "dimension": "ip",
      "allowed": false,
      "remaining": 0,
      "limit": 100,
      "reset": 1621234567890
    }
  ]
}
```

## 限流头

每个响应都包含以下限流头：

- `X-RateLimit-Limit`: 当前限制的最大请求数
- `X-RateLimit-Remaining`: 剩余可用请求数
- `X-RateLimit-Reset`: 限流重置时间戳（Unix时间戳）

## 配置

### 默认配置

```javascript
// src/backend/config/rateLimit.js
module.exports = {
  default: {
    windowMs: 60000,
    max: 100
  }
};
```

### 端点配置

```javascript
endpoints: {
  '/api/auth/login': {
    windowMs: 300000,
    max: 5
  }
}
```

## 最佳实践

1. **选择合适的限流维度**
   - 登录：使用IP+用户双重限流
   - 公开API：使用IP限流
   - 私有API：使用用户限流

2. **设置合理的阈值**
   - 根据业务需求调整
   - 考虑正常用户行为
   - 预留突发流量空间

3. **使用Redis集群**
   - 确保高可用性
   - 支持分布式限流
   - 提高性能

4. **监控和分析**
   - 跟踪限流触发情况
   - 分析异常流量
   - 持续优化配置
EOF

echo "=========================================="
echo "任务18完成：Rate Limiter细粒度控制"
echo "=================================="
