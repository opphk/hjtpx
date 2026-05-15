import { lightTheme, darkTheme } from './theme';

export const getGlobalStyles = (theme = lightTheme) => `
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');

  * {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  body {
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    font-size: ${theme.fontSize.md};
    color: ${theme.colors.text};
    background-color: ${theme.colors.background};
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
  }

  #root {
    min-height: 100vh;
  }

  button {
    font-family: inherit;
  }

  input, textarea {
    font-family: inherit;
  }

  a {
    color: ${theme.colors.primary};
    text-decoration: none;
  }

  a:hover {
    color: ${theme.colors.primaryHover};
  }

  ::selection {
    background-color: ${theme.colors.primary};
    color: #ffffff;
  }

  :focus-visible {
    outline: 2px solid ${theme.colors.primary};
    outline-offset: 2px;
  }

  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  ::-webkit-scrollbar-track {
    background: ${theme.colors.backgroundSecondary};
    border-radius: 4px;
  }

  ::-webkit-scrollbar-thumb {
    background: ${theme.colors.border};
    border-radius: 4px;
  }

  ::-webkit-scrollbar-thumb:hover {
    background: ${theme.colors.textTertiary};
  }

  @keyframes captchaFadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes captchaSlideUp {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes captchaPulse {
    0%, 100% {
      transform: scale(1);
    }
    50% {
      transform: scale(1.05);
    }
  }

  @keyframes captchaShake {
    0%, 100% {
      transform: translateX(0);
    }
    10%, 30%, 50%, 70%, 90% {
      transform: translateX(-4px);
    }
    20%, 40%, 60%, 80% {
      transform: translateX(4px);
    }
  }

  .captcha-animation-fade-in {
    animation: captchaFadeIn 0.3s ease;
  }

  .captcha-animation-slide-up {
    animation: captchaSlideUp 0.3s ease;
  }

  .captcha-animation-pulse {
    animation: captchaPulse 0.6s ease infinite;
  }

  .captcha-animation-shake {
    animation: captchaShake 0.5s ease;
  }
`;

export const injectGlobalStyles = (theme) => {
  if (typeof document === 'undefined') return;

  const styleId = 'captchax-global-styles';
  let styleElement = document.getElementById(styleId);

  if (!styleElement) {
    styleElement = document.createElement('style');
    styleElement.id = styleId;
    document.head.appendChild(styleElement);
  }

  styleElement.textContent = getGlobalStyles(theme);
};

export default {
  getGlobalStyles,
  injectGlobalStyles
};
