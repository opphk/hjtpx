'use client';

import { CaptchaButton } from '@captchax/nextjs';
import Link from 'next/link';

export default function HomePage() {
  const handleSuccess = (token: string) => {
    console.log('Verification successful, token:', token);
  };
  
  const handleError = (error: Error) => {
    console.error('Verification failed:', error.message);
  };
  
  return (
    <main style={{ padding: '40px', maxWidth: '800px', margin: '0 auto' }}>
      <h1 style={{ fontSize: '32px', marginBottom: '20px' }}>欢迎使用 CaptchaX</h1>
      
      <section style={{ marginBottom: '30px' }}>
        <h2 style={{ fontSize: '24px', marginBottom: '16px' }}>基础按钮验证</h2>
        <CaptchaButton 
          scene="login"
          onSuccess={handleSuccess}
          onError={handleError}
          text="点击验证"
          apiKey={process.env.NEXT_PUBLIC_CAPTCHA_API_KEY || ''}
        />
      </section>

      <section style={{ marginBottom: '30px' }}>
        <h2 style={{ fontSize: '24px', marginBottom: '16px' }}>登录流程</h2>
        <p style={{ marginBottom: '16px', color: '#666' }}>
          点击下方按钮完成验证后登录
        </p>
        <CaptchaButton 
          scene="login"
          onSuccess={(token) => {
            console.log('Login verified:', token);
            alert('验证成功！Token: ' + token);
          }}
          onError={handleError}
          text="登录验证"
          apiKey={process.env.NEXT_PUBLIC_CAPTCHA_API_KEY || ''}
        />
      </section>

      <section style={{ marginBottom: '30px' }}>
        <h2 style={{ fontSize: '24px', marginBottom: '16px' }}>注册验证</h2>
        <CaptchaButton 
          scene="register"
          onSuccess={(token) => {
            console.log('Register verified:', token);
          }}
          onError={handleError}
          text="注册验证"
          apiKey={process.env.NEXT_PUBLIC_CAPTCHA_API_KEY || ''}
        />
      </section>

      <section style={{ marginBottom: '30px' }}>
        <h2 style={{ fontSize: '24px', marginBottom: '16px' }}>评论验证</h2>
        <CaptchaButton 
          scene="comment"
          onSuccess={(token) => {
            console.log('Comment verified:', token);
          }}
          onError={handleError}
          text="发表评论"
          apiKey={process.env.NEXT_PUBLIC_CAPTCHA_API_KEY || ''}
        />
      </section>

      <nav style={{ marginTop: '40px', paddingTop: '20px', borderTop: '1px solid #e5e7eb' }}>
        <Link href="/login" style={{ marginRight: '20px', color: '#4F46E5' }}>
          登录页面示例 →
        </Link>
      </nav>
    </main>
  );
}
