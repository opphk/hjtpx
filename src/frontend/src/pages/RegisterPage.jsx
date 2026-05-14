import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import RegisterForm from '../components/RegisterForm';
import Alert from '../components/ui/Alert';
import { register } from '../services/auth';

const RegisterPage = () => {
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleRegister = async (formData) => {
    setError('');
    setLoading(true);
    
    try {
      const result = await register(formData);
      if (result.success) {
        navigate('/login', { 
          state: { message: '注册成功，请登录' }
        });
      } else {
        setError(result.message || '注册失败');
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
          <h1>创建账户</h1>
          <p>加入我们，开始探索</p>
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
            已有账户？ <Link to="/login">立即登录</Link>
          </p>
        </div>
      </div>
    </div>
  );
};

export default RegisterPage;
