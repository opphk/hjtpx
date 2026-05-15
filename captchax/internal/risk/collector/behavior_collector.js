/**
 * @fileoverview 鼠标轨迹采集器
 * @description 采集鼠标移动速度、加速度、方向变化、停留时间等行为数据
 * @module captchax/internal/risk/collector/behavior_collector
 */

'use strict';

class BehaviorCollector {
  constructor(config = {}) {
    this.config = {
      sampleRate: config.sampleRate || 50,
      maxTrackPoints: config.maxTrackPoints || 500,
      minTrackPoints: config.minTrackPoints || 10,
      velocityThreshold: config.velocityThreshold || 2000,
      accelerationThreshold: config.accelerationThreshold || 5000,
      ...config
    };
    
    this.tracks = [];
    this.clickTimes = [];
    this.keyTimes = [];
    this.startTime = null;
    this.lastPoint = null;
    this.lastVelocity = null;
    this.lastTime = null;
    this.dwellPoints = [];
    this.isCollecting = false;
  }

  start(sessionId = '') {
    this.reset();
    this.startTime = Date.now();
    this.sessionId = sessionId;
    this.isCollecting = true;
    return this;
  }

  stop() {
    this.isCollecting = false;
    return this.getData();
  }

  reset() {
    this.tracks = [];
    this.clickTimes = [];
    this.keyTimes = [];
    this.startTime = null;
    this.lastPoint = null;
    this.lastVelocity = null;
    this.lastTime = null;
    this.dwellPoints = [];
  }

  trackMouseMove(x, y, timestamp = Date.now()) {
    if (!this.isCollecting || this.tracks.length >= this.config.maxTrackPoints) {
      return;
    }

    const point = { x, y, timestamp };
    
    if (this.lastPoint && this.lastTime) {
      const dt = timestamp - this.lastTime;
      
      if (dt > 0) {
        const dx = x - this.lastPoint.x;
        const dy = y - this.lastPoint.y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        
        const velocity = distance / dt * 1000;
        const acceleration = this.lastVelocity !== null 
          ? Math.abs(velocity - this.lastVelocity) / dt * 1000 
          : 0;
        
        const direction = Math.atan2(dy, dx);
        const directionChange = this.tracks.length > 0 && this.tracks[this.tracks.length - 1].direction !== undefined
          ? Math.abs(direction - this.tracks[this.tracks.length - 1].direction)
          : 0;

        point.velocity = velocity;
        point.acceleration = acceleration;
        point.direction = direction;
        point.directionChange = directionChange;
        point.distance = distance;
        point.dt = dt;

        this.lastVelocity = velocity;
        
        if (dt > 100) {
          this.dwellPoints.push({
            x,
            y,
            duration: dt,
            timestamp
          });
        }
      }
    }

    this.tracks.push(point);
    this.lastPoint = point;
    this.lastTime = timestamp;
  }

  trackClick(timestamp = Date.now()) {
    if (!this.isCollecting) return;
    this.clickTimes.push(timestamp);
  }

  trackKeyPress(key, timestamp = Date.now()) {
    if (!this.isCollecting) return;
    
    const keyData = {
      key,
      timestamp,
      interval: this.keyTimes.length > 0 
        ? timestamp - this.keyTimes[this.keyTimes.length - 1].timestamp 
        : 0
    };
    
    this.keyTimes.push(keyData);
    return keyData;
  }

  trackKeyRelease(key, timestamp = Date.now()) {
    if (!this.isCollecting) return;
    
    const keyDown = this.keyTimes.find(k => k.key === key && !k.releaseTime);
    if (keyDown) {
      keyDown.releaseTime = timestamp;
      keyDown.pressDuration = timestamp - keyDown.timestamp;
    }
  }

  getData() {
    const slideStart = this.startTime || Date.now();
    const slideEnd = this.lastTime || Date.now();

    return {
      sessionId: this.sessionId,
      mouseTracks: this.tracks,
      clickTimes: this.clickTimes,
      keyTimes: this.keyTimes,
      dwellPoints: this.dwellPoints,
      slideStart,
      slideEnd,
      slideDuration: slideEnd - slideStart,
      trackPointCount: this.tracks.length,
      clickCount: this.clickTimes.length,
      keyPressCount: this.keyTimes.length,
      statistics: this.calculateStatistics()
    };
  }

