/**
 * @fileoverview AI增强型风控引擎 - 集成入口
 * @description 统一管理所有风控模块，提供完整的风险评估服务
 * @module captchax/internal/risk
 */

'use strict';

const { BehaviorCollector, DeviceFingerprint, CollectorManager } = require('./collector');
const { FeatureExtractor, AnomalyDetector } = require('./analysis');
const { RiskScorer, RiskLevel } = require('./scoring');
const { DifficultySelector } = require('./adaptive');
const { RiskLogger, RiskMonitor } = require('./monitoring');

class RiskEngine {
  constructor(config = {}) {
    this.config = {
      enableCollection: config.enableCollection !== false,
      enableAnalysis: config.enableAnalysis !== false,
      enableScoring: config.enableScoring !== false,
      enableAdaptive: config.enableAdaptive !== false,
      enableLogging: config.enableLogging !== false,
      enableMonitoring: config.enableMonitoring !== false,
      autoInitialize: config.autoInitialize !== false,
      ...config
    };

    this.collector = null;
    this.deviceFingerprint = null;
    this.collectorManager = null;
    this.featureExtractor = null;
    this.anomalyDetector = null;
    this.riskScorer = null;
    this.riskLevel = null;
    this.difficultySelector = null;
    this.riskLogger = null;
    this.riskMonitor = null;

    this.userProfiles = new Map();
    this.sessionData = new Map();

    if (this.config.autoInitialize) {
      this.initialize();
    }
  }

  initialize(customConfig = {}) {
    const finalConfig = { ...this.config, ...customConfig };

    if (finalConfig.enableCollection) {
      this.initializeCollector(finalConfig);
    }

    if (finalConfig.enableAnalysis) {
      this.initializeAnalysis(finalConfig);
    }

    if (finalConfig.enableScoring) {
      this.initializeScoring(finalConfig);
    }

    if (finalConfig.enableAdaptive) {
      this.initializeAdaptive(finalConfig);
    }

    if (finalConfig.enableLogging) {
      this.initializeLogging(finalConfig);
    }

    if (finalConfig.enableMonitoring) {
      this.initializeMonitoring(finalConfig);
    }

    return this;
  }

  initializeCollector(config) {
    this.collector = new BehaviorCollector(config.behaviorCollector);
    this.deviceFingerprint = new DeviceFingerprint(config.deviceFingerprint);
    this.collectorManager = new CollectorManager({
      behaviorCollector: config.behaviorCollector,
      deviceFingerprint: config.deviceFingerprint,
      ...config.collectorManager
    });
  }

  initializeAnalysis(config) {
    this.featureExtractor = new FeatureExtractor(config.featureExtractor);
    this.anomalyDetector = new AnomalyDetector(config.anomalyDetector);
  }

  initializeScoring(config) {
    this.riskScorer = new RiskScorer(config.riskScorer);
    this.riskLevel = new RiskLevel(config.riskLevel);
  }

  initializeAdaptive(config) {
    this.difficultySelector = new DifficultySelector(config.difficultySelector);
  }

  initializeLogging(config) {
    this.riskLogger = new RiskLogger(config.riskLogger);
  }

  initializeMonitoring(config) {
    this.riskMonitor = new RiskMonitor(config.riskMonitor);
    this.riskMonitor.startMonitoring();
  }

  async collectBehavior(ctx = {}) {
    if (!this.collectorManager) {
      throw new Error('Collector not initialized. Call initialize() first.');
    }

    try {
      const sessionId = ctx.sessionId || this.collectorManager.initialize(null, ctx.userId);
      
      this.sessionData.set(sessionId, {
        userId: ctx.userId,
        startTime: Date.now(),
        ip: ctx.ip,
        userAgent: ctx.userAgent
      });

      const data = await this.collectorManager.getFullData();

      if (this.riskMonitor) {
        this.riskMonitor.recordRequest({
          userId: ctx.userId,
          sessionId,
          riskLevel: null,
          captchaType: ctx.captchaType
        });
      }

      return {
        success: true,
        sessionId,
        data,
        timestamp: Date.now()
      };
    } catch (error) {
      console.error('Error collecting behavior:', error);
      return {
        success: false,
        error: error.message,
        timestamp: Date.now()
      };
    }
  }

  async analyzeFeatures(data) {
    if (!this.featureExtractor || !this.anomalyDetector) {
      throw new Error('Analysis modules not initialized. Call initialize() first.');
    }

    try {
      const features = this.featureExtractor.extract(data);

      const anomalyResult = this.anomalyDetector.detect(features);

      return {
        success: true,
        features,
        anomaly: anomalyResult,
        timestamp: Date.now()
      };
    } catch (error) {
      console.error('Error analyzing features:', error);
      return {
        success: false,
        error: error.message,
        timestamp: Date.now()
      };
    }
  }

