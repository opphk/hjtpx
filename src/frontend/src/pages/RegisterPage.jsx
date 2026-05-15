import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, Link } from 'react-router-dom';
import SEO, { getSEOMeta } from '../components/SEO';
import RegisterForm from '../components/RegisterForm';
import Alert from '../components/ui/Alert';
import { register } from '../services/auth';

const RegisterPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const seoMeta = getSEOMeta('register');

  const handleRegister = async (formData) => {
    setError('');
    setLoading(true);
    
    try {
      const result = await register(formData);
      if (result.success) {
        navigate('/login', { 
          state: { message: t('auth.registerSuccess') }
        });
      } else {
        setError(result.message || t('auth.registerFailed'));
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
            <h1>{t('auth.createAccount')}</h1>
            <p>{t('auth.joinUs')}</p>
          </div>
          
          {error && (
            <Alert 
              type="error" 
              message={error} 
              closable 
              onClose={() => setError('')}
            />
          )}
          
          <RegisterForm onSubmit={handleRegister} loading={loading} />
          
          <div className="auth-footer">
            <p>
              {t('auth.hasAccount')} <Link to="/login">{t('auth.signInNow')}</Link>
            </p>
          </div>
        </div>
      </div>
    </>
  );
};

export default RegisterPage;
