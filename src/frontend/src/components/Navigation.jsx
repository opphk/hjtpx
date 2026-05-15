import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../hooks/useAuth';
import LanguageSwitcher from './LanguageSwitcher';

const Navigation = () => {
  const { t } = useTranslation();
  const { user, logout, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [swiping, setSwiping] = useState(false);
  const [swipeStart, setSwipeStart] = useState(0);
  const drawerRef = useRef(null);
  const toggleButtonRef = useRef(null);
  const lastFocusedElementRef = useRef(null);

  useEffect(() => {
    setMobileMenuOpen(false);
  }, [location]);

  useEffect(() => {
    if (mobileMenuOpen) {
      lastFocusedElementRef.current = document.activeElement;
      document.body.style.overflow = 'hidden';
      
      const focusableElements = drawerRef.current?.querySelectorAll(
        'a, button, [tabindex]:not([tabindex="-1"])'
      );
      if (focusableElements?.[0]) {
        focusableElements[0].focus();
      }
    } else {
      document.body.style.overflow = '';
      if (lastFocusedElementRef.current) {
        lastFocusedElementRef.current.focus();
      }
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [mobileMenuOpen]);

  const focusTrap = useCallback((e) => {
    if (!drawerRef.current || !mobileMenuOpen) return;
    
    const focusableElements = drawerRef.current.querySelectorAll(
      'a, button, [tabindex]:not([tabindex="-1"])'
    );
    
    if (focusableElements.length === 0) return;
    
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    
    if (e.key === 'Tab') {
      if (e.shiftKey && document.activeElement === firstElement) {
        e.preventDefault();
        lastElement.focus();
      } else if (!e.shiftKey && document.activeElement === lastElement) {
        e.preventDefault();
        firstElement.focus();
      }
    }
  }, [mobileMenuOpen]);

  const handleEscape = useCallback((e) => {
    if (e.key === 'Escape' && mobileMenuOpen) {
      setMobileMenuOpen(false);
    }
  }, [mobileMenuOpen]);

  useEffect(() => {
    if (mobileMenuOpen) {
      document.addEventListener('keydown', handleEscape);
      document.addEventListener('keydown', focusTrap);
    }
    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.removeEventListener('keydown', focusTrap);
    };
  }, [mobileMenuOpen, handleEscape, focusTrap]);

  const handleLogout = async () => {
    setMobileMenuOpen(false);
    await logout();
    navigate('/login');
  };

  const handleOverlayClick = () => {
    setMobileMenuOpen(false);
  };

  const handleSwipeStart = (e) => {
    if (e.touches && e.touches.length === 1) {
      setSwipeStart(e.touches[0].clientX);
      setSwiping(true);
    }
  };

  const handleSwipeMove = (e) => {
    if (!swiping || !drawerRef.current) return;
    const currentX = e.touches[0].clientX;
    const diff = currentX - swipeStart;
    if (diff < 0) {
      const drawer = drawerRef.current;
      drawer.style.transform = `translateX(${Math.max(diff, 0)}px)`;
    }
  };

  const handleSwipeEnd = (e) => {
    if (!swiping) return;
    setSwiping(false);
    if (drawerRef.current) {
      drawerRef.current.style.transform = '';
      const endX = e.changedTouches[0].clientX;
      const diff = swipeStart - endX;
      if (diff > 80) {
        setMobileMenuOpen(false);
      }
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <>
      <nav className="navbar" role="navigation" aria-label={t('nav.mainNavigation')}>
        <div className="navbar-brand">
          <Link to="/dashboard" aria-label={`${t('common.appName')} - ${t('nav.home')}`}>
            {t('common.appName')}
          </Link>
        </div>

        <div className="navbar-menu" role="menubar" aria-label={t('nav.mainMenu')}>
          <Link
            to="/dashboard"
            className={`nav-link ${location.pathname === '/dashboard' ? 'active' : ''}`}
            role="menuitem"
            aria-current={location.pathname === '/dashboard' ? 'page' : undefined}
          >
            {t('nav.home')}
          </Link>
          <Link
            to="/users"
            className={`nav-link ${location.pathname === '/users' ? 'active' : ''}`}
            role="menuitem"
            aria-current={location.pathname === '/users' ? 'page' : undefined}
          >
            {t('nav.users')}
          </Link>
          <Link
            to="/settings"
            className={`nav-link ${location.pathname === '/settings' ? 'active' : ''}`}
            role="menuitem"
            aria-current={location.pathname === '/settings' ? 'page' : undefined}
          >
            {t('nav.settings')}
          </Link>
          <Link
            to="/logs"
            className={`nav-link ${location.pathname === '/logs' ? 'active' : ''}`}
            role="menuitem"
            aria-current={location.pathname === '/logs' ? 'page' : undefined}
          >
            {t('nav.logs')}
          </Link>
          <Link
            to="/admin/audit"
            className={`nav-link ${location.pathname === '/admin/audit' ? 'active' : ''}`}
            role="menuitem"
            aria-current={location.pathname === '/admin/audit' ? 'page' : undefined}
          >
            {t('nav.audit')}
          </Link>

          <div className="navbar-user">
            <LanguageSwitcher />
            <span className="user-info" aria-label={t('nav.currentUser', { username: user?.username })}>
              {user?.username}
            </span>
            <button 
              onClick={handleLogout} 
              className="btn btn-logout"
              aria-label={t('nav.logout')}
              type="button"
            >
              {t('nav.logout')}
            </button>
          </div>
        </div>

        <button
          ref={toggleButtonRef}
          className="mobile-nav-toggle"
          onClick={() => setMobileMenuOpen(true)}
          aria-label={t('nav.openMenu')}
          aria-expanded={mobileMenuOpen}
          aria-controls="mobile-nav-drawer"
          type="button"
        >
          <span style={{ display: 'flex', alignItems: 'center' }}>
            <svg
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <line x1="3" y1="6" x2="21" y2="6" />
              <line x1="3" y1="12" x2="21" y2="12" />
              <line x1="3" y1="18" x2="21" y2="18" />
            </svg>
          </span>
        </button>
      </nav>

      {mobileMenuOpen && (
        <div
          className={`mobile-nav-overlay ${mobileMenuOpen ? 'active' : ''}`}
          onClick={handleOverlayClick}
          role="presentation"
          aria-hidden="true"
        />
      )}

      <div
        id="mobile-nav-drawer"
        ref={drawerRef}
        className={`mobile-nav-drawer ${mobileMenuOpen ? 'open' : ''}`}
        onTouchStart={handleSwipeStart}
        onTouchMove={handleSwipeMove}
        onTouchEnd={handleSwipeEnd}
        role="dialog"
        aria-modal="true"
        aria-label={t('nav.mobileNavigation')}
        aria-hidden={!mobileMenuOpen}
        tabIndex={-1}
      >
        <div className="mobile-nav-header">
          <span 
            style={{ fontWeight: 600, fontSize: '18px' }}
            id="mobile-menu-title"
          >
            {t('nav.menu')}
          </span>
          <button
            className="mobile-nav-close"
            onClick={() => setMobileMenuOpen(false)}
            aria-label={t('nav.closeMenu')}
            type="button"
          >
            <svg
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div 
          className="mobile-nav-links" 
          role="menu" 
          aria-labelledby="mobile-menu-title"
        >
          <Link
            to="/dashboard"
            className={`mobile-nav-link ${location.pathname === '/dashboard' ? 'active' : ''}`}
            onClick={() => setMobileMenuOpen(false)}
            role="menuitem"
            aria-current={location.pathname === '/dashboard' ? 'page' : undefined}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              style={{ marginRight: '12px' }}
              aria-hidden="true"
            >
              <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
              <polyline points="9 22 9 12 15 12 15 22" />
            </svg>
            {t('nav.home')}
          </Link>
          <Link
            to="/users"
            className={`mobile-nav-link ${location.pathname === '/users' ? 'active' : ''}`}
            onClick={() => setMobileMenuOpen(false)}
            role="menuitem"
            aria-current={location.pathname === '/users' ? 'page' : undefined}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              style={{ marginRight: '12px' }}
              aria-hidden="true"
            >
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
              <path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
            {t('nav.users')}
          </Link>
          <Link
            to="/settings"
            className={`mobile-nav-link ${location.pathname === '/settings' ? 'active' : ''}`}
            onClick={() => setMobileMenuOpen(false)}
            role="menuitem"
            aria-current={location.pathname === '/settings' ? 'page' : undefined}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              style={{ marginRight: '12px' }}
              aria-hidden="true"
            >
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            {t('nav.settings')}
          </Link>
          <Link
            to="/logs"
            className={`mobile-nav-link ${location.pathname === '/logs' ? 'active' : ''}`}
            onClick={() => setMobileMenuOpen(false)}
            role="menuitem"
            aria-current={location.pathname === '/logs' ? 'page' : undefined}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              style={{ marginRight: '12px' }}
              aria-hidden="true"
            >
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
              <polyline points="14 2 14 8 20 8" />
              <line x1="16" y1="13" x2="8" y2="13" />
              <line x1="16" y1="17" x2="8" y2="17" />
              <polyline points="10 9 9 9 8 9" />
            </svg>
            {t('nav.logs')}
          </Link>
          <Link
            to="/admin/audit"
            className={`mobile-nav-link ${location.pathname === '/admin/audit' ? 'active' : ''}`}
            onClick={() => setMobileMenuOpen(false)}
            role="menuitem"
            aria-current={location.pathname === '/admin/audit' ? 'page' : undefined}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              style={{ marginRight: '12px' }}
              aria-hidden="true"
            >
              <path d="M12 20h9" />
              <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z" />
            </svg>
            {t('nav.audit')}
          </Link>
        </div>

        <div className="mobile-nav-footer">
          <div 
            style={{ marginBottom: '12px', padding: '12px', backgroundColor: '#f5f5f5', borderRadius: '6px' }}
            aria-label={t('nav.userInfo', { username: user?.username, role: user?.role === 'admin' ? t('users.admin') : t('users.user') })}
          >
            <div style={{ fontSize: '14px', fontWeight: 500, marginBottom: '4px' }}>
              {user?.username}
            </div>
            <div style={{ fontSize: '12px', color: '#8c8c8c' }}>
              {user?.role === 'admin' ? t('users.admin') : t('users.user')}
            </div>
          </div>
          <LanguageSwitcher />
          <button
            onClick={handleLogout}
            className="btn btn-logout"
            style={{ width: '100%', marginTop: '12px' }}
            aria-label={t('auth.logout')}
            type="button"
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              style={{ marginRight: '8px' }}
              aria-hidden="true"
            >
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1="21" y1="12" x2="9" y2="12" />
            </svg>
            {t('auth.logout')}
          </button>
        </div>
      </div>
    </>
  );
};

export default Navigation;
