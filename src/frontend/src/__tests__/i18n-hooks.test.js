import { render, screen, waitFor } from '@testing-library/react';
import { describe, test, expect, jest, beforeEach } from '@jest/globals';
import { I18nProvider } from 'react-i18next';
import i18n from '../i18n';
import { LanguageProvider } from '../components/LanguageSelector';
import useLanguage from '../hooks/useLanguage';
import { useDirection } from '../hooks/useDirection';

const TestComponent = () => {
  const { currentLanguage, changeLanguage, t, isRTL } = useLanguage();
  const { direction, isRTL: isRTLDirection } = useDirection();

  return (
    <div>
      <span data-testid="language">{currentLanguage}</span>
      <span data-testid="direction">{direction}</span>
      <span data-testid="is-rtl">{isRTL.toString()}</span>
      <span data-testid="is-rtl-direction">{isRTLDirection.toString()}</span>
      <span data-testid="translation">{t('common.save')}</span>
      <button onClick={() => changeLanguage('ar')} data-testid="change-ar">
        Change to Arabic
      </button>
      <button onClick={() => changeLanguage('en')} data-testid="change-en">
        Change to English
      </button>
    </div>
  );
};

const Wrapper = ({ children }) => (
  <I18nProvider i18n={i18n}>
    <LanguageProvider>{children}</LanguageProvider>
  </I18nProvider>
);

describe('Internationalization Hooks', () => {
  beforeEach(() => {
    i18n.changeLanguage('en');
  });

  describe('useLanguage', () => {
    test('should return current language', () => {
      render(
        <Wrapper>
          <TestComponent />
        </Wrapper>
      );

      expect(screen.getByTestId('language')).toHaveTextContent('en');
    });

    test('should return supported languages', () => {
      const { result } = renderHook(() => useLanguage(), {
        wrapper: Wrapper
      });

      expect(result.current.supportedLanguages).toContain('en');
      expect(result.current.supportedLanguages).toContain('zh');
      expect(result.current.supportedLanguages).toContain('ar');
    });

    test('should detect RTL languages', () => {
      const { result } = renderHook(() => useLanguage(), {
        wrapper: Wrapper
      });

      expect(result.current.isRTL).toBe(false);
    });

    test('should translate keys correctly', () => {
      render(
        <Wrapper>
          <TestComponent />
        </Wrapper>
      );

      expect(screen.getByTestId('translation')).toHaveTextContent('Save');
    });
  });

  describe('useDirection', () => {
    test('should initialize with LTR direction', () => {
      render(
        <Wrapper>
          <TestComponent />
        </Wrapper>
      );

      expect(screen.getByTestId('direction')).toHaveTextContent('ltr');
      expect(screen.getByTestId('is-rtl-direction')).toHaveTextContent('false');
    });

    test('should change to RTL direction for Arabic', async () => {
      render(
        <Wrapper>
          <TestComponent />
        </Wrapper>
      );

      const user = userEvent.setup();
      await user.click(screen.getByTestId('change-ar'));

      await waitFor(() => {
        expect(screen.getByTestId('direction')).toHaveTextContent('rtl');
        expect(screen.getByTestId('is-rtl-direction')).toHaveTextContent('true');
      });
    });

    test('should change back to LTR for English', async () => {
      render(
        <Wrapper>
          <TestComponent />
        </Wrapper>
      );

      const user = userEvent.setup();
      await user.click(screen.getByTestId('change-ar'));

      await waitFor(() => {
        expect(screen.getByTestId('direction')).toHaveTextContent('rtl');
      });

      await user.click(screen.getByTestId('change-en'));

      await waitFor(() => {
        expect(screen.getByTestId('direction')).toHaveTextContent('ltr');
      });
    });
  });

  describe('Dynamic Language Switching', () => {
    test('should switch between supported languages', async () => {
      render(
        <Wrapper>
          <TestComponent />
        </Wrapper>
      );

      const user = userEvent.setup();

      expect(screen.getByTestId('language')).toHaveTextContent('en');

      await user.click(screen.getByTestId('change-ar'));
      await waitFor(() => {
        expect(screen.getByTestId('language')).toHaveTextContent('ar');
      });

      await user.click(screen.getByTestId('change-en'));
      await waitFor(() => {
        expect(screen.getByTestId('language')).toHaveTextContent('en');
      });
    });
  });

  describe('Translation Interpolation', () => {
    test('should interpolate values in translations', () => {
      const { result } = renderHook(() => useLanguage(), {
        wrapper: Wrapper
      });

      const translation = result.current.t('dashboard.welcome', { name: 'John' });
      expect(translation).toContain('John');
    });
  });
});
