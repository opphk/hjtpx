const express = require('express');
const router = express.Router();
const userService = require('../../services/userService');
const validator = require('../../middleware/validator');
const auth = require('../../middleware/auth');

router.get('/', 
  validator('updateUserSchema', 'query'),
  async (req, res) => {
    try {
      const users = await userService.getAllUsers();
      res.success(users, 'Users retrieved successfully');
    } catch (error) {
      res.error(error.message, 500, 'FETCH_USERS_ERROR');
    }
  }
);

router.get('/:id', async (req, res) => {
  try {
    const user = await userService.getUserById(req.params.id);
    if (!user) {
      return res.notFound('User not found');
    }
    res.success(user, 'User retrieved successfully');
  } catch (error) {
    res.error(error.message, 500, 'FETCH_USER_ERROR');
  }
});

router.post('/',
  validator('userSchema', 'body'),
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

router.put('/:id',
  validator('updateUserSchema', 'body'),
  async (req, res) => {
    try {
      const user = await userService.updateUser(req.params.id, req.body);
      if (!user) {
        return res.notFound('User not found');
      }
      res.success(user, 'User updated successfully');
    } catch (error) {
      res.error(error.message, 500, 'UPDATE_USER_ERROR');
    }
  }
);

router.delete('/:id', async (req, res) => {
  try {
    await userService.deleteUser(req.params.id);
    res.noContent();
  } catch (error) {
    res.error(error.message, 500, 'DELETE_USER_ERROR');
  }
});

module.exports = router;
