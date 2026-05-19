(function() {
  'use strict';

  const MOBILE_ENHANCED_CONFIG = {
    touchFeedbackDuration: 120,
    longPressDelay: 500,
    doubleTapDelay: 250,
    swipeThreshold: 40,
    pinchZoomMin: 1,
    pinchZoomMax: 3,
    keyboardDebounceDelay: 300,
    animationDuration: 200,
    touchRippleDuration: 400,
    gestureVelocityThreshold: 0.5,
    multiTouchTimeout: 500,
    inertiaDuration: 800,
    snapBackDuration: 300
  };

  class MobileEnhancedAdapter {
    constructor() {
      this.touchStartTime = 0;
      this.touchStartPos = { x: 0, y: 0 };
      this.lastTapTime = 0;
      this.lastTapPos = { x: 0, y: 0 };
      this.longPressTimer = null;
      this.activeTouches = new Map();
      this.pinchStartDistance = 0;
      this.currentZoom = 1;
      this.isTouchDevice = 'ontouchstart' in document.documentElement;
      this.keyboardVisible = false;
      this.keyboardHeight = 0;
      this.animationQueue = [];
      this.isAnimating = false;
      this.gestureHistory = [];
      this.maxGestureHistory = 50;
      this.touchTargets = new WeakMap();
      this.rippleElements = new Set();
    }

    init() {
      console.log('[MobileEnhanced] 初始化移动端适配增强模块');
      this.setupViewportFix();
      this.setupTouchGestureOptimization();
      this.setupGestureFeedback();
      this.setupKeyboardInputOptimization();
      this.setupMobileAnimations();
      this.setupInertiaScrolling();
      this.setupTouchRipple();
      this.setupSwipeNavigation();
      this.setupPinchZoom();
      this.setupGestureHistory();
      this.injectEnhancedStyles();
      this.setupOrientationHandling();
      this.setupResizeObserver();
      this.bindGlobalEvents();

      if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => this.onDOMReady());
      } else {
        this.onDOMReady();
      }
    }

    onDOMReady() {
      this.optimizeInteractiveElements();
      this.setupPullToRefresh();
      this.setupTouchTargets();
      console.log('[MobileEnhanced] 移动端适配增强模块初始化完成');
    }

    setupViewportFix() {
      const viewport = document.querySelector('meta[name="viewport"]');
      if (viewport) {
        let content = viewport.getAttribute('content');
        const updates = [];

        if (!content.includes('viewport-fit=cover')) {
          updates.push('viewport-fit=cover');
        }
        if (!content.includes('user-scalable=no')) {
          updates.push('maximum-scale=1.0');
        }
        if (!content.includes('user-scalable=no')) {
          updates.push('user-scalable=no');
        }

        if (updates.length > 0) {
          content += ', ' + updates.join(', ');
          viewport.setAttribute('content', content);
        }
      }
    }

    setupTouchGestureOptimization() {
      const interactiveElements = this.getInteractiveElements();

      interactiveElements.forEach(el => {
        if (el.hasAttribute('data-touch-optimized')) return;

        el.setAttribute('data-touch-optimized', 'true');

        el.addEventListener('touchstart', (e) => this.handleTouchStart(e), { passive: false });
        el.addEventListener('touchmove', (e) => this.handleTouchMove(e), { passive: true });
        el.addEventListener('touchend', (e) => this.handleTouchEnd(e), { passive: false });
        el.addEventListener('touchcancel', (e) => this.handleTouchCancel(e), { passive: true });
      });
    }

    getInteractiveElements() {
      return document.querySelectorAll(
        '.captcha-slider-button, .captcha-click-marker, .btn, .nav-link, [role="button"], ' +
        '.captcha-tab, .captcha-refresh, .captcha-interactive, button, a, input, select, textarea'
      );
    }

    handleTouchStart(e) {
      const touch = e.touches[0];
      const target = e.currentTarget;

      this.touchStartTime = Date.now();
      this.touchStartPos = { x: touch.clientX, y: touch.clientY };

      this.activeTouches.set(touch.identifier || 'default', {
        startX: touch.clientX,
        startY: touch.clientY,
        currentX: touch.clientX,
        currentY: touch.clientY,
        startTime: Date.now()
      });

      target.classList.add('touch-active');
      this.createTouchRipple(target, touch.clientX, touch.clientY);

      this.longPressTimer = setTimeout(() => {
        this.triggerHapticFeedback('medium');
        target.classList.add('long-press-active');
        this.dispatchCustomEvent(target, 'mobilepress', {
          x: touch.clientX,
          y: touch.clientY,
          duration: MOBILE_ENHANCED_CONFIG.longPressDelay
        });
      }, MOBILE_ENHANCED_CONFIG.longPressDelay);

      if (e.touches.length === 2) {
        this.handleMultiTouchStart(e);
      }
    }

    handleTouchMove(e) {
      const touch = e.touches[0];
      const target = e.currentTarget;

      if (this.activeTouches.has(touch.identifier || 'default')) {
        const touchData = this.activeTouches.get(touch.identifier || 'default');
        touchData.currentX = touch.clientX;
        touchData.currentY = touch.clientY;
      }

      if (this.longPressTimer) {
        const deltaX = Math.abs(touch.clientX - this.touchStartPos.x);
        const deltaY = Math.abs(touch.clientY - this.touchStartPos.y);

        if (deltaX > 10 || deltaY > 10) {
          clearTimeout(this.longPressTimer);
          this.longPressTimer = null;
          target.classList.remove('long-press-active');
        }
      }

      if (e.touches.length === 2) {
        this.handleMultiTouchMove(e);
      }
    }

    handleTouchEnd(e) {
      const touch = e.changedTouches[0];
      const target = e.currentTarget;
      const touchDuration = Date.now() - this.touchStartTime;
      const deltaX = touch.clientX - this.touchStartPos.x;
      const deltaY = touch.clientY - this.touchStartPos.y;

      clearTimeout(this.longPressTimer);
      this.longPressTimer = null;

      target.classList.remove('touch-active', 'long-press-active');

      if (touchDuration < 300 && Math.abs(deltaX) < 15 && Math.abs(deltaY) < 15) {
        this.handleTap(e, target, touch);
      }

      this.recordGesture({
        type: 'touchend',
        x: touch.clientX,
        y: touch.clientY,
        duration: touchDuration,
        deltaX: deltaX,
        deltaY: deltaY
      });

      this.activeTouches.delete(touch.identifier || 'default');

      if (this.currentZoom !== 1) {
        this.animateTransform(target, `scale(1)`, MOBILE_ENHANCED_CONFIG.animationDuration);
        this.currentZoom = 1;
      }
    }

    handleTouchCancel(e) {
      clearTimeout(this.longPressTimer);
      this.longPressTimer = null;

      const target = e.currentTarget;
      target.classList.remove('touch-active', 'long-press-active');

      e.changedTouches.forEach(touch => {
        this.activeTouches.delete(touch.identifier || 'default');
      });
    }

    handleTap(e, target, touch) {
      const now = Date.now();
      const tapPos = { x: touch.clientX, y: touch.clientY };

      if (now - this.lastTapTime < MOBILE_ENHANCED_CONFIG.doubleTapDelay) {
        const deltaX = Math.abs(tapPos.x - this.lastTapPos.x);
        const deltaY = Math.abs(tapPos.y - this.lastTapPos.y);

        if (deltaX < 50 && deltaY < 50) {
          this.triggerHapticFeedback('light');
          this.dispatchCustomEvent(target, 'mobiledoubletap', tapPos);
          this.lastTapTime = 0;
          return;
        }
      }

      this.lastTapTime = now;
      this.lastTapPos = { ...tapPos };

      this.triggerHapticFeedback('light');
      this.dispatchCustomEvent(target, 'mobiletap', tapPos);
    }

    handleMultiTouchStart(e) {
      if (e.touches.length === 2) {
        const dx = e.touches[0].clientX - e.touches[1].clientX;
        const dy = e.touches[0].clientY - e.touches[1].clientY;
        this.pinchStartDistance = Math.sqrt(dx * dx + dy * dy);
      }
    }

    handleMultiTouchMove(e) {
      if (e.touches.length === 2 && this.pinchStartDistance > 0) {
        const dx = e.touches[0].clientX - e.touches[1].clientX;
        const dy = e.touches[0].clientY - e.touches[1].clientY;
        const currentDistance = Math.sqrt(dx * dx + dy * dy);

        const scale = currentDistance / this.pinchStartDistance;
        this.currentZoom = Math.max(
          MOBILE_ENHANCED_CONFIG.pinchZoomMin,
          Math.min(MOBILE_ENHANCED_CONFIG.pinchZoomMax, scale)
        );

        const centerX = (e.touches[0].clientX + e.touches[1].clientX) / 2;
        const centerY = (e.touches[0].clientY + e.touches[1].clientY) / 2;

        const target = e.currentTarget;
        if (target.classList.contains('captcha-click-image') || target.classList.contains('captcha-canvas')) {
          target.style.transform = `scale(${this.currentZoom})`;
          this.dispatchCustomEvent(target, 'mobilepinch', {
            scale: this.currentZoom,
            centerX: centerX,
            centerY: centerY
          });
        }
      }
    }

    setupGestureFeedback() {
      document.addEventListener('touchstart', (e) => {
        const target = e.currentTarget;
        if (target.classList.contains('feedback-enabled')) {
          this.showTouchFeedback(target, e.touches[0]);
        }
      }, { passive: true });

      document.addEventListener('touchmove', (e) => {
        if (e.touches.length === 1) {
          this.updateTouchFeedbackPosition(e.touches[0]);
        }
      }, { passive: true });

      document.addEventListener('touchend', () => {
        this.hideTouchFeedback();
      }, { passive: true });
    }

    showTouchFeedback(element, touch) {
      const feedback = document.createElement('div');
      feedback.className = 'mobile-touch-feedback';
      feedback.style.cssText = `
        position: fixed;
        left: ${touch.clientX - 30}px;
        top: ${touch.clientY - 30}px;
        width: 60px;
        height: 60px;
        border-radius: 50%;
        background: radial-gradient(circle, rgba(201, 169, 110, 0.4) 0%, transparent 70%);
        pointer-events: none;
        z-index: 10000;
        transform: scale(0);
        transition: transform 0.15s ease-out, opacity 0.15s ease-out;
        opacity: 1;
      `;

      document.body.appendChild(feedback);

      requestAnimationFrame(() => {
        feedback.style.transform = 'scale(1)';
      });

      setTimeout(() => {
        feedback.style.opacity = '0';
        setTimeout(() => feedback.remove(), 150);
      }, MOBILE_ENHANCED_CONFIG.touchFeedbackDuration);
    }

    updateTouchFeedbackPosition(touch) {
      const feedbacks = document.querySelectorAll('.mobile-touch-feedback');
      feedbacks.forEach(feedback => {
        feedback.style.left = `${touch.clientX - 30}px`;
        feedback.style.top = `${touch.clientY - 30}px`;
      });
    }

    hideTouchFeedback() {
    }

    triggerHapticFeedback(intensity = 'light') {
      if ('vibrate' in navigator) {
        const patterns = {
          light: 10,
          medium: 25,
          heavy: 50,
          success: [20, 50, 20],
          warning: [30, 30, 30],
          error: [50, 30, 50]
        };
        navigator.vibrate(patterns[intensity] || patterns.light);
      }
    }

    setupKeyboardInputOptimization() {
      this.setupInputFocusBehavior();
      this.setupKeyboardHeightDetection();
      this.setupInputAutocomplete();
      this.setupInputValidation();
      this.setupFormSubmitOptimization();
    }

    setupInputFocusBehavior() {
      const inputs = document.querySelectorAll('input, textarea, select');

      inputs.forEach(input => {
        input.addEventListener('focus', (e) => {
          this.handleInputFocus(e.target);
        });

        input.addEventListener('blur', (e) => {
          this.handleInputBlur(e.target);
        });

        input.addEventListener('input', (e) => {
          this.handleInputChange(e.target);
        });

        if (!input.hasAttribute('data-keyboard-optimized')) {
          input.setAttribute('data-keyboard-optimized', 'true');
          input.style.fontSize = '16px';
        }
      });
    }

    handleInputFocus(input) {
      this.keyboardVisible = true;

      setTimeout(() => {
        this.adjustViewportForKeyboard();
      }, 100);

      input.classList.add('keyboard-focused');

      if (input.hasAttribute('data-autocomplete')) {
        this.showAutocomplete(input);
      }

      this.scrollInputIntoView(input);
    }

    handleInputBlur(input) {
      this.keyboardVisible = false;
      input.classList.remove('keyboard-focused');

      this.resetViewportAfterKeyboard();

      if (input.hasAttribute('required') && !input.value.trim()) {
        this.showValidationError(input, '此字段为必填项');
      }
    }

    handleInputChange(input) {
      this.debounce(() => {
        this.validateInput(input);
        this.updateInputState(input);
      }, MOBILE_ENHANCED_CONFIG.keyboardDebounceDelay)();
    }

    validateInput(input) {
      const value = input.value;
      const type = input.type;
      const validations = [];

      if (input.required && !value.trim()) {
        validations.push({ type: 'required', passed: false });
      }

      if (type === 'email' && value) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        validations.push({ type: 'email', passed: emailRegex.test(value) });
      }

      if (type === 'tel' && value) {
        const telRegex = /^[\d\s\-\+\(\)]+$/;
        validations.push({ type: 'tel', passed: telRegex.test(value) });
      }

      if (input.hasAttribute('minlength')) {
        const minLength = parseInt(input.getAttribute('minlength'));
        validations.push({
          type: 'minlength',
          passed: value.length >= minLength
        });
      }

      if (input.hasAttribute('maxlength')) {
        const maxLength = parseInt(input.getAttribute('maxlength'));
        validations.push({
          type: 'maxlength',
          passed: value.length <= maxLength
        });
      }

      const allPassed = validations.every(v => v.passed);
      input.setAttribute('data-valid', String(allPassed));

      return allPassed;
    }

    updateInputState(input) {
      const value = input.value;
      const maxLength = input.getAttribute('maxlength');

      if (maxLength) {
        const remaining = parseInt(maxLength) - value.length;
        this.updateCharacterCount(input, remaining);
      }

      if (value.length > 0) {
        input.classList.add('has-content');
      } else {
        input.classList.remove('has-content');
      }
    }

    updateCharacterCount(input, remaining) {
      let counter = input.parentElement.querySelector('.char-counter');

      if (!counter && remaining < 20) {
        counter = document.createElement('div');
        counter.className = 'char-counter';
        input.parentElement.appendChild(counter);
      }

      if (counter) {
        counter.textContent = `${remaining}`;
        counter.className = `char-counter ${remaining < 0 ? 'text-danger' : remaining < 5 ? 'text-warning' : 'text-muted'}`;
      }
    }

    showValidationError(input, message) {
      input.classList.add('is-invalid');

      let errorEl = input.parentElement.querySelector('.invalid-feedback');
      if (!errorEl) {
        errorEl = document.createElement('div');
        errorEl.className = 'invalid-feedback';
        input.parentElement.appendChild(errorEl);
      }
      errorEl.textContent = message;

      setTimeout(() => {
        input.classList.remove('is-invalid');
      }, 3000);
    }

    adjustViewportForKeyboard() {
      const activeElement = document.activeElement;
      if (!activeElement || !['INPUT', 'TEXTAREA', 'SELECT'].includes(activeElement.tagName)) {
        return;
      }

      const inputRect = activeElement.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const keyboardHeight = viewportHeight - inputRect.bottom - (inputRect.height || 40);

      if (keyboardHeight > 0 && keyboardHeight < viewportHeight * 0.6) {
        this.keyboardHeight = keyboardHeight;
        const offset = keyboardHeight + 10;

        document.body.style.cssText += `
          position: fixed;
          top: ${-offset}px;
          height: ${viewportHeight + offset}px;
        `;
      }
    }

    resetViewportAfterKeyboard() {
      if (this.keyboardVisible) return;

      document.body.style.position = '';
      document.body.style.top = '';
      document.body.style.height = '';

      this.keyboardHeight = 0;
    }

    setupKeyboardHeightDetection() {
      if ('visualViewport' in window) {
        const visualViewport = window.visualViewport;

        visualViewport.addEventListener('resize', () => {
          const keyboardHeight = window.innerHeight - visualViewport.height;

          if (keyboardHeight > 0 && keyboardHeight < window.innerHeight * 0.6) {
            this.keyboardHeight = keyboardHeight;
            this.onKeyboardShow(keyboardHeight);
          } else {
            this.onKeyboardHide();
          }
        });

        visualViewport.addEventListener('scroll', () => {
          if (this.keyboardVisible) {
            this.adjustViewportForKeyboard();
          }
        });
      } else {
        window.addEventListener('resize', () => {
          if (this.keyboardVisible) {
            this.adjustViewportForKeyboard();
          }
        });
      }
    }

    onKeyboardShow(height) {
      this.keyboardVisible = true;
      document.body.classList.add('keyboard-visible');

      const focusedInput = document.activeElement;
      if (focusedInput && ['INPUT', 'TEXTAREA'].includes(focusedInput.tagName)) {
        this.scrollInputIntoView(focusedInput);
      }

      document.dispatchEvent(new CustomEvent('keyboardshow', {
        detail: { height: height }
      }));
    }

    onKeyboardHide() {
      this.keyboardVisible = false;
      document.body.classList.remove('keyboard-visible');
      this.resetViewportAfterKeyboard();

      document.dispatchEvent(new CustomEvent('keyboardhide'));
    }

    scrollInputIntoView(input) {
      const rect = input.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const keyboardHeight = this.keyboardHeight || 0;
      const visibleArea = viewportHeight - keyboardHeight;

      if (rect.bottom > visibleArea - 20) {
        const scrollOffset = rect.bottom - visibleArea + 20;
        const currentScroll = window.pageYOffset;
        const newScroll = currentScroll + scrollOffset;

        window.scrollTo({
          top: newScroll,
          behavior: 'smooth'
        });
      }
    }

    setupInputAutocomplete() {
      const inputs = document.querySelectorAll('input[autocomplete]');

      inputs.forEach(input => {
        const autocompleteValue = input.getAttribute('autocomplete');

        if (['on', 'off', 'name', 'email', 'tel', 'username'].includes(autocompleteValue)) {
          input.setAttribute('autocomplete', autocompleteValue);
          input.setAttribute('autocapitalize', 'off');
          input.setAttribute('autocorrect', 'off');
          input.setAttribute('spellcheck', 'false');
        }
      });
    }

    setupFormSubmitOptimization() {
      const forms = document.querySelectorAll('form');

      forms.forEach(form => {
        form.addEventListener('submit', (e) => {
          if (!this.validateForm(form)) {
            e.preventDefault();
            e.stopPropagation();
            return false;
          }

          this.handleFormSubmit(form);
        }, { passive: false });
      });
    }

    validateForm(form) {
      const inputs = form.querySelectorAll('input, textarea, select');
      let isValid = true;

      inputs.forEach(input => {
        if (!this.validateInput(input)) {
          isValid = false;
          input.classList.add('was-validated');
        }
      });

      return isValid;
    }

    handleFormSubmit(form) {
      const submitBtn = form.querySelector('[type="submit"]');

      if (submitBtn && !submitBtn.disabled) {
        submitBtn.disabled = true;
        submitBtn.classList.add('submitting');

        const originalText = submitBtn.textContent;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>提交中...';

        setTimeout(() => {
          submitBtn.disabled = false;
          submitBtn.classList.remove('submitting');
          submitBtn.textContent = originalText;
        }, 5000);
      }
    }

    showAutocomplete(input) {
    }

    setupMobileAnimations() {
      this.setupEnterAnimations();
      this.setupExitAnimations();
      this.setupScrollAnimations();
      this.setupTransitionAnimations();
    }

    setupEnterAnimations() {
      const animatedElements = document.querySelectorAll('[data-animate="fadeIn"], [data-animate="slideUp"], [data-animate="scaleIn"]');

      const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            this.playEnterAnimation(entry.target);
            observer.unobserve(entry.target);
          }
        });
      }, {
        threshold: 0.1,
        rootMargin: '50px'
      });

      animatedElements.forEach(el => observer.observe(el));
    }

    playEnterAnimation(element) {
      const animationType = element.getAttribute('data-animate') || 'fadeIn';
      const delay = parseInt(element.getAttribute('data-delay') || '0');
      const duration = parseInt(element.getAttribute('data-duration') || '400');

      setTimeout(() => {
        element.style.animation = `${animationType} ${duration}ms ease-out forwards`;
        element.classList.add('animated');
      }, delay);
    }

    setupExitAnimations() {
      document.addEventListener('click', (e) => {
        const closeButton = e.target.closest('[data-dismiss], .close, [data-close]');
        if (closeButton) {
          const targetId = closeButton.getAttribute('data-target') ||
                          closeButton.getAttribute('data-bs-target');
          if (targetId) {
            const target = document.querySelector(targetId);
            if (target) {
              this.playExitAnimation(target, () => {
                if (target.classList.contains('modal')) {
                  target.style.display = 'none';
                } else {
                  target.remove();
                }
              });
            }
          }
        }
      });
    }

    playExitAnimation(element, callback) {
      element.style.animation = 'fadeOut 200ms ease-in forwards';

      setTimeout(() => {
        if (callback) callback();
      }, 200);
    }

    setupScrollAnimations() {
      let ticking = false;

      window.addEventListener('scroll', () => {
        if (!ticking) {
          requestAnimationFrame(() => {
            this.updateScrollAnimations();
            ticking = false;
          });
          ticking = true;
        }
      }, { passive: true });
    }

    updateScrollAnimations() {
      const scrollY = window.pageYOffset;
      const viewportHeight = window.innerHeight;

      const parallaxElements = document.querySelectorAll('[data-parallax]');
      parallaxElements.forEach(el => {
        const speed = parseFloat(el.getAttribute('data-parallax') || '0.5');
        const offset = scrollY * speed;
        el.style.transform = `translateY(${offset}px)`;
      });

      const revealElements = document.querySelectorAll('[data-reveal]');
      revealElements.forEach(el => {
        const rect = el.getBoundingClientRect();
        if (rect.top < viewportHeight * 0.85) {
          el.classList.add('revealed');
        }
      });
    }

    setupTransitionAnimations() {
      document.addEventListener('click', (e) => {
        const toggle = e.target.closest('[data-toggle-class], [data-toggle-animation]');
        if (toggle) {
          const targetSelector = toggle.getAttribute('data-target');
          const className = toggle.getAttribute('data-toggle-class');
          const animation = toggle.getAttribute('data-toggle-animation');

          if (targetSelector && className) {
            const target = document.querySelector(targetSelector);
            if (target) {
              this.toggleClassWithAnimation(target, className, animation);
            }
          }
        }
      });
    }

    toggleClassWithAnimation(element, className, animation) {
      if (animation) {
        element.style.transition = animation;
      }

      element.classList.toggle(className);

      if (animation) {
        setTimeout(() => {
          element.style.transition = '';
        }, 300);
      }
    }

    animateTransform(element, transform, duration = MOBILE_ENHANCED_CONFIG.animationDuration) {
      return new Promise(resolve => {
        element.style.transition = `transform ${duration}ms ease-out`;
        element.style.transform = transform;

        setTimeout(() => {
          element.style.transition = '';
          resolve();
        }, duration);
      });
    }

    setupInertiaScrolling() {
      const scrollableElements = document.querySelectorAll('.overflow-auto, [data-inertia-scroll]');

      scrollableElements.forEach(el => {
        let velocity = 0;
        let lastY = 0;
        let lastTime = 0;
        let isScrolling = false;

        el.addEventListener('touchstart', (e) => {
          lastY = e.touches[0].clientY;
          lastTime = Date.now();
          isScrolling = true;
          velocity = 0;
        }, { passive: true });

        el.addEventListener('touchmove', (e) => {
          if (!isScrolling) return;

          const currentY = e.touches[0].clientY;
          const currentTime = Date.now();
          const deltaY = currentY - lastY;
          const deltaTime = currentTime - lastTime;

          if (deltaTime > 0) {
            velocity = deltaY / deltaTime;
          }

          lastY = currentY;
          lastTime = currentTime;
        }, { passive: true });

        el.addEventListener('touchend', () => {
          if (!isScrolling) return;
          isScrolling = false;

          if (Math.abs(velocity) > MOBILE_ENHANCED_CONFIG.gestureVelocityThreshold) {
            this.applyInertia(el, velocity);
          }
        }, { passive: true });
      });
    }

    applyInertia(element, velocity) {
      const startScrollTop = element.scrollTop;
      const startTime = Date.now();
      const duration = MOBILE_ENHANCED_CONFIG.inertiaDuration;

      const animate = () => {
        const elapsed = Date.now() - startTime;
        const progress = Math.min(elapsed / duration, 1);

        const easeOutQuart = 1 - Math.pow(1 - progress, 4);
        const deceleration = 0.95;

        const currentVelocity = velocity * Math.pow(deceleration, elapsed / 16);
        const displacement = currentVelocity * 16 * (1 - easeOutQuart);

        element.scrollTop = startScrollTop - displacement;

        if (progress < 1 && Math.abs(currentVelocity) > 0.1) {
          requestAnimationFrame(animate);
        }
      };

      requestAnimationFrame(animate);
    }

    setupTouchRipple() {
      const rippleTargets = document.querySelectorAll(
        '.btn, .captcha-slider-button, .captcha-click-marker, ' +
        '.captcha-tab, .captcha-refresh, [role="button"], button, a'
      );

      rippleTargets.forEach(el => {
        if (el.hasAttribute('data-ripple')) return;

        el.setAttribute('data-ripple', 'true');
        el.style.position = 'relative';
        el.style.overflow = 'hidden';

        el.addEventListener('touchstart', (e) => {
          this.createRipple(e.currentTarget, e.touches[0].clientX, e.touches[0].clientY);
        }, { passive: true });

        el.addEventListener('click', (e) => {
          this.createRipple(e.currentTarget, e.clientX, e.clientY);
        }, { passive: true });
      });
    }

    createRipple(element, x, y) {
      const rect = element.getBoundingClientRect();
      const ripple = document.createElement('span');
      const size = Math.max(rect.width, rect.height);

      const offsetX = x - rect.left;
      const offsetY = y - rect.top;

      ripple.className = 'touch-ripple';
      ripple.style.cssText = `
        position: absolute;
        width: ${size}px;
        height: ${size}px;
        left: ${offsetX - size / 2}px;
        top: ${offsetY - size / 2}px;
        border-radius: 50%;
        background: radial-gradient(circle, rgba(255, 255, 255, 0.3) 0%, transparent 70%);
        transform: scale(0);
        animation: rippleEffect ${MOBILE_ENHANCED_CONFIG.touchRippleDuration}ms ease-out forwards;
        pointer-events: none;
        z-index: 1;
      `;

      const existingRipple = element.querySelector('.touch-ripple');
      if (existingRipple) {
        existingRipple.remove();
      }

      element.appendChild(ripple);

      setTimeout(() => {
        ripple.remove();
      }, MOBILE_ENHANCED_CONFIG.touchRippleDuration);
    }

    createTouchRipple(element, x, y) {
      if (!element.classList.contains('ripple-enabled')) return;

      const rect = element.getBoundingClientRect();
      const ripple = document.createElement('div');
      const size = Math.max(rect.width, rect.height) * 2;

      ripple.className = 'touch-ripple-effect';
      ripple.style.cssText = `
        position: absolute;
        width: ${size}px;
        height: ${size}px;
        left: ${x - rect.left - size / 2}px;
        top: ${y - rect.top - size / 2}px;
        border-radius: 50%;
        background: rgba(201, 169, 110, 0.2);
        transform: scale(0);
        animation: touchRippleAnim 0.4s ease-out forwards;
        pointer-events: none;
      `;

      element.style.position = 'relative';
      element.style.overflow = 'hidden';
      element.appendChild(ripple);

      setTimeout(() => ripple.remove(), 400);
    }

    setupSwipeNavigation() {
      let swipeStartX = 0;
      let swipeStartY = 0;
      let swipeStartTime = 0;
      let isSwipe = false;

      document.addEventListener('touchstart', (e) => {
        if (e.touches.length === 1) {
          swipeStartX = e.touches[0].clientX;
          swipeStartY = e.touches[0].clientY;
          swipeStartTime = Date.now();
          isSwipe = true;
        }
      }, { passive: true });

      document.addEventListener('touchmove', (e) => {
        if (isSwipe && e.touches.length === 1) {
          const deltaX = e.touches[0].clientX - swipeStartX;
          const deltaY = e.touches[0].clientY - swipeStartY;

          if (Math.abs(deltaY) > Math.abs(deltaX) * 1.5) {
            isSwipe = false;
          }
        }
      }, { passive: true });

      document.addEventListener('touchend', (e) => {
        if (!isSwipe) return;

        const swipeEndX = e.changedTouches[0].clientX;
        const swipeEndY = e.changedTouches[0].clientY;
        const swipeEndTime = Date.now();

        const deltaX = swipeEndX - swipeStartX;
        const deltaY = swipeEndY - swipeStartY;
        const deltaTime = swipeEndTime - swipeStartTime;

        if (deltaTime < 500 && Math.abs(deltaX) > MOBILE_ENHANCED_CONFIG.swipeThreshold) {
          const direction = deltaX > 0 ? 'right' : 'left';
          this.triggerHapticFeedback('light');
          this.dispatchSwipeEvent(direction, Math.abs(deltaX), deltaTime);

          this.recordGesture({
            type: 'swipe',
            direction: direction,
            distance: Math.abs(deltaX),
            duration: deltaTime
          });
        }

        isSwipe = false;
      }, { passive: true });
    }

    dispatchSwipeEvent(direction, distance, duration) {
      const swipeTargets = document.querySelectorAll('[data-swipe-navigate]');

      swipeTargets.forEach(target => {
        const enabledDirections = (target.getAttribute('data-swipe-directions') || 'left,right').split(',');

        if (enabledDirections.includes(direction)) {
          target.dispatchEvent(new CustomEvent('swipenavigate', {
            bubbles: true,
            detail: { direction, distance, duration }
          }));
        }
      });
    }

    setupPinchZoom() {
      let currentScale = 1;
      let isPinching = false;

      if ('GestureEvent' in window) {
        document.addEventListener('gesturestart', (e) => {
          currentScale = e.scale;
          isPinching = true;
        });

        document.addEventListener('gesturechange', (e) => {
          if (isPinching) {
            const scale = Math.max(0.5, Math.min(3, e.scale));
            currentScale = scale;

            const zoomableElements = document.querySelectorAll('[data-pinch-zoom]');
            zoomableElements.forEach(el => {
              el.style.transform = `scale(${scale})`;
            });

            document.dispatchEvent(new CustomEvent('pinchzoom', {
              detail: { scale: scale, rotation: e.rotation }
            }));
          }
        });

        document.addEventListener('gestureend', () => {
          isPinching = false;

          if (currentScale < 1.1) {
            currentScale = 1;
            const zoomableElements = document.querySelectorAll('[data-pinch-zoom]');
            zoomableElements.forEach(el => {
              el.style.transform = 'scale(1)';
            });
          }
        });
      }
    }

    setupGestureHistory() {
      this.gestureHistory = [];

      document.addEventListener('mobiletap', (e) => {
        this.recordGesture({ type: 'tap', ...e.detail });
      });

      document.addEventListener('mobiledoubletap', (e) => {
        this.recordGesture({ type: 'doubletap', ...e.detail });
      });

      document.addEventListener('mobilepress', (e) => {
        this.recordGesture({ type: 'press', ...e.detail });
      });

      document.addEventListener('swipenavigate', (e) => {
        this.recordGesture({ type: 'swipe', ...e.detail });
      });
    }

    recordGesture(gesture) {
      gesture.timestamp = Date.now();

      this.gestureHistory.push(gesture);

      if (this.gestureHistory.length > this.maxGestureHistory) {
        this.gestureHistory.shift();
      }
    }

    getGestureHistory() {
      return [...this.gestureHistory];
    }

    clearGestureHistory() {
      this.gestureHistory = [];
    }

    setupOrientationHandling() {
      window.addEventListener('orientationchange', () => {
        setTimeout(() => {
          this.handleOrientationChange();
        }, 100);
      });

      window.addEventListener('resize', () => {
        this.debounce(() => {
          this.handleResize();
        }, 250)();
      });

      this.handleOrientationChange();
    }

    handleOrientationChange() {
      const isLandscape = window.innerWidth > window.innerHeight;
      const orientation = isLandscape ? 'landscape' : 'portrait';

      document.body.setAttribute('data-orientation', orientation);

      this.updateLayoutForOrientation(orientation);

      document.dispatchEvent(new CustomEvent('orientationchange', {
        detail: { orientation, width: window.innerWidth, height: window.innerHeight }
      }));
    }

    handleResize() {
      const width = window.innerWidth;
      let breakpoint = 'xl';

      if (width < 576) breakpoint = 'xs';
      else if (width < 768) breakpoint = 'sm';
      else if (width < 992) breakpoint = 'md';
      else if (width < 1200) breakpoint = 'lg';

      document.body.setAttribute('data-breakpoint', breakpoint);

      this.optimizeInteractiveElements();
    }

    updateLayoutForOrientation(orientation) {
      const captchaContainer = document.querySelector('.captcha-container');

      if (captchaContainer) {
        captchaContainer.classList.remove('orientation-portrait', 'orientation-landscape');
        captchaContainer.classList.add(`orientation-${orientation}`);
      }
    }

    setupResizeObserver() {
      if ('ResizeObserver' in window) {
        const resizeObserver = new ResizeObserver(entries => {
          entries.forEach(entry => {
            this.handleElementResize(entry.target, entry.contentRect);
          });
        });

        const monitoredElements = document.querySelectorAll('.captcha-container, .captcha-canvas');
        monitoredElements.forEach(el => resizeObserver.observe(el));
      }
    }

    handleElementResize(element, contentRect) {
      if (contentRect.width > 0 && contentRect.height > 0) {
        element.dispatchEvent(new CustomEvent('elementresize', {
          detail: {
            width: contentRect.width,
            height: contentRect.height
          }
        }));

        if (element.classList.contains('captcha-canvas')) {
          this.optimizeCanvasRendering(element);
        }
      }
    }

    optimizeCanvasRendering(canvas) {
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      const dpr = window.devicePixelRatio || 1;
      const rect = canvas.getBoundingClientRect();

      if (canvas.width !== rect.width * dpr || canvas.height !== rect.height * dpr) {
        const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);

        canvas.width = rect.width * dpr;
        canvas.height = rect.height * dpr;
        canvas.style.width = `${rect.width}px`;
        canvas.style.height = `${rect.height}px`;

        ctx.scale(dpr, dpr);
        ctx.putImageData(imageData, 0, 0);
      }
    }

    bindGlobalEvents() {
      document.addEventListener('visibilitychange', () => {
        if (document.hidden) {
          this.onPageHidden();
        } else {
          this.onPageVisible();
        }
      });

      window.addEventListener('offline', () => {
        this.onNetworkOffline();
      });

      window.addEventListener('online', () => {
        this.onNetworkOnline();
      });
    }

    onPageHidden() {
      this.pauseAnimations();
    }

    onPageVisible() {
      this.resumeAnimations();
    }

    onNetworkOffline() {
      document.body.classList.add('offline');
      console.log('[MobileEnhanced] 网络已断开');
    }

    onNetworkOnline() {
      document.body.classList.remove('offline');
      console.log('[MobileEnhanced] 网络已恢复');
    }

    pauseAnimations() {
      document.body.classList.add('animations-paused');
    }

    resumeAnimations() {
      document.body.classList.remove('animations-paused');
    }

    optimizeInteractiveElements() {
      const interactiveElements = this.getInteractiveElements();

      interactiveElements.forEach(el => {
        const rect = el.getBoundingClientRect();
        const minSize = 44;

        if (rect.width < minSize || rect.height < minSize) {
          el.classList.add('touch-target-small');
        }
      });
    }

    setupPullToRefresh() {
      if (!document.querySelector('[data-pull-refresh]')) return;

      let startY = 0;
      let currentY = 0;
      let isPulling = false;
      const pullThreshold = 80;

      document.body.style.overscrollBehavior = 'none';

      const indicator = document.createElement('div');
      indicator.className = 'pull-refresh-indicator';
      indicator.innerHTML = '<div class="pull-refresh-spinner"></div>';
      document.body.insertBefore(indicator, document.body.firstChild);

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

          if (pullDistance > 0) {
            const progress = Math.min(pullDistance / pullThreshold, 1);
            indicator.style.transform = `translateY(${pullDistance * 0.5}px)`;
            indicator.classList.toggle('active', pullDistance > 20);
            indicator.style.setProperty('--pull-progress', progress);

            document.dispatchEvent(new CustomEvent('pullrefresh', {
              detail: { distance: pullDistance, progress: progress }
            }));
          }
        }
      }, { passive: true });

      document.addEventListener('touchend', () => {
        if (isPulling) {
          const pullDistance = currentY - startY;

          if (pullDistance > pullThreshold) {
            indicator.classList.add('refreshing');
            document.dispatchEvent(new CustomEvent('pullrefreshcomplete'));

            setTimeout(() => {
              indicator.classList.remove('active', 'refreshing');
              indicator.style.transform = '';
            }, 1000);
          } else {
            indicator.classList.remove('active');
            indicator.style.transform = '';
          }
        }

        isPulling = false;
        startY = 0;
        currentY = 0;
      }, { passive: true });
    }

    setupTouchTargets() {
      const touchTargetElements = document.querySelectorAll(
        '.captcha-slider-button, .captcha-click-marker, .captcha-refresh, ' +
        'button:not(.btn-sm), a:not(.btn-sm)'
      );

      touchTargetElements.forEach(el => {
        const rect = el.getBoundingClientRect();
        const minSize = 44;

        if (rect.width < minSize || rect.height < minSize) {
          el.classList.add('touch-target-min');
        }
      });
    }

    dispatchCustomEvent(element, eventName, detail = {}) {
      const event = new CustomEvent(eventName, {
        bubbles: true,
        detail: detail
      });
      element.dispatchEvent(event);
    }

    debounce(func, wait) {
      let timeout;
      return function executedFunction(...args) {
        const later = () => {
          clearTimeout(timeout);
          func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
      };
    }

    injectEnhancedStyles() {
      if (document.getElementById('mobile-enhanced-styles')) {
        return;
      }

      const style = document.createElement('style');
      style.id = 'mobile-enhanced-styles';
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

        @keyframes rippleEffect {
          to {
            transform: scale(2);
            opacity: 0;
          }
        }

        @keyframes touchRippleAnim {
          0% {
            transform: scale(0);
            opacity: 1;
          }
          100% {
            transform: scale(2);
            opacity: 0;
          }
        }

        @keyframes fadeIn {
          from {
            opacity: 0;
            transform: translateY(20px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes fadeOut {
          from {
            opacity: 1;
            transform: translateY(0);
          }
          to {
            opacity: 0;
            transform: translateY(-20px);
          }
        }

        @keyframes slideUp {
          from {
            opacity: 0;
            transform: translateY(30px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes scaleIn {
          from {
            opacity: 0;
            transform: scale(0.8);
          }
          to {
            opacity: 1;
            transform: scale(1);
          }
        }

        .animated {
          opacity: 0;
        }

        .revealed {
          animation: fadeIn 0.5s ease-out forwards;
        }

        .keyboard-focused {
          border-color: var(--primary, #0d6efd) !important;
          box-shadow: 0 0 0 0.2rem rgba(13, 110, 253, 0.25) !important;
        }

        .is-invalid {
          border-color: #dc3545 !important;
          animation: shake 0.3s ease-in-out;
        }

        @keyframes shake {
          0%, 100% { transform: translateX(0); }
          25% { transform: translateX(-5px); }
          75% { transform: translateX(5px); }
        }

        .char-counter {
          font-size: 12px;
          text-align: right;
          margin-top: 4px;
        }

        @media (hover: none) and (pointer: coarse) {
          .captcha-slider-button,
          .captcha-click-marker,
          .btn,
          .nav-link,
          .captcha-tab,
          .captcha-refresh {
            min-height: 44px;
            min-width: 44px;
          }

          input, textarea, select {
            font-size: 16px !important;
          }

          input:focus, textarea:focus, select:focus {
            font-size: 16px !important;
          }
        }

        @media (max-width: 576px) {
          .captcha-container {
            margin: 0 !important;
            border-radius: 0 !important;
            border-left: none !important;
            border-right: none !important;
          }

          .captcha-container.orientation-landscape {
            max-width: 480px;
            margin: 0 auto !important;
            border-radius: 12px !important;
          }

          .captcha-slider-container {
            height: 50px !important;
            border-radius: 25px !important;
          }

          .captcha-slider-button {
            width: 46px !important;
            height: 46px !important;
          }

          .captcha-click-marker {
            width: 36px !important;
            height: 36px !important;
          }

          .nav-link {
            padding: 0.75rem 1rem !important;
          }
        }

        @media (prefers-reduced-motion: reduce) {
          *,
          *::before,
          *::after {
            animation-duration: 0.01ms !important;
            animation-iteration-count: 1 !important;
            transition-duration: 0.01ms !important;
          }

          .touch-active,
          .long-press-active {
            transition: none !important;
          }
        }

        .animations-paused *,
        .animations-paused *::before,
        .animations-paused *::after {
          animation-play-state: paused !important;
          transition-play-state: paused !important;
        }

        .offline::before {
          content: '离线模式';
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          background: #ffc107;
          color: #000;
          text-align: center;
          padding: 8px;
          z-index: 9999;
          font-size: 14px;
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
          transition: height 0.3s cubic-bezier(0.4, 0, 0.2, 1), transform 0.3s ease;
        }

        .pull-refresh-indicator.active {
          height: 65px;
        }

        .pull-refresh-indicator.refreshing {
          animation: refreshSpin 0.8s linear infinite;
        }

        @keyframes refreshSpin {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(360deg); }
        }

        .pull-refresh-spinner {
          width: 28px;
          height: 28px;
          border: 3px solid rgba(255,255,255,0.4);
          border-top-color: white;
          border-radius: 50%;
          opacity: 0;
          transition: opacity 0.3s ease;
        }

        .pull-refresh-indicator.active .pull-refresh-spinner {
          opacity: 1;
        }

        .touch-target-min {
          min-width: 44px !important;
          min-height: 44px !important;
        }

        .mobile-safe-area {
          padding-top: env(safe-area-inset-top);
          padding-bottom: env(safe-area-inset-bottom);
          padding-left: env(safe-area-inset-left);
          padding-right: env(safe-area-inset-right);
        }

        .keyboard-visible {
          overflow: hidden;
        }
      `;

      document.head.appendChild(style);
    }
  }

  const mobileEnhancedAdapter = new MobileEnhancedAdapter();

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => mobileEnhancedAdapter.init());
  } else {
    mobileEnhancedAdapter.init();
  }

  window.MobileEnhancedAdapter = mobileEnhancedAdapter;
})();
