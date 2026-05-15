const { consumerManager } = require('./consumers/streamConsumer');
const { producerManager } = require('./producers/streamProducer');

class EmailQueueService {
  constructor() {
    this.queueName = 'email';
  }

  async sendEmail(to, templateName, variables = {}, options = {}) {
    return await producerManager.send(
      this.queueName,
      {
        to,
        templateName,
        variables,
        options
      },
      {
        type: 'send_email',
        priority: options.priority || 0,
        correlationId: options.correlationId
      }
    );
  }

  async sendBulkEmails(recipients, templateName, variables = {}, options = {}) {
    const messages = recipients.map(recipient => ({
      to: recipient.email,
      templateName,
      variables: { ...variables, ...recipient.variables },
      options
    }));

    return await producerManager.sendBatch(
      this.queueName,
      messages.map(msg => ({
        ...msg,
        type: 'send_email'
      })),
      options
    );
  }

  async scheduleEmail(to, templateName, variables = {}, scheduledTime, options = {}) {
    const delay = new Date(scheduledTime).getTime() - Date.now();
    if (delay <= 0) {
      throw new Error('Scheduled time must be in the future');
    }

    return await producerManager.sendWithDelay(
      this.queueName,
      {
        to,
        templateName,
        variables,
        options
      },
      delay,
      {
        type: 'send_email',
        priority: options.priority || 0,
        correlationId: options.correlationId
      }
    );
  }

  async sendWelcomeEmail(user) {
    return await this.sendEmail(
      user.email,
      'welcome',
      {
        username: user.username,
        appUrl: process.env.APP_URL || 'http://localhost:3000'
      },
      { priority: 5 }
    );
  }

  async sendPasswordResetEmail(user, resetToken) {
    return await this.sendEmail(
      user.email,
      'resetPassword',
      {
        username: user.username,
        resetUrl: `${process.env.APP_URL || 'http://localhost:3000'}/reset-password?token=${resetToken}`,
        expiresIn: '1 hour'
      },
      { priority: 10 }
    );
  }

  async sendNotificationEmail(user, notification) {
    return await this.sendEmail(
      user.email,
      'notification',
      {
        title: notification.title,
        message: notification.message,
        timestamp: new Date().toISOString()
      },
      { priority: 1 }
    );
  }

  async startConsumer(options = {}) {
    const consumer = await consumerManager.createConsumer(this.queueName, options);

    consumer.registerHandler('send_email', async message => {
      const emailService = require('../emailService');

      await emailService.sendEmail(
        message.payload.to,
        message.payload.templateName,
        message.payload.variables
      );

      console.log(`[EmailQueue] Email sent to ${message.payload.to}`);
    });

    consumer.registerHandler('send_bulk_email', async message => {
      const emailService = require('../emailService');

      const results = await emailService.sendBulkEmails(
        message.payload.recipients,
        message.payload.templateName,
        message.payload.variables
      );

      console.log(`[EmailQueue] Bulk email sent to ${results.length} recipients`);
    });

    return consumer;
  }
}

const emailQueueService = new EmailQueueService();

module.exports = emailQueueService;
