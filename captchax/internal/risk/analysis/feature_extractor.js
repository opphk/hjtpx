/**
 * @fileoverview 特征提取器
 * @description 基础统计特征计算、时序特征提取、轨迹相似度算法
 * @module captchax/internal/risk/analysis/feature_extractor
 */

'use strict';

class FeatureExtractor {
  constructor(config = {}) {
    this.config = {
      smoothingWindow: config.smoothingWindow || 3,
      featureNormalization: config.featureNormalization !== false,
      similarityThreshold: config.similarityThreshold || 0.85,
      ...config
    };

    this.baselineFeatures = null;
    this.featureWeights = this.initializeWeights();
  }

  initializeWeights() {
    return {
      velocity: 1.0,
      acceleration: 1.0,
      direction: 1.0,
      smoothness: 1.0,
      jitter: 1.0,
      straightness: 1.0,
      dwell: 1.0,
      clickRhythm: 1.0,
      trackLength: 1.0,
      trackPointCount: 1.0
    };
  }

  extract(behaviorData) {
    if (!behaviorData || !behaviorData.mouseTracks) {
      return this.getDefaultFeatures();
    }

    const features = {
      basic: this.extractBasicFeatures(behaviorData),
      temporal: this.extractTemporalFeatures(behaviorData),
      statistical: this.extractStatisticalFeatures(behaviorData),
      behavioral: this.extractBehavioralFeatures(behaviorData),
      keyboard: this.extractKeyboardFeatures(behaviorData),
      metadata: this.extractMetadataFeatures(behaviorData)
    };

    features.combined = this.combineFeatures(features);
    features.normalized = this.normalizeFeatures(features.combined);
    
    return features;
  }

  extractBasicFeatures(data) {
    const tracks = data.mouseTracks || [];
    
    if (tracks.length < 2) {
      return {
        trackPointCount: 0,
        totalDistance: 0,
        averageSpeed: 0,
        maxSpeed: 0,
        minSpeed: 0,
        speedVariance: 0
      };
    }

    let totalDistance = 0;
    let maxSpeed = 0;
    let minSpeed = Infinity;
    let speedSum = 0;
    let speedCount = 0;

    const speeds = [];
    const velocities = tracks.map((t, i) => {
      if (i === 0) return 0;
      
      const dx = t.x - tracks[i - 1].x;
      const dy = t.y - tracks[i - 1].y;
      const dt = t.timestamp - tracks[i - 1].timestamp;
      const distance = Math.sqrt(dx * dx + dy * dy);
      totalDistance += distance;
      
      if (dt > 0) {
        const speed = distance / dt * 1000;
        speeds.push(speed);
        maxSpeed = Math.max(maxSpeed, speed);
        minSpeed = Math.min(minSpeed, speed);
        speedSum += speed;
        speedCount++;
        return speed;
      }
      return 0;
    });

    const averageSpeed = speedCount > 0 ? speedSum / speedCount : 0;
    const speedVariance = this.calculateVariance(speeds, averageSpeed);

    return {
      trackPointCount: tracks.length,
      totalDistance,
      averageSpeed,
      maxSpeed,
      minSpeed,
      speedVariance,
      speedStdDev: Math.sqrt(speedVariance),
      coefficientOfVariation: averageSpeed > 0 ? Math.sqrt(speedVariance) / averageSpeed : 0
    };
  }

  extractTemporalFeatures(data) {
    const tracks = data.mouseTracks || [];
    const duration = data.slideDuration || (tracks.length > 0 
      ? tracks[tracks.length - 1].timestamp - tracks[0].timestamp 
      : 0);

    const startPoint = tracks[0] || { x: 0, y: 0 };
    const endPoint = tracks[tracks.length - 1] || { x: 0, y: 0 };
    const directDistance = Math.sqrt(
      Math.pow(endPoint.x - startPoint.x, 2) + 
      Math.pow(endPoint.y - startPoint.y, 2)
    );

    const totalDistance = this.extractBasicFeatures(data).totalDistance;
    const straightness = totalDistance > 0 ? directDistance / totalDistance : 0;

    const averageTimeBetweenPoints = tracks.length > 1 
      ? duration / (tracks.length - 1) 
      : 0;

    return {
      duration,
      averageTimeBetweenPoints,
      startTime: tracks[0]?.timestamp || 0,
      endTime: tracks[tracks.length - 1]?.timestamp || 0,
      straightness,
      totalDistance,
      directDistance,
      efficiency: straightness * (totalDistance > 0 ? 1 : 0),
      startX: startPoint.x,
      startY: startPoint.y,
      endX: endPoint.x,
      endY: endPoint.y
    };
  }

