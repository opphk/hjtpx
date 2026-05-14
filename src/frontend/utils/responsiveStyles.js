export function getResponsiveStyles(styles, deviceType) {
  if (typeof styles === 'function') {
    return styles(deviceType);
  }

  if (typeof styles === 'object' && styles !== null) {
    const responsive = {};

    for (const [key, value] of Object.entries(styles)) {
      if (key === 'base') {
        Object.assign(responsive, value);
      } else if (key === 'mobile' && deviceType === 'mobile') {
        Object.assign(responsive, value);
      } else if (key === 'tablet' && deviceType === 'tablet') {
        Object.assign(responsive, value);
      } else if (key === 'desktop' && deviceType === 'desktop') {
        Object.assign(responsive, value);
      } else {
        responsive[key] = value;
      }
    }

    return responsive;
  }

  return styles;
}

export function createResponsiveComponent(styles) {
  return function ResponsiveComponent({ deviceType, children, ...props }) {
    const responsiveStyles = getResponsiveStyles(styles, deviceType);
    return children ? children(responsiveStyles, props) : <div style={responsiveStyles} {...props} />;
  };
}

export const spacingScale = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48
};

export const fontSizeScale = {
  xs: 12,
  sm: 14,
  md: 16,
  lg: 18,
  xl: 20,
  xxl: 24,
  xxxl: 32
};

export const borderRadiusScale = {
  none: 0,
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
  full: 9999
};

export function createSpacing(responsive = true) {
  return function spacing(key, multiplier = 1) {
    const value = spacingScale[key] || spacingScale.md;

    if (responsive) {
      return {
        padding: value * multiplier,
        margin: value * multiplier
      };
    }

    return {
      padding: `${value * multiplier}px`,
      margin: `${value * multiplier}px`
    };
  };
}

export function createResponsiveLayout(options = {}) {
  const {
    direction = 'column',
    align = 'center',
    justify = 'center',
    gap = 'md'
  } = options;

  return {
    flexDirection: direction,
    alignItems: align,
    justifyContent: justify,
    gap: spacingScale[gap] || gap,
    display: 'flex'
  };
}

export function createCardStyles(options = {}) {
  const {
    padding = 'md',
    borderRadius = 'lg',
    elevation = 'medium',
    backgroundColor = '#ffffff'
  } = options;

  const elevationStyles = {
    none: {},
    low: {
      boxShadow: '0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.24)'
    },
    medium: {
      boxShadow: '0 3px 6px rgba(0,0,0,0.15), 0 2px 4px rgba(0,0,0,0.12)'
    },
    high: {
      boxShadow: '0 10px 20px rgba(0,0,0,0.15), 0 3px 6px rgba(0,0,0,0.10)'
    }
  };

  return {
    padding: spacingScale[padding] || padding,
    borderRadius: borderRadiusScale[borderRadius] || borderRadius,
    backgroundColor,
    ...elevationStyles[elevation]
  };
}

export function createButtonStyles(options = {}) {
  const {
    size = 'md',
    variant = 'primary',
    fullWidth = false,
    disabled = false
  } = options;

  const sizes = {
    sm: {
      padding: '8px 16px',
      fontSize: 14,
      borderRadius: 4
    },
    md: {
      padding: '12px 24px',
      fontSize: 16,
      borderRadius: 6
    },
    lg: {
      padding: '16px 32px',
      fontSize: 18,
      borderRadius: 8
    }
  };

  const variants = {
    primary: {
      backgroundColor: '#2196f3',
      color: '#ffffff',
      border: 'none'
    },
    secondary: {
      backgroundColor: '#f5f5f5',
      color: '#333333',
      border: '1px solid #e0e0e0'
    },
    outline: {
      backgroundColor: 'transparent',
      color: '#2196f3',
      border: '2px solid #2196f3'
    },
    text: {
      backgroundColor: 'transparent',
      color: '#2196f3',
      border: 'none',
      padding: '8px 12px'
    }
  };

  const disabledStyles = disabled
    ? {
        opacity: 0.6,
        cursor: 'not-allowed'
      }
    : {};

  return {
    ...sizes[size],
    ...variants[variant],
    ...disabledStyles,
    width: fullWidth ? '100%' : 'auto',
    cursor: disabled ? 'not-allowed' : 'pointer',
    fontWeight: 500,
    textAlign: 'center',
    transition: 'all 0.2s ease-in-out'
  };
}

export function createInputStyles(options = {}) {
  const {
    size = 'md',
    error = false,
    disabled = false,
    fullWidth = true
  } = options;

  const sizes = {
    sm: {
      padding: '8px 12px',
      fontSize: 14
    },
    md: {
      padding: '12px 16px',
      fontSize: 16
    },
    lg: {
      padding: '16px 20px',
      fontSize: 18
    }
  };

  const baseStyles = {
    border: error ? '2px solid #f44336' : '1px solid #e0e0e0',
    borderRadius: 6,
    outline: 'none',
    transition: 'border-color 0.2s ease-in-out, box-shadow 0.2s ease-in-out',
    backgroundColor: disabled ? '#f5f5f5' : '#ffffff',
    color: disabled ? '#9e9e9e' : '#333333'
  };

  const focusStyles = {
    borderColor: error ? '#f44336' : '#2196f3',
    boxShadow: error
      ? '0 0 0 3px rgba(244, 67, 54, 0.2)'
      : '0 0 0 3px rgba(33, 150, 243, 0.2)'
  };

  return {
    ...sizes[size],
    ...baseStyles,
    width: fullWidth ? '100%' : 'auto',
    cursor: disabled ? 'not-allowed' : 'text',
    ':focus': focusStyles
  };
}

export const globalStyles = {
  html: {
    fontSize: 16,
    boxSizing: 'border-box'
  },
  body: {
    margin: 0,
    padding: 0,
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
    WebkitFontSmoothing: 'antialiased',
    MozOsxFontSmoothing: 'grayscale',
    overflowX: 'hidden'
  },
  '*': {
    boxSizing: 'inherit'
  },
  a: {
    color: '#2196f3',
    textDecoration: 'none'
  },
  button: {
    fontFamily: 'inherit'
  },
  input: {
    fontFamily: 'inherit'
  }
};

export default {
  getResponsiveStyles,
  createResponsiveComponent,
  spacingScale,
  fontSizeScale,
  borderRadiusScale,
  createSpacing,
  createResponsiveLayout,
  createCardStyles,
  createButtonStyles,
  createInputStyles,
  globalStyles
};
