const ApiUsageStats = require('../services/apiUsageStats');

const statsService = new ApiUsageStats();

const apiStatsMiddleware = (req, res, next) => {
  const startTime = Date.now();
  const { method, path } = req;

  const originalEnd = res.end;
  res.end = function (...args) {
    const responseTime = Date.now() - startTime;
    const statusCode = res.statusCode;
    
    if (!path.startsWith('/api-docs') && !path.startsWith('/docs') && !path.startsWith('/static')) {
      statsService.recordRequest(method, path, statusCode, responseTime);
    }
    
    originalEnd.apply(res, args);
  };

  next();
};

const getStatsService = () => statsService;

module.exports = {
  apiStatsMiddleware,
  getStatsService
};
