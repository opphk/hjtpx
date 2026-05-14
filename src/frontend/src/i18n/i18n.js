import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import zh from './locales/zh.json';
import ja from './locales/ja.json';
import ko from './locales/ko.json';

export const supportedLanguages = [
  { code: 'zh', name: '中文', nativeName: '中文', dir: 'ltr' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語', dir: 'ltr' },
  { code: 'ko', name: 'Korean', nativeName: '한국어', dir: 'ltr' },
  { code: 'en', name: 'English', nativeName: 'English', dir: 'ltr' }
];

export const defaultLanguage = 'zh';
export const fallbackLanguage = 'en';

const resources = {
  zh: { translation: zh },
  ja: { translation: ja },
  ko: { translation: ko },
  en: {
    translation: {
      app: {
        name: 'HJTPX System',
        description: 'Modern full-stack application'
      },
      nav: {
        home: 'Home',
        dashboard: 'Dashboard',
        users: 'Users',
        settings: 'Settings',
        profile: 'Profile',
        logout: 'Logout',
        login: 'Login',
        register: 'Register'
      },
      dashboard: {
        title: 'Dashboard',
        welcome: 'Welcome back',
        overview: 'Overview',
        recentActivity: 'Recent Activity',
        statistics: 'Statistics',
        notifications: 'Notifications',
        quickActions: 'Quick Actions'
      },
      users: {
        title: 'User Management',
        list: 'User List',
        add: 'Add User',
        edit: 'Edit User',
        delete: 'Delete User',
        search: 'Search Users',
        name: 'Name',
        email: 'Email',
        role: 'Role',
        status: 'Status',
        createdAt: 'Created At',
        actions: 'Actions',
        confirmDelete: 'Are you sure you want to delete this user?',
        success: 'User operation successful',
        error: 'User operation failed'
      },
      auth: {
        login: 'Login',
        logout: 'Logout',
        register: 'Register',
        email: 'Email address',
        password: 'Password',
        confirmPassword: 'Confirm password',
        rememberMe: 'Remember me',
        forgotPassword: 'Forgot password?',
        noAccount: "Don't have an account?",
        hasAccount: 'Already have an account?',
        loginSuccess: 'Login successful',
        loginError: 'Login failed',
        logoutSuccess: 'Logout successful'
      },
      common: {
        save: 'Save',
        cancel: 'Cancel',
        delete: 'Delete',
        edit: 'Edit',
        add: 'Add',
        search: 'Search',
        filter: 'Filter',
        export: 'Export',
        import: 'Import',
        refresh: 'Refresh',
        loading: 'Loading...',
        noData: 'No data available',
        success: 'Operation successful',
        error: 'Operation failed',
        warning: 'Warning',
        info: 'Information',
        confirm: 'Confirm',
        yes: 'Yes',
        no: 'No',
        ok: 'OK',
        close: 'Close',
        back: 'Back',
        next: 'Next',
        previous: 'Previous',
        submit: 'Submit',
        reset: 'Reset',
        required: 'Required field',
        optional: 'Optional field'
      },
      validation: {
        required: 'This field is required',
        email: 'Please enter a valid email address',
        minLength: 'Minimum {{count}} characters required',
        maxLength: 'Maximum {{count}} characters allowed',
        pattern: 'Invalid format',
        match: 'Values do not match',
        number: 'Please enter a number',
        integer: 'Please enter an integer',
        positive: 'Please enter a positive number',
        date: 'Please enter a valid date',
        url: 'Please enter a valid URL'
      },
      errors: {
        title: 'Error',
        404: 'Page not found',
        500: 'Server error',
        network: 'Network error',
        timeout: 'Request timeout',
        unauthorized: 'Unauthorized',
        forbidden: 'Access forbidden',
        notFound: 'Resource not found',
        serverError: 'Internal server error',
        tryAgain: 'Please try again',
        contactSupport: 'Please contact support'
      },
      notifications: {
        title: 'Notifications',
        markAllRead: 'Mark all as read',
        noNotifications: 'No notifications',
        new: 'New',
        earlier: 'Earlier'
      },
      settings: {
        title: 'Settings',
        general: 'General',
        account: 'Account',
        security: 'Security',
        notifications: 'Notifications',
        appearance: 'Appearance',
        language: 'Language',
        theme: 'Theme',
        darkMode: 'Dark mode',
        lightMode: 'Light mode',
        autoMode: 'Auto'
      },
      table: {
        showing: 'Showing {{start}} to {{end}} of {{total}} entries',
        rowsPerPage: 'Rows per page',
        page: 'Page',
        of: 'of',
        first: 'First',
        previous: 'Previous',
        next: 'Next',
        last: 'Last'
      },
      time: {
        now: 'Just now',
        minutesAgo: '{{count}} minutes ago',
        hoursAgo: '{{count}} hours ago',
        daysAgo: '{{count}} days ago',
        weeksAgo: '{{count}} weeks ago',
        monthsAgo: '{{count}} months ago',
        yearsAgo: '{{count}} years ago'
      }
    }
  }
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: fallbackLanguage,
    defaultNS: 'translation',
    ns: ['translation'],

    detection: {
      order: ['localStorage', 'navigator', 'htmlTag', 'path', 'subdomain'],
      caches: ['localStorage'],
      lookupLocalStorage: 'hjtpx-language',
      lookupSessionStorage: 'hjtpx-language-session',

      caches: ['localStorage', 'sessionStorage'],

      lookupCookie: 'hjtpx-language',
      cookieMinutes: 60 * 24 * 365,

      lookupFromPathIndex: 0,
      lookupFromSubdomainIndex: 0,

      htmlTag: document.documentElement,

      checkWhitelist: true
    },

    interpolation: {
      escapeValue: false,
      format: (value, format, lng) => {
        if (format === 'date') {
          return new Date(value).toLocaleDateString(lng, {
            year: 'numeric',
            month: 'long',
            day: 'numeric'
          });
        }
        if (format === 'time') {
          return new Date(value).toLocaleTimeString(lng, {
            hour: '2-digit',
            minute: '2-digit'
          });
        }
        if (format === 'datetime') {
          return new Date(value).toLocaleString(lng);
        }
        if (format === 'number') {
          return new Intl.NumberFormat(lng).format(value);
        }
        if (format === 'currency') {
          return new Intl.NumberFormat(lng, {
            style: 'currency',
            currency: value.currency || 'USD'
          }).format(value.amount || value);
        }
        if (format === 'relativeTime') {
          const rtf = new Intl.RelativeTimeFormat(lng, { numeric: 'auto' });
          const diff = value - Date.now();
          const absDiff = Math.abs(diff);
          const duration = absDiff < 60000
            ? { value: diff / 1000, unit: 'second' }
            : absDiff < 3600000
            ? { value: diff / 60000, unit: 'minute' }
            : absDiff < 86400000
            ? { value: diff / 3600000, unit: 'hour' }
            : { value: diff / 86400000, unit: 'day' };
          return rtf.format(Math.round(duration.value), duration.unit);
        }
        return value;
      }
    },

    react: {
      useSuspense: true,
      bindI18n: 'languageChanged loaded',
      bindI18nStore: 'added removed',
      nsMode: 'default',
      wait: true
    },

    backend: {
      loadPath: '/locales/{{lng}}/{{ns}}.json',
      addPath: '/locales/{{lng}}/{{ns}}.json',
      updatePath: '/locales/{{lng}}/{{ns}}.json',
      multiSeparator: '+',
      requestOptions: {
        mode: 'cors',
        credentials: 'same-origin'
      }
    },

    whitelist: supportedLanguages.map(lang => lang.code),

    nonExplicitWhitelist: false,

    load: 'languageOnly',

    preload: [defaultLanguage, fallbackLanguage],

    missingKeyNoValueFallbackToKey: true,

    skipOnVariables: false,

    keySeparator: '.',

    nsSeparator: ':',

    pluralSeparator: '_',

    contextSeparator: '_',

    appendNamespaceToCIMode: false,

    returnEmptyString: true,

    returnNull: false,

    returnObjects: false,

    returnedObjectHandler: (key, value, options) => value,

    simpleEmptyStringResult: true,

    parseMissingKey: 'empty',

    stringify: JSON.stringify,

    isEnabled: (lng, ns, options) => true,

    enableImplicit: true,

    enableMounting: true
  });

