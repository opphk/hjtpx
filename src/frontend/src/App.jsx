import React, { useEffect, useRef, lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import Loading from './components/ui/Loading';
import DashboardLayout from './components/DashboardLayout';
import PWAInstallPrompt from './components/PWAInstallPrompt';

const LoginPage = lazy(() => import('./pages/LoginPage'));
const RegisterPage = lazy(() => import('./pages/RegisterPage'));
const DashboardPage = lazy(() => import('./pages/DashboardPage'));
const AdminUsersPage = lazy(() => import('./pages/AdminUsersPage'));
const LogsPage = lazy(() => import('./pages/LogsPage'));
const SettingsPage = lazy(() => import('./pages/SettingsPage'));
const AuditDashboard = lazy(() => import('./pages/AuditDashboard'));
const UserList = lazy(() => import('./components/UserList'));

const ProtectedRoute = ({ children }) => {
  const { isAuthenticated, loading } = useAuth();
  
  if (loading) {
    return <Loading fullScreen text="加载中..." />;
  }
  
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  
  return children;
};

const PublicRoute = ({ children }) => {
  const { isAuthenticated, loading } = useAuth();
  
  if (loading) {
    return <Loading fullScreen text="加载中..." />;
  }
  
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }
  
  return children;
};

const RoutePreloader = ({ children, delay = 1000 }) => {
  const timeoutRef = useRef(null);
  
  useEffect(() => {
    timeoutRef.current = setTimeout(() => {
      if (children?.props?.children?.props?.path) {
        const path = children.props.children.props.path;
        if (path === '/dashboard') {
          DashboardPage.preload();
        } else if (path === '/users') {
          UserList.preload();
        }
      }
    }, delay);
    
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [children, delay]);
  
  return children;
};

const SuspenseFallback = () => (
  <Loading fullScreen text="加载中..." />
);

const AppRoutes = () => {
  return (
    <Routes>
      <Route 
        path="/login" 
        element={
          <PublicRoute>
            <Suspense fallback={<Loading fullScreen text="加载中..." />}>
              <LoginPage />
            </Suspense>
          </PublicRoute>
        } 
      />
      <Route 
        path="/register" 
        element={
          <PublicRoute>
            <Suspense fallback={<Loading fullScreen text="加载中..." />}>
              <RegisterPage />
            </Suspense>
          </PublicRoute>
        } 
      />
      <Route 
        path="/dashboard" 
        element={
          <ProtectedRoute>
            <Suspense fallback={<Loading fullScreen text="加载中..." />}>
              <DashboardPage />
            </Suspense>
          </ProtectedRoute>
        } 
      />
      <Route 
        path="/users" 
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <Suspense fallback={<Loading fullScreen text="加载中..." />}>
                <UserList />
              </Suspense>
            </DashboardLayout>
          </ProtectedRoute>
        } 
      />
      <Route 
        path="/admin/users" 
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <Suspense fallback={<Loading fullScreen text="加载中..." />}>
                <AdminUsersPage />
              </Suspense>
            </DashboardLayout>
          </ProtectedRoute>
        } 
      />
      <Route 
        path="/admin/logs" 
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <Suspense fallback={<Loading fullScreen text="加载中..." />}>
                <LogsPage />
              </Suspense>
            </DashboardLayout>
          </ProtectedRoute>
        } 
      />
      <Route 
        path="/admin/settings" 
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <Suspense fallback={<Loading fullScreen text="加载中..." />}>
                <SettingsPage />
              </Suspense>
            </DashboardLayout>
          </ProtectedRoute>
        } 
      />
      <Route 
        path="/admin/audit" 
        element={
          <ProtectedRoute>
            <DashboardLayout>
              <Suspense fallback={<Loading fullScreen text="加载中..." />}>
                <AuditDashboard />
              </Suspense>
            </DashboardLayout>
          </ProtectedRoute>
        } 
      />
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  );
};

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppRoutes />
        <PWAInstallPrompt delay={3000} />
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
