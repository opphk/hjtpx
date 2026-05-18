(function() {
  'use strict';

  const MOBILE_CONFIG = {
    touchFeedbackDuration: 150,
    longPressDelay: 500,
    doubleTapDelay: 300,
    swipeThreshold: 50,
    pinchZoomMin: 1,
    pinchZoomMax: 3,
    imageCompressionQuality: 0.8,
    lazyLoadThreshold: 200
  };

  class MobileOptimizer {
    constructor() {
      this.touchStartTime = 0;
      this.touchStartPos = { x: 0, y: 0 };
      this.touchStartDistance = 0;
      this.lastTapTime = 0;
      this.lastTapPos = { x: 0, y: 0 };
      this.longPressTimer = null;
      this.activeTouches = [];
      this.isScrolling = false;
      this.pinchStartDistance = 0;
      this.currentZoom = 1;
    }

    init() {
      console.log('[Mobile] 初始化移动端优化模块');
      this.setupTouchHandlers();
      this.setupGestureHandlers();
      this.setupPullToRefresh();
      this.setupImageOptimization();
      this.setupLazyLoading();
      this.injectMobileStyles();
      this.setupViewportFix();
      this.setupFastClick();
    }

    setupViewportFix() {
      const viewport = document.querySelector('meta[name="viewport"]');
      if (viewport) {
        let content = viewport.getAttribute('content');
        if (!content.includes('user-scalable=no')) {
          viewport.setAttribute('content', content + ', user-scalable=no');
        }
      }
    }

    setupFastClick() {
      if ('ontouchstart' in document.documentElement) {
        document.body.style.cssText += '; -webkit-tap-highlight-color: transparent; -webkit-touch-callout: none;';
        document.querySelectorAll('a, button, [role="button"]').forEach(el => {
          el.style.cssText += '; -webkit-tap-highlight-color: rgba(0,0,0,0.1);';
        });
      }
    }

    setupTouchHandlers() {
      const interactiveElements = document.querySelectorAll('.captcha-slider-button, .captcha-click-marker, .btn, .nav-link, [role="button"]');

      interactiveElements.forEach(el => {
        el.addEventListener('touchstart', (e) => this.handleTouchStart(e), { passive: false });
        el.addEventListener('touchmove', (e) => this.handleTouchMove(e), { passive: false });
        el.addEventListener('touchend', (e) => this.handleTouchEnd(e), { passive: false });
        el.addEventListener('touchcancel', (e) => this.handleTouchCancel(e), { passive: false });
      });
    }

    handleTouchStart(e) {
      const touch = e.touches[0];
      this.touchStartTime = Date.now();
      this.touchStartPos = { x: touch.clientX, y: touch.clientY };

      const target = e.currentTarget;
      target.classList.add('touch-active');

      this.longPressTimer = setTimeout(() => {
        this.triggerHapticFeedback();
        target.classList.add('long-press-active');
        target.dispatchEvent(new CustomEvent('longpress', { bubbles: true }));
      }, MOBILE_CONFIG.longPressDelay);

      if (e.touches.length === 2) {
        this.pinchStartDistance = this.getTouchDistance(e.touches);
        e.preventDefault();
      }

      this.activeTouches = Array.from(e.touches);
    }

    handleTouchMove(e) {
      if (this.longPressTimer) {
        const touch = e.touches[0];
        const deltaX = Math.abs(touch.clientX - this.touchStartPos.x);
        const deltaY = Math.abs(touch.clientY - this.touchStartPos.y);

        if (deltaX > 10 || deltaY > 10) {
          clearTimeout(this.longPressTimer);
          this.longPressTimer = null;
          e.currentTarget.classList.remove('long-press-active');
        }
      }

      if (e.touches.length === 2) {
        const currentDistance = this.getTouchDistance(e.touches);
        const scale = currentDistance / this.pinchStartDistance;
        this.currentZoom = Math.max(MOBILE_CONFIG.pinchZoomMin, Math.min(MOBILE_CONFIG.pinchZoomMax, scale));

        const element = e.currentTarget;
        if (element.classList.contains('captcha-click-image') || element.classList.contains('captcha-canvas')) {
          element.style.transform = `scale(${this.currentZoom})`;
        }
      }
    }

    handleTouchEnd(e) {
      const touchDuration = Date.now() - this.touchStartTime;
      const touch = e.changedTouches[0];
      const deltaX = touch.clientX - this.touchStartPos.x;
      const deltaY = touch.clientY - this.touchStartPos.y;

      clearTimeout(this.longPressTimer);
      this.longPressTimer = null;

      const target = e.currentTarget;
      target.classList.remove('touch-active', 'long-press-active');

      if (touchDuration < 300 && Math.abs(deltaX) < 10 && Math.abs(deltaY) < 10) {
        this.handleTap(e, target);
      }

      if (this.currentZoom !== 1) {
        target.style.transform = 'scale(1)';
        this.currentZoom = 1;
      }
    }

    handleTouchCancel(e) {
      clearTimeout(this.longPressTimer);
      this.longPressTimer = null;
      e.currentTarget.classList.remove('touch-active', 'long-press-active');
    }

    handleTap(e, target) {
      const now = Date.now();

      if (now - this.lastTapTime < MOBILE_CONFIG.doubleTapDelay) {
        const deltaX = Math.abs(e.changedTouches[0].clientX - this.lastTapPos.x);
        const deltaY = Math.abs(e.changedTouches[0].clientY - this.lastTapPos.y);

        if (deltaX < 50 && deltaY < 50) {
          target.dispatchEvent(new CustomEvent('doubletap', {
            bubbles: true,
            detail: { x: e.changedTouches[0].clientX, y: e.changedTouches[0].clientY }
          }));
          this.lastTapTime = 0;
          return;
        }
      }

      this.lastTapTime = now;
      this.lastTapPos = { x: e.changedTouches[0].clientX, y: e.changedTouches[0].clientY };

      this.triggerHapticFeedback();

      target.dispatchEvent(new CustomEvent('tap', {
        bubbles: true,
        detail: { x: e.changedTouches[0].clientX, y: e.changedTouches[0].clientY }
      }));
    }

    getTouchDistance(touches) {
      const dx = touches[0].clientX - touches[1].clientX;
      const dy = touches[0].clientY - touches[1].clientY;
      return Math.sqrt(dx * dx + dy * dy);
    }

    triggerHapticFeedback() {
      if ('vibrate' in navigator) {
        navigator.vibrate(50);
      }
    }

    setupGestureHandlers() {
      let swipeStartX = 0;
      let swipeStartY = 0;
      let swipeStartTime = 0;

      document.addEventListener('touchstart', (e) => {
        if (e.touches.length === 1) {
          swipeStartX = e.touches[0].clientX;
          swipeStartY = e.touches[0].clientY;
          swipeStartTime = Date.now();
        }
      }, { passive: true });

      document.addEventListener('touchend', (e) => {
        if (e.changedTouches.length === 1) {
          const swipeEndX = e.changedTouches[0].clientX;
          const swipeEndY = e.changedTouches[0].clientY;
          const swipeEndTime = Date.now();

          const deltaX = swipeEndX - swipeStartX;
          const deltaY = swipeEndY - swipeStartY;
          const deltaTime = swipeEndTime - swipeStartTime;

          if (deltaTime < 500 && Math.abs(deltaX) > MOBILE_CONFIG.swipeThreshold) {
            const direction = deltaX > 0 ? 'right' : 'left';
            document.dispatchEvent(new CustomEvent('swipe', {
              bubbles: true,
              detail: {
                direction: direction,
                distance: Math.abs(deltaX),
                duration: deltaTime
              }
            }));
          }

          if (deltaTime < 500 && Math.abs(deltaY) > MOBILE_CONFIG.swipeThreshold && Math.abs(deltaY) > Math.abs(deltaX)) {
            const direction = deltaY > 0 ? 'down' : 'up';
            document.dispatchEvent(new CustomEvent('swipe', {
              bubbles: true,
              detail: {
                direction: direction,
                distance: Math.abs(deltaY),
                duration: deltaTime
              }
            }));
          }
        }
      }, { passive: true });
    }

    setupPullToRefresh() {
      let startY = 0;
      let currentY = 0;
      let isPulling = false;
      const pullThreshold = 80;

      document.body.style.overscrollBehavior = 'none';

      document.addEventListener('touchstart', (e) => {
        if (window.scrollY === 0 && e.touches[0].clientY > 0) {
          startY = e.touches[0].clientY;
          isPulling = true;
        }
      }, { passive: true });

      document.addEventListener('touchmove', (e) => {
        if (isPulling) {
          currentY = e.touches[0].clientY;
          const pullDistance = currentY - startY;

          if (pullDistance > 0 && pullDistance < pullThreshold * 2) {
            document.dispatchEvent(new CustomEvent('pullrefresh', {
              bubbles: true,
              detail: {
                distance: pullDistance,
                progress: Math.min(pullDistance / pullThreshold, 1)
              }
            }));
          }
        }
      }, { passive: true });

      document.addEventListener('touchend', (e) => {
        if (isPulling && (currentY - startY) > pullThreshold) {
          document.dispatchEvent(new CustomEvent('pullrefreshcomplete', { bubbles: true }));
        }
        isPulling = false;
        startY = 0;
        currentY = 0;
      }, { passive: true });
    }

    setupImageOptimization() {
      const images = document.querySelectorAll('img');

      images.forEach(img => {
        if (!img.hasAttribute('data-optimized')) {
          img.setAttribute('data-optimized', 'true');

          if (img.complete) {
            this.optimizeImage(img);
          } else {
            img.addEventListener('load', () => this.optimizeImage(img));
          }
        }
      });
    }

    optimizeImage(img) {
      const src = img.src;

      if (src.startsWith('data:') || src.startsWith('blob:')) {
        return;
      }

      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');

      const maxWidth = img.naturalWidth || img.width;
      const maxHeight = img.naturalHeight || img.height;

      let width = maxWidth;
      let height = maxHeight;

      const maxDimension = 800;
      if (width > maxDimension || height > maxDimension) {
        if (width > height) {
          height = (height / width) * maxDimension;
          width = maxDimension;
        } else {
          width = (width / height) * maxDimension;
          height = maxDimension;
        }
      }

      canvas.width = width;
      canvas.height = height;
      ctx.drawImage(img, 0, 0, width, height);

      const optimizedSrc = canvas.toDataURL('image/jpeg', MOBILE_CONFIG.imageCompressionQuality);

      img.src = optimizedSrc;
      img.setAttribute('data-optimized', 'true');
    }

    setupLazyLoading() {
      if ('IntersectionObserver' in window) {
        const lazyImages = document.querySelectorAll('img[data-src]');

        const imageObserver = new IntersectionObserver((entries) => {
          entries.forEach(entry => {
            if (entry.isIntersecting) {
              const img = entry.target;
              const src = img.getAttribute('data-src');

              if (src) {
                img.src = src;
                img.removeAttribute('data-src');
                img.classList.add('lazy-loaded');
                imageObserver.unobserve(img);
              }
            }
          });
        }, {
          rootMargin: `${MOBILE_CONFIG.lazyLoadThreshold}px`,
          threshold: 0.01
        });

        lazyImages.forEach(img => imageObserver.observe(img));
      }
    }

    injectMobileStyles() {
      if (document.getElementById('mobile-optimization-styles')) {
        return;
      }

      const style = document.createElement('style');
      style.id = 'mobile-optimization-styles';
      style.textContent = `
        .touch-active {
          transform: scale(0.95);
          opacity: 0.8;
        }

        .long-press-active {
          background-color: rgba(102, 126, 234, 0.2) !important;
        }

        @media (hover: none) and (pointer: coarse) {
          .captcha-slider-button,
          .captcha-click-marker,
          .btn,
          .nav-link {
            min-height: 44px;
            min-width: 44px;
          }

          .captcha-refresh {
            min-width: 44px;
            min-height: 44px;
          }
        }

        @media (max-width: 576px) {
          .captcha-container {
            margin: 0 !important;
            border-radius: 0 !important;
          }

          .page-hero {
            padding: 6rem 0 3rem !important;
          }

          .demo-card {
            margin-bottom: 1rem;
          }

          .demo-card-body {
            padding: 1rem !important;
          }

          .captcha-slider-container {
            height: 48px !important;
            border-radius: 24px !important;
          }

          .captcha-slider-button {
            width: 44px !important;
            height: 44px !important;
          }

          .captcha-slider-track {
            height: 44px !important;
          }

          .captcha-slider-text {
            line-height: 48px !important;
            font-size: 14px !important;
          }

          .captcha-click-marker {
            width: 36px !important;
            height: 36px !important;
            font-size: 16px !important;
          }

          .nav-link {
            padding: 0.6rem 0.8rem !important;
            font-size: 13px !important;
          }
        }

        @media (max-width: 400px) {
          .captcha-image-wrapper {
            margin-bottom: 8px !important;
          }

          .captcha-actions {
            flex-direction: column !important;
          }

          .captcha-btn {
            width: 100% !important;
            justify-content: center !important;
            margin-bottom: 0.5rem !important;
          }
        }

        .pull-refresh-indicator {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          height: 0;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          z-index: 10000;
          display: flex;
          align-items: center;
          justify-content: center;
          overflow: hidden;
          transition: height 0.3s ease;
        }

        .pull-refresh-indicator.active {
          height: 60px;
        }

        .pull-refresh-spinner {
          width: 24px;
          height: 24px;
          border: 3px solid rgba(255,255,255,0.3);
          border-top-color: white;
          border-radius: 50%;
          animation: mobile-spin 1s linear infinite;
        }

        @keyframes mobile-spin {
          to { transform: rotate(360deg); }
        }

        img[lazy] {
          opacity: 0;
          transition: opacity 0.3s ease;
        }

        img.lazy-loaded {
          opacity: 1;
        }

        @media (prefers-reduced-motion: reduce) {
          .touch-active,
          .long-press-active,
          .captcha-slider-button,
          .captcha-click-marker,
          .btn,
          .nav-link {
            transition: none !important;
          }

          .pull-refresh-indicator,
          img {
            transition: none !important;
          }
        }

        input, textarea, select {
          font-size: 16px !important;
        }

        @media (hover: none) and (pointer: coarse) {
          input:focus, textarea:focus, select:focus {
            font-size: 16px !important;
          }
        }

        .mobile-safe-area {
          padding-top: env(safe-area-inset-top);
          padding-bottom: env(safe-area-inset-bottom);
          padding-left: env(safe-area-inset-left);
          padding-right: env(safe-area-inset-right);
        }

        .touch-target-44 {
          min-width: 44px;
          min-height: 44px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
        }
      `;

      document.head.appendChild(style);
    }
  }

  const mobileOptimizer = new MobileOptimizer();

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => mobileOptimizer.init());
  } else {
    mobileOptimizer.init();
  }

  window.MobileOptimizer = mobileOptimizer;
})();
