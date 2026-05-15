/**
 * @fileoverview 异常检测器
 * @description 基于统计的异常检测、基于距离的异常检测、异常评分计算
 * @module captchax/internal/risk/analysis/anomaly_detector
 */

'use strict';

class AnomalyDetector {
  constructor(config = {}) {
    this.config = {
      zScoreThreshold: config.zScoreThreshold || 3,
      iqrMultiplier: config.iqrMultiplier || 1.5,
      lofK: config.lofK || 5,
      contamination: config.contamination || 0.1,
      minSamples: config.minSamples || 10,
      enableAdaptiveThreshold: config.enableAdaptiveThreshold !== false,
      ...config
    };

    this.baselineData = [];
    this.baselineStats = null;
    this.models = {
      zScore: null,
      iqr: null,
      isolation: null
    };
  }

  setBaseline(data) {
    this.baselineData = data;
    this.baselineStats = this.calculateBaselineStats(data);
    this.models.zScore = this.calculateZScoreModel(data);
    this.models.iqr = this.calculateIQRModel(data);
    this.models.isolation = this.buildIsolationTree(data);
  }

  detect(features) {
    const results = {
      isAnomaly: false,
      anomalyScore: 0,
      methods: {},
      factors: []
    };

    if (!this.baselineStats) {
      return results;
    }

    const zScoreResult = this.detectByZScore(features);
    results.methods.zScore = zScoreResult;

    const iqrResult = this.detectByIQR(features);
    results.methods.iqr = iqrResult;

    const distanceResult = this.detectByDistance(features);
    results.methods.distance = distanceResult;

    const velocityResult = this.detectVelocityAnomaly(features);
    results.methods.velocity = velocityResult;

    const behavioralResult = this.detectBehavioralAnomaly(features);
    results.methods.behavioral = behavioralResult;

    const combinedScore = this.combineScores([
      zScoreResult.score,
      iqrResult.score,
      distanceResult.score,
      velocityResult.score,
      behavioralResult.score
    ]);

    results.anomalyScore = combinedScore;
    results.isAnomaly = combinedScore > this.config.contamination * 100;
    results.factors = this.aggregateFactors([
      zScoreResult,
      iqrResult,
      distanceResult,
      velocityResult,
      behavioralResult
    ]);

    return results;
  }

  detectByZScore(features) {
    const result = {
      method: 'zScore',
      score: 0,
      anomaly: false,
      details: {}
    };

    if (!this.models.zScore) return result;

    let totalZScore = 0;
    let count = 0;

    const numericFeatures = [
      'basic.averageSpeed',
      'basic.maxSpeed',
      'basic.speedVariance',
      'statistical.smoothness',
      'statistical.jitter',
      'behavioral.humanLikeness',
      'temporal.straightness'
    ];

    for (const featureKey of numericFeatures) {
      const value = this.getFeatureValue(features, featureKey);
      if (value === null) continue;

      const model = this.models.zScore[featureKey];
      if (!model) continue;

      const zScore = Math.abs((value - model.mean) / model.stdDev);
      totalZScore += zScore;
      count++;

      result.details[featureKey] = {
        value,
        zScore,
        threshold: this.config.zScoreThreshold,
        anomaly: zScore > this.config.zScoreThreshold
      };
    }

    result.score = count > 0 ? (totalZScore / count) * 33.33 : 0;
    result.anomaly = result.score > 50;

    return result;
  }

  detectByIQR(features) {
    const result = {
      method: 'iqr',
      score: 0,
      anomaly: false,
      details: {}
    };

    if (!this.models.iqr) return result;

    let totalScore = 0;
    let count = 0;

    const numericFeatures = [
      'basic.averageSpeed',
      'basic.maxSpeed',
      'basic.speedVariance',
      'statistical.smoothness',
      'behavioral.humanLikeness'
    ];

    for (const featureKey of numericFeatures) {
      const value = this.getFeatureValue(features, featureKey);
      if (value === null) continue;

      const model = this.models.iqr[featureKey];
      if (!model) continue;

      const iqr = model.q3 - model.q1;
      const lowerBound = model.q1 - this.config.iqrMultiplier * iqr;
      const upperBound = model.q3 + this.config.iqrMultiplier * iqr;

      let deviation = 0;
      let anomaly = false;

      if (value < lowerBound) {
        deviation = (lowerBound - value) / (lowerBound - model.min || 1);
        anomaly = true;
      } else if (value > upperBound) {
        deviation = (value - upperBound) / (model.max - upperBound || 1);
        anomaly = true;
      }

      totalScore += Math.min(deviation * 100, 100);
      count++;

      result.details[featureKey] = {
        value,
        lowerBound,
        upperBound,
        deviation,
        anomaly
      };
    }

    result.score = count > 0 ? (totalScore / count) * 0.33 : 0;
    result.anomaly = result.score > 50;

    return result;
  }

