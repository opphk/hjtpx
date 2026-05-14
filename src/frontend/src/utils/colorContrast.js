export function hexToRgb(hex) {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result ? {
    r: parseInt(result[1], 16),
    g: parseInt(result[2], 16),
    b: parseInt(result[3], 16)
  } : null;
}

export function rgbToHex(r, g, b) {
  return "#" + [r, g, b].map(x => {
    const hex = x.toString(16);
    return hex.length === 1 ? '0' + hex : hex;
  }).join('');
}

export function getLuminance(r, g, b) {
  const [rs, gs, bs] = [r, g, b].map(c => {
    c = c / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
}

export function getContrastRatio(color1, color2) {
  const rgb1 = typeof color1 === 'string' ? hexToRgb(color1) : color1;
  const rgb2 = typeof color2 === 'string' ? hexToRgb(color2) : color2;
  
  if (!rgb1 || !rgb2) return 0;
  
  const l1 = getLuminance(rgb1.r, rgb1.g, rgb1.b);
  const l2 = getLuminance(rgb2.r, rgb2.g, rgb2.b);
  
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  
  return (lighter + 0.05) / (darker + 0.05);
}

export function meetsWCAGLevel(color1, color2, level = 'AA') {
  const ratio = getContrastRatio(color1, color2);
  
  if (level === 'AAA') {
    return ratio >= 7;
  }
  
  if (level === 'AA') {
    return ratio >= 4.5;
  }
  
  if (level === 'AA-large') {
    return ratio >= 3;
  }
  
  return false;
}

export function suggestAccessibleColor(badColor, goodColor, level = 'AA') {
  const rgb = hexToRgb(badColor);
  if (!rgb) return badColor;
  
  const targetRatio = level === 'AAA' ? 7 : level === 'AA-large' ? 3 : 4.5;
  const currentRatio = getContrastRatio(badColor, goodColor);
  
  if (currentRatio >= targetRatio) {
    return badColor;
  }
  
  let { r, g, b } = rgb;
  const goodRgb = hexToRgb(goodColor);
  if (!goodRgb) return badColor;
  
  const goodLuminance = getLuminance(goodRgb.r, goodRgb.g, goodRgb.b);
  const targetLuminance = goodLuminance > 0.5 ? 0 : 1;
  
  for (let i = 0; i < 10; i++) {
    const luminance = getLuminance(r, g, b);
    
    if (Math.abs(luminance - targetLuminance) < 0.1) {
      break;
    }
    
    if (luminance < targetLuminance) {
      r = Math.min(255, r + 25);
      g = Math.min(255, g + 25);
      b = Math.min(255, b + 25);
    } else {
      r = Math.max(0, r - 25);
      g = Math.max(0, g - 25);
      b = Math.max(0, b - 25);
    }
  }
  
  return rgbToHex(Math.round(r), Math.round(g), Math.round(b));
}

export const colorTokens = {
  primary: {
    default: '#0066cc',
    hover: '#0052a3',
    active: '#004080',
    disabled: '#99c2e8'
  },
  secondary: {
    default: '#6c757d',
    hover: '#5a6268',
    active: '#4a5258',
    disabled: '#c8cbcf'
  },
  success: {
    default: '#28a745',
    hover: '#218838',
    active: '#1e7e34',
    disabled: '#a3d7b0'
  },
  danger: {
    default: '#dc3545',
    hover: '#c82333',
    active: '#bd2130',
    disabled: '#f1a4ad'
  },
  warning: {
    default: '#ffc107',
    hover: '#e0a800',
    active: '#d39e00',
    text: '#212529',
    disabled: '#ffeaa7'
  },
  info: {
    default: '#17a2b8',
    hover: '#138496',
    active: '#117a8b',
    disabled: '#a8dbe5'
  },
  light: {
    default: '#f8f9fa',
    text: '#212529',
    border: '#dee2e6'
  },
  dark: {
    default: '#343a40',
    text: '#ffffff',
    border: '#495057'
  }
};

export function validateColorToken(token, backgroundColor = '#ffffff') {
  const results = [];
  
  Object.entries(token).forEach(([variant, color]) => {
    if (variant === 'text') {
      const ratio = getContrastRatio(color, backgroundColor);
      results.push({
        token: `${token}.${variant}`,
        color,
        ratio,
        passesAA: ratio >= 4.5,
        passesAAA: ratio >= 7,
        passesAALarge: ratio >= 3
      });
    } else if (!['disabled'].includes(variant)) {
      const ratio = getContrastRatio(color, backgroundColor);
      results.push({
        token: `${token}.${variant}`,
        color,
        ratio,
        passesAA: ratio >= 4.5,
        passesAAA: ratio >= 7,
        passesAALarge: ratio >= 3
      });
    }
  });
  
  return results;
}

export function generateAccessiblePalette(baseColor, options = {}) {
  const {
    lightRatio = 1.5,
    darkRatio = 7,
    textColor = '#ffffff'
  } = options;
  
  const baseRgb = hexToRgb(baseColor);
  if (!baseRgb) return {};
  
  const baseLuminance = getLuminance(baseRgb.r, baseRgb.g, baseRgb.b);
  
  let lighter = { ...baseRgb };
  let darker = { ...baseRgb };
  
  for (let i = 0; i < 20; i++) {
    const lighterLuminance = getLuminance(lighter.r, lighter.g, lighter.b);
    const darkerLuminance = getLuminance(darker.r, darker.g, darker.b);
    
    if (lighterLuminance < lightRatio / (lightRatio + 1)) {
      lighter = {
        r: Math.min(255, lighter.r + 15),
        g: Math.min(255, lighter.g + 15),
        b: Math.min(255, lighter.b + 15)
      };
    }
    
    if (darkerLuminance > 1 / darkRatio) {
      darker = {
        r: Math.max(0, darker.r - 15),
        g: Math.max(0, darker.g - 15),
        b: Math.max(0, darker.b - 15)
      };
    }
    
    if (
      lighterLuminance >= lightRatio / (lightRatio + 1) &&
      darkerLuminance <= 1 / darkRatio
    ) {
      break;
    }
  }
  
  return {
    lighter: rgbToHex(Math.round(lighter.r), Math.round(lighter.g), Math.round(lighter.b)),
    base: baseColor,
    darker: rgbToHex(Math.round(darker.r), Math.round(darker.g), Math.round(darker.b)),
    text: textColor,
    textOnLighter: '#212529',
    textOnDarker: '#ffffff'
  };
}
