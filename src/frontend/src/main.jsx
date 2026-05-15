import React, { createContext, useContext } from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.jsx';
import './styles/global.css';
import './i18n';
import PWAInstallPrompt from './components/PWAInstallPrompt.jsx';
import useServiceWorker from './hooks/useServiceWorker.js';
import usePushNotifications from './hooks/usePushNotifications.js';

const PWAContext = createContext(null);

export const usePWA = () => {
  const context = useContext(PWAContext);
  if (!context) {
    throw new Error('usePWA must be used within a PWAProvider');
  }
  return context;
};

const PWAProvider = ({ children }) => {
  const sw = useServiceWorker();
  const push = usePushNotifications();

  return (
    <PWAContext.Provider value={{ sw, push }}>
      {children}
    </PWAContext.Provider>
  );
};

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <PWAProvider>
      <App />
      <PWAInstallPrompt />
    </PWAProvider>
  </React.StrictMode>,
);
