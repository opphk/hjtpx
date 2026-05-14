import React from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import Button from '../components/Button';
import '../styles/components.css';

function DashboardPage() {
  const { user, logout } = useAuth();

  const handleLogout = () => {
    logout();
  };

  return (
    <div className="dashboard-page">
      <header className="dashboard-header">
        <h1>Dashboard</h1>
        <div className="user-info">
          <span>Welcome, {user?.username || user?.email}</span>
          <Button onClick={handleLogout} variant="secondary">
            Logout
          </Button>
        </div>
      </header>

      <nav className="dashboard-nav">
        <ul>
          <li>
            <Link to="/dashboard">Home</Link>
          </li>
          <li>
            <Link to="/users">Users</Link>
          </li>
        </ul>
      </nav>

      <main className="dashboard-content">
        <div className="dashboard-card">
          <h2>Welcome to Your Dashboard</h2>
          <p>
            This is your personal dashboard. You can manage your account and view
            various information here.
          </p>
          {user && (
            <div className="user-details">
              <h3>User Details</h3>
              <p><strong>Username:</strong> {user.username}</p>
              <p><strong>Email:</strong> {user.email}</p>
              {user.role && <p><strong>Role:</strong> {user.role}</p>}
            </div>
          )}
        </div>

        <div className="dashboard-stats">
          <div className="stat-card">
            <h3>Active Users</h3>
            <p className="stat-number">1,234</p>
          </div>
          <div className="stat-card">
            <h3>Total Projects</h3>
            <p className="stat-number">567</p>
          </div>
          <div className="stat-card">
            <h3>Tasks Completed</h3>
            <p className="stat-number">8,901</p>
          </div>
        </div>
      </main>
    </div>
  );
}

export default DashboardPage;
