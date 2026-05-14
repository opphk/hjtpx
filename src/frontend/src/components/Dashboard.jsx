import React from 'react';
import ChartComponent from './ChartComponent';
import { useTranslation } from 'react-i18next';

const DashboardWidget = ({ title, children, className = '' }) => {
  const { t } = useTranslation();

  return (
    <div className={`dashboard-widget ${className}`}>
      {title && <h3 className="widget-title">{title}</h3>}
      <div className="widget-content">
        {children}
      </div>
    </div>
  );
};

const StatCard = ({ title, value, change, changeType = 'positive', icon }) => {
  const { t } = useTranslation();

  return (
    <div className="stat-card">
      <div className="stat-icon">{icon}</div>
      <div className="stat-info">
        <h4 className="stat-title">{title}</h4>
        <p className="stat-value">{value}</p>
        {change && (
          <span className={`stat-change ${changeType}`}>
            {changeType === 'positive' ? '↑' : '↓'} {change}
          </span>
        )}
      </div>
    </div>
  );
};

const Dashboard = ({ data = {}, widgets = [] }) => {
  const { t } = useTranslation();

  const defaultWidgets = [
    {
      id: 'users',
      title: t('dashboard.totalUsers'),
      type: 'stat',
      value: data.totalUsers || 0
    },
    {
      id: 'active',
      title: t('dashboard.activeUsers'),
      type: 'stat',
      value: data.activeUsers || 0
    },
    {
      id: 'chart',
      title: t('analytics.trendAnalysis'),
      type: 'chart',
      chartType: 'line',
      dataKey: 'date',
      valueKeys: ['users', 'sessions']
    }
  ];

  const allWidgets = widgets.length > 0 ? widgets : defaultWidgets;

  return (
    <div className="dashboard">
      <div className="dashboard-header">
        <h1>{t('dashboard.welcome')}</h1>
        <p>{t('dashboard.overview')}</p>
      </div>

      <div className="dashboard-grid">
        {allWidgets.map((widget) => {
          switch (widget.type) {
            case 'stat':
              return (
                <DashboardWidget key={widget.id} title={widget.title}>
                  <StatCard
                    title={widget.title}
                    value={widget.value}
                    change={widget.change}
                    changeType={widget.changeType}
                    icon={widget.icon}
                  />
                </DashboardWidget>
              );

            case 'chart':
              return (
                <DashboardWidget key={widget.id} title={widget.title}>
                  <ChartComponent
                    type={widget.chartType}
                    data={widget.data || []}
                    xKey={widget.dataKey}
                    yKeys={widget.valueKeys}
                    height={widget.height || 300}
                  />
                </DashboardWidget>
              );

            default:
              return null;
          }
        })}
      </div>
    </div>
  );
};

export { Dashboard, DashboardWidget, StatCard, ChartComponent };
export default Dashboard;
