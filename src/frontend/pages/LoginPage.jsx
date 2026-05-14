import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useForm } from '../hooks/useForm';
import Button from '../components/Button';
import Input from '../components/Input';
import Alert from '../components/Alert';
import '../styles/components.css';

function LoginPage() {
  const navigate = useNavigate();
  const { login } = useAuth();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const { values, handleChange, handleSubmit, errors, validate } = useForm({
    initialValues: {
      email: '',
      password: ''
    },
    validate: (values) => {
      const errors = {};
      if (!values.email) {
        errors.email = 'Email is required';
      } else if (!/\S+@\S+\.\S+/.test(values.email)) {
        errors.email = 'Email is invalid';
      }
      if (!values.password) {
        errors.password = 'Password is required';
      } else if (values.password.length < 6) {
        errors.password = 'Password must be at least 6 characters';
      }
      return errors;
    },
    onSubmit: async (values) => {
      setError('');
      setLoading(true);
      try {
        await login(values.email, values.password);
        navigate('/dashboard');
      } catch (err) {
        setError(err.message || 'Login failed. Please try again.');
      } finally {
        setLoading(false);
      }
    }
  });

  return (
    <div className="login-page">
      <div className="login-container">
        <h1>Login</h1>
        {error && <Alert type="error" message={error} onClose={() => setError('')} />}
        <form onSubmit={handleSubmit}>
          <Input
            type="email"
            name="email"
            label="Email"
            value={values.email}
            onChange={handleChange}
            error={errors.email}
            placeholder="Enter your email"
            required
          />
          <Input
            type="password"
            name="password"
            label="Password"
            value={values.password}
            onChange={handleChange}
            error={errors.password}
            placeholder="Enter your password"
            required
          />
          <Button type="submit" loading={loading} disabled={loading}>
            Login
          </Button>
        </form>
        <p>
          Don't have an account? <Link to="/register">Register</Link>
        </p>
      </div>
    </div>
  );
}

export default LoginPage;
