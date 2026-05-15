/**
 * @fileoverview 风险评分器
 * @description 多维度评分权重配置、实时风险评分计算、历史行为对比分析
 * @module captchax/internal/risk/scoring/risk_scorer
 */

'use strict';

class RiskScorer {
  constructor(config = {}) {
    this.config = {
      baseWeights: config.baseWeights || this.getDefaultWeights(),
      scoreRanges: config.scoreRanges || this.getDefaultRanges(),
      enableHistoryComparison: config.enableHistoryComparison !== false,
      enableRealTimeUpdate: config.enableRealTimeUpdate !== false,
      decayFactor: config.decayFactor || 0.95,
      ...config
    };

    this.userHistory = new Map();
    this.globalBaseline = null;
    this.weightModifiers = {
      velocity: 1.0,
      smoothness: 1.0,
      jitter: 1.0,
      humanLikeness: 1.0,
      historical: 1.0,
      device: 1.0,
      behavioral: 1.0
    };
  }

  getDefaultWeights() {
    return {
      velocity: {
        weight: 0.15,
        features: ['basic.averageSpeed', 'basic.maxSpeed', 'basic.speedVariance'],
        thresholds: {
          suspicious: { avgSpeed: { min: 20, max: 500 }, maxSpeed: { max: 2000 } },
          critical: { avgSpeed: { min: 10, max: 800 }, maxSpeed: { max: 3000 } }
        }
      },
      smoothness: {
        weight: 0.20,
        features: ['statistical.smoothness', 'temporal.straightness'],
        thresholds: {
          suspicious: { smoothness: { min: 0.95 } },
          critical: { smoothness: { min: 0.98 } }
        }
      },
      jitter: {
        weight: 0.10,
        features: ['statistical.jitter', 'behavioral.dwellPointCount'],
        thresholds: {
          suspicious: { jitter: { max: 2 } },
          critical: { jitter: { max: 1 } }
        }
      },
      humanLikeness: {
        weight: 0.20,
        features: ['behavioral.humanLikeness', 'behavioral.mechanicalScore'],
        thresholds: {
          suspicious: { humanLikeness: { max: 0.4 } },
          critical: { humanLikeness: { max: 0.2 } }
        }
      },
      historical: {
        weight: 0.15,
        features: ['similarityToHistory', 'deviationFromBaseline'],
        thresholds: {
          suspicious: { similarity: { max: 0.3 } },
          critical: { similarity: { max: 0.1 } }
        }
      },
      device: {
        weight: 0.10,
        features: ['deviceConsistency', 'fpAnomalyScore'],
        thresholds: {
          suspicious: { consistency: { max: 0.7 } },
          critical: { consistency: { max: 0.5 } }
        }
      },
      behavioral: {
        weight: 0.10,
        features: ['clickCount', 'errorRate', 'avgDwellDuration'],
        thresholds: {
          suspicious: { errorRate: { min: 0.3 } },
          critical: { errorRate: { min: 0.5 } }
        }
      }
    };
  }

  getDefaultRanges() {
    return {
      low: { min: 0, max: 30 },
      medium: { min: 31, max: 60 },
      high: { min: 61, max: 80 },
      critical: { min: 81, max: 100 }
    };
  }

  calculateScore(features, context = {}) {
    const scores = {};
    let totalScore = 0;
    let totalWeight = 0;
    const details = [];

    for (const [dimension, config] of Object.entries(this.config.baseWeights)) {
      const modifiedWeight = config.weight * (this.weightModifiers[dimension] || 1.0);
      const dimensionScore = this.calculateDimensionScore(dimension, config, features, context);
      
      scores[dimension] = {
        rawScore: dimensionScore,
        weight: modifiedWeight,
        weightedScore: dimensionScore * modifiedWeight
      };

      totalWeight += modifiedWeight;
      totalScore += dimensionScore * modifiedWeight;

      details.push({
        dimension,
        score: dimensionScore,
        weight: modifiedWeight,
        weightedScore: dimensionScore * modifiedWeight
      });
    }

    const normalizedScore = totalWeight > 0 ? totalScore / totalWeight : 0;
    const roundedScore = Math.round(Math.min(100, Math.max(0, normalizedScore)));

    let historicalBonus = 0;
    if (this.config.enableHistoryComparison && context.userId) {
      historicalBonus = this.calculateHistoricalBonus(context.userId, features);
    }

    const finalScore = Math.min(100, roundedScore + historicalBonus);

    return {
      totalScore: finalScore,
      normalizedScore,
      roundedScore,
      scores,
      details,
      historicalBonus,
      modifiers: { ...this.weightModifiers }
    };
  }

  calculateDimensionScore(dimension, config, features, context) {
    switch (dimension) {
      case 'velocity':
        return this.calculateVelocityScore(config, features);
      
      case 'smoothness':
        return this.calculateSmoothnessScore(config, features);
      
      case 'jitter':
        return this.calculateJitterScore(config, features);
      
      case 'humanLikeness':
        return this.calculateHumanLikenessScore(config, features);
      
      case 'historical':
        return this.calculateHistoricalScore(config, features, context);
      
      case 'device':
        return this.calculateDeviceScore(config, features, context);
      
      case 'behavioral':
        return this.calculateBehavioralScore(config, features);
      
      default:
        return 0;
    }
  }