  extractStatisticalFeatures(data) {
    const tracks = data.mouseTracks || [];
    
    if (tracks.length < 3) {
      return this.getDefaultStatisticalFeatures();
    }

    const xValues = tracks.map(t => t.x);
    const yValues = tracks.map(t => t.y);
    
    const xMean = this.calculateMean(xValues);
    const yMean = this.calculateMean(yValues);
    const xVariance = this.calculateVariance(xValues, xMean);
    const yVariance = this.calculateVariance(yValues, yMean);
    
    const velocities = tracks
      .slice(1)
      .map((t, i) => {
        const dx = t.x - tracks[i].x;
        const dy = t.y - tracks[i].y;
        const dt = t.timestamp - tracks[i].timestamp;
        return dt > 0 ? Math.sqrt(dx * dx + dy * dy) / dt * 1000 : 0;
      });

    const accelerations = velocities.slice(1).map((v, i) => Math.abs(v - velocities[i]));
    
    const velocityMean = this.calculateMean(velocities);
    const velocityVariance = this.calculateVariance(velocities, velocityMean);
    const accelerationMean = accelerations.length > 0 ? this.calculateMean(accelerations) : 0;
    const accelerationVariance = accelerations.length > 0 ? this.calculateVariance(accelerations, accelerationMean) : 0;

    const smoothness = this.calculateSmoothness(tracks);
    const jitter = this.calculateJitter(tracks);

    const directions = tracks.slice(1).map((t, i) => {
      const dx = t.x - tracks[i].x;
      const dy = t.y - tracks[i].y;
      return Math.atan2(dy, dx);
    });

    const directionChanges = directions.slice(1).map((d, i) => {
      let diff = Math.abs(d - directions[i]);
      if (diff > Math.PI) diff = 2 * Math.PI - diff;
      return diff;
    });

    const avgDirectionChange = directionChanges.length > 0 
      ? this.calculateMean(directionChanges) 
      : 0;
    const directionChangeVariance = directionChanges.length > 0 
      ? this.calculateVariance(directionChanges, avgDirectionChange) 
      : 0;

    return {
      xMean,
      yMean,
      xVariance,
      yVariance,
      xStdDev: Math.sqrt(xVariance),
      yStdDev: Math.sqrt(yVariance),
      velocityMean,
      velocityVariance,
      velocityStdDev: Math.sqrt(velocityVariance),
      accelerationMean,
      accelerationVariance,
      accelerationStdDev: Math.sqrt(accelerationVariance),
      smoothness,
      jitter,
      avgDirectionChange,
      directionChangeVariance,
      directionChangeStdDev: Math.sqrt(directionChangeVariance),
      peakCount: this.countPeaks(velocities)
    };
  }

  extractBehavioralFeatures(data) {
    const tracks = data.mouseTracks || [];
    const clicks = data.clickTimes || [];
    const dwellPoints = data.dwellPoints || [];

    const statistical = this.extractStatisticalFeatures(data);
    
    const mechanicalScore = this.calculateMechanicalScore(statistical);
    const humanLikeness = this.calculateHumanLikeness(statistical, dwellPoints);

    return {
      mechanicalScore,
      humanLikeness,
      clickCount: clicks.length,
      dwellPointCount: dwellPoints.length,
      avgDwellDuration: dwellPoints.length > 0 
        ? this.calculateMean(dwellPoints.map(p => p.duration)) 
        : 0,
      maxDwellDuration: dwellPoints.length > 0 
        ? Math.max(...dwellPoints.map(p => p.duration)) 
        : 0,
      totalDwellTime: dwellPoints.reduce((sum, p) => sum + p.duration, 0),
      hesitationCount: dwellPoints.filter(p => p.duration > 200).length,
      confidenceScore: humanLikeness * 100
    };
  }

