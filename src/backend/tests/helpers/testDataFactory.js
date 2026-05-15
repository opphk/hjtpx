const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');
const pool = require('../../../config/database/db');

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';
const SALT_ROUNDS = 10;

class TestDataFactory {
  constructor() {
    this.createdEntities = [];
  }

  async createUser(overrides = {}) {
    const defaultUser = {
      email: `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}@example.com`,
      name: `Test User ${Date.now()}`,
      password: 'TestPassword123!',
      role: 'user'
    };

    const userData = { ...defaultUser, ...overrides };
    const hashedPassword = await bcrypt.hash(userData.password, SALT_ROUNDS);

    const result = await pool.query(
      'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4) RETURNING id, email, name, role, created_at',
      [userData.email, userData.name, hashedPassword, userData.role]
    );

    const user = result.rows[0];
    this.createdEntities.push({ type: 'user', id: user.id });

    return {
      ...user,
      plainPassword: userData.password
    };
  }

  async createAdminUser(overrides = {}) {
    return this.createUser({ role: 'admin', ...overrides });
  }

  async createRegularUser(overrides = {}) {
    return this.createUser({ role: 'user', ...overrides });
  }

  async createToken(user) {
    return jwt.sign(
      { id: user.id, email: user.email, role: user.role },
      JWT_SECRET,
      { expiresIn: '1h' }
    );
  }

  async createExpiredToken(user) {
    return jwt.sign(
      { id: user.id, email: user.email, role: user.role },
      JWT_SECRET,
      { expiresIn: '-1h' }
    );
  }

  async createFile(userId, overrides = {}) {
    const defaultFile = {
      original_name: `file_${Date.now()}.txt`,
      storage_path: `/uploads/file_${Date.now()}.txt`,
      mime_type: 'text/plain',
      size: Math.floor(Math.random() * 10000) + 100,
      folder: 'test'
    };

    const fileData = { ...defaultFile, ...overrides };

    const result = await pool.query(
      'INSERT INTO files (user_id, original_name, storage_path, mime_type, size, folder) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, user_id, original_name, storage_path, mime_type, size, folder, created_at',
      [userId, fileData.original_name, fileData.storage_path, fileData.mime_type, fileData.size, fileData.folder]
    );

    const file = result.rows[0];
    this.createdEntities.push({ type: 'file', id: file.id });

    return file;
  }

  async createNotification(userId, overrides = {}) {
    const defaultNotification = {
      title: `Notification ${Date.now()}`,
      message: `Test notification message`,
      type: 'info',
      channels: ['in_app'],
      status: 'unread'
    };

    const notificationData = { ...defaultNotification, ...overrides };

    try {
      const result = await pool.query(
        'INSERT INTO notifications (user_id, title, message, type, channels, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, user_id, title, message, type, status, created_at',
        [userId, notificationData.title, notificationData.message, notificationData.type, JSON.stringify(notificationData.channels), notificationData.status]
      );

      const notification = result.rows[0];
      this.createdEntities.push({ type: 'notification', id: notification.id });

      return notification;
    } catch (error) {
      console.log('Could not create notification:', error.message);
      return null;
    }
  }

  async createSession(userId, overrides = {}) {
    const defaultSession = {
      token: `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      ip_address: '127.0.0.1',
      user_agent: 'Test User Agent'
    };

    const sessionData = { ...defaultSession, ...overrides };

    const result = await pool.query(
      'INSERT INTO sessions (user_id, token, ip_address, user_agent) VALUES ($1, $2, $3, $4) RETURNING id, user_id, token, ip_address, created_at',
      [userId, sessionData.token, sessionData.ip_address, sessionData.user_agent]
    );

    const session = result.rows[0];
    this.createdEntities.push({ type: 'session', id: session.id });

    return session;
  }

  async createUserWithToken(overrides = {}) {
    const user = await this.createUser(overrides);
    const token = await this.createToken(user);

    return { user, token };
  }

  async createAdminWithToken(overrides = {}) {
    return this.createUserWithToken({ role: 'admin', ...overrides });
  }

  async cleanup() {
    for (const entity of this.createdEntities.reverse()) {
      try {
        switch (entity.type) {
          case 'user':
            await pool.query('DELETE FROM users WHERE id = $1', [entity.id]);
            break;
          case 'file':
            await pool.query('DELETE FROM files WHERE id = $1', [entity.id]);
            break;
          case 'notification':
            await pool.query('DELETE FROM notifications WHERE id = $1', [entity.id]);
            break;
          case 'session':
            await pool.query('DELETE FROM sessions WHERE id = $1', [entity.id]);
            break;
        }
      } catch (error) {
        console.log(`Cleanup failed for ${entity.type} ${entity.id}:`, error.message);
      }
    }

    this.createdEntities = [];
  }
}

const factory = new TestDataFactory();

module.exports = {
  TestDataFactory,
  factory
};
