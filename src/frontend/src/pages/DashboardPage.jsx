import React from 'react';
import { useAuth } from '../hooks/useAuth';
import DashboardLayout from '../components/DashboardLayout';
import Navigation from '../components/Navigation';
import Alert from '../components/ui/Alert';

const DashboardPage = () => {
  const { user, isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return (
      <Alert 
        type="warning" 
        message="请先登录" 
      />
    );
  }

  return (
    <DashboardLayout>
      <div className="dashboard-content">
        <div className="dashboard-header">
          <h1>欢迎回来, {user?.username || '用户'}!</h1>
          <p>这是您的个人仪表板</p>
        </div>
        
        <div className="dashboard-cards">
          <div className="card">
            <h3>个人信息</h3>
            <div className="card-content">
              <p><strong>用户名:</strong> {user?.username}</p>
              <p><strong>邮箱:</strong> {user?.email}</p>
              <p><strong>角色:</strong> {user?.role || '用户'}</p>
            </div>
          </div>
          
          <div className="card">
            <h3>账户统计</h3>
            <div className="card-content">
              <p><strong>注册时间:</strong> {user?.createdAt || '未知'}</p>
              <p><strong>最后登录:</strong> {user?.lastLogin || '未知'}</p>
            </div>
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
};

export default DashboardPage;