  extractKeyboardFeatures(data) {
    const keyTimes = data.keyTimes || [];
    
    if (keyTimes.length < 2) {
      return {
        keyPressCount: keyTimes.length,
        avgInterval: 0,
        intervalVariance: 0,
        fastestInterval: 0,
        slowestInterval: 0,
        isMechanical: false,
        avgPressDuration: 0,
        errorRate: 0
      };
    }

    const intervals = keyTimes.slice(1).map((k, i) => k.timestamp - keyTimes[i].timestamp);
    const avgInterval = this.calculateMean(intervals);
    const intervalVariance = this.calculateVariance(intervals, avgInterval);
    const fastestInterval = Math.min(...intervals);
    const slowestInterval = Math.max(...intervals);

    const pressDurations = keyTimes
      .filter(k => k.pressDuration !== undefined)
      .map(k => k.pressDuration);
    const avgPressDuration = pressDurations.length > 0 
      ? this.calculateMean(pressDurations) 
      : 0;

    const errorIndicators = keyTimes.filter(k => 
      (k.pressDuration !== undefined && k.pressDuration < 30) ||
      (k.interval !== undefined && k.interval < 50)
    ).length;
    const errorRate = keyTimes.length > 0 ? errorIndicators / keyTimes.length : 0;

    return {
      keyPressCount: keyTimes.length,
      avgInterval,
      intervalVariance,
      fastestInterval,
      slowestInterval,
      isMechanical: intervalVariance < 500 && intervals.length > 5,
      avgPressDuration,
      errorRate,
      typingSpeed: avgInterval > 0 ? 60000 / avgInterval : 0
    };
  }

  extractMetadataFeatures(data) {
    return {
      sessionId: data.sessionId,
      trackPointCount: data.trackPointCount || 0,
      clickCount: data.clickCount || 0,
      keyPressCount: data.keyPressCount || 0,
      duration: data.slideDuration || 0
    };
  }

  combineFeatures(features) {
    const combined = {};
    
    Object.keys(features.basic).forEach(key => {
      combined[`basic.${key}`] = features.basic[key];
    });
    
    Object.keys(features.temporal).forEach(key => {
      combined[`temporal.${key}`] = features.temporal[key];
    });
    
    Object.keys(features.statistical).forEach(key => {
      combined[`statistical.${key}`] = features.statistical[key];
    });
    
    Object.keys(features.behavioral).forEach(key => {
      combined[`behavioral.${key}`] = features.behavioral[key];
    });
    
    Object.keys(features.keyboard).forEach(key => {
      combined[`keyboard.${key}`] = features.keyboard[key];
    });

    return combined;
  }

  normalizeFeatures(features) {
    if (!this.baselineFeatures) {
      return features;
    }

    const normalized = {};
    
    for (const [key, value] of Object.entries(features)) {
      if (typeof value !== 'number' || isNaN(value)) {
        normalized[key] = value;
        continue;
      }

      const baselineKey = key.replace(/^(basic|temporal|statistical|behavioral|keyboard)\./, '');
      const baselineValue = this.baselineFeatures[key] || this.getDefaultValue(key);
      
      if (baselineValue !== 0) {
        normalized[key] = (value - baselineValue) / Math.abs(baselineValue);
      } else {
        normalized[key] = value;
      }
    }

    return normalized;
  }

  getDefaultValue(key) {
    const defaults = {
      'basic.averageSpeed': 100,
      'basic.maxSpeed': 500,
      'statistical.smoothness': 0.8,
      'statistical.jitter': 5,
      'behavioral.humanLikeness': 0.7,
      'temporal.straightness': 0.5
    };
    return defaults[key] || 0;
  }

  setBaseline(features) {
    this.baselineFeatures = this.combineFeatures(features);
  }

