/**
 * @fileoverview 风险等级管理器
 * @description 风险等级定义、阈值配置管理、风险等级判定逻辑
 * @module captchax/internal/risk/scoring/risk_level
 */

'use strict';

class RiskLevel {
  constructor(config = {}) {
    this.config = {
      thresholds: config.thresholds || this.getDefaultThresholds(),
      labels: config.labels || this.getDefaultLabels(),
      actions: config.actions || this.getDefaultActions(),
      enableDynamicThresholds: config.enableDynamicThresholds !== false,
      ...config
    };

    this.dynamicThresholds = null;
    this.adjustmentHistory = [];
  }

  getDefaultThresholds() {
    return {
      low: { min: 0, max: 30 },
      medium: { min: 31, max: 60 },
      high: { min: 61, max: 80 },
      critical: { min: 81, max: 100 }
    };
  }

  getDefaultLabels() {
    return {
      low: '低风险',
      medium: '中风险',
      high: '高风险',
      critical: '严重风险'
    };
  }

  getDefaultActions() {
    return {
      low: 'allow',
      medium: 'verify',
      high: 'verify',
      critical: 'block'
    };
  }

  determineLevel(score) {
    const thresholds = this.dynamicThresholds || this.config.thresholds;

    for (const [level, range] of Object.entries(thresholds)) {
      if (score >= range.min && score <= range.max) {
        return {
          level,
          label: this.config.labels[level],
          action: this.config.actions[level],
          score,
          thresholds: range
        };
      }
    }

    if (score < thresholds.low.min) {
      return {
        level: 'low',
        label: this.config.labels.low,
        action: this.config.actions.low,
        score,
        thresholds: thresholds.low
      };
    }

    return {
      level: 'critical',
      label: this.config.labels.critical,
      action: this.config.actions.critical,
      score,
      thresholds: thresholds.critical
    };
  }

  getLevelFromScore(score) {
    return this.determineLevel(score).level;
  }

  getActionFromScore(score) {
    return this.determineLevel(score).action;
  }

  getLabelFromScore(score) {
    return this.determineLevel(score).label;
  }

  isHighRisk(score) {
    const level = this.getLevelFromScore(score);
    return level === 'high' || level === 'critical';
  }

  isLowRisk(score) {
    return this.getLevelFromScore(score) === 'low';
  }

  requiresVerification(score) {
    const action = this.getActionFromScore(score);
    return action === 'verify';
  }

  shouldBlock(score) {
    return this.getActionFromScore(score) === 'block';
  }

  shouldAllow(score) {
    return this.getActionFromScore(score) === 'allow';
  }

  adjustThresholds(context = {}) {
    if (!this.config.enableDynamicThresholds) {
      return this.config.thresholds;
    }

    const baseThresholds = { ...this.config.thresholds };
    
    const timeAdjustment = this.calculateTimeAdjustment(context.timeOfDay);
    const volumeAdjustment = this.calculateVolumeAdjustment(context.requestVolume);
    const threatLevelAdjustment = this.calculateThreatLevelAdjustment(context.currentThreatLevel);

    const totalAdjustment = timeAdjustment + volumeAdjustment + threatLevelAdjustment;
    
    for (const level of Object.keys(baseThresholds)) {
      const adjustment = totalAdjustment * this.getLevelSensitivity(level);
      baseThresholds[level].min = Math.max(0, baseThresholds[level].min + adjustment);
      baseThresholds[level].max = Math.min(100, baseThresholds[level].max + adjustment);
    }

    this.dynamicThresholds = baseThresholds;

    this.adjustmentHistory.push({
      timestamp: Date.now(),
      context,
      adjustments: {
        time: timeAdjustment,
        volume: volumeAdjustment,
        threatLevel: threatLevelAdjustment,
        total: totalAdjustment
      },
      resultingThresholds: { ...baseThresholds }
    });

    if (this.adjustmentHistory.length > 100) {
      this.adjustmentHistory.shift();
    }

    return baseThresholds;
  }

  calculateTimeAdjustment(timeOfDay) {
    if (!timeOfDay) {
      const hour = new Date().getHours();
      timeOfDay = hour;
    }

    const isNightTime = timeOfDay >= 0 && timeOfDay < 6;
    const isRushHour = (timeOfDay >= 7 && timeOfDay <= 9) || (timeOfDay >= 17 && timeOfDay <= 19);

    if (isNightTime) {
      return -5;
    }
    if (isRushHour) {
      return -3;
    }

    return 0;
  }

  calculateVolumeAdjustment(requestVolume) {
    if (!requestVolume) return 0;

    if (requestVolume > 10000) {
      return -10;
    }
    if (requestVolume > 5000) {
      return -5;
    }
    if (requestVolume > 1000) {
      return -2;
    }

    return 0;
  }

  calculateThreatLevelAdjustment(currentThreatLevel) {
    if (!currentThreatLevel) return 0;

    const threatMultipliers = {
      low: 0,
      medium: 5,
      high: 10,
      critical: 15
    };

    return threatMultipliers[currentThreatLevel] || 0;
  }

  getLevelSensitivity(level) {
    const sensitivities = {
      low: 1.0,
      medium: 0.8,
      high: 0.6,
      critical: 0.4
    };

    return sensitivities[level] || 0.5;
  }

  resetThresholds() {
    this.dynamicThresholds = null;
  }

  getCurrentThresholds() {
    return this.dynamicThresholds || this.config.thresholds;
  }

  getAdjustmentHistory(limit = 10) {
    return this.adjustmentHistory.slice(-limit);
  }

  getStatistics() {
    const stats = {
      currentThresholds: this.getCurrentThresholds(),
      isDynamic: this.dynamicThresholds !== null,
      adjustmentCount: this.adjustmentHistory.length,
      recentAdjustments: this.adjustmentHistory.slice(-5).map(a => ({
        timestamp: a.timestamp,
        totalAdjustment: a.adjustments.total
      }))
    };

    return stats;
  }

  setThresholds(thresholds) {
    this.config.thresholds = { ...thresholds };
    this.dynamicThresholds = null;
  }

  getThresholdForLevel(level) {
    const thresholds = this.getCurrentThresholds();
    return thresholds[level] || null;
  }

  isScoreInRange(score, level) {
    const threshold = this.getThresholdForLevel(level);
    if (!threshold) return false;

    return score >= threshold.min && score <= threshold.max;
  }

  getNextLevel(level) {
    const levels = ['low', 'medium', 'high', 'critical'];
    const currentIndex = levels.indexOf(level);

    if (currentIndex < levels.length - 1) {
      return levels[currentIndex + 1];
    }

    return level;
  }

  getPreviousLevel(level) {
    const levels = ['low', 'medium', 'high', 'critical'];
    const currentIndex = levels.indexOf(level);

    if (currentIndex > 0) {
      return levels[currentIndex - 1];
    }

    return level;
  }

  compareLevels(level1, level2) {
    const levels = ['low', 'medium', 'high', 'critical'];
    const index1 = levels.indexOf(level1);
    const index2 = levels.indexOf(level2);

    if (index1 > index2) return 1;
    if (index1 < index2) return -1;
    return 0;
  }

  isLevelHigher(level1, level2) {
    return this.compareLevels(level1, level2) > 0;
  }

  isLevelLower(level1, level2) {
    return this.compareLevels(level1, level2) < 0;
  }

  getAllLevels() {
    return ['low', 'medium', 'high', 'critical'];
  }

  getLevelInfo(level) {
    return {
      level,
      label: this.config.labels[level],
      action: this.config.actions[level],
      thresholds: this.getThresholdForLevel(level)
    };
  }
}

module.exports = RiskLevel;