  async calculateRiskScore(features, context = {}) {
    if (!this.riskScorer || !this.riskLevel) {
      throw new Error('Scoring modules not initialized. Call initialize() first.');
    }

    try {
      const scoreResult = this.riskScorer.calculateScore(features, context);
      const levelResult = this.riskLevel.determineLevel(scoreResult.totalScore);

      const riskResult = {
        success: true,
        score: scoreResult.totalScore,
        normalizedScore: scoreResult.normalizedScore,
        riskLevel: levelResult.level,
        riskLabel: levelResult.label,
        action: levelResult.action,
        details: {
          scores: scoreResult.scores,
          details: scoreResult.details,
          factors: this.aggregateRiskFactors(scoreResult, context)
        },
        timestamp: Date.now()
      };

      if (context.userId) {
        this.riskScorer.updateHistory(context.userId, features, scoreResult.totalScore);

        this.updateUserProfile(context.userId, {
          lastRiskScore: scoreResult.totalScore,
          lastRiskLevel: levelResult.level,
          lastActivity: Date.now()
        });
      }

      if (this.riskLogger) {
        this.riskLogger.logRiskEvent({
          userId: context.userId,
          sessionId: context.sessionId,
          riskScore: scoreResult.totalScore,
          riskLevel: levelResult.level,
          action: levelResult.action,
          factors: riskResult.details.factors,
          features,
          fingerprint: context.fingerprint,
          ip: context.ip,
          userAgent: context.userAgent
        });
      }

      if (this.riskMonitor) {
        this.riskMonitor.recordRequest({
          userId: context.userId,
          sessionId: context.sessionId,
          riskScore: scoreResult.totalScore,
          riskLevel: levelResult.level,
          action: levelResult.action
        });
      }

      return riskResult;
    } catch (error) {
      console.error('Error calculating risk score:', error);
      return {
        success: false,
        error: error.message,
        timestamp: Date.now()
      };
    }
  }

  aggregateRiskFactors(scoreResult, context) {
    const factors = [];

    for (const detail of scoreResult.details) {
      if (detail.score > 60) {
        factors.push({
          category: detail.dimension,
          score: detail.score,
          weight: detail.weight,
          description: this.getFactorDescription(detail.dimension, detail.score)
        });
      }
    }

    if (context.anomalyScore && context.anomalyScore > 50) {
      factors.push({
        category: 'anomaly',
        score: context.anomalyScore,
        weight: 1.0,
        description: '检测到异常行为模式'
      });
    }

    return factors;
  }

  getFactorDescription(category, score) {
    const descriptions = {
      velocity: score > 80 ? '速度异常：过快或过慢' : '速度略异常',
      smoothness: score > 80 ? '轨迹异常平滑：疑似机器行为' : '轨迹略平滑',
      jitter: score > 80 ? '抖动异常低：缺乏人类特征' : '抖动略低',
      humanLikeness: score > 80 ? '人类特征明显缺失' : '人类特征略低',
      historical: score > 80 ? '与历史行为差异大' : '与历史行为略有差异',
      device: score > 80 ? '设备指纹异常' : '设备指纹略有变化',
      behavioral: score > 80 ? '行为模式异常' : '行为模式略有异常'
    };

    return descriptions[category] || '存在风险特征';
  }

  async selectDifficulty(riskScore, context = {}) {
    if (!this.difficultySelector) {
      throw new Error('Adaptive module not initialized. Call initialize() first.');
    }

    try {
      const selection = this.difficultySelector.selectDifficulty(riskScore, {
        userId: context.userId,
        userHistory: context.userHistory
      });

      return {
        success: true,
        difficulty: selection.difficulty,
        captchaType: selection.captchaType.id,
        captchaName: selection.captchaType.name,
        config: selection.config,
        confidence: selection.confidence,
        timestamp: Date.now()
      };
    } catch (error) {
      console.error('Error selecting difficulty:', error);
      return {
        success: false,
        error: error.message,
        timestamp: Date.now()
      };
    }
  }

  async logRiskEvent(event) {
    if (!this.riskLogger) {
      throw new Error('Logger not initialized. Call initialize() first.');
    }

    try {
      return this.riskLogger.logRiskEvent(event);
    } catch (error) {
      console.error('Error logging risk event:', error);
      return null;
    }
  }

  async getRiskReport(userId) {
    if (!userId) {
      return null;
    }

    const profile = this.userProfiles.get(userId) || {};
    const history = this.riskScorer ? this.riskScorer.getUserHistory(userId) : [];
    const alerts = this.riskMonitor ? this.riskMonitor.getAlerts({ since: Date.now() - 86400000 }) : [];

    const recentAlerts = alerts.filter(a => a.userId === userId);

    const avgScore = history.length > 0
      ? history.reduce((sum, h) => sum + h.score, 0) / history.length
      : 0;

    const riskTrend = this.calculateUserRiskTrend(history);

    return {
      userId,
      profile: {
        lastActivity: profile.lastActivity,
        totalAttempts: history.length,
        averageRiskScore: avgScore,
        currentRiskLevel: profile.lastRiskLevel || 'unknown'
      },
      history: history.slice(-20),
      riskTrend,
      recentAlerts,
      recommendations: this.generateRecommendations(avgScore, riskTrend, recentAlerts),
      generatedAt: Date.now()
    };
  }

