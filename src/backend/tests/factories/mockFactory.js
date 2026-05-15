const bcrypt = require('bcryptjs');
const { v4: uuidv4 } = require('uuid');

const testUsers = new Map();
const testFiles = new Map();
const testNotifications = [];

const defaultUserAttributes = {
  name: 'Test User',
  role: 'user',
  status: 'active'
};

async function createUser(overrides = {}) {
  const attributes = { ...defaultUserAttributes, ...overrides };

  if (!attributes.email) {
    attributes.email = `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}@example.com`;
  }

  if (!attributes.password) {
    attributes.password = await bcrypt.hash('TestPassword123!', 10);
  } else if (!attributes.password.startsWith('$2')) {
    attributes.password = await bcrypt.hash(attributes.password, 10);
  }

  const user = {
    id: Math.floor(Math.random() * 1000000) + 1,
    email: attributes.email,
    name: attributes.name,
    password: attributes.password,
    role: attributes.role,
    status: attributes.status,
    created_at: new Date()
  };

  testUsers.set(user.id, user);

  return user;
}

async function createAdmin(overrides = {}) {
  return createUser({
    ...overrides,
    role: 'admin',
    name: overrides.name || 'Admin User'
  });
}

async function createMultipleUsers(count, overrides = {}) {
  const users = [];
  for (let i = 0; i < count; i++) {
    const user = await createUser({
      ...overrides,
      email: `user_${Date.now()}_${i}_${Math.random().toString(36).substr(2, 9)}@example.com`
    });
    users.push(user);
  }
  return users;
}

async function deleteUser(userId) {
  testUsers.delete(userId);
}

async function deleteUsers(userIds) {
  for (const userId of userIds) {
    await deleteUser(userId);
  }
}

const defaultFileAttributes = {
  filename: 'test-file.txt',
  originalName: 'test-file.txt',
  mimeType: 'text/plain',
  size: 1024,
  folder: 'test'
};

async function createFile(userId, overrides = {}) {
  const attributes = { ...defaultFileAttributes, ...overrides };

  const file = {
    id: Math.floor(Math.random() * 1000000) + 1,
    user_id: userId,
    filename: attributes.filename,
    original_name: attributes.originalName,
    mime_type: attributes.mimeType,
    size: attributes.size,
    folder: attributes.folder,
    file_path: `/uploads/${attributes.folder}/${attributes.filename}`,
    created_at: new Date()
  };

  testFiles.set(file.id, file);

  return file;
}

async function createMultipleFiles(userId, count, overrides = {}) {
  const files = [];
  for (let i = 0; i < count; i++) {
    const file = await createFile(userId, {
      ...overrides,
      filename: `test-file-${i}-${Date.now()}.txt`,
      originalName: `test-file-${i}.txt`
    });
    files.push(file);
  }
  return files;
}

async function deleteFile(fileId) {
  testFiles.delete(fileId);
}

async function deleteFiles(fileIds) {
  for (const fileId of fileIds) {
    await deleteFile(fileId);
  }
}

async function deleteUserFiles(userId) {
  for (const [id, file] of testFiles) {
    if (file.user_id === userId) {
      testFiles.delete(id);
    }
  }
}

const defaultNotificationAttributes = {
  type: 'info',
  title: 'Test Notification',
  message: 'This is a test notification message',
  status: 'unread',
  channels: ['in_app'],
  priority: 'medium'
};

async function createNotification(userId, overrides = {}) {
  const attributes = {
    ...defaultNotificationAttributes,
    userId,
    ...overrides
  };

  const notification = {
    _id: new (require('mongoose').Types.ObjectId)(),
    userId: attributes.userId,
    title: attributes.title,
    message: attributes.message,
    type: attributes.type,
    status: attributes.status,
    channels: attributes.channels,
    priority: attributes.priority,
    createdAt: new Date(),
    updatedAt: new Date()
  };

  testNotifications.push(notification);
  return notification;
}

async function createMultipleNotifications(userId, count, overrides = {}) {
  const notifications = [];
  for (let i = 0; i < count; i++) {
    const notification = await createNotification(userId, {
      ...overrides,
      title: `${overrides.title || 'Test Notification'} ${i + 1}`
    });
    notifications.push(notification);
  }
  return notifications;
}

async function createReadNotification(userId, overrides = {}) {
  return createNotification(userId, {
    ...overrides,
    status: 'read'
  });
}

async function createUnreadNotification(userId, overrides = {}) {
  return createNotification(userId, {
    ...overrides,
    status: 'unread'
  });
}

async function deleteNotification(notificationId) {
  const index = testNotifications.findIndex(n => n._id.toString() === notificationId);
  if (index !== -1) {
    testNotifications.splice(index, 1);
  }
}

async function deleteNotifications(notificationIds) {
  for (const notificationId of notificationIds) {
    await deleteNotification(notificationId);
  }
}

async function deleteUserNotifications(userId) {
  const filtered = testNotifications.filter(n => n.userId !== userId);
  testNotifications.length = 0;
  testNotifications.push(...filtered);
}

const NOTIFICATION_TYPES = ['info', 'success', 'warning', 'error'];
const NOTIFICATION_PRIORITIES = ['low', 'medium', 'high', 'urgent'];

module.exports = {
  userFactory: {
    createUser,
    createAdmin,
    createMultipleUsers,
    deleteUser,
    deleteUsers,
    defaultUserAttributes
  },
  fileFactory: {
    createFile,
    createMultipleFiles,
    deleteFile,
    deleteFiles,
    deleteUserFiles,
    defaultFileAttributes
  },
  notificationFactory: {
    createNotification,
    createMultipleNotifications,
    createReadNotification,
    createUnreadNotification,
    deleteNotification,
    deleteNotifications,
    deleteUserNotifications,
    defaultNotificationAttributes,
    NOTIFICATION_TYPES,
    NOTIFICATION_PRIORITIES
  }
};
