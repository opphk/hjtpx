import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import Input from './ui/Input';
import Button from './ui/Button';

const LoginForm = ({ onSubmit, loading }) => {
  const { t } = useTranslation();
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
      newErrors.email = t('validation.emailRequired');
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = t('validation.email');
    }
    
    if (!formData.password) {
      newErrors.password = t('validation.passwordRequired');
    } else if (formData.password.length < 8) {
      newErrors.password = t('validation.passwordMin', { min: 8 });
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

  return (
    <form onSubmit={handleSubmit} className="auth-form">
      <Input
        label={t('form.email')}
        name="email"
        type="email"
        value={formData.email}
        onChange={handleChange}
        placeholder={t('form.placeholder.email')}
        error={errors.email}
        required
      />
      
      <Input
        label={t('form.password')}
        name="password"
        type="password"
        value={formData.password}
        onChange={handleChange}
        placeholder={t('form.placeholder.password')}
        error={errors.password}
        required
      />
      
      <Button 
        type="submit" 
        loading={loading}
        className="auth-submit"
      >
        {t('auth.login')}
      </Button>
    </form>
  );
};

export default LoginForm;
