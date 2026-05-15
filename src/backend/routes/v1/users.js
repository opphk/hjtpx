const express = require('express');

const router = express.Router();
const { auth } = require('../../middleware/auth');
const { apiCache, invalidateCacheByTag } = require('../../middleware/cacheMiddleware');
const { checkRole, ROLES } = require('../../middleware/rbac');
const validator = require('../../middleware/validator');
const userService = require('../../services/userService');

/**
 * @swagger
 * tags:
 *   name: Users
 *   description: User management endpoints
 */

/**
 * @swagger
 * /api/v1/users:
 *   get:
 *     summary: Get all users
 *     description: Retrieve a list of all users (Admin only)
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: Users retrieved successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   type: array
 *                   items:
 *                     $ref: '#/components/schemas/User'
 *                 message:
 *                   type: string
 *                   example: Users retrieved successfully
 *       401:
 *         $ref: '#/components/responses/Unauthorized'
 *       403:
 *         $ref: '#/components/responses/Forbidden'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
router.get(
  '/',
  auth,
  checkRole(ROLES.ADMIN),
  apiCache(60, { tags: ['users'] }),
  async (req, res) => {
    try {
      const users = await userService.getAllUsers();
      res.success(users, 'Users retrieved successfully');
    } catch (error) {
      res.error(error.message, 500, 'FETCH_USERS_ERROR');
    }
  }
);

/**
 * @swagger
 * /api/v1/users/me:
 *   get:
 *     summary: Get current user
 *     description: Retrieve the currently authenticated user's profile
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: User retrieved successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/User'
 *                 message:
 *                   type: string
 *       401:
 *         $ref: '#/components/responses/Unauthorized'
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api/v1/users/{id}:
 *   get:
 *     summary: Get user by ID
 *     description: Retrieve a specific user by their ID
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *           format: uuid
 *         description: User ID
 *     responses:
 *       200:
 *         description: User retrieved successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/User'
 *                 message:
 *                   type: string
 *       401:
 *         $ref: '#/components/responses/Unauthorized'
 *       403:
 *         $ref: '#/components/responses/Forbidden'
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
router.get(
  '/:id',
  auth,
  checkRole(ROLES.ADMIN, ROLES.USER),
  apiCache(60, { tags: ['user'] }),
  async (req, res) => {
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
  }
);

/**
 * @swagger
 * /api/v1/users:
 *   post:
 *     summary: Create new user
 *     description: Create a new user account (Admin only)
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - email
 *               - name
 *               - password
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *                 example: newuser@example.com
 *               name:
 *                 type: string
 *                 example: John Doe
 *               password:
 *                 type: string
 *                 format: password
 *                 example: securePassword123
 *               role:
 *                 type: string
 *                 enum: [user, admin, moderator]
 *                 example: user
 *     responses:
 *       201:
 *         description: User created successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/User'
 *                 message:
 *                   type: string
 *       400:
 *         $ref: '#/components/responses/ValidationError'
 *       401:
 *         $ref: '#/components/responses/Unauthorized'
 *       403:
 *         $ref: '#/components/responses/Forbidden'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api/v1/users/me:
 *   put:
 *     summary: Update current user
 *     description: Update the currently authenticated user's profile
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               name:
 *                 type: string
 *                 example: John Updated
 *               email:
 *                 type: string
 *                 format: email
 *                 example: updated@example.com
 *     responses:
 *       200:
 *         description: User updated successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/User'
 *                 message:
 *                   type: string
 *       400:
 *         $ref: '#/components/responses/ValidationError'
 *       401:
 *         $ref: '#/components/responses/Unauthorized'
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
router.put(
  '/me',
  auth,
  validator('updateUserSchema', 'body'),
  invalidateCacheByTag('user'),
  async (req, res) => {
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
  }
);

/**
 * @swagger
 * /api/v1/users/{id}:
 *   put:
 *     summary: Update user by ID
 *     description: Update a specific user's information
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *           format: uuid
 *         description: User ID
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               name:
 *                 type: string
 *               email:
 *                 type: string
 *                 format: email
 *               role:
 *                 type: string
 *                 enum: [user, admin, moderator]
 *     responses:
 *       200:
 *         description: User updated successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                 data:
 *                   $ref: '#/components/schemas/User'
 *                 message:
 *                   type: string
 *       400:
 *         $ref: '#/components/responses/ValidationError'
 *       401:
 *         $ref: '#/components/responses/Unauthorized'
 *       403:
 *         $ref: '#/components/responses/Forbidden'
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
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

/**
 * @swagger
 * /api/v1/users/{id}:
 *   delete:
 *     summary: Delete user
 *     description: Delete a user account (Admin only)
 *     tags: [Users]
 *     security:
 *       - bearerAuth: []
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *           format: uuid
 *         description: User ID
 *     responses:
 *       204:
 *         description: User deleted successfully
 *       401:
 *         $ref: '#/components/responses/Unauthorized'
 *       403:
 *         $ref: '#/components/responses/Forbidden'
 *       404:
 *         $ref: '#/components/responses/NotFound'
 *       500:
 *         $ref: '#/components/responses/InternalServerError'
 */
router.delete(
  '/:id',
  auth,
  checkRole(ROLES.ADMIN),
  invalidateCacheByTag('users'),
  async (req, res) => {
    try {
      await userService.deleteUser(req.params.id);
      res.noContent();
    } catch (error) {
      res.error(error.message, 500, 'DELETE_USER_ERROR');
    }
  }
);

module.exports = router;
