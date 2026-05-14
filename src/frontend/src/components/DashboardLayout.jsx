import React from 'react';
import Navigation from './Navigation';
import Loading from './ui/Loading';

const DashboardLayout = ({ children }) => {
  return (
    <div className="dashboard-layout">
      <Navigation />
      <main className="dashboard-main">
        <div className="container">
          {children}
        </div>
      </main>
    </div>
  );
};

export default DashboardLayout;