  calculateVelocityScore(config, features) {
    let score = 0;
    let count = 0;

    const avgSpeed = this.getFeatureValue(features, 'basic.averageSpeed');
    const maxSpeed = this.getFeatureValue(features, 'basic.maxSpeed');
    const speedVariance = this.getFeatureValue(features, 'basic.speedVariance');

    if (avgSpeed !== null) {
      if (avgSpeed > 500 || avgSpeed < 20) {
        score += 80;
      } else if (avgSpeed > 300 || avgSpeed < 50) {
        score += 50;
      } else {
        score += 20;
      }
      count++;
    }

    if (maxSpeed !== null) {
      if (maxSpeed > 2000) {
        score += 100;
      } else if (maxSpeed > 1000) {
        score += 60;
      } else {
        score += 20;
      }
      count++;
    }

    if (speedVariance !== null) {
      if (speedVariance < 10) {
        score += 100;
      } else if (speedVariance < 50) {
        score += 50;
      } else {
        score += 10;
      }
      count++;
    }

    return count > 0 ? score / count : 50;
  }

  calculateSmoothnessScore(config, features) {
    let score = 0;
    let count = 0;

    const smoothness = this.getFeatureValue(features, 'statistical.smoothness');
    const straightness = this.getFeatureValue(features, 'temporal.straightness');

    if (smoothness !== null) {
      if (smoothness > 0.95) {
        score += 100;
      } else if (smoothness > 0.85) {
        score += 60;
      } else if (smoothness > 0.7) {
        score += 30;
      } else {
        score += 10;
      }
      count++;
    }

    if (straightness !== null) {
      if (straightness > 0.9) {
        score += 100;
      } else if (straightness > 0.7) {
        score += 60;
      } else if (straightness > 0.5) {
        score += 30;
      } else {
        score += 10;
      }
      count++;
    }

    return count > 0 ? score / count : 50;
  }

  calculateJitterScore(config, features) {
    let score = 0;
    let count = 0;

    const jitter = this.getFeatureValue(features, 'statistical.jitter');
    const dwellCount = this.getFeatureValue(features, 'behavioral.dwellPointCount');

    if (jitter !== null) {
      if (jitter < 2) {
        score += 100;
      } else if (jitter < 5) {
        score += 60;
      } else if (jitter < 10) {
        score += 30;
      } else {
        score += 10;
      }
      count++;
    }

    if (dwellCount !== null) {
      if (dwellCount === 0) {
        score += 100;
      } else if (dwellCount < 3) {
        score += 60;
      } else {
        score += 20;
      }
      count++;
    }

    return count > 0 ? score / count : 50;
  }

  calculateHumanLikenessScore(config, features) {
    let score = 0;
    let count = 0;

    const humanLikeness = this.getFeatureValue(features, 'behavioral.humanLikeness');
    const mechanicalScore = this.getFeatureValue(features, 'behavioral.mechanicalScore');

    if (humanLikeness !== null) {
      if (humanLikeness < 0.2) {
        score += 100;
      } else if (humanLikeness < 0.4) {
        score += 70;
      } else if (humanLikeness < 0.6) {
        score += 40;
      } else {
        score += 10;
      }
      count++;
    }

    if (mechanicalScore !== null) {
      if (mechanicalScore > 80) {
        score += 100;
      } else if (mechanicalScore > 60) {
        score += 70;
      } else if (mechanicalScore > 40) {
        score += 40;
      } else {
        score += 10;
      }
      count++;
    }

    return count > 0 ? score / count : 50;
  }

  calculateHistoricalScore(config, features, context) {
    if (!this.config.enableHistoryComparison || !context.userId) {
      return 50;
    }

    const userId = context.userId;
    const history = this.userHistory.get(userId);

    if (!history || history.length === 0) {
      return 50;
    }

    const recentHistory = history.slice(-10);
    const avgScore = recentHistory.reduce((sum, h) => sum + h.score, 0) / recentHistory.length;
    const variance = this.calculateVariance(
      recentHistory.map(h => h.score),
      avgScore
    );

    const currentFeatures = this.getFeatureVector(features);
    let totalDistance = 0;

    for (const historyItem of recentHistory) {
      const distance = this.calculateDistance(currentFeatures, historyItem.features);
      totalDistance += distance;
    }

    const avgDistance = totalDistance / recentHistory.length;

    let score = 50;

    if (avgDistance > 0.8) {
      score += 40;
    } else if (avgDistance > 0.5) {
      score += 20;
    }

    if (variance < 10) {
      score += 30;
    } else if (variance < 50) {
      score += 15;
    }

    return score;
  }

