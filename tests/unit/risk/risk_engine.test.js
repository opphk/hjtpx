/**
 * @fileoverview AI增强型风控引擎测试
 * @description 验证核心功能的单元测试
 * @module captchax/internal/risk/test
 */

'use strict';

const assert = require('assert');

const { BehaviorCollector, DeviceFingerprint, CollectorManager } = require('../../../captchax/internal/risk/collector');
const { FeatureExtractor, AnomalyDetector } = require('../../../captchax/internal/risk/analysis');
const { RiskScorer, RiskLevel } = require('../../../captchax/internal/risk/scoring');
const { DifficultySelector } = require('../../../captchax/internal/risk/adaptive');
const { RiskLogger, RiskMonitor } = require('../../../captchax/internal/risk/monitoring');
const RiskEngine = require('../../../captchax/internal/risk/index');

describe('AI增强型风控引擎测试', function() {
  describe('BehaviorCollector', function() {
    it('应该正确初始化', function() {
      const collector = new BehaviorCollector();
      assert.ok(collector);
      assert.deepStrictEqual(collector.tracks, []);
    });

    it('应该开始和停止采集', function() {
      const collector = new BehaviorCollector();
      collector.start('test-session');
      assert.strictEqual(collector.sessionId, 'test-session');
      assert.ok(collector.isCollecting);

      collector.stop();
      assert.ok(!collector.isCollecting);
    });

    it('应该采集鼠标轨迹', function() {
      const collector = new BehaviorCollector();
      collector.start('test-session');

      collector.trackMouseMove(100, 100, 1000);
      collector.trackMouseMove(110, 105, 1050);
      collector.trackMouseMove(120, 110, 1100);

      assert.strictEqual(collector.tracks.length, 3);
      assert.strictEqual(collector.tracks[0].x, 100);
    });

    it('应该采集点击事件', function() {
      const collector = new BehaviorCollector();
      collector.start('test-session');

      collector.trackClick(1000);
      collector.trackClick(1100);

      assert.strictEqual(collector.clickTimes.length, 2);
    });

    it('应该计算统计数据', function() {
      const collector = new BehaviorCollector();
      collector.start('test-session');

      for (let i = 0; i < 10; i++) {
        collector.trackMouseMove(100 + i * 10, 100 + i * 5, 1000 + i * 100);
      }

      const data = collector.getData();
      assert.ok(data.statistics);
      assert.ok(data.statistics.totalDistance > 0);
    });
  });

  describe('DeviceFingerprint', function() {
    it('应该正确初始化', function() {
      const fingerprint = new DeviceFingerprint();
      assert.ok(fingerprint);
    });

    it('应该生成哈希值', function() {
      const fingerprint = new DeviceFingerprint();
      const hash = fingerprint.hashString('test-string');
      assert.ok(hash);
      assert.strictEqual(typeof hash, 'string');
    });

    it('应该生成唯一ID', function() {
      const id1 = DeviceFingerprint.generateId();
      const id2 = DeviceFingerprint.generateId();
      assert.ok(id1);
      assert.ok(id2);
      assert.notStrictEqual(id1, id2);
    });
  });

  describe('CollectorManager', function() {
    it('应该正确初始化', function() {
      const manager = new CollectorManager({ autoCollect: false });
      assert.ok(manager);
      assert.ok(manager.behaviorCollector);
      assert.ok(manager.deviceFingerprint);
    });

    it('应该创建会话', function() {
      const manager = new CollectorManager({ autoCollect: false });
      const sessionId = manager.initialize('test-session');
      assert.ok(sessionId);
    });

    it('应该追踪鼠标移动', function() {
      const manager = new CollectorManager({ autoCollect: false });
      manager.initialize('test-session');
      manager.trackMouseMove(100, 100);
      manager.trackMouseMove(110, 110);
      assert.ok(manager.behaviorCollector.tracks.length >= 2);
    });
  });

  describe('FeatureExtractor', function() {
    it('应该正确初始化', function() {
      const extractor = new FeatureExtractor();
      assert.ok(extractor);
    });

    it('应该提取默认特征', function() {
      const extractor = new FeatureExtractor();
      const features = extractor.extract(null);
      assert.ok(features);
      assert.ok(features.basic);
      assert.ok(features.temporal);
      assert.ok(features.statistical);
    });

    it('应该提取鼠标轨迹特征', function() {
      const extractor = new FeatureExtractor();
      const behaviorData = {
        mouseTracks: [
          { x: 0, y: 0, timestamp: 0 },
          { x: 100, y: 100, timestamp: 1000 },
          { x: 200, y: 200, timestamp: 2000 }
        ],
        clickTimes: [],
        keyTimes: [],
        slideDuration: 2000
      };

      const features = extractor.extract(behaviorData);
      assert.ok(features.basic.trackPointCount > 0);
      assert.ok(features.basic.totalDistance > 0);
    });

    it('应该计算欧几里得距离', function() {
      const extractor = new FeatureExtractor();
      const features1 = { 'basic.averageSpeed': 100, 'basic.maxSpeed': 200 };
      const features2 = { 'basic.averageSpeed': 110, 'basic.maxSpeed': 220 };

      const distance = extractor.calculateEuclideanDistance(features1, features2);
      assert.ok(distance >= 0);
    });

    it('应该计算余弦相似度', function() {
      const extractor = new FeatureExtractor();
      const features1 = { 'basic.averageSpeed': 100, 'basic.maxSpeed': 200 };
      const features2 = { 'basic.averageSpeed': 100, 'basic.maxSpeed': 200 };

      const similarity = extractor.calculateCosineSimilarity(features1, features2);
      assert.strictEqual(similarity, 1);
    });
  });

  describe('AnomalyDetector', function() {
    it('应该正确初始化', function() {
      const detector = new AnomalyDetector();
      assert.ok(detector);
    });

    it('应该检测异常', function() {
      const detector = new AnomalyDetector();

      const baselineData = [];
      for (let i = 0; i < 20; i++) {
        baselineData.push({
          combined: {
            'basic.averageSpeed': 100 + Math.random() * 50,
            'basic.maxSpeed': 200 + Math.random() * 100,
            'statistical.smoothness': 0.7 + Math.random() * 0.2
          }
        });
      }

      detector.setBaseline(baselineData);

      const features = {
        combined: {
          'basic.averageSpeed': 500,
          'basic.maxSpeed': 1000,
          'statistical.smoothness': 0.99,
          'basic.speedVariance': 5,
          'behavioral.humanLikeness': 0.1,
          'temporal.straightness': 0.95,
          'behavioral.dwellPointCount': 0,
          'keyboard.errorRate': 0
        }
      };

      const result = detector.detect(features);
      assert.ok(result);
      assert.strictEqual(typeof result.isAnomaly, 'boolean');
      assert.strictEqual(typeof result.anomalyScore, 'number');
    });

    it('应该添加训练样本', function() {
      const detector = new AnomalyDetector();
      const features = {
        combined: {
          'basic.averageSpeed': 100,
          'basic.maxSpeed': 200
        }
      };

      detector.addSample(features);
      assert.strictEqual(detector.baselineData.length, 1);
    });
  });

  describe('RiskScorer', function() {
    it('应该正确初始化', function() {
      const scorer = new RiskScorer();
      assert.ok(scorer);
      assert.ok(scorer.config);
    });

    it('应该计算风险评分', function() {
      const scorer = new RiskScorer();

      const features = {
        combined: {
          'basic.averageSpeed': 100,
          'basic.maxSpeed': 500,
          'basic.speedVariance': 30,
          'statistical.smoothness': 0.5,
          'statistical.jitter': 5,
          'behavioral.humanLikeness': 0.6,
          'temporal.straightness': 0.6,
          'behavioral.dwellPointCount': 3,
          'behavioral.clickCount': 2,
          'keyboard.errorRate': 0.1,
          'behavioral.avgDwellDuration': 150
        }
      };

      const result = scorer.calculateScore(features, {});
      assert.ok(result);
      assert.ok(typeof result.totalScore === 'number');
      assert.ok(result.totalScore >= 0 && result.totalScore <= 100);
    });

    it('应该更新用户历史', function() {
      const scorer = new RiskScorer();
      const features = {
        combined: {
          'basic.averageSpeed': 100,
          'basic.maxSpeed': 200,
          'statistical.smoothness': 0.8,
          'statistical.jitter': 5,
          'behavioral.humanLikeness': 0.6,
          'temporal.straightness': 0.6
        }
      };

      scorer.updateHistory('user-1', features, 30);
      const history = scorer.getUserHistory('user-1');
      assert.strictEqual(history.length, 1);
    });
  });

  describe('RiskLevel', function() {
    it('应该正确初始化', function() {
      const level = new RiskLevel();
      assert.ok(level);
    });

    it('应该判定风险等级', function() {
      const level = new RiskLevel();

      assert.strictEqual(level.determineLevel(15).level, 'low');
      assert.strictEqual(level.determineLevel(45).level, 'medium');
      assert.strictEqual(level.determineLevel(70).level, 'high');
      assert.strictEqual(level.determineLevel(90).level, 'critical');
    });

    it('应该返回正确的动作', function() {
      const level = new RiskLevel();

      assert.strictEqual(level.getActionFromScore(15), 'allow');
      assert.strictEqual(level.getActionFromScore(45), 'verify');
      assert.strictEqual(level.getActionFromScore(90), 'block');
    });

    it('应该调整阈值', function() {
      const level = new RiskLevel();
      const adjusted = level.adjustThresholds({ timeOfDay: 3 });

      assert.ok(adjusted);
      assert.ok(level.dynamicThresholds !== null);
    });

    it('应该重置阈值', function() {
      const level = new RiskLevel();
      level.adjustThresholds({ timeOfDay: 3 });
      level.resetThresholds();
      assert.strictEqual(level.dynamicThresholds, null);
    });
  });

  describe('DifficultySelector', function() {
    it('应该正确初始化', function() {
      const selector = new DifficultySelector();
      assert.ok(selector);
      assert.ok(selector.difficultyLevels);
      assert.ok(selector.captchaTypes);
    });

    it('应该选择难度', function() {
      const selector = new DifficultySelector();

      const result = selector.selectDifficulty(30, {});
      assert.ok(result);
      assert.ok(result.difficulty);
      assert.ok(result.captchaType);
    });

    it('应该为高风险选择更高难度', function() {
      const selector = new DifficultySelector();

      const lowRisk = selector.selectDifficulty(10, {});
      const highRisk = selector.selectDifficulty(80, {});

      const lowLevel = selector.difficultyLevels[lowRisk.difficulty].level;
      const highLevel = selector.difficultyLevels[highRisk.difficulty].level;

      assert.ok(highLevel >= lowLevel);
    });

    it('应该记录选择', function() {
      const selector = new DifficultySelector();
      selector.selectDifficulty(30, {});

      assert.strictEqual(selector.history.length, 1);
    });

    it('应该返回统计信息', function() {
      const selector = new DifficultySelector();
      selector.selectDifficulty(30, {});
      selector.selectDifficulty(50, {});

      const stats = selector.getStatistics();
      assert.ok(stats);
      assert.strictEqual(stats.totalSelections, 2);
    });
  });

  describe('RiskLogger', function() {
    it('应该正确初始化', function() {
      const logger = new RiskLogger();
      assert.ok(logger);
    });

    it('应该记录风险事件', function() {
      const logger = new RiskLogger();

      const event = logger.logRiskEvent({
        userId: 'user-1',
        riskScore: 50,
        riskLevel: 'medium',
        action: 'verify'
      });

      assert.ok(event);
      assert.strictEqual(logger.logs.length, 1);
    });

    it('应该记录验证尝试', function() {
      const logger = new RiskLogger();

      logger.logVerificationAttempt({
        userId: 'user-1',
        captchaType: 'slider',
        success: true,
        timeSpent: 3000,
        riskScore: 30
      });

      assert.strictEqual(logger.logs.length, 1);
    });

    it('应该记录异常事件', function() {
      const logger = new RiskLogger();

      logger.logAnomaly({
        userId: 'user-1',
        anomalyScore: 80,
        method: 'zScore'
      });

      assert.strictEqual(logger.logs.length, 1);
    });

    it('应该返回统计信息', function() {
      const logger = new RiskLogger();
      logger.logRiskEvent({
        userId: 'user-1',
        riskScore: 50,
        riskLevel: 'medium',
        action: 'verify'
      });

      const stats = logger.getStatistics();
      assert.ok(stats);
      assert.strictEqual(stats.totalLogs, 1);
    });

    it('应该清除日志', function() {
      const logger = new RiskLogger();
      logger.logRiskEvent({
        userId: 'user-1',
        riskScore: 50,
        riskLevel: 'medium',
        action: 'verify'
      });

      logger.clear();
      assert.strictEqual(logger.logs.length, 0);
    });
  });

  describe('RiskMonitor', function() {
    it('应该正确初始化', function() {
      const monitor = new RiskMonitor();
      assert.ok(monitor);
    });

    it('应该记录请求', function() {
      const monitor = new RiskMonitor();

      monitor.recordRequest({
        riskScore: 50,
        riskLevel: 'medium',
        action: 'verify'
      });

      assert.strictEqual(monitor.metrics.totalRequests, 1);
    });

    it('应该记录验证结果', function() {
      const monitor = new RiskMonitor();

      monitor.recordVerification({
        success: true,
        timeSpent: 3000
      });

      assert.strictEqual(monitor.metrics.totalVerifications, 1);
    });

    it('应该计算效果指标', function() {
      const monitor = new RiskMonitor();

      monitor.recordRequest({ riskScore: 30, riskLevel: 'low', action: 'allow' });
      monitor.recordRequest({ riskScore: 50, riskLevel: 'medium', action: 'verify' });
      monitor.recordRequest({ riskScore: 90, riskLevel: 'critical', action: 'block' });

      const metrics = monitor.calculateEffectivenessMetrics();
      assert.ok(metrics);
      assert.strictEqual(metrics.totalRequests, 3);
    });

    it('应该触发告警', function() {
      const monitor = new RiskMonitor();
      let alertTriggered = false;

      monitor.onAlert((alert) => {
        alertTriggered = true;
      });

      monitor.recordAnomaly({
        score: 90,
        severity: 'critical',
        userId: 'user-1'
      });

      assert.ok(alertTriggered);
    });

    it('应该返回健康状态', function() {
      const monitor = new RiskMonitor();

      const health = monitor.performHealthCheck();
      assert.ok(health);
      assert.strictEqual(typeof health.healthy, 'boolean');
    });
  });

  describe('RiskEngine', function() {
    it('应该正确初始化', function() {
      const engine = new RiskEngine();
      assert.ok(engine);
    });

    it('应该自动初始化所有模块', function() {
      const engine = new RiskEngine({ autoInitialize: true });
      const status = engine.getStatus();

      assert.ok(status.initialized.collector);
      assert.ok(status.initialized.analysis);
      assert.ok(status.initialized.scoring);
      assert.ok(status.initialized.adaptive);
      assert.ok(status.initialized.logging);
      assert.ok(status.initialized.monitoring);
    });

    it('应该计算风险评分', async function() {
      const engine = new RiskEngine({
        autoInitialize: true,
        enableCollection: false
      });

      const features = {
        combined: {
          'basic.averageSpeed': 100,
          'basic.maxSpeed': 500,
          'basic.speedVariance': 30,
          'statistical.smoothness': 0.5,
          'statistical.jitter': 5,
          'behavioral.humanLikeness': 0.6,
          'temporal.straightness': 0.6,
          'behavioral.dwellPointCount': 3,
          'behavioral.clickCount': 2,
          'keyboard.errorRate': 0.1,
          'behavioral.avgDwellDuration': 150
        }
      };

      const result = await engine.calculateRiskScore(features, { userId: 'user-1' });
      assert.ok(result);
      assert.ok(typeof result.score === 'number');
      assert.ok(typeof result.riskLevel === 'string');
      assert.ok(typeof result.action === 'string');
    });

    it('应该选择难度', async function() {
      const engine = new RiskEngine({
        autoInitialize: true
      });

      const result = await engine.selectDifficulty(50, { userId: 'user-1' });
      assert.ok(result);
      assert.ok(result.difficulty);
      assert.ok(result.captchaType);
    });

    it('应该记录风险事件', async function() {
      const engine = new RiskEngine({
        autoInitialize: true
      });

      const result = await engine.logRiskEvent({
        userId: 'user-1',
        riskScore: 50,
        riskLevel: 'medium',
        action: 'verify'
      });

      assert.ok(result);
    });

    it('应该获取风险报告', async function() {
      const engine = new RiskEngine({
        autoInitialize: true
      });

      engine.riskScorer.updateHistory('user-1', { combined: {} }, 30);
      engine.riskScorer.updateHistory('user-1', { combined: {} }, 35);

      const report = await engine.getRiskReport('user-1');
      assert.ok(report);
      assert.strictEqual(report.userId, 'user-1');
      assert.ok(report.profile);
    });

    it('应该返回状态', function() {
      const engine = new RiskEngine({
        autoInitialize: true
      });

      const status = engine.getStatus();
      assert.ok(status);
      assert.ok(status.initialized);
      assert.ok(status.stats);
    });
  });

  describe('集成测试', function() {
    it('应该完整处理风控流程', async function() {
      const engine = new RiskEngine({
        autoInitialize: true
      });

      const result = await engine.process({
        userId: 'user-integration-test',
        sessionId: 'session-integration-test'
      });

      assert.ok(result);
    });

    it('应该处理正常用户', async function() {
      const engine = new RiskEngine({
        autoInitialize: true
      });

      const features = {
        combined: {
          'basic.averageSpeed': 80,
          'basic.maxSpeed': 400,
          'basic.speedVariance': 200,
          'statistical.smoothness': 0.5,
          'statistical.jitter': 8,
          'behavioral.humanLikeness': 0.7,
          'temporal.straightness': 0.5,
          'behavioral.dwellPointCount': 5,
          'behavioral.clickCount': 3,
          'keyboard.errorRate': 0.05,
          'behavioral.avgDwellDuration': 200
        }
      };

      const result = await engine.calculateRiskScore(features, {
        userId: 'normal-user'
      });

      assert.ok(result.score < 60);
    });

    it('应该检测机器人行为', async function() {
      const engine = new RiskEngine({
        autoInitialize: true
      });

      const features = {
        combined: {
          'basic.averageSpeed': 1000,
          'basic.maxSpeed': 3000,
          'basic.speedVariance': 1,
          'statistical.smoothness': 0.99,
          'statistical.jitter': 0.1,
          'behavioral.humanLikeness': 0.05,
          'temporal.straightness': 0.98,
          'behavioral.dwellPointCount': 0,
          'behavioral.clickCount': 0,
          'keyboard.errorRate': 0,
          'behavioral.avgDwellDuration': 0
        }
      };

      const result = await engine.calculateRiskScore(features, {
        userId: 'bot-user'
      });

      assert.ok(result.score > 60);
    });
  });
});
