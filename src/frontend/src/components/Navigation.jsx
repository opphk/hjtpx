import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

const Navigation = () => {
  const { user, logout, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  return (
    <nav className="navbar">
      <div className="navbar-brand">
        <Link to="/dashboard">HJTPX 系统</Link>
      </div>
      
      {isAuthenticated && (
        <div className="navbar-menu">
          <Link to="/dashboard" className="nav-link">首页</Link>
          <Link to="/users" className="nav-link">用户管理</Link>
          
          <div className="navbar-user">
            <span className="user-info">
              {user?.username}
            </span>
            <button 
              onClick={handleLogout}
              className="btn btn-logout"
            >
              退出
            </button>
          </div>
        </div>
      )}
    </nav>
  );
};

export default Navigation;