  calculateEuclideanDistance(features1, features2) {
    let sum = 0;
    let count = 0;

    for (const key of Object.keys(features1)) {
      if (typeof features1[key] === 'number' && typeof features2[key] === 'number') {
        const diff = features1[key] - features2[key];
        sum += diff * diff;
        count++;
      }
    }

    return count > 0 ? Math.sqrt(sum / count) : 0;
  }

  calculateCosineSimilarity(features1, features2) {
    let dotProduct = 0;
    let norm1 = 0;
    let norm2 = 0;

    for (const key of Object.keys(features1)) {
      if (typeof features1[key] === 'number' && typeof features2[key] === 'number') {
        dotProduct += features1[key] * features2[key];
        norm1 += features1[key] * features1[key];
        norm2 += features2[key] * features2[key];
      }
    }

    const denominator = Math.sqrt(norm1) * Math.sqrt(norm2);
    return denominator > 0 ? dotProduct / denominator : 0;
  }

  calculateManhattanDistance(features1, features2) {
    let sum = 0;
    let count = 0;

    for (const key of Object.keys(features1)) {
      if (typeof features1[key] === 'number' && typeof features2[key] === 'number') {
        sum += Math.abs(features1[key] - features2[key]);
        count++;
      }
    }

    return count > 0 ? sum / count : 0;
  }

  calculateDTWDistance(sequence1, sequence2, window = null) {
    const n = sequence1.length;
    const m = sequence2.length;
    
    if (n === 0 || m === 0) return 0;

    const dtw = Array(n + 1).fill(null).map(() => Array(m + 1).fill(Infinity));
    dtw[0][0] = 0;

    const w = window || Math.max(n, m);

    for (let i = 1; i <= n; i++) {
      for (let j = Math.max(1, i - w); j <= Math.min(m, i + w); j++) {
        const cost = Math.abs(sequence1[i - 1] - sequence2[j - 1]);
        dtw[i][j] = cost + Math.min(
          dtw[i - 1][j],
          dtw[i][j - 1],
          dtw[i - 1][j - 1]
        );
      }
    }

    return dtw[n][m];
  }

  getSimilarityScore(features1, features2) {
    const euclidean = this.calculateEuclideanDistance(features1, features2);
    const cosine = this.calculateCosineSimilarity(features1, features2);
    
    const normalizedEuclidean = 1 / (1 + euclidean);
    const combinedScore = (normalizedEuclidean + cosine) / 2;
    
    return combinedScore;
  }

  getDefaultFeatures() {
    return {
      basic: {
        trackPointCount: 0,
        totalDistance: 0,
        averageSpeed: 0,
        maxSpeed: 0,
        minSpeed: 0,
        speedVariance: 0
      },
      temporal: {
        duration: 0,
        averageTimeBetweenPoints: 0,
        straightness: 0,
        totalDistance: 0,
        directDistance: 0
      },
      statistical: this.getDefaultStatisticalFeatures(),
      behavioral: {
        mechanicalScore: 0,
        humanLikeness: 0,
        clickCount: 0,
        dwellPointCount: 0
      },
      keyboard: {
        keyPressCount: 0,
        avgInterval: 0,
        intervalVariance: 0
      },
      metadata: {
        sessionId: '',
        trackPointCount: 0,
        clickCount: 0,
        keyPressCount: 0,
        duration: 0
      }
    };
  }

  getDefaultStatisticalFeatures() {
    return {
      xMean: 0,
      yMean: 0,
      xVariance: 0,
      yVariance: 0,
      xStdDev: 0,
      yStdDev: 0,
      velocityMean: 0,
      velocityVariance: 0,
      velocityStdDev: 0,
      accelerationMean: 0,
      accelerationVariance: 0,
      accelerationStdDev: 0,
      smoothness: 0,
      jitter: 0,
      avgDirectionChange: 0,
      directionChangeVariance: 0,
      directionChangeStdDev: 0,
      peakCount: 0
    };
  }

  calculateMean(values) {
    if (values.length === 0) return 0;
    return values.reduce((sum, val) => sum + val, 0) / values.length;
  }