export const changeLanguage = async (lng) => {
  if (!supportedLanguages.find(lang => lang.code === lng)) {
    console.warn(`Language '${lng}' is not supported. Falling back to '${fallbackLanguage}'.`);
    lng = fallbackLanguage;
  }

  await i18n.changeLanguage(lng);

  localStorage.setItem('hjtpx-language', lng);
  localStorage.setItem('hjtpx-language-version', '2');

  document.documentElement.lang = lng;
  document.documentElement.dir = supportedLanguages.find(l => l.code === lng)?.dir || 'ltr';

  if (i18n.options?.detection?.caches?.includes('localStorage')) {
    localStorage.setItem(i18n.options.detection.lookupLocalStorage, lng);
  }

  return lng;
};

export const getCurrentLanguage = () => {
  return i18n.language || defaultLanguage;
};

export const getLanguageInfo = (lng) => {
  return supportedLanguages.find(lang => lang.code === lng) || supportedLanguages[0];
};

export const getAvailableLanguages = () => {
  return supportedLanguages;
};

export const isLanguageSupported = (lng) => {
  return supportedLanguages.some(lang => lang.code === lng);
};

export const loadLanguageBundle = async (lng, ns = 'translation') => {
  try {
    await i18n.loadLanguages([lng]);
    return true;
  } catch (error) {
    console.error(`Failed to load language bundle for '${lng}':`, error);
    return false;
  }
};

export const reloadLanguageBundle = async (lng, ns = 'translation') => {
  try {
    const bundle = await import(`./locales/${lng}.json`);
    i18n.addResourceBundle(lng, ns, bundle.default || bundle, true, true);
    return true;
  } catch (error) {
    console.error(`Failed to reload language bundle for '${lng}':`, error);
    return false;
  }
};

export default i18n;
