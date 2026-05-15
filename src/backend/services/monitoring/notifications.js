const notificationService = {
  channels: {
    email: [],
    sms: [],
    webhook: []
  },
  
  // 添加邮件订阅
  subscribeEmail(email, alertIds = []) {
    this.channels.email.push({
      email,
      alertIds: alertIds.length > 0 ? alertIds : ['*'], // * 表示订阅所有
      createdAt: new Date().toISOString()
    });
    return true;
  },
  
  // 添加短信订阅
  subscribeSMS(phone, alertIds = []) {
    this.channels.sms.push({
      phone,
      alertIds: alertIds.length > 0 ? alertIds : ['*'],
      createdAt: new Date().toISOString()
    });
    return true;
  },
  
  // 添加Webhook订阅
  subscribeWebhook(url, alertIds = [], headers = {}) {
    this.channels.webhook.push({
      url,
      alertIds: alertIds.length > 0 ? alertIds : ['*'],
      headers,
      createdAt: new Date().toISOString()
    });
    return true;
  },
  
  // 发送通知
  async sendNotification(alert) {
    const promises = [];
    
    // 邮件通知
    const emailSubscribers = this.channels.email.filter(sub => 
      sub.alertIds.includes('*') || sub.alertIds.includes(alert.ruleId)
    );
    
    for (const subscriber of emailSubscribers) {
      promises.push(this.sendEmail(subscriber.email, alert));
    }
    
    // 短信通知
    const smsSubscribers = this.channels.sms.filter(sub =>
      sub.alertIds.includes('*') || sub.alertIds.includes(alert.ruleId)
    );
    
    for (const subscriber of smsSubscribers) {
      promises.push(this.sendSMS(subscriber.phone, alert));
    }
    
    // Webhook通知
    const webhookSubscribers = this.channels.webhook.filter(sub =>
      sub.alertIds.includes('*') || sub.alertIds.includes(alert.ruleId)
    );
    
    for (const subscriber of webhookSubscribers) {
      promises.push(this.sendWebhook(subscriber.url, alert, subscriber.headers));
    }
    
    return Promise.allSettled(promises);
  },
  
  // 发送邮件
  async sendEmail(email, alert) {
    console.log(`[EMAIL] Sending alert to ${email}:`, alert);
    
    const subject = `[${alert.severity.toUpperCase()}] ${alert.name}`;
    const body = this.formatEmailBody(alert);
    
    // 实际发送邮件逻辑
    // await emailClient.send({ to: email, subject, body });
    
    return { success: true, channel: 'email', recipient: email };
  },
  
  // 发送短信
  async sendSMS(phone, alert) {
    console.log(`[SMS] Sending alert to ${phone}:`, alert);
    
    const message = this.formatSMSMessage(alert);
    
    // 实际发送短信逻辑
    // await smsClient.send({ to: phone, message });
    
    return { success: true, channel: 'sms', recipient: phone };
  },
  
  // 发送Webhook
  async sendWebhook(url, alert, headers = {}) {
    console.log(`[WEBHOOK] Sending alert to ${url}:`, alert);
    
    const payload = {
      alert,
      timestamp: new Date().toISOString(),
      source: 'HJTPX Monitoring'
    };
    
    // 实际发送Webhook逻辑
    // await fetch(url, {
    //   method: 'POST',
    //   headers: { 'Content-Type': 'application/json', ...headers },
    //   body: JSON.stringify(payload)
    // });
    
    return { success: true, channel: 'webhook', recipient: url };
  },
  
  formatEmailBody(alert) {
    return `
Alert: ${alert.name}
Severity: ${alert.severity.toUpperCase()}
Description: ${alert.description}
Value: ${alert.value}
Threshold: ${alert.threshold}
Triggered At: ${alert.triggeredAt}

Please take action to resolve this issue.

---
HJTPX Monitoring System
    `.trim();
  },
  
  formatSMSMessage(alert) {
    return `[${alert.severity.toUpperCase()}] ${alert.name}: ${alert.description}`;
  }
};

module.exports = notificationService;
