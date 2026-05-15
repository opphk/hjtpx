export const lightTheme = {
  name: 'light',
  colors: {
    primary: '#1890ff',
    primaryHover: '#40a9ff',
    primaryActive: '#096dd9',
    success: '#52c41a',
    successHover: '#73d13d',
    error: '#ff4d4f',
    errorHover: '#ff7875',
    warning: '#faad14',
    info: '#1890ff',
    background: '#ffffff',
    backgroundSecondary: '#fafafa',
    backgroundTertiary: '#f5f5f5',
    text: '#000000',
    textSecondary: '#595959',
    textTertiary: '#8c8c8c',
    border: '#d9d9d9',
    borderSecondary: '#f0f0f0',
    shadow: 'rgba(0, 0, 0, 0.1)',
    overlay: 'rgba(0, 0, 0, 0.45)'
  },
  spacing: {
    xs: '4px',
    sm: '8px',
    md: '12px',
    lg: '16px',
    xl: '20px',
    xxl: '24px'
  },
  borderRadius: {
    sm: '2px',
    md: '4px',
    lg: '8px',
    xl: '12px',
    round: '50%'
  },
  fontSize: {
    xs: '12px',
    sm: '13px',
    md: '14px',
    lg: '16px',
    xl: '20px',
    xxl: '24px'
  },
  transitions: {
    fast: '0.15s ease',
    normal: '0.2s ease',
    slow: '0.3s ease'
  }
};

export const darkTheme = {
  name: 'dark',
  colors: {
    primary: '#1890ff',
    primaryHover: '#40a9ff',
    primaryActive: '#096dd9',
    success: '#52c41a',
    successHover: '#73d13d',
    error: '#ff4d4f',
    errorHover: '#ff7875',
    warning: '#faad14',
    info: '#1890ff',
    background: '#141414',
    backgroundSecondary: '#1f1f1f',
    backgroundTertiary: '#262626',
    text: '#ffffff',
    textSecondary: '#d9d9d9',
    textTertiary: '#a6a6a6',
    border: '#434343',
    borderSecondary: '#303030',
    shadow: 'rgba(0, 0, 0, 0.3)',
    overlay: 'rgba(0, 0, 0, 0.75)'
  },
  spacing: lightTheme.spacing,
  borderRadius: lightTheme.borderRadius,
  fontSize: lightTheme.fontSize,
  transitions: lightTheme.transitions
};

export const createTheme = (options = {}) => {
  const { mode = 'light', primaryColor, ...custom } = options;

  const baseTheme = mode === 'dark' ? darkTheme : lightTheme;

  if (primaryColor) {
    baseTheme.colors.primary = primaryColor;
    baseTheme.colors.primaryHover = adjustColor(primaryColor, 20);
    baseTheme.colors.primaryActive = adjustColor(primaryColor, -10);
  }

  return {
    ...baseTheme,
    ...custom
  };
};

const adjustColor = (hex, percent) => {
  const num = parseInt(hex.replace('#', ''), 16);
  const amt = Math.round(2.55 * percent);
  const R = (num >> 16) + amt;
  const G = (num >> 8 & 0x00FF) + amt;
  const B = (num & 0x0000FF) + amt;

  return '#' + (
    0x1000000 +
    (R < 255 ? (R < 1 ? 0 : R) : 255) * 0x10000 +
    (G < 255 ? (G < 1 ? 0 : G) : 255) * 0x100 +
    (B < 255 ? (B < 1 ? 0 : B) : 255)
  ).toString(16).slice(1);
};

export default {
  lightTheme,
  darkTheme,
  createTheme
};
