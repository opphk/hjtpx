/**
 * @fileoverview 风控数据采集模块导出
 * @module captchax/internal/risk/collector
 */

'use strict';

const BehaviorCollector = require('./behavior_collector');
const DeviceFingerprint = require('./device_fingerprint');
const CollectorManager = require('./collector_manager');

module.exports = {
  BehaviorCollector,
  DeviceFingerprint,
  CollectorManager
};
