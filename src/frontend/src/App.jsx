import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Spin } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';

import Layout from '@components/Layout';
import ErrorBoundary from '@components/ErrorBoundary';
import AuthGuard from '@components/AuthGuard';

const HomePage = lazy(() => import('@pages/Home'));
const DashboardPage = lazy(() => import('@pages/Dashboard'));
const UsersPage = lazy(() => import('@pages/Users'));
const ProfilePage = lazy(() => import('@pages/Profile'));
const SettingsPage = lazy(() => import('@pages/Settings'));
const LoginPage = lazy(() => import('@pages/Login'));
const RegisterPage = lazy(() => import('@pages/Register'));
const NotFoundPage = lazy(() => import('@pages/NotFound'));
const CaptchaDemoPage = lazy(() => import('@pages/CaptchaDemo'));
const AnalyticsPage = lazy(() => import('@pages/Analytics'));

const LoadingSpinner = () => (
  <div style={{
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    minHeight: '100vh',
    background: '#f0f2f5'
  }}>
    <Spin indicator={<LoadingOutlined style={{ fontSize: 48 }} spin />} />
  </div>
);

const PageError = ({ error }) => (
  <div style={{
    padding: 40,
    textAlign: 'center',
    color: '#ff4d4f'
  }}>
    <h2>页面加载失败</h2>
    <p>{error?.message || '未知错误'}</p>
    <button onClick={() => window.location.reload()}>
      重新加载
    </button>
  </div>
);

function App() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Suspense fallback={<LoadingSpinner />}>
          <Routes>
            <Route path="/login" element={
              <Suspense fallback={<LoadingSpinner />}>
                <LoginPage />
              </Suspense>
            } />
            
            <Route path="/register" element={
              <Suspense fallback={<LoadingSpinner />}>
                <RegisterPage />
              </Suspense>
            } />
            
            <Route path="/" element={<Layout />}>
              <Route index element={<Navigate to="/home" replace />} />
              
              <Route path="home" element={
                <Suspense fallback={<LoadingSpinner />}>
                  <HomePage />
                </Suspense>
              } />
              
              <Route path="dashboard" element={
                <AuthGuard>
                  <Suspense fallback={<LoadingSpinner />}>
                    <DashboardPage />
                  </Suspense>
                </AuthGuard>
              } />
              
              <Route path="users" element={
                <AuthGuard>
                  <Suspense fallback={<LoadingSpinner />}>
                    <UsersPage />
                  </Suspense>
                </AuthGuard>
              } />
              
              <Route path="profile" element={
                <AuthGuard>
                  <Suspense fallback={<LoadingSpinner />}>
                    <ProfilePage />
                  </Suspense>
                </AuthGuard>
              } />
              
              <Route path="settings" element={
                <AuthGuard>
                  <Suspense fallback={<LoadingSpinner />}>
                    <SettingsPage />
                  </Suspense>
                </AuthGuard>
              } />
              
              <Route path="captcha-demo" element={
                <Suspense fallback={<LoadingSpinner />}>
                  <CaptchaDemoPage />
                </Suspense>
              } />
              
              <Route path="analytics" element={
                <AuthGuard>
                  <Suspense fallback={<LoadingSpinner />}>
                    <AnalyticsPage />
                  </Suspense>
                </AuthGuard>
              } />
              
              <Route path="*" element={
                <Suspense fallback={<LoadingSpinner />}>
                  <NotFoundPage />
                </Suspense>
              } />
            </Route>
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ErrorBoundary>
  );
}

export default App;
