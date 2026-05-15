/**
 * @fileoverview 行为特征分析引擎模块导出
 * @module captchax/internal/risk/analysis
 */

'use strict';

const FeatureExtractor = require('./feature_extractor');
const AnomalyDetector = require('./anomaly_detector');

module.exports = {
  FeatureExtractor,
  AnomalyDetector
};
