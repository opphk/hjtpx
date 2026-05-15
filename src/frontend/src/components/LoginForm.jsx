import React, { useState } from 'react';
import Input from './ui/Input';
import Button from './ui/Button';

const LoginForm = ({ onSubmit, loading }) => {
  const [formData, setFormData] = useState({
    email: '',
    password: ''
  });
  const [errors, setErrors] = useState({});

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
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (validate()) {
      onSubmit(formData);
    }
  };

  const formId = React.useId();

  return (
    <form 
      onSubmit={handleSubmit} 
      className="auth-form"
      aria-label="登录表单"
    >
      <Input
        label="邮箱"
        name="email"
        value={formData.email}
        onChange={handleChange}
        placeholder="请输入邮箱"
        error={errors.email}
        required
        aria-describedby={errors.email ? `${formId}-email-error` : undefined}
      />
      {errors.email && (
        <span id={`${formId}-email-error`} className="sr-only" role="alert">
          {errors.email}
        </span>
      )}
      
      <Input
        label="密码"
        name="password"
        type="password"
        value={formData.password}
        onChange={handleChange}
        placeholder="请输入密码"
        error={errors.password}
        required
        aria-describedby={errors.password ? `${formId}-password-error` : undefined}
      />
      {errors.password && (
        <span id={`${formId}-password-error`} className="sr-only" role="alert">
          {errors.password}
        </span>
      )}
      
      <Button 
        type="submit" 
        loading={loading}
        className="auth-submit"
        aria-label="提交登录"
      >
        登录
      </Button>
    </form>
  );
};

export default LoginForm;
