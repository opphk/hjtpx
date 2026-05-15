/**
 * @fileoverview 风控数据采集管理器
 * @description 统一管理所有采集器，实现数据聚合、格式化和缓存
 * @module captchax/internal/risk/collector/collector_manager
 */

'use strict';

const BehaviorCollector = require('./behavior_collector');
const DeviceFingerprint = require('./device_fingerprint');

class CollectorManager {
  constructor(config = {}) {
    this.config = {
      cacheEnabled: config.cacheEnabled !== false,
      cacheTTL: config.cacheTTL || 300000,
      maxCacheSize: config.maxCacheSize || 1000,
      autoCollect: config.autoCollect !== false,
      ...config
    };

    this.behaviorCollector = new BehaviorCollector(config.behaviorCollector);
    this.deviceFingerprint = new DeviceFingerprint(config.deviceFingerprint);
    
    this.cache = new Map();
    this.sessions = new Map();
    this.eventListeners = new Map();
    this.isAutoCollecting = false;
  }

  initialize(element = null, sessionId = null) {
    const session = sessionId || this.generateSessionId();
    
    this.behaviorCollector.start(session);
    
    if (this.config.autoCollect) {
      this.startAutoCollection(element);
    }

    this.sessions.set(session, {
      id: session,
      startTime: Date.now(),
      lastActivity: Date.now(),
      data: null
    });

    return session;
  }

  generateSessionId() {
    const timestamp = Date.now().toString(36);
    const randomPart = Math.random().toString(36).substring(2, 15);
    const randomPart2 = Math.random().toString(36).substring(2, 8);
    return `captchax_${timestamp}_${randomPart}_${randomPart2}`;
  }

  startAutoCollection(element = null) {
    if (this.isAutoCollecting) return;

    const target = element || document;
    
    this.mouseMoveHandler = (e) => {
      this.behaviorCollector.trackMouseMove(e.clientX, e.clientY, Date.now());
      this.updateSessionActivity();
    };
    
    this.clickHandler = (e) => {
      this.behaviorCollector.trackClick(Date.now());
      this.updateSessionActivity();
    };
    
    this.keyDownHandler = (e) => {
      this.behaviorCollector.trackKeyPress(e.key, Date.now());
      this.updateSessionActivity();
    };
    
    this.keyUpHandler = (e) => {
      this.behaviorCollector.trackKeyRelease(e.key, Date.now());
      this.updateSessionActivity();
    };

    target.addEventListener('mousemove', this.mouseMoveHandler, { passive: true });
    target.addEventListener('click', this.clickHandler);
    target.addEventListener('keydown', this.keyDownHandler);
    target.addEventListener('keyup', this.keyUpHandler);

    this.isAutoCollecting = true;
  }

  stopAutoCollection(element = null) {
    if (!this.isAutoCollecting) return;

    const target = element || document;
    
    if (this.mouseMoveHandler) {
      target.removeEventListener('mousemove', this.mouseMoveHandler);
    }
    if (this.clickHandler) {
      target.removeEventListener('click', this.clickHandler);
    }
    if (this.keyDownHandler) {
      target.removeEventListener('keydown', this.keyDownHandler);
    }
    if (this.keyUpHandler) {
      target.removeEventListener('keyup', this.keyUpHandler);
    }

    this.isAutoCollecting = false;
  }

  updateSessionActivity() {
    const session = this.sessions.get(this.behaviorCollector.sessionId);
    if (session) {
      session.lastActivity = Date.now();
    }
  }

  trackMouseMove(x, y, timestamp = Date.now()) {
    this.behaviorCollector.trackMouseMove(x, y, timestamp);
    this.updateSessionActivity();
  }

  trackClick(timestamp = Date.now()) {
    this.behaviorCollector.trackClick(timestamp);
    this.updateSessionActivity();
  }

  trackKeyPress(key, timestamp = Date.now()) {
    const keyData = this.behaviorCollector.trackKeyPress(key, timestamp);
    this.updateSessionActivity();
    return keyData;
  }

