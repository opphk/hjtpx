const productionLogger = require('./productionLogger');

module.exports = {
  Logger: productionLogger.Logger,
  createChildLogger: productionLogger.createChildLogger,
  logRequest: productionLogger.logRequest,
  logError: productionLogger.logError,
  logSecurity: productionLogger.logSecurity,
  logPerformance: productionLogger.logPerformance,
  logInfo: productionLogger.logInfo,
  logWarn: productionLogger.logWarn,
  logDebug: productionLogger.logDebug,
  logHttp: productionLogger.logHttp,
  logPerformanceMetric: productionLogger.logPerformanceMetric,
  generateRequestId: productionLogger.generateRequestId,
  sensitiveDataMasker: productionLogger.sensitiveDataMasker
};
