const express = require('express');
const router = express.Router();
const monitoringService = require('../services/monitoring');
const alertRules = require('../services/monitoring/alertRules');
const alertHistory = require('../services/monitoring/alertHistory');
const notificationService = require('../services/monitoring/notifications');

// 获取实时统计
router.get('/stats/realtime', (req, res) => {
  try {
    const stats = monitoringService.getRealtimeStats();
    res.json({ success: true, data: stats });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 获取指标时间序列
router.get('/metrics/:name/timeseries', (req, res) => {
  try {
    const { name } = req.params;
    const { interval } = req.query;
    const data = monitoringService.getTimeSeriesData(name, parseInt(interval) || 60000);
    res.json({ success: true, data });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 获取告警规则列表
router.get('/alerts/rules', (req, res) => {
  try {
    const rules = alertRules.getAllRules();
    res.json({ success: true, data: rules });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 创建告警规则
router.post('/alerts/rules', (req, res) => {
  try {
    const rule = alertRules.addRule(req.body);
    res.status(201).json({ success: true, data: rule });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 更新告警规则
router.put('/alerts/rules/:id', (req, res) => {
  try {
    const rule = alertRules.updateRule(req.params.id, req.body);
    res.json({ success: true, data: rule });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 删除告警规则
router.delete('/alerts/rules/:id', (req, res) => {
  try {
    alertRules.deleteRule(req.params.id);
    res.json({ success: true });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 获取告警历史
router.get('/alerts/history', (req, res) => {
  try {
    const history = alertHistory.getHistory(req.query);
    res.json({ success: true, data: history });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 确认告警
router.post('/alerts/:id/acknowledge', (req, res) => {
  try {
    const { userId, note } = req.body;
    const alert = alertHistory.acknowledgeAlert(req.params.id, userId, note);
    res.json({ success: true, data: alert });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 解决告警
router.post('/alerts/:id/resolve', (req, res) => {
  try {
    const { userId, note } = req.body;
    const alert = alertHistory.resolveAlert(req.params.id, userId, note);
    res.json({ success: true, data: alert });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 获取告警统计
router.get('/alerts/statistics', (req, res) => {
  try {
    const timeRange = parseInt(req.query.timeRange) || 86400000;
    const stats = alertHistory.getStatistics(timeRange);
    res.json({ success: true, data: stats });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

// 订阅通知
router.post('/subscriptions', (req, res) => {
  try {
    const { type, target, alertIds } = req.body;
    
    switch (type) {
      case 'email':
        notificationService.subscribeEmail(target, alertIds);
        break;
      case 'sms':
        notificationService.subscribeSMS(target, alertIds);
        break;
      case 'webhook':
        notificationService.subscribeWebhook(target, alertIds);
        break;
      default:
        return res.status(400).json({ success: false, error: 'Invalid subscription type' });
    }
    
    res.status(201).json({ success: true });
  } catch (error) {
    res.status(500).json({ success: false, error: error.message });
  }
});

module.exports = router;
