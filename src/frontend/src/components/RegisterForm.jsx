import React, { useState } from 'react';
import Input from '../ui/Input';
import Button from '../ui/Button';

const RegisterForm = ({ onSubmit, loading }) => {
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: ''
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
    
    if (!formData.username.trim()) {
      newErrors.username = '用户名不能为空';
    } else if (formData.username.length < 3) {
      newErrors.username = '用户名至少3个字符';
    }
    
    if (!formData.email.trim()) {
      newErrors.email = '邮箱不能为空';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = '请输入有效的邮箱地址';
    }
    
    if (!formData.password) {
      newErrors.password = '密码不能为空';
    } else if (formData.password.length < 6) {
      newErrors.password = '密码至少6个字符';
    }
    
    if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = '两次输入的密码不一致';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (validate()) {
      onSubmit({
        username: formData.username,
        email: formData.email,
        password: formData.password
      });
    }
  };

  return (
    <form onSubmit={handleSubmit} className="auth-form">
      <Input
        label="用户名"
        name="username"
        value={formData.username}
        onChange={handleChange}
        placeholder="请输入用户名"
        error={errors.username}
        required
      />
      
      <Input
        label="邮箱"
        name="email"
        type="email"
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
      
      <Input
        label="确认密码"
        name="confirmPassword"
        type="password"
        value={formData.confirmPassword}
        onChange={handleChange}
        placeholder="请再次输入密码"
        error={errors.confirmPassword}
        required
      />
      
      <Button 
        type="submit" 
        loading={loading}
        className="auth-submit"
      >
        注册
      </Button>
    </form>
  );
};

export default RegisterForm;
