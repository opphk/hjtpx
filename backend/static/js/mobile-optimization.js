(function() {
  'use strict';

  const MOBILE_CONFIG = {
    touchFeedbackDuration: 120,
    longPressDelay: 500,
    doubleTapDelay: 250,
    swipeThreshold: 40,
    pinchZoomMin: 1,
    pinchZoomMax: 3,
    imageCompressionQuality: 0.85,
    lazyLoadThreshold: 300,
    touchTargetMinSize: 44
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
      this.isTouchDevice = 'ontouchstart' in document.documentElement;
    }

    init() {
      console.log('[Mobile] 初始化移动端优化模块');
      this.setupViewportFix();
      this.setupFastClick();
      this.setupTouchHandlers();
      this.setupGestureHandlers();
      this.setupPullToRefresh();
      this.setupImageOptimization();
      this.setupLazyLoading();
      this.setupOrientationHandling();
      this.setupKeyboardAvoidance();
      this.injectMobileStyles();
    }

    setupViewportFix() {
      const viewport = document.querySelector('meta[name="viewport"]');
      if (viewport) {
        let content = viewport.getAttribute('content');
        const needsUpdate = !content.includes('viewport-fit=cover') || 
                           !content.includes('maximum-scale=1.0');
        
        if (needsUpdate) {
          if (!content.includes('viewport-fit=cover')) {
            content += ', viewport-fit=cover';
          }
          if (!content.includes('maximum-scale')) {
            content += ', maximum-scale=1.0';
          }
          viewport.setAttribute('content', content);
        }
      }
    }

    setupFastClick() {
      if (this.isTouchDevice) {
        document.body.style.cssText += `
          ; -webkit-tap-highlight-color: transparent;
          -webkit-touch-callout: none;
          touch-action: manipulation;
        `;
        
        document.querySelectorAll('a, button, [role="button"], .captcha-slider-button, .captcha-click-marker').forEach(el => {
          el.style.cssText += '; -webkit-tap-highlight-color: rgba(201, 169, 110, 0.2);';
        });
      }
    }

    setupTouchHandlers() {
      const interactiveElements = document.querySelectorAll(
        '.captcha-slider-button, .captcha-click-marker, .btn, .nav-link, [role="button"], .captcha-tab, .captcha-refresh'
      );

      interactiveElements.forEach(el => {
        el.addEventListener('touchstart', (e) => this.handleTouchStart(e), { passive: true });
        el.addEventListener('touchmove', (e) => this.handleTouchMove(e), { passive: true });
        el.addEventListener('touchend', (e) => this.handleTouchEnd(e), { passive: true });
        el.addEventListener('touchcancel', (e) => this.handleTouchCancel(e), { passive: true });
      });
    }

    handleTouchStart(e) {
      const touch = e.touches[0];
      this.touchStartTime = Date.now();
      this.touchStartPos = { x: touch.clientX, y: touch.clientY };

      const target = e.currentTarget;
      target.classList.add('touch-active');

      this.longPressTimer = setTimeout(() => {
        this.triggerHapticFeedback('medium');
        target.classList.add('long-press-active');
        target.dispatchEvent(new CustomEvent('longpress', { bubbles: true }));
      }, MOBILE_CONFIG.longPressDelay);

      if (e.touches.length === 2) {
        this.pinchStartDistance = this.getTouchDistance(e.touches);
      }

      this.activeTouches = Array.from(e.touches);
    }

    handleTouchMove(e) {
      if (this.longPressTimer) {
        const touch = e.touches[0];
        const deltaX = Math.abs(touch.clientX - this.touchStartPos.x);
        const deltaY = Math.abs(touch.clientY - this.touchStartPos.y);

        if (deltaX > 15 || deltaY > 15) {
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

      if (touchDuration < 300 && Math.abs(deltaX) < 15 && Math.abs(deltaY) < 15) {
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

      this.triggerHapticFeedback('light');

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

    triggerHapticFeedback(intensity = 'light') {
      if ('vibrate' in navigator) {
        const patterns = {
          light: 10,
          medium: 25,
          heavy: 50
        };
        navigator.vibrate(patterns[intensity] || patterns.light);
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
        if (window.scrollY === 0 && e.touches[0].clientY < 100) {
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
      const images = document.querySelectorAll('img[data-src], img:not([data-optimized])');

      images.forEach(img => {
        if (!img.hasAttribute('data-optimized')) {
          img.setAttribute('data-optimized', 'true');

          if (img.complete) {
            this.optimizeImage(img);
          } else {
            img.addEventListener('load', () => this.optimizeImage(img), { once: true });
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

      const maxDimension = 600;
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
        const lazyImages = document.querySelectorAll('img[data-src], img.lazy');

        const imageObserver = new IntersectionObserver((entries) => {
          entries.forEach(entry => {
            if (entry.isIntersecting) {
              const img = entry.target;
              const src = img.getAttribute('data-src') || img.getAttribute('data-lazy-src');

              if (src) {
                img.src = src;
                img.removeAttribute('data-src');
                img.removeAttribute('data-lazy-src');
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

    setupOrientationHandling() {
      window.addEventListener('orientationchange', () => {
        setTimeout(() => {
          window.dispatchEvent(new Event('resize'));
        }, 100);
      });

      window.addEventListener('resize', () => {
        this.adjustLayoutForOrientation();
      });

      this.adjustLayoutForOrientation();
    }

    adjustLayoutForOrientation() {
      const isLandscape = window.innerWidth > window.innerHeight;
      const captchaContainer = document.querySelector('.captcha-container');
      
      if (captchaContainer) {
        captchaContainer.classList.toggle('landscape', isLandscape);
      }
    }

    setupKeyboardAvoidance() {
      const focusedElement = document.activeElement;
      if (focusedElement && ['INPUT', 'TEXTAREA'].includes(focusedElement.tagName)) {
        focusedElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }

      window.addEventListener('resize', () => {
        const activeElement = document.activeElement;
        if (activeElement && ['INPUT', 'TEXTAREA'].includes(activeElement.tagName)) {
          setTimeout(() => {
            activeElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
          }, 100);
        }
      });
    }

    injectMobileStyles() {
      if (document.getElementById('mobile-optimization-styles')) {
        return;
      }

      const style = document.createElement('style');
      style.id = 'mobile-optimization-styles';
      style.textContent = `
        .touch-active {
          transform: scale(0.96);
          opacity: 0.9;
          transition: transform 0.15s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.15s ease;
        }

        .long-press-active {
          background-color: rgba(201, 169, 110, 0.25) !important;
          box-shadow: inset 0 0 20px rgba(201, 169, 110, 0.1);
        }

        @media (hover: none) and (pointer: coarse) {
          .captcha-slider-button,
          .captcha-click-marker,
          .btn,
          .nav-link,
          .captcha-tab,
          .captcha-refresh {
            min-height: ${MOBILE_CONFIG.touchTargetMinSize}px;
            min-width: ${MOBILE_CONFIG.touchTargetMinSize}px;
          }

          .captcha-refresh {
            min-width: ${MOBILE_CONFIG.touchTargetMinSize + 4}px;
            min-height: ${MOBILE_CONFIG.touchTargetMinSize + 4}px;
          }
        }

        @media (max-width: 576px) {
          .captcha-container {
            margin: 0 !important;
            border-radius: 0 !important;
            border-left: none !important;
            border-right: none !important;
          }

          .captcha-container.landscape {
            max-width: 480px;
            margin: 0 auto !important;
            border-radius: 12px !important;
            border: 1px solid rgba(201, 169, 110, 0.2) !important;
          }

          .page-hero {
            padding: 5rem 0 2.5rem !important;
          }

          .demo-card {
            margin-bottom: 1rem;
            border-radius: 10px;
          }

          .demo-card-body {
            padding: 1rem !important;
          }

          .captcha-slider-container {
            height: 50px !important;
            border-radius: 25px !important;
          }

          .captcha-slider-button {
            width: 46px !important;
            height: 46px !important;
          }

          .captcha-slider-track {
            height: 46px !important;
          }

          .captcha-slider-text {
            line-height: 50px !important;
            font-size: 14px !important;
          }

          .captcha-click-marker {
            width: 36px !important;
            height: 36px !important;
            font-size: 16px !important;
          }

          .nav-link {
            padding: 0.75rem 1rem !important;
            font-size: 14px !important;
          }

          .captcha-tabs {
            gap: 4px;
          }

          .captcha-tab {
            padding: 9px 14px !important;
            font-size: 13px !important;
          }
        }

        @media (max-width: 400px) {
          .captcha-image-wrapper {
            margin-bottom: 10px !important;
          }

          .captcha-actions {
            flex-direction: column !important;
            gap: 8px !important;
          }

          .captcha-btn {
            width: 100% !important;
            justify-content: center !important;
            margin-bottom: 0 !important;
          }

          .captcha-header {
            padding: 12px !important;
          }

          .captcha-header h3 {
            font-size: 16px !important;
          }
        }

        .pull-refresh-indicator {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          height: 0;
          background: linear-gradient(135deg, #c9a96e 0%, #d4b87a 100%);
          z-index: 10000;
          display: flex;
          align-items: center;
          justify-content: center;
          overflow: hidden;
          transition: height 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .pull-refresh-indicator.active {
          height: 65px;
        }

        .pull-refresh-spinner {
          width: 28px;
          height: 28px;
          border: 3px solid rgba(255,255,255,0.4);
          border-top-color: white;
          border-radius: 50%;
          animation: mobile-spin 0.8s linear infinite;
        }

        @keyframes mobile-spin {
          to { transform: rotate(360deg); }
        }

        img[lazy], img[data-src] {
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
          .nav-link,
          .captcha-tab {
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

        .mobile-safe-area-bottom {
          padding-bottom: env(safe-area-inset-bottom);
        }

        .touch-target-44 {
          min-width: 44px;
          min-height: 44px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
        }

        .touch-target-48 {
          min-width: 48px;
          min-height: 48px;
          display: inline-flex;
          align-items: center;
          justify-content: center;
        }

        .captcha-container.landscape {
          flex-direction: row;
          max-width: 600px;
        }

        .captcha-container.landscape .captcha-header {
          flex-shrink: 0;
          min-width: 120px;
        }

        .captcha-container.landscape .captcha-body {
          flex: 1;
        }

        .hide-in-portrait {
          display: none;
        }

        .hide-in-landscape {
          display: block;
        }

        @media (orientation: landscape) {
          .hide-in-portrait {
            display: block;
          }
          .hide-in-landscape {
            display: none;
          }
        }

        ::-webkit-scrollbar {
          width: 6px;
          height: 6px;
        }

        ::-webkit-scrollbar-track {
          background: #f1f1f1;
          border-radius: 3px;
        }

        ::-webkit-scrollbar-thumb {
          background: #c9a96e;
          border-radius: 3px;
        }

        ::-webkit-scrollbar-thumb:hover {
          background: #b8954f;
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