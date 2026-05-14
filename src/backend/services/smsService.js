class SMSService {
  constructor() {
    this.provider = process.env.SMS_PROVIDER || 'mock';
    this.apiKey = process.env.SMS_API_KEY;
    this.apiSecret = process.env.SMS_API_SECRET;
    this.senderId = process.env.SMS_SENDER_ID || 'HJTPX';
    this.templates = new Map();
    this.initTemplates();
  }

  initTemplates() {
    this.templates.set('verification', {
      id: process.env.SMS_TEMPLATE_VERIFICATION || 'VERIFICATION',
      content: 'Your HJTPX verification code is {{code}}. Valid for {{expiresIn}} minutes.'
    });

    this.templates.set('notification', {
      id: process.env.SMS_TEMPLATE_NOTIFICATION || 'NOTIFICATION',
      content: 'HJTPX: {{message}}'
    });

    this.templates.set('alert', {
      id: process.env.SMS_TEMPLATE_ALERT || 'ALERT',
      content: 'HJTPX Alert: {{message}} at {{timestamp}}'
    });
  }

  async sendSMS(phoneNumber, templateName, variables = {}) {
    try {
      const template = this.templates.get(templateName);
      if (!template) {
        throw new Error(`SMS template "${templateName}" not found`);
      }

      let content = template.content;
      Object.entries(variables).forEach(([key, value]) => {
        content = content.replace(new RegExp(`{{${key}}}`, 'g'), value);
      });

      if (this.provider === 'mock' || !this.apiKey) {
        console.log(`[Mock SMS] To: ${phoneNumber}, Content: ${content}`);
        return {
          success: true,
          messageId: `mock_sms_${Date.now()}`,
          provider: 'mock'
        };
      }

      const result = await this.sendViaProvider(phoneNumber, content, template.id);
      return result;
    } catch (error) {
      console.error(`❌ Failed to send SMS to ${phoneNumber}:`, error);
      return { success: false, error: error.message };
    }
  }

  async sendViaProvider(phoneNumber, content, templateId) {
    switch (this.provider) {
      case 'twilio':
        return await this.sendViaTwilio(phoneNumber, content, templateId);
      case 'nexmo':
        return await this.sendViaNexmo(phoneNumber, content, templateId);
      case 'aws-sns':
        return await this.sendViaAWSSNS(phoneNumber, content, templateId);
      default:
        console.log(`[SMS] Provider "${this.provider}" not implemented, using mock`);
        return {
          success: true,
          messageId: `mock_${Date.now()}`,
          provider: this.provider
        };
    }
  }

  async sendViaTwilio(phoneNumber, content, templateId) {
    const twilio = require('twilio');
    const client = twilio(this.apiKey, this.apiSecret);

    const message = await client.messages.create({
      body: content,
      from: this.senderId,
      to: phoneNumber
    });

    return {
      success: true,
      messageId: message.sid,
      provider: 'twilio'
    };
  }

  async sendViaNexmo(phoneNumber, content, templateId) {
    const { Vonage } = require('@vonage/server-sdk');
    const vonage = new Vonage({
      apiKey: this.apiKey,
      apiSecret: this.apiSecret
    });

    const result = await vonage.sms.send({ to: phoneNumber, from: this.senderId, text: content });

    return {
      success: result.messages[0]['status'] === '0',
      messageId: result.messages[0]['message-id'],
      provider: 'nexmo'
    };
  }

  async sendViaAWSSNS(phoneNumber, content, templateId) {
    const AWS = require('aws-sdk');
    const sns = new AWS.SNS({
      apiVersion: '2010-03-31',
      accessKeyId: this.apiKey,
      secretAccessKey: this.apiSecret,
      region: process.env.AWS_REGION || 'us-east-1'
    });

    const result = await sns.publish({
      PhoneNumber: phoneNumber,
      Message: content,
      MessageAttributes: {
        'AWS.SNS.SMS.SenderID': {
          DataType: 'String',
          StringValue: this.senderId
        }
      }
    }).promise();

    return {
      success: true,
      messageId: result.MessageId,
      provider: 'aws-sns'
    };
  }

  async sendBulkSMS(recipients, templateName, variables = {}) {
    const results = [];

    for (const recipient of recipients) {
      const result = await this.sendSMS(recipient.phone, templateName, {
        ...variables,
        name: recipient.name || recipient.username
      });
      results.push({
        phone: recipient.phone,
        ...result
      });
    }

    return results;
  }

  async validatePhoneNumber(phoneNumber) {
    const phoneRegex = /^[+]?[(]?[0-9]{1,4}[)]?[-\s./0-9]*$/;
    return phoneRegex.test(phoneNumber);
  }

  async getBalance() {
    if (this.provider === 'mock' || !this.apiKey) {
      return { balance: 0, currency: 'USD', mock: true };
    }

    switch (this.provider) {
      case 'twilio':
        const twilio = require('twilio');
        const client = twilio(this.apiKey, this.apiSecret);
        const balance = await client.balance.fetch();
        return { balance: parseFloat(balance.balance), currency: balance.currency };

      default:
        return { balance: 0, currency: 'USD', mock: true };
    }
  }
}

const smsService = new SMSService();

module.exports = smsService;
