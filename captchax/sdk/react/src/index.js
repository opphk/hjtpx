export { CaptchaButton } from './components/CaptchaButton';
export { CaptchaDialog } from './components/CaptchaDialog';
export { CaptchaCard } from './components/CaptchaCard';
export { CaptchaInput } from './components/CaptchaInput';
export { CaptchaSlider } from './components/CaptchaSlider';

export { SliderCaptcha } from './components/captchas/SliderCaptcha';
export { ClickCaptcha } from './components/captchas/ClickCaptcha';
export { RotateCaptcha } from './components/captchas/RotateCaptcha';
export { PuzzleCaptcha } from './components/captchas/PuzzleCaptcha';
export { TextCaptcha } from './components/captchas/TextCaptcha';
export { IconCaptcha } from './components/captchas/IconCaptcha';

export { CaptchaProvider, useCaptcha } from './CaptchaProvider';
export { useCaptchaState } from './hooks/useCaptchaState';

export { lightTheme, darkTheme, createTheme } from './styles/theme';
export { getGlobalStyles, injectGlobalStyles } from './styles/globalStyles';

export * from './utils/constants';
export * from './utils/api';
export * from './utils/fingerprint';

const version = '1.0.0';

export { version };

export default {
  version,
  CaptchaButton,
  CaptchaDialog,
  CaptchaCard,
  CaptchaInput,
  CaptchaSlider,
  SliderCaptcha,
  ClickCaptcha,
  RotateCaptcha,
  PuzzleCaptcha,
  TextCaptcha,
  IconCaptcha,
  CaptchaProvider,
  useCaptcha,
  useCaptchaState,
  lightTheme,
  darkTheme,
  createTheme
};
