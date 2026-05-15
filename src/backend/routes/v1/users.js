const express = require('express');

const router = express.Router();
const { auth } = require('../../middleware/auth');
const { checkRole, ROLES } = require('../../middleware/rbac');
const { apiCache, invalidateCacheByTag } = require('../../middleware/cacheMiddleware');
const validator = require('../../middleware/validator');
const userService = require('../../services/userService');

router.get('/', auth, checkRole(ROLES.ADMIN), apiCache(60, { tags: ['users'] }), async (req, res) => {
  try {
    const users = await userService.getAllUsers();
    res.success(users, 'Users retrieved successfully');
  } catch (error) {
    res.error(error.message, 500, 'FETCH_USERS_ERROR');
  }
});

router.get('/me', auth, apiCache(60, { tags: ['user'] }), async (req, res) => {
  try {
    const user = await userService.getUserById(req.user.id);
    if (!user) {
      return res.notFound('User not found');
    }
    res.success(user, 'User retrieved successfully');
  } catch (error) {
    res.error(error.message, 500, 'FETCH_USER_ERROR');
  }
});

router.get('/:id', auth, checkRole(ROLES.ADMIN, ROLES.USER), apiCache(60, { tags: ['user'] }), async (req, res) => {
  try {
    if (req.user.role !== ROLES.ADMIN && req.user.id !== req.params.id) {
      return res.status(403).json({
        success: false,
        error: 'Access denied'
      });
    }
    const user = await userService.getUserById(req.params.id);
    if (!user) {
      return res.notFound('User not found');
    }
    res.success(user, 'User retrieved successfully');
  } catch (error) {
    res.error(error.message, 500, 'FETCH_USER_ERROR');
  }
});

router.post(
  '/',
  auth,
  checkRole(ROLES.ADMIN),
  validator('userSchema', 'body'),
  invalidateCacheByTag('users'),
  async (req, res) => {
    try {
      const { email, name, password } = req.body;
      const user = await userService.createUser({ email, name, password });
      res.created(user, 'User created successfully', 201);
    } catch (error) {
      if (error.code === '23505') {
        return res.badRequest('Email already exists');
      }
      res.error(error.message, 500, 'CREATE_USER_ERROR');
    }
  }
);

router.put('/me', auth, validator('updateUserSchema', 'body'), invalidateCacheByTag('user'), async (req, res) => {
  try {
    const updateData = req.body;
    delete updateData.role;
    delete updateData.id;

    const user = await userService.updateUser(req.user.id, updateData);
    if (!user) {
      return res.notFound('User not found');
    }
    res.success(user, 'User updated successfully');
  } catch (error) {
    res.error(error.message, 500, 'UPDATE_USER_ERROR');
  }
});

router.put(
  '/:id',
  auth,
  checkRole(ROLES.ADMIN, ROLES.USER),
  validator('updateUserSchema', 'body'),
  invalidateCacheByTag('user'),
  async (req, res) => {
    try {
      if (req.user.role !== ROLES.ADMIN && req.user.id !== req.params.id) {
        return res.status(403).json({
          success: false,
          error: 'Access denied'
        });
      }

      const updateData = req.body;
      if (req.user.role !== ROLES.ADMIN) {
        delete updateData.role;
      }

      const user = await userService.updateUser(req.params.id, updateData);
      if (!user) {
        return res.notFound('User not found');
      }
      res.success(user, 'User updated successfully');
    } catch (error) {
      res.error(error.message, 500, 'UPDATE_USER_ERROR');
    }
  }
);

router.delete('/:id', auth, checkRole(ROLES.ADMIN), invalidateCacheByTag('users'), async (req, res) => {
  try {
    await userService.deleteUser(req.params.id);
    res.noContent();
  } catch (error) {
    res.error(error.message, 500, 'DELETE_USER_ERROR');
  }
});

module.exports = router;
