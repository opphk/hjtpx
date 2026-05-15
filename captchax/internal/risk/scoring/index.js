/**
 * @fileoverview 风险评分模块导出
 * @module captchax/internal/risk/scoring
 */

'use strict';

const RiskScorer = require('./risk_scorer');
const RiskLevel = require('./risk_level');

module.exports = {
  RiskScorer,
  RiskLevel
};
