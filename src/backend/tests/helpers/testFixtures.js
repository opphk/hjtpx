const jwt = require('jsonwebtoken');

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-secret-key-change-in-production';

function generateToken(user) {
  return jwt.sign(
    { id: user.id, email: user.email, role: user.role },
    JWT_SECRET,
    { expiresIn: '1h' }
  );
}

const testPassword = 'TestPassword123!';

const validUserCredentials = {
  email: 'test@example.com',
  password: testPassword
};

const invalidUserCredentials = {
  email: 'invalid@example.com',
  password: 'wrongpassword'
};

const validRegistrationData = {
  email: 'newuser@example.com',
  name: 'New User',
  password: testPassword
};

const userUpdateData = {
  name: 'Updated User Name',
  bio: 'This is an updated bio'
};

const invalidEmailFormat = {
  email: 'invalid-email',
  password: testPassword
};

const weakPasswordData = {
  email: 'weak@example.com',
  name: 'Weak Password',
  password: '123'
};

const notificationData = {
  title: 'Test Notification',
  message: 'This is a test notification message',
  type: 'info',
  channels: ['in_app']
};

const fileUploadData = {
  folder: 'test-folder'
};

const paginationParams = {
  page: 1,
  limit: 10
};

const filterParams = {
  status: 'unread',
  type: 'info'
};

const ROLES = {
  ADMIN: 'admin',
  USER: 'user'
};

const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  NO_CONTENT: 204,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  INTERNAL_SERVER_ERROR: 500
};

module.exports = {
  generateToken,
  testPassword,
  validUserCredentials,
  invalidUserCredentials,
  validRegistrationData,
  userUpdateData,
  invalidEmailFormat,
  weakPasswordData,
  notificationData,
  fileUploadData,
  paginationParams,
  filterParams,
  ROLES,
  HTTP_STATUS
};
