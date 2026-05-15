import React from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../hooks/useAuth';
import SEO, { getSEOMeta } from '../components/SEO';
import DashboardLayout from '../components/DashboardLayout';
import Navigation from '../components/Navigation';
import Alert from '../components/ui/Alert';
import { formatDate } from '../i18n/dateFormat';

const DashboardPage = () => {
  const { t } = useTranslation();
  const { user, isAuthenticated } = useAuth();
  const seoMeta = getSEOMeta('dashboard');

  if (!isAuthenticated) {
    return (
      <>
        <SEO {...seoMeta} />
        <Alert 
        type="warning" 
        message={t('common.error')} 
      />
      </>
    );
  }

  return (
    <>
      <SEO {...seoMeta} />
      <DashboardLayout>
      <div className="dashboard-content">
        <div className="dashboard-header">
          <h1>{t('dashboard.welcomeBack')}, {user?.username || t('users.user')}!</h1>
          <p>{t('dashboard.thisIsYourDashboard')}</p>
        </div>
        
        <div className="dashboard-cards">
          <div className="card">
            <h3>{t('dashboard.personalInfo')}</h3>
            <div className="card-content">
              <p><strong>{t('auth.username')}:</strong> {user?.username}</p>
              <p><strong>{t('auth.email')}:</strong> {user?.email}</p>
              <p><strong>{t('dashboard.role')}:</strong> {user?.role === 'admin' ? t('users.admin') : t('users.user')}</p>
            </div>
          </div>
          
          <div className="card">
            <h3>{t('dashboard.accountStats')}</h3>
            <div className="card-content">
              <p><strong>{t('dashboard.registrationDate')}:</strong> {user?.createdAt ? formatDate(user.createdAt) : t('common.error')}</p>
              <p><strong>{t('dashboard.lastLogin')}:</strong> {user?.lastLogin ? formatDate(user.lastLogin) : t('common.error')}</p>
            </div>
          </div>
        </div>
      </div>
    </DashboardLayout>
    </>
  );
};

export default DashboardPage;