  calculateVariance(values, mean) {
    if (values.length < 2) return 0;
    const squaredDiffs = values.map(v => Math.pow(v - mean, 2));
    return squaredDiffs.reduce((sum, val) => sum + val, 0) / values.length;
  }

  calculateSmoothness(tracks) {
    if (tracks.length < 3) return 1;
    
    let totalAngleChange = 0;
    
    for (let i = 1; i < tracks.length - 1; i++) {
      const dx1 = tracks[i].x - tracks[i - 1].x;
      const dy1 = tracks[i].y - tracks[i - 1].y;
      const dx2 = tracks[i + 1].x - tracks[i].x;
      const dy2 = tracks[i + 1].y - tracks[i].y;
      
      const dot = dx1 * dx2 + dy1 * dy2;
      const mag1 = Math.sqrt(dx1 * dx1 + dy1 * dy1);
      const mag2 = Math.sqrt(dx2 * dx2 + dy2 * dy2);
      
      if (mag1 > 0 && mag2 > 0) {
        const cosAngle = Math.max(-1, Math.min(1, dot / (mag1 * mag2)));
        const angle = Math.acos(cosAngle);
        totalAngleChange += angle;
      }
    }
    
    const maxPossibleChange = (tracks.length - 2) * Math.PI;
    return maxPossibleChange > 0 ? 1 - (totalAngleChange / maxPossibleChange) : 1;
  }

  calculateJitter(tracks) {
    if (tracks.length < 3) return 0;
    
    let jitterSum = 0;
    let count = 0;
    
    for (let i = 2; i < tracks.length; i++) {
      const dx1 = tracks[i - 1].x - tracks[i - 2].x;
      const dy1 = tracks[i - 1].y - tracks[i - 2].y;
      const dx2 = tracks[i].x - tracks[i - 1].x;
      const dy2 = tracks[i].y - tracks[i - 1].y;
      
      const jitter = Math.sqrt(Math.pow(dx2 - dx1, 2) + Math.pow(dy2 - dy1, 2));
      jitterSum += jitter;
      count++;
    }
    
    return count > 0 ? jitterSum / count : 0;
  }

  countPeaks(values) {
    if (values.length < 3) return 0;
    
    let peakCount = 0;
    const threshold = this.calculateMean(values) * 1.5;
    
    for (let i = 1; i < values.length - 1; i++) {
      if (values[i] > values[i - 1] && 
          values[i] > values[i + 1] && 
          values[i] > threshold) {
        peakCount++;
      }
    }
    
    return peakCount;
  }

  calculateMechanicalScore(statistical) {
    let score = 0;
    let weightSum = 0;

    if (statistical.smoothness > 0.95) {
      score += 30;
      weightSum += 1;
    }

    if (statistical.jitter < 2) {
      score += 20;
      weightSum += 1;
    }

    if (statistical.velocityVariance < 100) {
      score += 25;
      weightSum += 1;
    }

    if (statistical.accelerationVariance < 50) {
      score += 25;
      weightSum += 1;
    }

    return weightSum > 0 ? score / weightSum : 0;
  }

  calculateHumanLikeness(statistical, dwellPoints) {
    let humanScore = 0;
    let totalWeight = 0;

    const smoothnessWeight = 2;
    humanScore += (1 - statistical.smoothness) * smoothnessWeight;
    totalWeight += smoothnessWeight;

    const jitterWeight = 1.5;
    const normalizedJitter = Math.min(statistical.jitter / 10, 1);
    humanScore += normalizedJitter * jitterWeight;
    totalWeight += jitterWeight;

    const dwellWeight = 1;
    if (dwellPoints.length > 0) {
      const avgDwell = this.calculateMean(dwellPoints.map(p => p.duration));
      const normalizedDwell = Math.min(avgDwell / 300, 1);
      humanScore += normalizedDwell * dwellWeight;
      totalWeight += dwellWeight;
    }

    return totalWeight > 0 ? humanScore / totalWeight : 0;
  }
}

module.exports = FeatureExtractor;
