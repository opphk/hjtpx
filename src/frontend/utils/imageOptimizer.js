export function createResponsiveImage(src, options = {}) {
  const {
    widths = [320, 640, 960, 1280, 1920],
    format = 'webp',
    quality = 80
  } = options;

  const cdnBase = process.env.VITE_CDN_URL || '';

  if (!cdnBase) {
    return {
      src,
      srcSet: widths.map(w => `${src} ${w}w`).join(', '),
      sizes: '(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw'
    };
  }

  const pathParts = src.split('/');
  const filename = pathParts.pop();
  const basePath = pathParts.join('/');

  const srcSet = widths
    .map(w => `${cdnBase}${basePath}/${w}w-${filename} ${w}w`)
    .join(', ');

  return {
    src: `${cdnBase}${basePath}/${filename}`,
    srcSet,
    sizes: '(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw'
  };
}

export function optimizeImageUrl(src, options = {}) {
  const {
    width,
    height,
    format = 'webp',
    quality = 80,
    fit = 'cover'
  } = options;

  const cdnBase = process.env.VITE_CDN_URL || '';

  if (!cdnBase) {
    const params = new URLSearchParams();
    if (width) params.set('w', width);
    if (height) params.set('h', height);
    if (quality) params.set('q', quality);
    if (format) params.set('fm', format);

    const separator = src.includes('?') ? '&' : '?';
    return `${src}${separator}${params.toString()}`;
  }

  const transforms = [];
  if (width) transforms.push(`w_${width}`);
  if (height) transforms.push(`h_${height}`);
  if (quality) transforms.push(`q_${quality}`);
  if (format) transforms.push(`f_${format}`);
  if (fit) transforms.push(`fit_${fit}`);

  return `${cdnBase}/${src}${transforms.length > 0 ? '?' + transforms.join(',') : ''}`;
}

export function preloadImage(src, options = {}) {
  const { as = 'image', type, crossOrigin } = options;

  if (typeof document === 'undefined') return;

  let link = document.querySelector(`link[rel="preload"][href="${src}"]`);

  if (!link) {
    link = document.createElement('link');
    link.rel = 'preload';
    link.href = src;
    link.as = as;
    if (type) link.type = type;
    if (crossOrigin) link.crossOrigin = crossOrigin;
    document.head.appendChild(link);
  }

  return link;
}

export function preloadImages(sources) {
  return sources.map(src => preloadImage(src));
}

export function detectImageFormat(src) {
  const ext = src.split('.').pop().toLowerCase().split('?')[0];

  const formats = {
    webp: 'image/webp',
    avif: 'image/avif',
    png: 'image/png',
    jpg: 'image/jpeg',
    jpeg: 'image/jpeg',
    gif: 'image/gif',
    svg: 'image/svg+xml',
    ico: 'image/x-icon'
  };

  return formats[ext] || 'image/jpeg';
}

export class ImageCache {
  constructor(maxSize = 50) {
    this.cache = new Map();
    this.maxSize = maxSize;
  }

  get(key) {
    if (this.cache.has(key)) {
      const entry = this.cache.get(key);
      if (Date.now() - entry.timestamp < 3600000) {
        return entry.image;
      }
      this.cache.delete(key);
    }
    return null;
  }

  set(key, image) {
    if (this.cache.size >= this.maxSize) {
      const firstKey = this.cache.keys().next().value;
      this.cache.delete(firstKey);
    }

    this.cache.set(key, {
      image,
      timestamp: Date.now()
    });
  }

  has(key) {
    return this.cache.has(key) && Date.now() - this.cache.get(key).timestamp < 3600000;
  }

  clear() {
    this.cache.clear();
  }
}

export default {
  createResponsiveImage,
  optimizeImageUrl,
  preloadImage,
  preloadImages,
  detectImageFormat,
  ImageCache
};
