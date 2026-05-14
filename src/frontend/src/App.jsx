import React, { Suspense, lazy, useEffect, useRef } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import Loading from './components/ui/Loading';
import PWAInstallPrompt from './components/PWAInstallPrompt';

const LoginPage = lazy(() => import('./pages/LoginPage'));
const RegisterPage = lazy(() => import('./pages/RegisterPage'));
const DashboardPage = lazy(() => import('./pages/DashboardPage'));
const AdminUsersPage = lazy(() => import('./pages/AdminUsersPage'));
const LogsPage = lazy(() => import('./pages/LogsPage'));
const SettingsPage = lazy(() => import('./pages/SettingsPage'));
const AuditDashboard = lazy(() => import('./pages/AuditDashboard'));
const UserList = lazy(() => import('./components/UserList'));
const DashboardLayout = lazy(() => import('./components/DashboardLayout'));

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
  useEffect(() => {
    const prefetchRoutes = () => {
      const dashboardLink = document.querySelector('a[href="/dashboard"]');
      const usersLink = document.querySelector('a[href="/users"]');
      
      if (dashboardLink) {
        dashboardLink.addEventListener('mouseenter', () => {
          DashboardPage.preload();
        }, { once: true });
      }
      
      if (usersLink) {
        usersLink.addEventListener('mouseenter', () => {
          UserList.preload();
        }, { once: true });
      }
    };
    
    const timer = setTimeout(prefetchRoutes, 1000);
    return () => clearTimeout(timer);
  }, []);

  return (
    <Suspense fallback={<SuspenseFallback />}>
      <Routes>
        <Route 
          path="/login" 
          element={
            <PublicRoute>
              <LoginPage />
            </PublicRoute>
          } 
        />
        <Route 
          path="/register" 
          element={
            <PublicRoute>
              <RegisterPage />
            </PublicRoute>
          } 
        />
        <Route 
          path="/dashboard" 
          element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          } 
        />
        <Route 
          path="/users" 
          element={
            <ProtectedRoute>
              <DashboardLayout>
                <UserList />
              </DashboardLayout>
            </ProtectedRoute>
          } 
        />
        <Route 
          path="/admin/users" 
          element={
            <ProtectedRoute>
              <DashboardLayout>
                <AdminUsersPage />
              </DashboardLayout>
            </ProtectedRoute>
          } 
        />
        <Route 
          path="/admin/logs" 
          element={
            <ProtectedRoute>
              <DashboardLayout>
                <LogsPage />
              </DashboardLayout>
            </ProtectedRoute>
          } 
        />
        <Route 
          path="/admin/settings" 
          element={
            <ProtectedRoute>
              <DashboardLayout>
                <SettingsPage />
              </DashboardLayout>
            </ProtectedRoute>
          } 
        />
        <Route 
          path="/admin/audit" 
          element={
            <ProtectedRoute>
              <DashboardLayout>
                <AuditDashboard />
              </DashboardLayout>
            </ProtectedRoute>
          } 
        />
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </Suspense>
  );
};

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppRoutes />
        <PWAInstallPrompt />
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
