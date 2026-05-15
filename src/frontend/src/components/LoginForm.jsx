import React, { useState } from 'react';
import Input from './ui/Input';
import Button from './ui/Button';
import CaptchaX from './CaptchaX';

const LoginForm = ({ onSubmit, loading }) => {
  const [formData, setFormData] = useState({
    email: '',
    password: ''
  });
  const [errors, setErrors] = useState({});
  const [captchaVerified, setCaptchaVerified] = useState(false);
  const [captchaToken, setCaptchaToken] = useState(null);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
    
    if (errors[name]) {
      setErrors(prev => ({
        ...prev,
        [name]: ''
      }));
    }
  };

  const handleCaptchaSuccess = (result) => {
    setCaptchaVerified(true);
    setCaptchaToken(result.token || result.captcha_id);
  };

  const handleCaptchaError = () => {
    setCaptchaVerified(false);
    setCaptchaToken(null);
  };

  const handleCaptchaRefresh = () => {
    setCaptchaVerified(false);
    setCaptchaToken(null);
  };

  const validate = () => {
    const newErrors = {};
    
    if (!formData.email.trim()) {
      newErrors.email = '邮箱不能为空';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = '请输入有效的邮箱地址';
    }
    
    if (!formData.password) {
      newErrors.password = '密码不能为空';
    } else if (formData.password.length < 8) {
      newErrors.password = '密码至少8个字符';
    }
    
    if (!captchaVerified) {
      newErrors.captcha = '请先完成验证码验证';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (validate()) {
      onSubmit({
        ...formData,
        captchaToken
      });
    }
  };

  return (
    <form onSubmit={handleSubmit} className="auth-form">
      <Input
        label="邮箱"
        name="email"
        value={formData.email}
        onChange={handleChange}
        placeholder="请输入邮箱"
        error={errors.email}
        required
      />
      
      <Input
        label="密码"
        name="password"
        type="password"
        value={formData.password}
        onChange={handleChange}
        placeholder="请输入密码"
        error={errors.password}
        required
      />

      <div className="captcha-field-wrapper">
        <label className="captcha-label">安全验证</label>
        <CaptchaX
          type="slider"
          onSuccess={handleCaptchaSuccess}
          onError={handleCaptchaError}
          onRefresh={handleCaptchaRefresh}
          width={300}
          height={150}
        />
        {errors.captcha && (
          <span className="input-error">{errors.captcha}</span>
        )}
      </div>
      
      <Button 
        type="submit" 
        loading={loading}
        className="auth-submit"
        disabled={!captchaVerified}
      >
        登录
      </Button>
    </form>
  );
};

export default LoginForm;
