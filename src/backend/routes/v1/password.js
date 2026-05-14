const express = require('express');
const router = express.Router();
const authService = require('../../services/authService');
const validator = require('../../middleware/validator');

router.post('/forgot',
  validator('forgotPasswordSchema', 'body'),
  async (req, res) => {
    try {
      const { email } = req.body;
      const result = await authService.forgotPassword(email);

      res.success(result, 'Password reset email sent if account exists');
    } catch (error) {
      res.error(error.message, 500, 'FORGOT_PASSWORD_ERROR');
    }
  }
);

router.post('/reset',
  validator('resetPasswordSchema', 'body'),
  async (req, res) => {
    try {
      const { token, newPassword } = req.body;
      const result = await authService.resetPassword({ token, newPassword });

      res.success(result, 'Password successfully reset');
    } catch (error) {
      if (error.message === 'Invalid or expired reset token') {
        return res.badRequest('Invalid or expired reset token');
      }
      if (error.message.includes('Password must')) {
        return res.badRequest(error.message);
      }
      res.error(error.message, 500, 'RESET_PASSWORD_ERROR');
    }
  }
);

module.exports = router;
