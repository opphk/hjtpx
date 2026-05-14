const entities = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#x27;',
  '/': '&#x2F;',
  '`': '&#x60;',
  '=': '&#x3D;'
};

const domAttributes = [
  'onerror', 'onload', 'onclick', 'onmouseover', 'onfocus',
  'onblur', 'onchange', 'onsubmit', 'onkeydown', 'onkeyup',
  'onkeypress', 'ondblclick', 'oncontextmenu', 'oncopy', 'oncut',
  'onpaste', 'onerror', 'onabort', 'onresize', 'onscroll'
];

function escapeHtml(str) {
  if (typeof str !== 'string') {
    return str;
  }
  
  return str.replace(/[&<>"'`=/]/g, (char) => entities[char] || char);
}

function escapeHtmlDeep(obj) {
  if (typeof obj === 'string') {
    return escapeHtml(obj);
  }
  
  if (Array.isArray(obj)) {
    return obj.map(item => escapeHtmlDeep(item));
  }
  
  if (obj && typeof obj === 'object') {
    const result = {};
    for (const key in obj) {
      result[key] = escapeHtmlDeep(obj[key]);
    }
    return result;
  }
  
  return obj;
}

function removeXssPatterns(str) {
  if (typeof str !== 'string') {
    return str;
  }
  
  let sanitized = str;
  
  domAttributes.forEach(attr => {
    const regex = new RegExp(`\\s*${attr}\\s*=\\s*`, 'gi');
    sanitized = sanitized.replace(regex, ' ');
  });
  
  sanitized = sanitized.replace(/javascript\s*:/gi, '');
  sanitized = sanitized.replace(/data\s*:\s*text\/html/gi, '');
  sanitized = sanitized.replace(/vbscript\s*:/gi, '');
  
  return sanitized;
}

function sanitizeObject(obj, options = {}) {
  const {
    allowSafeTags = false,
    maxLength = null,
    stripAllTags = false
  } = options;
  
  if (typeof obj === 'string') {
    let sanitized = stripAllTags 
      ? obj.replace(/<[^>]*>/g, '') 
      : obj;
    
    sanitized = removeXssPatterns(sanitized);
    sanitized = escapeHtmlDeep(sanitized);
    
    if (maxLength && sanitized.length > maxLength) {
      sanitized = sanitized.substring(0, maxLength);
    }
    
    return sanitized;
  }
  
  if (Array.isArray(obj)) {
    return obj.map(item => sanitizeObject(item, options));
  }
  
  if (obj && typeof obj === 'object') {
    const result = {};
    for (const key in obj) {
      result[key] = sanitizeObject(obj[key], options);
    }
    return result;
  }
  
  return obj;
}

function validateAndSanitize(value, rules = {}) {
  const {
    type = 'string',
    minLength = null,
    maxLength = null,
    pattern = null,
    allowedChars = null,
    stripAllTags = false
  } = rules;
  
  let sanitized = sanitizeObject(value, { stripAllTags });
  
  if (typeof sanitized !== 'string') {
    sanitized = String(sanitized);
  }
  
  if (minLength !== null && sanitized.length < minLength) {
    return { valid: false, value: sanitized, error: `Minimum length is ${minLength}` };
  }
  
  if (maxLength !== null && sanitized.length > maxLength) {
    sanitized = sanitized.substring(0, maxLength);
  }
  
  if (allowedChars !== null) {
    const allowedRegex = new RegExp(`[^${allowedChars}]`, 'g');
    if (allowedRegex.test(sanitized)) {
      return { valid: false, value: sanitized, error: 'Contains invalid characters' };
    }
  }
  
  if (pattern !== null) {
    const regex = new RegExp(pattern);
    if (!regex.test(sanitized)) {
      return { valid: false, value: sanitized, error: 'Does not match required pattern' };
    }
  }
  
  return { valid: true, value: sanitized };
}

function createSanitizer(options = {}) {
  return {
    sanitize: (data) => sanitizeObject(data, options),
    validate: (value, rules) => validateAndSanitize(value, { ...options, ...rules }),
    escapeHtml,
    escapeHtmlDeep,
    removeXssPatterns
  };
}

module.exports = {
  escapeHtml,
  escapeHtmlDeep,
  removeXssPatterns,
  sanitizeObject,
  validateAndSanitize,
  createSanitizer,
  entities,
  domAttributes
};
