module.exports = {
  messageQueue: {
    enabled: process.env.MQ_ALERTS_ENABLED !== 'false',
    channels: {
      email: process.env.MQ_ALERT_EMAIL || 'admin@hjtpx.com',
      slack: process.env.MQ_ALERT_SLACK_WEBHOOK || null,
      webhook: process.env.MQ_ALERT_WEBHOOK_URL || null
    },
    thresholds: {
      queueLength: {
        warning: parseInt(process.env.MQ_ALERT_QUEUE_LENGTH_WARN || '500'),
        critical: parseInt(process.env.MQ_ALERT_QUEUE_LENGTH || '1000')
      },
      processingTime: {
        warning: parseInt(process.env.MQ_ALERT_PROCESSING_TIME_WARN || '15000'),
        critical: parseInt(process.env.MQ_ALERT_PROCESSING_TIME || '30000')
      },
      failureRate: {
        warning: parseFloat(process.env.MQ_ALERT_FAILURE_RATE_WARN || '0.05'),
        critical: parseFloat(process.env.MQ_ALERT_FAILURE_RATE || '0.1')
      },
      dlqLength: {
        warning: parseInt(process.env.MQ_ALERT_DLQ_LENGTH_WARN || '10'),
        critical: parseInt(process.env.MQ_ALERT_DLQ_LENGTH || '50')
      },
      consumerLag: {
        warning: parseInt(process.env.MQ_ALERT_CONSUMER_LAG_WARN || '100'),
        critical: parseInt(process.env.MQ_ALERT_CONSUMER_LAG || '500')
      }
    },
    notifications: {
      onWarning: process.env.MQ_NOTIFY_ON_WARNING !== 'false',
      onCritical: true,
      cooldown: parseInt(process.env.MQ_ALERT_COOLDOWN || '300000')
    }
  },

  alertRules: [
    {
      name: 'queue_length_high',
      condition: 'queueLength > thresholds.queueLength.critical',
      severity: 'critical',
      message: 'Queue {{queue}} has excessive messages ({{value}})'
    },
    {
      name: 'queue_length_warning',
      condition: 'queueLength > thresholds.queueLength.warning',
      severity: 'warning',
      message: 'Queue {{queue}} message count elevated ({{value}})'
    },
    {
      name: 'processing_time_high',
      condition: 'processingTime > thresholds.processingTime.critical',
      severity: 'critical',
      message: 'Slow processing detected on {{queue}} ({{value}}ms)'
    },
    {
      name: 'failure_rate_high',
      condition: 'failureRate > thresholds.failureRate.critical',
      severity: 'critical',
      message: 'High failure rate on {{queue}} ({{value}}%)'
    },
    {
      name: 'dlq_messages',
      condition: 'dlqLength > thresholds.dlqLength.warning',
      severity: 'warning',
      message: 'Messages in DLQ for {{queue}} ({{value}})'
    },
    {
      name: 'no_consumers',
      condition: 'consumerCount === 0',
      severity: 'critical',
      message: 'No active consumers for {{queue}}'
    }
  ]
};