  calculateStatistics() {
    if (this.tracks.length < 2) {
      return {
        avgVelocity: 0,
        maxVelocity: 0,
        minVelocity: 0,
        velocityVariance: 0,
        avgAcceleration: 0,
        maxAcceleration: 0,
        avgDirectionChange: 0,
        totalDirectionChanges: 0,
        avgDwellDuration: 0,
        dwellPointCount: 0,
        totalDistance: 0,
        straightness: 0
      };
    }

    const velocities = this.tracks.map(t => t.velocity || 0).filter(v => v > 0);
    const accelerations = this.tracks.map(t => t.acceleration || 0).filter(a => a >= 0);
    const directionChanges = this.tracks.map(t => t.directionChange || 0);

    const avgVelocity = velocities.length > 0 
      ? velocities.reduce((a, b) => a + b, 0) / velocities.length 
      : 0;
    
    const maxVelocity = velocities.length > 0 ? Math.max(...velocities) : 0;
    const minVelocity = velocities.length > 0 ? Math.min(...velocities) : 0;
    
    const velocityVariance = this.calculateVariance(velocities, avgVelocity);
    const avgAcceleration = accelerations.length > 0
      ? accelerations.reduce((a, b) => a + b, 0) / accelerations.length
      : 0;
    const maxAcceleration = accelerations.length > 0 ? Math.max(...accelerations) : 0;

    const avgDirectionChange = directionChanges.length > 0
      ? directionChanges.reduce((a, b) => a + b, 0) / directionChanges.length
      : 0;
    const totalDirectionChanges = directionChanges.filter(dc => dc > Math.PI / 4).length;

    const totalDistance = this.tracks.reduce((sum, t) => sum + (t.distance || 0), 0);
    
    const startPoint = this.tracks[0];
    const endPoint = this.tracks[this.tracks.length - 1];
    const directDistance = startPoint && endPoint
      ? Math.sqrt(Math.pow(endPoint.x - startPoint.x, 2) + Math.pow(endPoint.y - startPoint.y, 2))
      : 0;
    const straightness = totalDistance > 0 ? directDistance / totalDistance : 0;

    const avgDwellDuration = this.dwellPoints.length > 0
      ? this.dwellPoints.reduce((sum, p) => sum + p.duration, 0) / this.dwellPoints.length
      : 0;

    return {
      avgVelocity,
      maxVelocity,
      minVelocity,
      velocityVariance,
      avgAcceleration,
      maxAcceleration,
      avgDirectionChange,
      totalDirectionChanges,
      avgDwellDuration,
      dwellPointCount: this.dwellPoints.length,
      totalDistance,
      straightness,
      smoothness: this.calculateSmoothness(),
      jitter: this.calculateJitter()
    };
  }

  calculateVariance(values, mean) {
    if (values.length < 2) return 0;
    const squaredDiffs = values.map(v => Math.pow(v - mean, 2));
    return squaredDiffs.reduce((a, b) => a + b, 0) / values.length;
  }

  calculateSmoothness() {
    if (this.tracks.length < 3) return 1;
    
    let totalAngleChange = 0;
    const angles = this.tracks.map(t => t.direction || 0);
    
    for (let i = 1; i < angles.length; i++) {
      let diff = Math.abs(angles[i] - angles[i - 1]);
      if (diff > Math.PI) diff = 2 * Math.PI - diff;
      totalAngleChange += diff;
    }
    
    const maxPossibleChange = (this.tracks.length - 1) * Math.PI;
    return maxPossibleChange > 0 ? 1 - (totalAngleChange / maxPossibleChange) : 1;
  }

  calculateJitter() {
    if (this.tracks.length < 3) return 0;
    
    let jitterSum = 0;
    let count = 0;
    
    for (let i = 2; i < this.tracks.length; i++) {
      const dx1 = this.tracks[i - 1].x - this.tracks[i - 2].x;
      const dy1 = this.tracks[i - 1].y - this.tracks[i - 2].y;
      const dx2 = this.tracks[i].x - this.tracks[i - 1].x;
      const dy2 = this.tracks[i].y - this.tracks[i - 1].y;
      
      const jitter = Math.sqrt(Math.pow(dx2 - dx1, 2) + Math.pow(dy2 - dy1, 2));
      jitterSum += jitter;
      count++;
    }
    
    return count > 0 ? jitterSum / count : 0;
  }

  getKeyboardRhythm() {
    if (this.keyTimes.length < 2) {
      return {
        avgInterval: 0,
        variance: 0,
        fastestInterval: 0,
        slowestInterval: 0,
        isMechanical: false,
        errorRate: 0,
        avgTypingSpeed: 0
      };
    }

    const intervals = this.keyTimes.map(k => k.interval).filter(i => i > 0);
    const avgInterval = intervals.length > 0
      ? intervals.reduce((a, b) => a + b, 0) / intervals.length
      : 0;
    const variance = this.calculateVariance(intervals, avgInterval);
    const fastestInterval = intervals.length > 0 ? Math.min(...intervals) : 0;
    const slowestInterval = intervals.length > 0 ? Math.max(...intervals) : 0;
    const isMechanical = variance < 500 && intervals.length > 5;

    const pressDurations = this.keyTimes
      .filter(k => k.pressDuration !== undefined)
      .map(k => k.pressDuration);
    const avgPressDuration = pressDurations.length > 0
      ? pressDurations.reduce((a, b) => a + b, 0) / pressDurations.length
      : 0;

    const errorIndicators = this.keyTimes.filter(k => 
      k.pressDuration < 30 || (k.interval > 0 && k.interval < 50)
    ).length;
    const errorRate = this.keyTimes.length > 0 ? errorIndicators / this.keyTimes.length : 0;

    return {
      avgInterval,
      variance,
      fastestInterval,
      slowestInterval,
      isMechanical,
      errorRate,
      avgTypingSpeed: avgInterval > 0 ? 60000 / avgInterval : 0,
      avgPressDuration
    };
  }

  isValid() {
    return this.tracks.length >= this.config.minTrackPoints;
  }

  static fromData(data) {
    const collector = new BehaviorCollector();
    collector.tracks = data.mouseTracks || [];
    collector.clickTimes = data.clickTimes || [];
    collector.keyTimes = data.keyTimes || [];
    collector.startTime = data.slideStart;
    collector.lastTime = data.slideEnd;
    collector.sessionId = data.sessionId;
    return collector;
  }
}

module.exports = BehaviorCollector;
