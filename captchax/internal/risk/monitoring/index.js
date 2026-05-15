/**
 * @fileoverview 风控日志与监控模块导出
 * @module captchax/internal/risk/monitoring
 */

'use strict';

const RiskLogger = require('./risk_logger');
const RiskMonitor = require('./risk_monitor');

module.exports = {
  RiskLogger,
  RiskMonitor
};
