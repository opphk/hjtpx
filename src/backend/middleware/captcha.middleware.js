
const captchaService = require('../services/captcha.service');

const requireCaptcha = (req, res, next) =&gt; {
  if (!captchaService.isEnabled()) {
    return next();
  }

  const token = req.headers['x-captcha-token'] || req.body.captchaToken;

  if (!token) {
    return res.status(400).json({
      success: false,
      message: '验证码 Token 不能为空',
      code: 'CAPTCHA_TOKEN_REQUIRED',
    });
  }

  try {
    const valid = captchaService.verifyToken(token);
    if (!valid) {
      return res.status(400).json({
        success: false,
        message: '验证码验证失败',
        code: 'CAPTCHA_VERIFICATION_FAILED',
      });
    }
  } catch (error) {
    console.error('验证码验证异常:', error);
    return res.status(500).json({
      success: false,
      message: '验证码服务错误',
      code: 'CAPTCHA_SERVICE_ERROR',
    });
  }

  next();
};

module.exports = {
  requireCaptcha,
};

