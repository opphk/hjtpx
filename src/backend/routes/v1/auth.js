const express = require('express');

const router = express.Router();
const { authRateLimiter } = require('../../middleware/rateLimiter');
const validator = require('../../middleware/validator');
const { requireCaptcha } = require('../../middleware/captcha.middleware');
const authService = require('../../services/authService');

router.post('/login', authRateLimiter, requireCaptcha, validator('loginSchema', 'body'), async (req, res) =&gt; {
  try {
    const { email, password } = req.body;
    const result = await authService.login({ email, password });

    res.success(result, 'Login successful');
  } catch (error) {
    if (error.message === 'Invalid credentials') {
      return res.unauthorized('Invalid email or password');
    }
    res.error(error.message, 500, 'AUTH_ERROR');
  }
});

router.post('/register', authRateLimiter, requireCaptcha, validator('userSchema', 'body'), async (req, res) =&gt; {
  try {
    const { email, name, password } = req.body;
    const userService = require('../../services/userService');
    const user = await userService.createUser({ email, name, password });

    const result = await authService.login({ email, password });

    res.created(result, 'Registration successful');
  } catch (error) {
    if (error.code === '23505') {
      return res.badRequest('Email already exists');
    }
    res.error(error.message, 500, 'REGISTRATION_ERROR');
  }
});

router.post('/verify', async (req, res) => {
  try {
    const { token } = req.body;
    if (!token) {
      return res.badRequest('Token is required');
    }

    const decoded = await authService.verifyToken(token);
    res.success({ valid: true, user: decoded }, 'Token is valid');
  } catch (error) {
    res.unauthorized('Invalid or expired token');
  }
});

router.post('/refresh', async (req, res) => {
  try {
    const { token } = req.body;
    if (!token) {
      return res.badRequest('Token is required');
    }

    const result = await authService.refreshToken(token);
    res.success(result, 'Token refreshed successfully');
  } catch (error) {
    res.unauthorized('Invalid or expired token');
  }
});

router.post('/logout', async (req, res) => {
  res.success({ message: 'Logout successful' }, 'Logout successful');
});

module.exports = router;