  async collect(sessionId = null) {
    const targetSessionId = sessionId || this.behaviorCollector.sessionId;
    
    if (!targetSessionId) {
      throw new Error('No active session. Call initialize() first.');
    }

    const cached = this.getFromCache(targetSessionId);
    if (cached && this.config.cacheEnabled) {
      return cached;
    }

    const behaviorData = this.behaviorCollector.stop();
    const fingerprint = await this.deviceFingerprint.collect();

    const result = {
      sessionId: targetSessionId,
      timestamp: Date.now(),
      behavior: behaviorData,
      fingerprint: fingerprint,
      metadata: this.getMetadata(),
      isValid: this.validateData(behaviorData)
    };

    if (this.config.cacheEnabled) {
      this.addToCache(targetSessionId, result);
    }

    this.emit('dataCollected', result);

    return result;
  }

  getMetadata() {
    const session = this.sessions.get(this.behaviorCollector.sessionId);
    
    return {
      collectDuration: session ? Date.now() - session.startTime : 0,
      idleTime: session ? Date.now() - session.lastActivity : 0,
      cacheSize: this.cache.size,
      activeSessions: this.sessions.size
    };
  }

  validateData(behaviorData) {
    if (!behaviorData) return false;
    
    const isValidTrackLength = behaviorData.trackPointCount >= this.behaviorCollector.config.minTrackPoints;
    const isValidDuration = behaviorData.slideDuration > 0;
    
    return isValidTrackLength && isValidDuration;
  }

  getFromCache(key) {
    if (!this.config.cacheEnabled) return null;
    
    const cached = this.cache.get(key);
    if (!cached) return null;

    if (Date.now() - cached.timestamp > this.config.cacheTTL) {
      this.cache.delete(key);
      return null;
    }

    return cached.data;
  }

  addToCache(key, data) {
    if (!this.config.cacheEnabled) return;

    if (this.cache.size >= this.maxCacheSize) {
      const oldestKey = this.cache.keys().next().value;
      this.cache.delete(oldestKey);
    }

    this.cache.set(key, {
      timestamp: Date.now(),
      data
    });
  }

  clearCache() {
    this.cache.clear();
  }

  clearSession(sessionId = null) {
    const targetSessionId = sessionId || this.behaviorCollector.sessionId;
    
    if (targetSessionId) {
      this.cache.delete(targetSessionId);
      this.sessions.delete(targetSessionId);
    }
    
    this.behaviorCollector.reset();
    this.stopAutoCollection();
  }

  on(event, callback) {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event).push(callback);
    return this;
  }

  off(event, callback) {
    if (!this.eventListeners.has(event)) return this;
    
    const listeners = this.eventListeners.get(event);
    const index = listeners.indexOf(callback);
    if (index > -1) {
      listeners.splice(index, 1);
    }
    return this;
  }

  emit(event, data) {
    if (!this.eventListeners.has(event)) return;
    
    const listeners = this.eventListeners.get(event);
    for (const listener of listeners) {
      try {
        listener(data);
      } catch (error) {
        console.error(`Error in event listener for ${event}:`, error);
      }
    }
  }

  async getBehaviorData(sessionId = null) {
    const collector = this.behaviorCollector;
    const originalSessionId = collector.sessionId;
    
    if (sessionId && sessionId !== originalSessionId) {
      const cached = this.getFromCache(sessionId);
      if (cached) {
        return cached.behavior;
      }
    }
    
    return collector.getData();
  }

  async getFingerprint() {
    return await this.deviceFingerprint.collect();
  }

  async getFullData() {
    return await this.collect();
  }

  getActiveSession() {
    return this.behaviorCollector.sessionId;
  }

  getSessionInfo(sessionId = null) {
    const targetSessionId = sessionId || this.behaviorCollector.sessionId;
    return this.sessions.get(targetSessionId);
  }

  getCacheStats() {
    return {
      size: this.cache.size,
      maxSize: this.config.maxCacheSize,
      ttl: this.config.cacheTTL,
      entries: Array.from(this.cache.keys())
    };
  }

  destroy() {
    this.stopAutoCollection();
    this.clearCache();
    this.sessions.clear();
    this.eventListeners.clear();
    this.behaviorCollector.reset();
    this.deviceFingerprint.clearCache();
  }
}

module.exports = CollectorManager;
