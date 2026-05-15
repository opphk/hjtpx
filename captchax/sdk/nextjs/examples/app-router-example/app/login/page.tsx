'use client';

import { useState } from 'react';
import { CaptchaDialog } from '@captchax/nextjs';
import { useCaptchaVerify } from '@captchax/nextjs';

export default function LoginPage() {
  const [showCaptcha, setShowCaptcha] = useState(false);
  const [formData, setFormData] = useState({ email: '', password: '' });
  const [captchaToken, setCaptchaToken] = useState<string | null>(null);
  
  const { token, loading, error, verify, reset, isVerified } = useCaptchaVerify({
    scene: 'login',
    apiKey: process.env.NEXT_PUBLIC_CAPTCHA_API_KEY || '',
    onSuccess: (token) => {
      console.log('Verification successful:', token);
      setCaptchaToken(token);
      setShowCaptcha(false);
    }
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!captchaToken && !isVerified) {
      setShowCaptcha(true);
      return;
    }

    try {
      const response = await fetch('/api/captcha/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: formData.email,
          password: formData.password,
          captchaToken: token || captchaToken
        })
      });
      
      const data = await response.json();
      
      if (data.success) {
        alert('登录成功！');
      } else {
        alert('登录失败: ' + data.error);
      }
    } catch (err) {
      console.error('Login error:', err);
    }
  };

  const handleCaptchaSuccess = (token: string) => {
    setCaptchaToken(token);
    setShowCaptcha(false);
  };

  return (
    <main style={{ padding: '40px', maxWidth: '400px', margin: '0 auto' }}>
      <h1 style={{ fontSize: '28px', marginBottom: '24px', textAlign: 'center' }}>用户登录</h1>
      
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
        <div>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: 500 }}>
            邮箱地址
          </label>
          <input
            type="email"
            value={formData.email}
            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
            required
            style={{
              width: '100%',
              padding: '12px',
              border: '1px solid #d1d5db',
              borderRadius: '6px',
              fontSize: '14px'
            }}
          />
        </div>

        <div>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: 500 }}>
            密码
          </label>
          <input
            type="password"
            value={formData.password}
            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
            required
            style={{
              width: '100%',
              padding: '12px',
              border: '1px solid #d1d5db',
              borderRadius: '6px',
              fontSize: '14px'
            }}
          />
        </div>

        {isVerified || captchaToken ? (
          <div style={{
            padding: '12px',
            backgroundColor: '#dcfce7',
            borderRadius: '6px',
            color: '#166534',
            textAlign: 'center'
          }}>
            ✓ 验证已完成
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setShowCaptcha(true)}
            style={{
              padding: '12px',
              backgroundColor: '#f3f4f6',
              border: '1px solid #d1d5db',
              borderRadius: '6px',
              cursor: 'pointer',
              fontSize: '14px'
            }}
          >
            {loading ? '验证中...' : '点击进行安全验证'}
          </button>
        )}

        <button
          type="submit"
          disabled={loading}
          style={{
            padding: '12px',
            backgroundColor: '#4F46E5',
            color: 'white',
            border: 'none',
            borderRadius: '6px',
            fontSize: '16px',
            fontWeight: 500,
            cursor: loading ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.6 : 1
          }}
        >
          {loading ? '处理中...' : '登录'}
        </button>

        {error && (
          <div style={{ color: '#ef4444', fontSize: '14px', textAlign: 'center' }}>
            {error.message}
          </div>
        )}
      </form>

      <CaptchaDialog
        open={showCaptcha}
        onClose={() => setShowCaptcha(false)}
        onSuccess={handleCaptchaSuccess}
        scene="login"
        type="slider"
        apiKey={process.env.NEXT_PUBLIC_CAPTCHA_API_KEY || ''}
      />
    </main>
  );
}