  calculateDeviceScore(config, features, context) {
    let score = 50;
    let count = 0;

    if (context.fingerprint) {
      const fpHash = context.fingerprint.hash || '';
      const userId = context.userId;

      if (userId) {
        const knownFps = this.userHistory.get(`${userId}_fingerprints`) || [];
        
        if (knownFps.length > 0) {
          const isKnown = knownFps.includes(fpHash);
          
          if (!isKnown) {
            score += 40;
          } else {
            score -= 10;
          }
          count++;
        }

        knownFps.push(fpHash);
        if (knownFps.length > 10) {
          knownFps.shift();
        }
        this.userHistory.set(`${userId}_fingerprints`, knownFps);
      }
    }

    return count > 0 ? score : 50;
  }

  calculateBehavioralScore(config, features) {
    let score = 0;
    let count = 0;

    const clickCount = this.getFeatureValue(features, 'behavioral.clickCount');
    const errorRate = this.getFeatureValue(features, 'keyboard.errorRate');
    const avgDwell = this.getFeatureValue(features, 'behavioral.avgDwellDuration');

    if (clickCount !== null) {
      if (clickCount === 0) {
        score += 80;
      } else if (clickCount < 2) {
        score += 50;
      } else {
        score += 20;
      }
      count++;
    }

    if (errorRate !== null) {
      if (errorRate > 0.5) {
        score += 100;
      } else if (errorRate > 0.3) {
        score += 60;
      } else if (errorRate > 0.1) {
        score += 30;
      } else {
        score += 10;
      }
      count++;
    }

    if (avgDwell !== null) {
      if (avgDwell < 50) {
        score += 80;
      } else if (avgDwell < 100) {
        score += 50;
      } else {
        score += 20;
      }
      count++;
    }

    return count > 0 ? score / count : 50;
  }

  calculateHistoricalBonus(userId, features) {
    const history = this.userHistory.get(userId);
    
    if (!history || history.length < 3) {
      return 0;
    }

    const recentItems = history.slice(-5);
    const currentFeatures = this.getFeatureVector(features);

    let totalDeviation = 0;
    for (const item of recentItems) {
      const distance = this.calculateDistance(currentFeatures, item.features);
      totalDeviation += distance;
    }

    const avgDeviation = totalDeviation / recentItems.length;

    if (avgDeviation > 0.8) {
      return 15;
    } else if (avgDeviation > 0.5) {
      return 5;
    }

    return 0;
  }

  updateHistory(userId, features, score) {
    if (!userId) return;

    const history = this.userHistory.get(userId) || [];
    
    history.push({
      timestamp: Date.now(),
      score,
      features: this.getFeatureVector(features)
    });

    if (history.length > 100) {
      history.shift();
    }

    this.userHistory.set(userId, history);
  }

  getFeatureVector(features) {
    const combined = features.combined || features;
    
    return {
      avgSpeed: combined['basic.averageSpeed'] || 0,
      maxSpeed: combined['basic.maxSpeed'] || 0,
      smoothness: combined['statistical.smoothness'] || 0,
      jitter: combined['statistical.jitter'] || 0,
      humanLikeness: combined['behavioral.humanLikeness'] || 0,
      straightness: combined['temporal.straightness'] || 0
    };
  }

  calculateDistance(vec1, vec2) {
    const keys = Object.keys(vec1);
    let sum = 0;
    let count = 0;

    for (const key of keys) {
      if (vec2[key] !== undefined) {
        const diff = vec1[key] - vec2[key];
        sum += diff * diff;
        count++;
      }
    }

    return count > 0 ? Math.sqrt(sum / count) : 0;
  }

  calculateVariance(values, mean) {
    if (values.length < 2) return 0;
    const squaredDiffs = values.map(v => Math.pow(v - mean, 2));
    return squaredDiffs.reduce((sum, val) => sum + val, 0) / values.length;
  }

  getFeatureValue(features, key) {
    if (!features) return null;
    
    const combined = features.combined || features;
    
    if (combined[key] !== undefined) {
      return combined[key];
    }

    const parts = key.split('.');
    if (parts.length === 2 && features[parts[0]]) {
      return features[parts[0]][parts[1]];
    }

    return features[key];
  }

  setWeightModifier(dimension, modifier) {
    if (this.weightModifiers[dimension] !== undefined) {
      this.weightModifiers[dimension] = modifier;
    }
  }

  resetWeightModifiers() {
    for (const key of Object.keys(this.weightModifiers)) {
      this.weightModifiers[key] = 1.0;
    }
  }

  setGlobalBaseline(features) {
    this.globalBaseline = this.getFeatureVector(features);
  }

  getUserHistory(userId) {
    return this.userHistory.get(userId) || [];
  }

  clearUserHistory(userId) {
    if (userId) {
      this.userHistory.delete(userId);
      this.userHistory.delete(`${userId}_fingerprints`);
    }
  }

  getStats() {
    return {
      userCount: this.userHistory.size / 2,
      globalBaseline: this.globalBaseline !== null,
      weightModifiers: { ...this.weightModifiers }
    };
  }
}

module.exports = RiskScorer;
