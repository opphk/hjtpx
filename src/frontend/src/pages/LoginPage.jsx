import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import SEO, { getSEOMeta } from '../components/SEO';
import LoginForm from '../components/LoginForm';
import Alert from '../components/ui/Alert';

const LoginPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { login } = useAuth();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const seoMeta = getSEOMeta('login');

  const handleLogin = async (formData) => {
    setError('');
    setLoading(true);
    
    try {
      const result = await login(formData);
      if (result.success) {
        navigate('/dashboard');
      } else {
        setError(result.message || t('auth.loginFailed'));
      }
    } catch (err) {
      setError(err.message || t('users.networkError'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <SEO {...seoMeta} />
      <div className="auth-page">
        <div className="auth-container">
          <div className="auth-header">
            <h1>{t('auth.welcomeBack')}</h1>
            <p>{t('auth.pleaseLogin')}</p>
          </div>
          
          {error && (
            <Alert 
              type="error" 
              message={error} 
              closable 
              onClose={() => setError('')}
            />
          )}
          
          <LoginForm onSubmit={handleLogin} loading={loading} />
          
          <div className="auth-footer">
            <p>
              {t('auth.noAccount')} <Link to="/register">{t('auth.signUpNow')}</Link>
            </p>
          </div>
        </div>
      </div>
    </>
  );
};

export default LoginPage;
