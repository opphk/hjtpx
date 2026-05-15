const fs = require('fs');
const path = require('path');

const nodemailer = require('nodemailer');

class EmailService {
  constructor() {
    this.transporter = null;
    this.templates = new Map();
    this.initTransporter();
    this.loadTemplates();
  }

  initTransporter() {
    const config = {
      host: process.env.SMTP_HOST || 'smtp.example.com',
      port: parseInt(process.env.SMTP_PORT || '587'),
      secure: process.env.SMTP_SECURE === 'true',
      auth: {
        user: process.env.SMTP_USER,
        pass: process.env.SMTP_PASS
      }
    };

    if (config.auth.user && config.auth.pass) {
      this.transporter = nodemailer.createTransport(config);
      console.log('✅ Email service initialized with SMTP');
    } else {
      console.log('⚠️  Email service: SMTP credentials not configured, using mock mode');
      this.transporter = {
        sendMail: async () => {
          console.log('[Mock Email] Email would be sent');
          return { messageId: `mock_${Date.now()}` };
        }
      };
    }
  }

  loadTemplates() {
    const templatesDir = path.join(__dirname, '../../templates/email');

    if (fs.existsSync(templatesDir)) {
      const files = fs.readdirSync(templatesDir);
      files.forEach(file => {
        if (file.endsWith('.html') || file.endsWith('.txt')) {
          const templateName = path.basename(file, path.extname(file));
          const content = fs.readFileSync(path.join(templatesDir, file), 'utf8');
          this.templates.set(templateName, content);
        }
      });
    }

    this.templates.set('welcome', this.getDefaultTemplate('welcome'));
    this.templates.set('notification', this.getDefaultTemplate('notification'));
    this.templates.set('resetPassword', this.getDefaultTemplate('resetPassword'));
  }

  getDefaultTemplate(type) {
    const templates = {
      welcome: {
        subject: 'Welcome to HJTPX',
        html: '<h1>Welcome to HJTPX!</h1><p>Thank you for registering, {{username}}.</p><p>Get started by exploring our features.</p><a href="{{appUrl}}">Open App</a>',
        text: 'Welcome to HJTPX! Thank you for registering, {{username}}.'
      },
      notification: {
        subject: 'New Notification: {{title}}',
        html: '<h2>{{title}}</h2><p>{{message}}</p><small>Sent at {{timestamp}}</small>',
        text: '{{title}}\n\n{{message}}\n\nSent at {{timestamp}}'
      },
      resetPassword: {
        subject: 'Reset Your Password',
        html: '<h1>Password Reset Request</h1><p>Hi {{username}},</p><p>You requested a password reset. Click the link below:</p><a href="{{resetUrl}}">Reset Password</a><p>This link expires in {{expiresIn}}.</p><p>If you didn\'t request this, please ignore this email.</p>',
        text: 'Password Reset Request\n\nHi {{username}},\n\nYou requested a password reset. Visit: {{resetUrl}}\n\nThis link expires in {{expiresIn}}.'
      }
    };

    return templates[type] || { subject: '', html: '', text: '' };
  }

  renderTemplate(templateName, variables) {
    let template = this.templates.get(templateName);

    if (!template) {
      console.warn(`Template "${templateName}" not found, using notification template`);
      template = this.templates.get('notification');
    }

    let subject = template.subject;
    let html = template.html;
    let text = template.text;

    Object.entries(variables).forEach(([key, value]) => {
      const regex = new RegExp(`{{${key}}}`, 'g');
      subject = subject.replace(regex, value);
      html = html.replace(regex, value);
      text = text.replace(regex, value);
    });

    return { subject, html, text };
  }

  async sendEmail(to, templateName, variables = {}) {
    try {
      const { subject, html, text } = this.renderTemplate(templateName, {
        ...variables,
        timestamp: new Date().toISOString(),
        appUrl: process.env.APP_URL || 'http://localhost:3000'
      });

      const mailOptions = {
        from: `"HJTPX" <${process.env.SMTP_FROM || 'noreply@hjtpx.com'}>`,
        to,
        subject,
        text,
        html
      };

      const result = await this.transporter.sendMail(mailOptions);
      console.log(`✅ Email sent to ${to}: ${subject}`);
      return { success: true, messageId: result.messageId };
    } catch (error) {
      console.error(`❌ Failed to send email to ${to}:`, error);
      return { success: false, error: error.message };
    }
  }

  async sendBulkEmails(recipients, templateName, variables = {}) {
    const results = [];

    for (const recipient of recipients) {
      const result = await this.sendEmail(recipient.email, templateName, {
        ...variables,
        username: recipient.username || recipient.name
      });
      results.push({
        email: recipient.email,
        ...result
      });
    }

    return results;
  }

  async verifyConnection() {
    try {
      if (this.transporter && this.transporter.verify) {
        await this.transporter.verify();
        return { connected: true };
      }
      return { connected: true, mock: true };
    } catch (error) {
      return { connected: false, error: error.message };
    }
  }
}

const emailService = new EmailService();

module.exports = emailService;
