import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import Input from './ui/Input';
import Button from './ui/Button';

const RegisterForm = ({ onSubmit, loading }) => {
  const { t } = useTranslation();
  const [formData, setFormData] = useState({
    name: '',
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
    
    if (!formData.name.trim()) {
      newErrors.name = t('validation.nameRequired');
    } else if (formData.name.length < 2) {
      newErrors.name = t('validation.usernameMin', { min: 2 });
    }
    
    if (!formData.email.trim()) {
      newErrors.email = t('validation.emailRequired');
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = t('validation.email');
    }
    
    if (!formData.password) {
      newErrors.password = t('validation.passwordRequired');
    } else if (formData.password.length < 8) {
      newErrors.password = t('validation.passwordMin', { min: 8 });
    } else if (!/(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/.test(formData.password)) {
      newErrors.password = t('validation.passwordComplexity');
    }
    
    if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = t('validation.passwordMatch');
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (validate()) {
      onSubmit({
        name: formData.name,
        email: formData.email,
        password: formData.password
      });
    }
  };

  return (
    <form onSubmit={handleSubmit} className="auth-form">
      <Input
        label={t('form.name')}
        name="name"
        value={formData.name}
        onChange={handleChange}
        placeholder={t('form.placeholder.name')}
        error={errors.name}
        required
      />
      
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
      
      <Input
        label={t('form.confirmPassword')}
        name="confirmPassword"
        type="password"
        value={formData.confirmPassword}
        onChange={handleChange}
        placeholder={t('form.placeholder.confirmPassword')}
        error={errors.confirmPassword}
        required
      />
      
      <Button 
        type="submit" 
        loading={loading}
        className="auth-submit"
      >
        {t('auth.register')}
      </Button>
    </form>
  );
};

export default RegisterForm;
