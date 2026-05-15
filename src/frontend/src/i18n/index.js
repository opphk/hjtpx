import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

const resources = {};
const loadedLanguages = new Set();
const languageCache = new Map();

const supportedLanguages = ['en', 'zh', 'fr', 'de', 'es', 'ru', 'ja', 'ko', 'ar', 'pt', 'it', 'nl'];
const rtlLanguages = ['ar', 'he', 'fa', 'ur'];

const savedLanguage = localStorage.getItem('language') || navigator.language?.split('-')[0] || 'en';
const initialLanguage = supportedLanguages.includes(savedLanguage) ? savedLanguage : 'en';

export const preloadLanguage = async (lng) => {
  if (loadedLanguages.has(lng)) {
    return languageCache.get(lng);
  }

  if (!resources[lng]) {
    const translations = await import(`./locales/${lng}.js`);
    resources[lng] = { translation: translations.default };
    i18n.addResourceBundle(lng, 'translation', translations.default, true, true);
    loadedLanguages.add(lng);
    languageCache.set(lng, translations.default);
  }

  return languageCache.get(lng);
};

export const preloadAllLanguages = async () => {
  const loadPromises = supportedLanguages
    .filter(lng => lng !== initialLanguage)
    .map(lng => preloadLanguage(lng));
  
  await Promise.all(loadPromises);
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    lng: initialLanguage,
    fallbackLng: 'en',
    debug: false,
    interpolation: {
      escapeValue: false
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage']
    },
    react: {
      useSuspense: false
    },
    partialBundledLanguages: true
  });

const initialLoad = async () => {
  const en = await import('./locales/en.js');
  resources.en = { translation: en.default };
  loadedLanguages.add('en');
  languageCache.set('en', en.default);

  const savedLang = localStorage.getItem('language') || initialLanguage;
  if (savedLang !== 'en') {
    await preloadLanguage(savedLang);
  }
};

initialLoad().catch(console.error);

i18n.on('languageChanged', (lng) => {
  localStorage.setItem('language', lng);
  const isRTL = rtlLanguages.includes(lng);
  document.documentElement.dir = isRTL ? 'rtl' : 'ltr';
  document.documentElement.lang = lng;
  document.documentElement.classList.toggle('rtl', isRTL);
  document.documentElement.classList.toggle('ltr', !isRTL);
  
  document.body.dir = isRTL ? 'rtl' : 'ltr';
  
  preloadLanguage(lng).catch(err => {
    console.warn(`Failed to preload language ${lng}:`, err);
  });
});

export const changeLanguage = async (lng) => {
  if (!loadedLanguages.has(lng)) {
    await preloadLanguage(lng);
  }
  await i18n.changeLanguage(lng);
  return lng;
};

export const getCurrentLanguage = () => i18n.language;

export const isRTL = (lng = i18n.language) => rtlLanguages.includes(lng);

export const getSupportedLanguages = () => supportedLanguages;

export const isLanguageLoaded = (lng) => loadedLanguages.has(lng);

export const getLoadedLanguages = () => Array.from(loadedLanguages);

export default i18n;

export const languages = [
  { code: 'en', name: 'English', nativeName: 'English', flag: '🇬🇧', dir: 'ltr', region: 'Europe' },
  { code: 'zh', name: 'Chinese', nativeName: '中文', flag: '🇨🇳', dir: 'ltr', region: 'Asia' },
  { code: 'fr', name: 'French', nativeName: 'Français', flag: '🇫🇷', dir: 'ltr', region: 'Europe' },
  { code: 'de', name: 'German', nativeName: 'Deutsch', flag: '🇩🇪', dir: 'ltr', region: 'Europe' },
  { code: 'es', name: 'Spanish', nativeName: 'Español', flag: '🇪🇸', dir: 'ltr', region: 'Europe' },
  { code: 'ru', name: 'Russian', nativeName: 'Русский', flag: '🇷🇺', dir: 'ltr', region: 'Europe' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語', flag: '🇯🇵', dir: 'ltr', region: 'Asia' },
  { code: 'ko', name: 'Korean', nativeName: '한국어', flag: '🇰🇷', dir: 'ltr', region: 'Asia' },
  { code: 'ar', name: 'Arabic', nativeName: 'العربية', flag: '🇸🇦', dir: 'rtl', region: 'Middle East' },
  { code: 'pt', name: 'Portuguese', nativeName: 'Português', flag: '🇧🇷', dir: 'ltr', region: 'Europe' },
  { code: 'it', name: 'Italian', nativeName: 'Italiano', flag: '🇮🇹', dir: 'ltr', region: 'Europe' },
  { code: 'nl', name: 'Dutch', nativeName: 'Nederlands', flag: '🇳🇱', dir: 'ltr', region: 'Europe' }
];
