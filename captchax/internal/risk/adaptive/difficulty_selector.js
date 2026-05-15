/**
 * @fileoverview 自适应难度选择器
 * @description 基于风险等级选择验证码难度
 * @module captchax/internal/risk/adaptive/difficulty_selector
 */

'use strict';

class DifficultySelector {
  constructor(config = {}) {
    this.config = {
      defaultDifficulty: config.defaultDifficulty || 'medium',
      enableDynamicAdjustment: config.enableDynamicAdjustment !== false,
      cacheEnabled: config.cacheEnabled !== false,
      minConfidenceThreshold: config.minConfidenceThreshold || 0.7,
      ...config
    };

    this.difficultyLevels = this.initializeDifficultyLevels();
    this.captchaTypes = this.initializeCaptchaTypes();
    this.history = [];
    this.userDifficultyCache = new Map();
    this.performanceMetrics = new Map();
  }

  initializeDifficultyLevels() {
    return {
      trivial: {
        level: 0,
        label: '极简',
        description: '最简单的验证方式',
        riskRange: { min: 0, max: 10 },
        responseTime: { min: 500, max: 2000 },
        successRate: { min: 0.95 }
      },
      easy: {
        level: 1,
        label: '简单',
        description: '简单的验证，友好用户体验',
        riskRange: { min: 0, max: 30 },
        responseTime: { min: 1000, max: 3000 },
        successRate: { min: 0.90 }
      },
      medium: {
        level: 2,
        label: '中等',
        description: '中等难度，需要一定操作',
        riskRange: { min: 31, max: 60 },
        responseTime: { min: 2000, max: 5000 },
        successRate: { min: 0.80 }
      },
      hard: {
        level: 3,
        label: '困难',
        description: '较难的验证，能有效阻挡机器',
        riskRange: { min: 61, max: 80 },
        responseTime: { min: 3000, max: 8000 },
        successRate: { min: 0.70 }
      },
      extreme: {
        level: 4,
        label: '极难',
        description: '高难度验证，安全优先',
        riskRange: { min: 81, max: 100 },
        responseTime: { min: 5000, max: 15000 },
        successRate: { min: 0.60 }
      }
    };
  }

  initializeCaptchaTypes() {
    return {
      text: {
        id: 'text',
        name: '文字验证',
        category: 'simple',
        difficulties: ['trivial', 'easy'],
        config: {
          charCount: { trivial: 3, easy: 4, medium: 5, hard: 6, extreme: 8 },
          distortion: { trivial: 0.1, easy: 0.2, medium: 0.4, hard: 0.6, extreme: 0.8 },
          noiseLevel: { trivial: 0.1, easy: 0.2, medium: 0.3, hard: 0.5, extreme: 0.7 }
        }
      },
      icon: {
        id: 'icon',
        name: '图标选择',
        category: 'simple',
        difficulties: ['trivial', 'easy'],
        config: {
          optionCount: { trivial: 3, easy: 4, medium: 6, hard: 8, extreme: 12 },
          similarIcons: { trivial: 0, easy: 1, medium: 2, hard: 3, extreme: 4 }
        }
      },
      slider: {
        id: 'slider',
        name: '滑块验证',
        category: 'interactive',
        difficulties: ['easy', 'medium', 'hard'],
        config: {
          targetSize: { easy: 60, medium: 50, hard: 40, extreme: 35 },
          trackLength: { easy: 300, medium: 350, hard: 400, extreme: 450 },
         干扰Strength: { easy: 0.1, medium: 0.3, hard: 0.5, extreme: 0.7 }
        }
      },
      click: {
        id: 'click',
        name: '点选验证',
        category: 'interactive',
        difficulties: ['easy', 'medium', 'hard'],
        config: {
          targetCount: { easy: 2, medium: 3, hard: 4, extreme: 5 },
          imageComplexity: { easy: 'simple', medium: 'medium', hard: 'complex', extreme: 'very_complex' }
        }
      },
      rotate: {
        id: 'rotate',
        name: '旋转验证',
        category: 'rotation',
        difficulties: ['medium', 'hard', 'extreme'],
        config: {
          rotationRange: { medium: 90, hard: 180, extreme: 270 },
          precisionRequired: { medium: 15, hard: 10, extreme: 5 }
        }
      },
      puzzle: {
        id: 'puzzle',
        name: '拼图验证',
        category: 'rotation',
        difficulties: ['hard', 'extreme'],
        config: {
          pieceCount: { hard: 9, extreme: 16 },
          shapeComplexity: { hard: 'simple', extreme: 'complex' },
          moveCount: { hard: 3, extreme: 5 }
        }
      }
    };
  }

