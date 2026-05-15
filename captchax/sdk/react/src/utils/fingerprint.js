const FINGERPRINT_VERSION = '1.0.0';

const hashString = async (str) => {
  const encoder = new TextEncoder();
  const data = encoder.encode(str);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
};

const getCanvasFingerprint = () => {
  try {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    canvas.width = 200;
    canvas.height = 50;

    ctx.textBaseline = 'top';
    ctx.font = '14px Arial';
    ctx.fillStyle = '#f60';
    ctx.fillRect(125, 1, 62, 20);
    ctx.fillStyle = '#069';
    ctx.fillText('CaptchaX', 2, 15);
    ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
    ctx.fillText('CaptchaX', 4, 17);

    return canvas.toDataURL();
  } catch (e) {
    return null;
  }
};

const getWebGLFingerprint = () => {
  try {
    const canvas = document.createElement('canvas');
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
    if (!gl) return null;

    const webgl = gl.getExtension('WEBGL_debug_renderer_info');
    const vendor = webgl ? gl.getParameter(webgl.UNMASKED_VENDOR_WEBGL) : '';
    const renderer = webgl ? gl.getParameter(webgl.UNMASKED_RENDERER_WEBGL) : '';

    return `${vendor}~${renderer}`;
  } catch (e) {
    return null;
  }
};

const getScreenInfo = () => {
  return {
    width: window.screen.width,
    height: window.screen.height,
    colorDepth: window.screen.colorDepth,
    pixelRatio: window.devicePixelRatio
  };
};

const getTimezoneInfo = () => {
  const offset = new Date().getTimezoneOffset();
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  return { offset, timezone };
};

const getLanguageInfo = () => {
  return {
    language: navigator.language,
    languages: navigator.languages?.join(','),
    doNotTrack: navigator.doNotTrack
  };
};

const getTouchSupport = () => {
  return {
    maxTouchPoints: navigator.maxTouchPoints,
    touchSupport: 'ontouchstart' in window,
    pointerSupport: window.matchMedia('(pointer: fine)').matches
  };
};

export const generateFingerprint = async () => {
  const components = {
    canvas: getCanvasFingerprint(),
    webgl: getWebGLFingerprint(),
    screen: getScreenInfo(),
    timezone: getTimezoneInfo(),
    language: getLanguageInfo(),
    touch: getTouchSupport(),
    platform: navigator.platform,
    hardwareConcurrency: navigator.hardwareConcurrency,
    version: FINGERPRINT_VERSION
  };

  const rawString = JSON.stringify(components);
  const hash = await hashString(rawString);

  return {
    hash,
    components
  };
};

export const generateSessionId = () => {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
};

export const generateToken = () => {
  return `${generateSessionId()}-${crypto.randomUUID()}`;
};

export default {
  generateFingerprint,
  generateSessionId,
  generateToken
};
