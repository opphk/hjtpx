const { factory } = require('./testDataFactory');

describe('Test Data Factory', () => {
  afterEach(async () => {
    await factory.cleanup();
  });

  describe('createUser', () => {
    it('should create a user with default values', async () => {
      const user = await factory.createUser();

      expect(user).toHaveProperty('id');
      expect(user).toHaveProperty('email');
      expect(user).toHaveProperty('name');
      expect(user).toHaveProperty('role', 'user');
      expect(user).toHaveProperty('plainPassword', 'TestPassword123!');
      expect(user.email).toMatch(/@example\.com$/);
    });

    it('should create a user with custom values', async () => {
      const user = await factory.createUser({
        email: 'custom@example.com',
        name: 'Custom User',
        role: 'admin'
      });

      expect(user.email).toBe('custom@example.com');
      expect(user.name).toBe('Custom User');
      expect(user.role).toBe('admin');
    });

    it('should create an admin user', async () => {
      const user = await factory.createAdminUser();

      expect(user.role).toBe('admin');
    });

    it('should create a regular user', async () => {
      const user = await factory.createRegularUser();

      expect(user.role).toBe('user');
    });
  });

  describe('createToken', () => {
    it('should create a valid JWT token', async () => {
      const user = await factory.createUser();
      const token = await factory.createToken(user);

      expect(typeof token).toBe('string');
      expect(token.split('.')).toHaveLength(3);
    });

    it('should create an expired token', async () => {
      const user = await factory.createUser();
      const token = await factory.createExpiredToken(user);

      expect(typeof token).toBe('string');
      expect(token.split('.')).toHaveLength(3);
    });
  });

  describe('createUserWithToken', () => {
    it('should create a user and token together', async () => {
      const { user, token } = await factory.createUserWithToken();

      expect(user).toHaveProperty('id');
      expect(typeof token).toBe('string');
      expect(token.split('.')).toHaveLength(3);
    });
  });

  describe('createFile', () => {
    it('should create a file with default values', async () => {
      const user = await factory.createUser();
      const file = await factory.createFile(user.id);

      expect(file).toHaveProperty('id');
      expect(file).toHaveProperty('user_id', user.id);
      expect(file).toHaveProperty('original_name');
      expect(file).toHaveProperty('mime_type', 'text/plain');
      expect(file).toHaveProperty('size');
      expect(file).toHaveProperty('folder', 'test');
    });

    it('should create a file with custom values', async () => {
      const user = await factory.createUser();
      const file = await factory.createFile(user.id, {
        original_name: 'custom.txt',
        mime_type: 'application/pdf',
        folder: 'documents'
      });

      expect(file.original_name).toBe('custom.txt');
      expect(file.mime_type).toBe('application/pdf');
      expect(file.folder).toBe('documents');
    });
  });

  describe('createNotification', () => {
    it('should create a notification', async () => {
      const user = await factory.createUser();
      const notification = await factory.createNotification(user.id);

      if (notification) {
        expect(notification).toHaveProperty('id');
        expect(notification).toHaveProperty('user_id', user.id);
        expect(notification).toHaveProperty('title');
        expect(notification).toHaveProperty('message');
        expect(notification).toHaveProperty('type', 'info');
      }
    });

    it('should create a notification with custom values', async () => {
      const user = await factory.createUser();
      const notification = await factory.createNotification(user.id, {
        title: 'Custom Title',
        message: 'Custom message',
        type: 'warning',
        status: 'read'
      });

      if (notification) {
        expect(notification.title).toBe('Custom Title');
        expect(notification.message).toBe('Custom message');
        expect(notification.type).toBe('warning');
      }
    });
  });

  describe('createSession', () => {
    it('should create a session', async () => {
      const user = await factory.createUser();
      const session = await factory.createSession(user.id);

      expect(session).toHaveProperty('id');
      expect(session).toHaveProperty('user_id', user.id);
      expect(session).toHaveProperty('token');
      expect(session).toHaveProperty('ip_address', '127.0.0.1');
    });
  });

  describe('cleanup', () => {
    it('should clean up all created entities', async () => {
      const user = await factory.createUser();
      const file = await factory.createFile(user.id);
      const session = await factory.createSession(user.id);

      expect(factory.createdEntities).toHaveLength(3);

      await factory.cleanup();

      expect(factory.createdEntities).toHaveLength(0);
    });
  });
});