  selectDifficulty(riskScore, context = {}) {
    const riskLevel = this.getRiskLevel(riskScore);
    const difficulty = this.mapRiskToDifficulty(riskLevel);
    
    const captchaType = this.selectCaptchaType(difficulty, context);
    const captchaConfig = this.generateCaptchaConfig(captchaType, difficulty, context);

    const result = {
      riskScore,
      riskLevel,
      difficulty,
      captchaType,
      config: captchaConfig,
      timestamp: Date.now(),
      confidence: this.calculateConfidence(riskScore, context)
    };

    if (context.userId) {
      this.updateUserDifficultyCache(context.userId, result);
    }

    this.recordSelection(result);

    return result;
  }

  getRiskLevel(score) {
    if (score <= 10) return 'trivial';
    if (score <= 30) return 'easy';
    if (score <= 60) return 'medium';
    if (score <= 80) return 'hard';
    return 'extreme';
  }

  mapRiskToDifficulty(riskLevel) {
    const mapping = {
      trivial: 'trivial',
      easy: 'easy',
      medium: 'medium',
      high: 'hard',
      critical: 'extreme'
    };

    return mapping[riskLevel] || this.config.defaultDifficulty;
  }

  selectCaptchaType(difficulty, context = {}) {
    const availableTypes = Object.values(this.captchaTypes).filter(type => 
      type.difficulties.includes(difficulty)
    );

    if (availableTypes.length === 0) {
      return this.captchaTypes.slider;
    }

    const userHistory = context.userId ? this.getUserHistory(context.userId) : [];
    const recentlyUsed = new Set(userHistory.slice(-3).map(h => h.captchaType));
    const availableAndRecent = availableTypes.filter(t => !recentlyUsed.has(t.id));

    const finalOptions = availableAndRecent.length > 0 ? availableAndRecent : availableTypes;

    const weights = finalOptions.map(type => this.calculateTypeWeight(type, difficulty, context));
    const totalWeight = weights.reduce((sum, w) => sum + w, 0);
    const random = Math.random() * totalWeight;

    let cumulative = 0;
    for (let i = 0; i < finalOptions.length; i++) {
      cumulative += weights[i];
      if (random <= cumulative) {
        return finalOptions[i];
      }
    }

    return finalOptions[0];
  }

  calculateTypeWeight(captchaType, difficulty, context) {
    let weight = 1.0;

    if (context.userId && this.performanceMetrics.has(context.userId)) {
      const metrics = this.performanceMetrics.get(context.userId);
      const typeMetrics = metrics[captchaType.id];

      if (typeMetrics) {
        const successRate = typeMetrics.successes / typeMetrics.attempts;
        if (successRate > 0.9) {
          weight *= 1.2;
        } else if (successRate < 0.5) {
          weight *= 0.5;
        }

        const avgTime = typeMetrics.totalTime / typeMetrics.attempts;
        if (avgTime > 10000) {
          weight *= 0.7;
        }
      }
    }

    const categoryWeights = {
      simple: { trivial: 2.0, easy: 1.5, medium: 0.8, hard: 0.3, extreme: 0.1 },
      interactive: { trivial: 0.5, easy: 1.2, medium: 1.5, hard: 1.0, extreme: 0.5 },
      rotation: { trivial: 0.1, easy: 0.3, medium: 0.8, hard: 1.5, extreme: 2.0 }
    };

    weight *= categoryWeights[captchaType.category]?.[difficulty] || 1.0;

    return weight;
  }

  generateCaptchaConfig(captchaType, difficulty, config = {}) {
    const baseConfig = captchaType.config || {};
    const generatedConfig = {};

    for (const [key, values] of Object.entries(baseConfig)) {
      if (typeof values === 'object' && values !== null) {
        generatedConfig[key] = values[difficulty] || values.easy || Object.values(values)[0];
      } else {
        generatedConfig[key] = values;
      }
    }

    generatedConfig.timeout = this.difficultyLevels[difficulty].responseTime.max;
    generatedConfig.retryLimit = difficulty === 'trivial' ? 5 : (difficulty === 'extreme' ? 2 : 3);
    generatedConfig.showHints = difficulty === 'trivial' || difficulty === 'easy';

    return generatedConfig;
  }

  calculateConfidence(riskScore, context = {}) {
    let confidence = 0.5;

    if (context.dataQuality) {
      if (context.dataQuality > 0.8) confidence += 0.2;
      else if (context.dataQuality > 0.5) confidence += 0.1;
      else confidence -= 0.1;
    }

    if (context.historicalDataPoints && context.historicalDataPoints > 10) {
      confidence += 0.15;
    }

    const deviation = this.calculateDeviation(context);
    if (deviation < 0.2) {
      confidence += 0.1;
    } else if (deviation > 0.5) {
      confidence -= 0.1;
    }

    return Math.max(0, Math.min(1, confidence));
  }