  detectByDistance(features) {
    const result = {
      method: 'distance',
      score: 0,
      anomaly: false,
      details: {}
    };

    if (this.baselineData.length < this.config.minSamples) {
      return result;
    }

    const distances = this.baselineData.map(baseline => {
      return this.calculateDistance(features, baseline);
    });

    distances.sort((a, b) => a - b);

    const k = Math.min(this.config.lofK, distances.length);
    const kDistances = distances.slice(0, k);
    const avgKDistance = kDistances.reduce((sum, d) => sum + d, 0) / k;

    const threshold = this.calculatePercentile(distances, 90);
    const isAnomaly = avgKDistance > threshold;

    result.score = threshold > 0 ? Math.min((avgKDistance / threshold) * 100, 100) : 0;
    result.anomaly = isAnomaly;
    result.details = {
      avgKDistance,
      threshold,
      distancePercentile: 90,
      kNearest: k
    };

    return result;
  }

  detectVelocityAnomaly(features) {
    const result = {
      method: 'velocity',
      score: 0,
      anomaly: false,
      details: {}
    };

    const avgSpeed = this.getFeatureValue(features, 'basic.averageSpeed');
    const maxSpeed = this.getFeatureValue(features, 'basic.maxSpeed');
    const speedVariance = this.getFeatureValue(features, 'basic.speedVariance');

    let score = 0;

    if (maxSpeed > 2000) {
      score += 30;
      result.details.excessiveSpeed = { value: maxSpeed, threshold: 2000 };
    }

    if (speedVariance < 10) {
      score += 25;
      result.details.lowVariance = { value: speedVariance, threshold: 10 };
    }

    if (avgSpeed > 500 || avgSpeed < 20) {
      score += 20;
      result.details.abnormalAverageSpeed = { value: avgSpeed, range: [20, 500] };
    }

    const smoothness = this.getFeatureValue(features, 'statistical.smoothness');
    if (smoothness !== null && smoothness > 0.95) {
      score += 25;
      result.details.overSmooth = { value: smoothness, threshold: 0.95 };
    }

    result.score = score;
    result.anomaly = score > 50;

    return result;
  }

  detectBehavioralAnomaly(features) {
    const result = {
      method: 'behavioral',
      score: 0,
      anomaly: false,
      details: {}
    };

    let score = 0;

    const humanLikeness = this.getFeatureValue(features, 'behavioral.humanLikeness');
    if (humanLikeness !== null && humanLikeness < 0.3) {
      score += 40;
      result.details.lowHumanLikeness = { value: humanLikeness, threshold: 0.3 };
    }

    const mechanicalScore = this.getFeatureValue(features, 'behavioral.mechanicalScore');
    if (mechanicalScore !== null && mechanicalScore > 80) {
      score += 35;
      result.details.highMechanicalScore = { value: mechanicalScore, threshold: 80 };
    }

    const straightness = this.getFeatureValue(features, 'temporal.straightness');
    if (straightness !== null && straightness > 0.9) {
      score += 25;
      result.details.overStraight = { value: straightness, threshold: 0.9 };
    }

    result.score = score;
    result.anomaly = score > 50;

    return result;
  }

  calculateBaselineStats(data) {
    const stats = {
      count: data.length,
      features: {}
    };

    const featureKeys = [
      'basic.averageSpeed',
      'basic.maxSpeed',
      'basic.speedVariance',
      'statistical.smoothness',
      'statistical.jitter',
      'behavioral.humanLikeness',
      'temporal.straightness'
    ];

    for (const key of featureKeys) {
      const values = data
        .map(d => this.getFeatureValue(d, key))
        .filter(v => v !== null && !isNaN(v));

      if (values.length > 0) {
        stats.features[key] = {
          min: Math.min(...values),
          max: Math.max(...values),
          mean: values.reduce((sum, v) => sum + v, 0) / values.length,
          count: values.length
        };
      }
    }

    return stats;
  }

  calculateZScoreModel(data) {
    const model = {};

    const featureKeys = [
      'basic.averageSpeed',
      'basic.maxSpeed',
      'basic.speedVariance',
      'statistical.smoothness',
      'statistical.jitter',
      'behavioral.humanLikeness',
      'temporal.straightness'
    ];

    for (const key of featureKeys) {
      const values = data
        .map(d => this.getFeatureValue(d, key))
        .filter(v => v !== null && !isNaN(v));

      if (values.length >= 2) {
        const mean = values.reduce((sum, v) => sum + v, 0) / values.length;
        const variance = values.reduce((sum, v) => sum + Math.pow(v - mean, 2), 0) / values.length;
        const stdDev = Math.sqrt(variance);

        model[key] = { mean, stdDev, count: values.length };
      }
    }

    return model;
  }

