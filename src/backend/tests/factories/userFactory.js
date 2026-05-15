const bcrypt = require('bcryptjs');
const pool = require('../../../config/database/db');

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
  
  const result = await pool.query(
    'INSERT INTO users (email, name, password, role, status) VALUES ($1, $2, $3, $4, $5) RETURNING *',
    [attributes.email, attributes.name, attributes.password, attributes.role, attributes.status]
  );
  
  return result.rows[0];
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
  try {
    await pool.query('DELETE FROM users WHERE id = $1', [userId]);
  } catch (error) {
    console.error('Error deleting user:', error);
  }
}

async function deleteUsers(userIds) {
  for (const userId of userIds) {
    await deleteUser(userId);
  }
}

module.exports = {
  createUser,
  createAdmin,
  createMultipleUsers,
  deleteUser,
  deleteUsers,
  defaultUserAttributes
};
