import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import LoginForm from '../components/LoginForm';
import Alert from '../components/ui/Alert';

const LoginPage = () => {
  const navigate = useNavigate();
  const { login } = useAuth();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleLogin = async (formData) => {
    setError('');
    setLoading(true);
    
    try {
      const result = await login(formData);
      if (result.success) {
        navigate('/dashboard');
      } else {
        setError(result.message || '登录失败');
      }
    } catch (err) {
      setError(err.message || '网络错误，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-container">
        <div className="auth-header">
          <h1>欢迎回来</h1>
          <p>请登录您的账户</p>
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
            还没有账户？ <Link to="/register">立即注册</Link>
          </p>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;
