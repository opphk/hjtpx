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
