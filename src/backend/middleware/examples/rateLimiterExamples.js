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