  calculateUserRiskTrend(history) {
    if (history.length < 2) {
      return { direction: 'stable', change: 0 };
    }

    const recent = history.slice(-5);
    const avgRecent = recent.reduce((sum, h) => sum + h.score, 0) / recent.length;

    const previous = history.slice(-10, -5);
    const avgPrevious = previous.length > 0
      ? previous.reduce((sum, h) => sum + h.score, 0) / previous.length
      : avgRecent;

    const change = avgRecent - avgPrevious;

    let direction = 'stable';
    if (change > 10) direction = 'increasing';
    else if (change < -10) direction = 'decreasing';

    return { direction, change, avgRecent, avgPrevious };
  }

  generateRecommendations(avgScore, trend, alerts) {
    const recommendations = [];

    if (avgScore > 80) {
      recommendations.push({
        priority: 'high',
        action: '加强监控',
        reason: '用户平均风险评分持续较高'
      });
    }

    if (trend.direction === 'increasing') {
      recommendations.push({
        priority: 'medium',
        action: '关注趋势',
        reason: '用户风险评分呈上升趋势'
      });
    }

    if (alerts.length > 5) {
      recommendations.push({
        priority: 'high',
        action: '审查用户',
        reason: '用户近期产生多次告警'
      });
    }

    return recommendations;
  }

  updateUserProfile(userId, data) {
    const profile = this.userProfiles.get(userId) || {};
    this.userProfiles.set(userId, { ...profile, ...data });
  }

  async process(ctx = {}) {
    const result = {
      sessionId: null,
      features: null,
      riskScore: null,
      difficulty: null,
      action: 'verify',
      timestamp: Date.now()
    };

    try {
      const collectResult = await this.collectBehavior(ctx);
      if (!collectResult.success) {
        throw new Error(collectResult.error);
      }

      result.sessionId = collectResult.sessionId;

      const analyzeResult = await this.analyzeFeatures(collectResult.data);
      if (!analyzeResult.success) {
        throw new Error(analyzeResult.error);
      }

      result.features = analyzeResult.features;

      const riskResult = await this.calculateRiskScore(analyzeResult.features, {
        userId: ctx.userId,
        sessionId: result.sessionId,
        fingerprint: ctx.fingerprint,
        ip: ctx.ip,
        userAgent: ctx.userAgent,
        anomalyScore: analyzeResult.anomaly?.anomalyScore
      });

      result.riskScore = riskResult.score;
      result.action = riskResult.action;

      const difficultyResult = await this.selectDifficulty(riskResult.score, {
        userId: ctx.userId
      });

      result.difficulty = difficultyResult;

      result.success = true;

      return result;
    } catch (error) {
      console.error('Error in risk engine process:', error);
      return {
        success: false,
        error: error.message,
        timestamp: Date.now()
      };
    }
  }

  setBaseline(features) {
    if (this.featureExtractor) {
      this.featureExtractor.setBaseline(features);
    }
    if (this.anomalyDetector) {
      this.anomalyDetector.setBaseline(features);
    }
  }

  addTrainingData(features, label) {
    if (label === 'normal') {
      this.anomalyDetector.addSample(features);
    }
  }

  getStatus() {
    return {
      initialized: {
        collector: this.collector !== null,
        analysis: this.featureExtractor !== null,
        scoring: this.riskScorer !== null,
        adaptive: this.difficultySelector !== null,
        logging: this.riskLogger !== null,
        monitoring: this.riskMonitor !== null
      },
      stats: {
        userProfiles: this.userProfiles.size,
        sessionData: this.sessionData.size,
        anomalyDetector: this.anomalyDetector?.getStats() || null,
        riskScorer: this.riskScorer?.getStats() || null,
        riskMonitor: this.riskMonitor?.getHealthMetrics() || null
      }
    };
  }

  reset() {
    if (this.riskMonitor) {
      this.riskMonitor.stopMonitoring();
    }
    if (this.riskLogger) {
      this.riskLogger.destroy();
    }

    this.userProfiles.clear();
    this.sessionData.clear();

    this.initialize(this.config);
  }

  destroy() {
    if (this.riskMonitor) {
      this.riskMonitor.destroy();
    }
    if (this.riskLogger) {
      this.riskLogger.destroy();
    }
    if (this.collectorManager) {
      this.collectorManager.destroy();
    }

    this.userProfiles.clear();
    this.sessionData.clear();
  }
}

module.exports = RiskEngine;
