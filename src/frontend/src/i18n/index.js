import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import en from './locales/en';
import zh from './locales/zh';
import fr from './locales/fr';
import de from './locales/de';
import es from './locales/es';
import ru from './locales/ru';
import ja from './locales/ja';
import ko from './locales/ko';
import ar from './locales/ar';
import pt from './locales/pt';
import it from './locales/it';
import nl from './locales/nl';

const resources = {
  en: { translation: en },
  zh: { translation: zh },
  fr: { translation: fr },
  de: { translation: de },
  es: { translation: es },
  ru: { translation: ru },
  ja: { translation: ja },
  ko: { translation: ko },
  ar: { translation: ar },
  pt: { translation: pt },
  it: { translation: it },
  nl: { translation: nl }
};

const savedLanguage = localStorage.getItem('language') || navigator.language?.split('-')[0] || 'en';
const supportedLanguages = ['en', 'zh', 'fr', 'de', 'es', 'ru', 'ja', 'ko', 'ar', 'pt', 'it', 'nl'];
const initialLanguage = supportedLanguages.includes(savedLanguage) ? savedLanguage : 'en';

const rtlLanguages = ['ar', 'he', 'fa', 'ur'];

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
    }
  });

i18n.on('languageChanged', (lng) => {
  localStorage.setItem('language', lng);
  const isRTL = rtlLanguages.includes(lng);
  document.documentElement.dir = isRTL ? 'rtl' : 'ltr';
  document.documentElement.lang = lng;
  document.documentElement.classList.toggle('rtl', isRTL);
  document.documentElement.classList.toggle('ltr', !isRTL);
  
  document.body.dir = isRTL ? 'rtl' : 'ltr';
});

export const changeLanguage = async (lng) => {
  await i18n.changeLanguage(lng);
  return lng;
};

export const getCurrentLanguage = () => i18n.language;

export const isRTL = (lng = i18n.language) => rtlLanguages.includes(lng);

export const getSupportedLanguages = () => supportedLanguages;

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
