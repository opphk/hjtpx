const rateLimit = require('express-rate-limit');
const RedisStore = require('rate-limit-redis').default;
const redisClient = require('../../../config/redis/client');

const redisStore = new RedisStore({
  sendCommand: (...args) => redisClient.sendCommand(args)
});

const createRateLimiter = (options = {}) => {
  const defaultOptions = {
    windowMs: 15 * 60 * 1000,
    max: 100,
    message: {
      success: false,
      error: {
        code: 'TOO_MANY_REQUESTS',
        message: 'Too many requests, please try again later.'
      }
    },
    standardHeaders: true,
    legacyHeaders: false,
    handler: (req, res, next, options) => {
      return res.status(429).json({
        success: false,
        error: {
          code: 'TOO_MANY_REQUESTS',
          message: 'Too many requests, please try again later.',
          retryAfter: Math.ceil(options.windowMs / 1000)
        }
      });
    }
  };

  return rateLimit({ ...defaultOptions, ...options });
};

const ipRateLimiter = createRateLimiter({
  windowMs: 15 * 60 * 1000,
  max: 100,
  keyGenerator: (req) => {
    return req.ip || req.connection.remoteAddress;
  }
});

const userRateLimiter = createRateLimiter({
  windowMs: 15 * 60 * 1000,
  max: 200,
  keyGenerator: (req) => {
    return req.user?.id || req.ip || req.connection.remoteAddress;
  }
});

const strictRateLimiter = createRateLimiter({
  windowMs: 60 * 1000,
  max: 10,
  message: {
    success: false,
    error: {
      code: 'TOO_MANY_REQUESTS',
      message: 'Too many requests from this IP, please try again after a minute.'
    }
  }
});

const authRateLimiter = createRateLimiter({
  windowMs: 15 * 60 * 1000,
  max: 5,
  skipSuccessfulRequests: true,
  keyGenerator: (req) => {
    return `auth:${req.ip || req.connection.remoteAddress}`;
  }
});

module.exports = {
  createRateLimiter,
  ipRateLimiter,
  userRateLimiter,
  strictRateLimiter,
  authRateLimiter,
  redisStore
};