  calculateDeviation(context = {}) {
    if (!context.userId) return 0.5;

    const history = this.getUserHistory(context.userId);
    if (history.length < 2) return 0.5;

    const recentSelections = history.slice(-5);
    const difficulties = recentSelections.map(h => this.difficultyLevels[h.difficulty].level);
    
    const mean = difficulties.reduce((sum, d) => sum + d, 0) / difficulties.length;
    const variance = difficulties.reduce((sum, d) => sum + Math.pow(d - mean, 2), 0) / difficulties.length;

    return Math.sqrt(variance);
  }

  updateUserDifficultyCache(userId, selection) {
    const cache = this.userDifficultyCache.get(userId) || [];
    cache.push(selection);

    if (cache.length > 20) {
      cache.shift();
    }

    this.userDifficultyCache.set(userId, cache);
  }

  getUserHistory(userId) {
    return this.userDifficultyCache.get(userId) || [];
  }

  recordSelection(selection) {
    this.history.push(selection);

    if (this.history.length > 1000) {
      this.history.shift();
    }
  }

  recordAttempt(userId, captchaType, success, timeSpent) {
    if (!userId) return;

    const metrics = this.performanceMetrics.get(userId) || {};
    const typeMetrics = metrics[captchaType] || { attempts: 0, successes: 0, totalTime: 0 };

    typeMetrics.attempts++;
    if (success) {
      typeMetrics.successes++;
    }
    typeMetrics.totalTime += timeSpent;

    metrics[captchaType] = typeMetrics;
    this.performanceMetrics.set(userId, metrics);
  }

  getCaptchaTypeInfo(typeId) {
    return this.captchaTypes[typeId] || null;
  }

  getAllCaptchaTypes() {
    return Object.values(this.captchaTypes);
  }

  getDifficultyInfo(difficulty) {
    return this.difficultyLevels[difficulty] || null;
  }

  getAllDifficulties() {
    return Object.values(this.difficultyLevels);
  }

  getUserPerformance(userId) {
    return this.performanceMetrics.get(userId) || {};
  }

  adjustDifficulty(currentDifficulty, attemptResult) {
    const currentLevel = this.difficultyLevels[currentDifficulty];

    if (attemptResult.success) {
      if (attemptResult.timeSpent < currentLevel.responseTime.min) {
        return this.getNextHigherDifficulty(currentDifficulty);
      }
      return currentDifficulty;
    } else {
      const failedAttempts = attemptResult.failedAttempts || 1;
      if (failedAttempts >= 2) {
        return this.getNextLowerDifficulty(currentDifficulty);
      }
      return currentDifficulty;
    }
  }

  getNextHigherDifficulty(difficulty) {
    const order = ['trivial', 'easy', 'medium', 'hard', 'extreme'];
    const index = order.indexOf(difficulty);

    if (index < order.length - 1) {
      return order[index + 1];
    }

    return difficulty;
  }

  getNextLowerDifficulty(difficulty) {
    const order = ['trivial', 'easy', 'medium', 'hard', 'extreme'];
    const index = order.indexOf(difficulty);

    if (index > 0) {
      return order[index - 1];
    }

    return difficulty;
  }

  getStatistics() {
    const stats = {
      totalSelections: this.history.length,
      difficultyDistribution: {},
      captchaTypeDistribution: {},
      averageRiskScore: 0
    };

    for (const selection of this.history) {
      stats.difficultyDistribution[selection.difficulty] = 
        (stats.difficultyDistribution[selection.difficulty] || 0) + 1;
      
      stats.captchaTypeDistribution[selection.captchaType.id] = 
        (stats.captchaTypeDistribution[selection.captchaType.id] || 0) + 1;
      
      stats.averageRiskScore += selection.riskScore;
    }

    if (this.history.length > 0) {
      stats.averageRiskScore /= this.history.length;
    }

    stats.cachedUsers = this.userDifficultyCache.size;
    stats.usersWithMetrics = this.performanceMetrics.size;

    return stats;
  }

  clearCache() {
    this.userDifficultyCache.clear();
  }

  clearHistory() {
    this.history = [];
  }

  clearUserData(userId) {
    this.userDifficultyCache.delete(userId);
    this.performanceMetrics.delete(userId);
  }
}

module.exports = DifficultySelector;