  calculateIQRModel(data) {
    const model = {};

    const featureKeys = [
      'basic.averageSpeed',
      'basic.maxSpeed',
      'basic.speedVariance',
      'statistical.smoothness',
      'behavioral.humanLikeness'
    ];

    for (const key of featureKeys) {
      const values = data
        .map(d => this.getFeatureValue(d, key))
        .filter(v => v !== null && !isNaN(v))
        .sort((a, b) => a - b);

      if (values.length >= 4) {
        const q1 = this.calculatePercentile(values, 25);
        const q3 = this.calculatePercentile(values, 75);

        model[key] = {
          q1,
          q3,
          min: values[0],
          max: values[values.length - 1]
        };
      }
    }

    return model;
  }

  buildIsolationTree(data) {
    const tree = {
      size: data.length,
      feature: null,
      threshold: null,
      left: null,
      right: null,
      isLeaf: data.length <= this.config.minSamples
    };

    if (!tree.isLeaf && data.length > 0) {
      const featureKeys = Object.keys(data[0] || {});
      const randomFeature = featureKeys[Math.floor(Math.random() * featureKeys.length)];
      
      const values = data.map(d => this.getFeatureValue(d, randomFeature)).filter(v => v !== null);
      
      if (values.length > 0) {
        const min = Math.min(...values);
        const max = Math.max(...values);
        
        if (min !== max) {
          tree.feature = randomFeature;
          tree.threshold = min + Math.random() * (max - min);
          
          const leftData = data.filter(d => {
            const val = this.getFeatureValue(d, randomFeature);
            return val !== null && val < tree.threshold;
          });
          
          const rightData = data.filter(d => {
            const val = this.getFeatureValue(d, randomFeature);
            return val !== null && val >= tree.threshold;
          });
          
          tree.left = this.buildIsolationTree(leftData);
          tree.right = this.buildIsolationTree(rightData);
        }
      }
    }

    return tree;
  }

  calculateIsolationScore(features, tree, depth = 0, limit = 10) {
    if (!tree || tree.isLeaf || depth >= limit) {
      return depth;
    }

    const value = this.getFeatureValue(features, tree.feature);
    if (value === null) return depth;

    if (value < tree.threshold) {
      return this.calculateIsolationScore(features, tree.left, depth + 1, limit);
    } else {
      return this.calculateIsolationScore(features, tree.right, depth + 1, limit);
    }
  }

  calculateDistance(features1, features2) {
    let sum = 0;
    let count = 0;

    const keys1 = Object.keys(features1).filter(k => 
      typeof features1[k] === 'number'
    );
    const keys2 = Object.keys(features2).filter(k => 
      typeof features2[k] === 'number'
    );

    const commonKeys = keys1.filter(k => keys2.includes(k));

    for (const key of commonKeys) {
      const diff = features1[key] - features2[key];
      sum += diff * diff;
      count++;
    }

    return count > 0 ? Math.sqrt(sum / count) : 0;
  }

  combineScores(scores) {
    const weights = [0.2, 0.2, 0.25, 0.2, 0.15];
    
    let weightedSum = 0;
    let totalWeight = 0;

    for (let i = 0; i < scores.length; i++) {
      if (scores[i] !== undefined) {
        weightedSum += scores[i] * weights[i];
        totalWeight += weights[i];
      }
    }

    return totalWeight > 0 ? weightedSum / totalWeight : 0;
  }

  aggregateFactors(results) {
    const factors = [];

    for (const result of results) {
      if (!result.details) continue;

      for (const [key, detail] of Object.entries(result.details)) {
        if (detail.anomaly) {
          factors.push({
            method: result.method,
            feature: key,
            value: detail.value,
            detail
          });
        }
      }
    }

    return factors;
  }

  getFeatureValue(features, key) {
    if (features === null || features === undefined) return null;
    
    const combined = features.combined || features;
    
    if (combined[key] !== undefined) {
      return combined[key];
    }

    const parts = key.split('.');
    if (parts.length === 2 && features[parts[0]] && features[parts[0]][parts[1]] !== undefined) {
      return features[parts[0]][parts[1]];
    }

    if (features[key] !== undefined) {
      return features[key];
    }

    return null;
  }

  calculatePercentile(sortedValues, percentile) {
    if (sortedValues.length === 0) return 0;
    
    const index = (percentile / 100) * (sortedValues.length - 1);
    const lower = Math.floor(index);
    const upper = Math.ceil(index);
    const fraction = index - lower;

    if (upper >= sortedValues.length) {
      return sortedValues[sortedValues.length - 1];
    }

    return sortedValues[lower] * (1 - fraction) + sortedValues[upper] * fraction;
  }

  addSample(features) {
    if (this.baselineData.length >= 1000) {
      this.baselineData.shift();
    }
    
    this.baselineData.push(features);
    this.setBaseline(this.baselineData);
  }

  clearBaseline() {
    this.baselineData = [];
    this.baselineStats = null;
    this.models = {
      zScore: null,
      iqr: null,
      isolation: null
    };
  }

  getStats() {
    return {
      baselineSize: this.baselineData.length,
      config: this.config,
      hasModels: {
        zScore: this.models.zScore !== null,
        iqr: this.models.iqr !== null,
        isolation: this.models.isolation !== null
      }
    };
  }
}

module.exports = AnomalyDetector;
