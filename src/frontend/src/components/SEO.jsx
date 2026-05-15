import React, { useEffect } from 'react';

const SEO = ({ 
  title, 
  description, 
  keywords, 
  ogImage,
  ogType = 'website',
  canonical,
  noIndex = false 
}) => {
  useEffect(() => {
    const baseTitle = 'HJTPX 系统';
    const fullTitle = title ? `${title} | ${baseTitle}` : baseTitle;
    
    document.title = fullTitle;
    
    updateMetaTag('description', description || 'HJTPX 系统 - 现代化全栈应用程序，提供用户管理、数据分析、审计追踪等功能。支持实时通知、多语言界面和响应式设计。');
    updateMetaTag('keywords', keywords || 'HJTPX,全栈应用,用户管理,数据分析,系统管理,React,Node.js,实时通知,Web应用');
    
    updateMetaTag('og:title', fullTitle, 'property');
    updateMetaTag('og:description', description, 'property');
    updateMetaTag('og:image', ogImage || 'https://hjtpx.com/og-image.png', 'property');
    updateMetaTag('og:type', ogType, 'property');
    updateMetaTag('og:url', canonical || window.location.href, 'property');
    
    updateMetaTag('twitter:title', fullTitle);
    updateMetaTag('twitter:description', description);
    updateMetaTag('twitter:image', ogImage || 'https://hjtpx.com/og-image.png');
    
    if (canonical) {
      updateCanonical(canonical);
    }
    
    if (noIndex) {
      updateMetaTag('robots', 'noindex, nofollow');
    }
  }, [title, description, keywords, ogImage, ogType, canonical, noIndex]);

  return null;
};

const updateMetaTag = (name, content, type = 'name') => {
  if (!content) return;
  
  let selector = type === 'property' 
    ? `meta[property="${name}"]` 
    : `meta[name="${name}"]`;
  
  let metaTag = document.querySelector(selector);
  
  if (!metaTag) {
    metaTag = document.createElement('meta');
    if (type === 'property') {
      metaTag.setAttribute('property', name);
    } else {
      metaTag.setAttribute('name', name);
    }
    document.head.appendChild(metaTag);
  }
  
  metaTag.setAttribute('content', content);
};

const updateCanonical = (url) => {
  let canonical = document.querySelector('link[rel="canonical"]');
  
  if (!canonical) {
    canonical = document.createElement('link');
    canonical.setAttribute('rel', 'canonical');
    document.head.appendChild(canonical);
  }
  
  canonical.setAttribute('href', url);
};

export const getSEOMeta = (page) => {
  const metaConfig = {
    login: {
      title: '用户登录',
      description: '登录 HJTPX 系统，访问您的个人仪表板，管理您的账户和设置。',
      keywords: '登录,HJTPX,用户登录,账户管理',
      ogType: 'website',
      canonical: 'https://hjtpx.com/login'
    },
    register: {
      title: '用户注册',
      description: '注册 HJTPX 系统账户，开始使用强大的用户管理、数据分析和审计追踪功能。',
      keywords: '注册,HJTPX,新用户,账户创建',
      ogType: 'website',
      canonical: 'https://hjtpx.com/register'
    },
    dashboard: {
      title: '个人仪表板',
      description: '访问您的 HJTPX 个人仪表板，查看账户统计、个人信息和最近活动。',
      keywords: '仪表板,HJTPX,个人中心,账户统计',
      ogType: 'website',
      canonical: 'https://hjtpx.com/dashboard'
    },
    settings: {
      title: '系统设置',
      description: '配置 HJTPX 系统设置，管理通知偏好、功能开关和系统参数。',
      keywords: '设置,HJTPX,系统配置,通知管理',
      ogType: 'website',
      canonical: 'https://hjtpx.com/settings'
    },
    adminUsers: {
      title: '用户管理',
      description: '管理 HJTPX 系统中的所有用户账户、权限分配和用户状态。',
      keywords: '用户管理,HJTPX,管理员,账户管理',
      ogType: 'website',
      canonical: 'https://hjtpx.com/admin/users',
      noIndex: true
    },
    logs: {
      title: '日志管理',
      description: '查看和管理 HJTPX 系统的操作日志、错误日志和安全日志。',
      keywords: '日志管理,HJTPX,操作日志,错误追踪',
      ogType: 'website',
      canonical: 'https://hjtpx.com/logs',
      noIndex: true
    },
    audit: {
      title: '审计仪表板',
      description: '监控 HJTPX 系统的用户活动、系统变更和安全事件。',
      keywords: '审计,HJTPX,安全监控,活动追踪',
      ogType: 'website',
      canonical: 'https://hjtpx.com/audit',
      noIndex: true
    },
    home: {
      title: '首页',
      description: 'HJTPX 系统 - 现代化全栈应用程序，提供用户管理、数据分析、审计追踪等功能。支持实时通知、多语言界面和响应式设计。',
      keywords: 'HJTPX,全栈应用,用户管理,数据分析,系统管理,React,Node.js,实时通知,Web应用',
      ogType: 'website',
      canonical: 'https://hjtpx.com'
    }
  };

  return metaConfig[page] || metaConfig.home;
};

export default SEO;
