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
