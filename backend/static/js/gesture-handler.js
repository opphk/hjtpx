(function() {
  'use strict';

  const GESTURE_CONFIG = {
    minSwipeDistance: 30,
    maxSwipeTime: 1000,
    rotationSensitivity: 0.1,
    pinchThreshold: 0.02,
    dragThreshold: 5,
    velocityThreshold: 0.5,
    touchFeedbackDuration: 120,
    longPressDelay: 500,
    doubleTapDelay: 300
  };

  class GestureHandler {
    constructor() {
      this.gestureStartPoints = new Map();
      this.activeGestures = new Map();
      this.gestureHandlers = new Map();
      this.pinchStartDistance = 0;
      this.pinchStartScale = 1;
      this.rotationStartAngle = 0;
      this.lastTapTime = 0;
      this.tapCount = 0;
      this.tapTimeout = null;
    }

    init() {
      console.log('[Gesture] 初始化手势处理模块');
      this.setupTouchGestures();
      this.setupSwipeGestures();
      this.setupPinchZoom();
      this.setupDragAndDrop();
      this.setupDoubleTap();
      this.setupLongPress();
      this.injectGestureStyles();
    }

    setupLongPress() {
      const longPressTargets = document.querySelectorAll('[data-long-press="true"], .captcha-interactive, .captcha-refresh');

      longPressTargets.forEach(el => {
        let longPressTimer = null;
        let isMoving = false;
        let startX = 0;
        let startY = 0;

        el.addEventListener('touchstart', (e) => {
          startX = e.touches[0].clientX;
          startY = e.touches[0].clientY;
          isMoving = false;

          longPressTimer = setTimeout(() => {
            this.triggerHapticFeedback('medium');
            el.classList.add('long-press-active');
            this.dispatchGestureEvent('longpress', el, {
              x: startX,
              y: startY,
              duration: GESTURE_CONFIG.longPressDelay
            });
          }, GESTURE_CONFIG.longPressDelay);
        }, { passive: true });

        el.addEventListener('touchmove', (e) => {
          const deltaX = Math.abs(e.touches[0].clientX - startX);
          const deltaY = Math.abs(e.touches[0].clientY - startY);

          if (deltaX > 10 || deltaY > 10) {
            isMoving = true;
            if (longPressTimer) {
              clearTimeout(longPressTimer);
              longPressTimer = null;
              el.classList.remove('long-press-active');
            }
          }
        }, { passive: true });

        el.addEventListener('touchend', () => {
          if (longPressTimer) {
            clearTimeout(longPressTimer);
            longPressTimer = null;
          }
          el.classList.remove('long-press-active');
        }, { passive: true });

        el.addEventListener('touchcancel', () => {
          if (longPressTimer) {
            clearTimeout(longPressTimer);
            longPressTimer = null;
          }
          el.classList.remove('long-press-active');
        });
      });
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

    setupTouchGestures() {
      document.addEventListener('touchstart', (e) => this.handleGestureStart(e), { passive: false });
      document.addEventListener('touchmove', (e) => this.handleGestureMove(e), { passive: false });
      document.addEventListener('touchend', (e) => this.handleGestureEnd(e), { passive: false });
      document.addEventListener('touchcancel', (e) => this.handleGestureCancel(e), { passive: false });
    }

    handleGestureStart(e) {
      const touch = e.touches[0];
      const target = e.currentTarget;
      const pointId = touch.identifier || 'default';

      this.gestureStartPoints.set(pointId, {
        x: touch.clientX,
        y: touch.clientY,
        time: Date.now(),
        target: target
      });

      if (e.touches.length === 2) {
        this.handlePinchStart(e);
      }
    }

    handleGestureMove(e) {
      if (e.touches.length === 2) {
        this.handlePinchMove(e);
      }
    }

    handleGestureEnd(e) {
      const touch = e.changedTouches[0];
      const pointId = touch.identifier || 'default';
      const startPoint = this.gestureStartPoints.get(pointId);

      if (startPoint) {
        const deltaX = touch.clientX - startPoint.x;
        const deltaY = touch.clientY - startPoint.y;
        const deltaTime = Date.now() - startPoint.time;

        if (Math.abs(deltaX) > GESTURE_CONFIG.dragThreshold || Math.abs(deltaY) > GESTURE_CONFIG.dragThreshold) {
          this.dispatchGestureEvent('dragend', startPoint.target, {
            deltaX: deltaX,
            deltaY: deltaY,
            velocityX: deltaX / deltaTime,
            velocityY: deltaY / deltaTime
          });
        }

        this.gestureStartPoints.delete(pointId);
      }

      if (e.touches.length < 2) {
        this.pinchStartDistance = 0;
      }
    }

    handleGestureCancel(e) {
      this.gestureStartPoints.clear();
      this.pinchStartDistance = 0;
      this.activeGestures.clear();
    }

    handlePinchStart(e) {
      if (e.touches.length === 2) {
        const dx = e.touches[0].clientX - e.touches[1].clientX;
        const dy = e.touches[0].clientY - e.touches[1].clientY;
        this.pinchStartDistance = Math.sqrt(dx * dx + dy * dy);
        this.pinchStartScale = 1;
      }
    }

    handlePinchMove(e) {
      if (e.touches.length === 2 && this.pinchStartDistance > 0) {
        const dx = e.touches[0].clientX - e.touches[1].clientX;
        const dy = e.touches[0].clientY - e.touches[1].clientY;
        const currentDistance = Math.sqrt(dx * dx + dy * dy);

        const scaleChange = (currentDistance - this.pinchStartDistance) / this.pinchStartDistance;

        if (Math.abs(scaleChange) > GESTURE_CONFIG.pinchThreshold) {
          const newScale = Math.max(0.5, Math.min(3, this.pinchStartScale + scaleChange));

          this.dispatchGestureEvent('pinch', document.body, {
            scale: newScale,
            scaleChange: scaleChange,
            center: {
              x: (e.touches[0].clientX + e.touches[1].clientX) / 2,
              y: (e.touches[0].clientY + e.touches[1].clientY) / 2
            }
          });

          this.pinchStartScale = newScale;
        }
      }
    }

    setupSwipeGestures() {
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

          if (Math.abs(deltaX) > GESTURE_CONFIG.dragThreshold || Math.abs(deltaY) > GESTURE_CONFIG.dragThreshold) {
            if (Math.abs(deltaX) < Math.abs(deltaY)) {
              isSwipe = false;
            }
          }
        }
      }, { passive: true });

      document.addEventListener('touchend', (e) => {
        if (isSwipe) {
          const swipeEndX = e.changedTouches[0].clientX;
          const swipeEndY = e.changedTouches[0].clientY;
          const swipeEndTime = Date.now();

          const deltaX = swipeEndX - swipeStartX;
          const deltaY = swipeEndY - swipeStartY;
          const deltaTime = swipeEndTime - swipeStartTime;

          const velocityX = Math.abs(deltaX) / deltaTime;
          const velocityY = Math.abs(deltaY) / deltaTime;

          if (deltaTime < GESTURE_CONFIG.maxSwipeTime) {
            if (Math.abs(deltaX) > GESTURE_CONFIG.minSwipeDistance) {
              const direction = deltaX > 0 ? 'right' : 'left';
              this.triggerHapticFeedback('light');
              this.dispatchGestureEvent('swipe', document.body, {
                direction: direction,
                distance: Math.abs(deltaX),
                velocity: velocityX,
                duration: deltaTime
              });
            }

            if (Math.abs(deltaY) > GESTURE_CONFIG.minSwipeDistance) {
              const direction = deltaY > 0 ? 'down' : 'up';
              this.triggerHapticFeedback('light');
              this.dispatchGestureEvent('swipe', document.body, {
                direction: direction,
                distance: Math.abs(deltaY),
                velocity: velocityY,
                duration: deltaTime
              });
            }
          }

          isSwipe = false;
        }
      }, { passive: true });
    }

    setupPinchZoom() {
      let currentScale = 1;
      let isPinching = false;

      document.addEventListener('gesturestart', (e) => {
        currentScale = e.scale;
        isPinching = true;
      });

      document.addEventListener('gesturechange', (e) => {
        if (isPinching) {
          const scale = Math.max(0.5, Math.min(3, e.scale));
          currentScale = scale;

          this.dispatchGestureEvent('pinch', document.body, {
            scale: scale,
            rotation: e.rotation,
            center: {
              x: e.center ? e.center.x : window.innerWidth / 2,
              y: e.center ? e.center.y : window.innerHeight / 2
            }
          });
        }
      });

      document.addEventListener('gestureend', () => {
        isPinching = false;
      });
    }

    setupDragAndDrop() {
      const draggableElements = document.querySelectorAll('[data-draggable="true"], .captcha-slider-button, .captcha-click-marker');

      draggableElements.forEach(el => {
        el.setAttribute('data-draggable', 'true');

        el.addEventListener('touchstart', (e) => this.handleDragStart(e), { passive: false });
        el.addEventListener('touchmove', (e) => this.handleDragMove(e), { passive: false });
        el.addEventListener('touchend', (e) => this.handleDragEnd(e), { passive: false });
      });
    }

    handleDragStart(e) {
      const target = e.currentTarget;
      const touch = e.touches[0];

      target.classList.add('dragging');

      this.dispatchGestureEvent('dragstart', target, {
        x: touch.clientX,
        y: touch.clientY,
        element: target
      });
    }

    handleDragMove(e) {
      const target = e.currentTarget;
      if (!target.classList.contains('dragging')) {
        return;
      }

      e.preventDefault();
      const touch = e.touches[0];

      this.dispatchGestureEvent('dragmove', target, {
        x: touch.clientX,
        y: touch.clientY,
        deltaX: this.lastDragX ? touch.clientX - this.lastDragX : 0,
        deltaY: this.lastDragY ? touch.clientY - this.lastDragY : 0
      });

      this.lastDragX = touch.clientX;
      this.lastDragY = touch.clientY;
    }

    handleDragEnd(e) {
      const target = e.currentTarget;
      if (!target.classList.contains('dragging')) {
        return;
      }

      target.classList.remove('dragging');

      const touch = e.changedTouches[0];

      this.dispatchGestureEvent('dragend', target, {
        x: touch.clientX,
        y: touch.clientY,
        element: target
      });

      this.lastDragX = null;
      this.lastDragY = null;
    }

    setupDoubleTap() {
      const doubleTapTargets = document.querySelectorAll('.captcha-click-image, .captcha-canvas, img');

      doubleTapTargets.forEach(el => {
        el.addEventListener('touchend', (e) => {
          const now = Date.now();
          const touch = e.changedTouches[0];

          if (now - this.lastTapTime < 300) {
            const deltaX = Math.abs(touch.clientX - this.lastTapPos.x);
            const deltaY = Math.abs(touch.clientY - this.lastTapPos.y);

            if (deltaX < 50 && deltaY < 50) {
              this.handleDoubleTap(el, touch);
              this.lastTapTime = 0;
              return;
            }
          }

          this.lastTapTime = now;
          this.lastTapPos = { x: touch.clientX, y: touch.clientY };
        });
      });
    }

    handleDoubleTap(target, touch) {
      const currentTransform = target.style.transform || '';
      const scaleMatch = currentTransform.match(/scale\(([\d.]+)\)/);
      const currentScale = scaleMatch ? parseFloat(scaleMatch[1]) : 1;

      const newScale = currentScale === 1 ? 2 : 1;

      target.style.transition = 'transform 0.3s ease';
      target.style.transform = `scale(${newScale})`;

      this.dispatchGestureEvent('doubletap', target, {
        x: touch.clientX,
        y: touch.clientY,
        scale: newScale
      });

      setTimeout(() => {
        target.style.transition = '';
      }, 300);
    }

    dispatchGestureEvent(eventType, target, detail) {
      const event = new CustomEvent(eventType, {
        bubbles: true,
        detail: detail
      });
      target.dispatchEvent(event);
    }

    registerGestureHandler(gestureType, handler) {
      this.gestureHandlers.set(gestureType, handler);
      document.addEventListener(gestureType, handler);
    }

    unregisterGestureHandler(gestureType) {
      const handler = this.gestureHandlers.get(gestureType);
      if (handler) {
        document.removeEventListener(gestureType, handler);
        this.gestureHandlers.delete(gestureType);
      }
    }

    injectGestureStyles() {
      if (document.getElementById('gesture-handler-styles')) {
        return;
      }

      const style = document.createElement('style');
      style.id = 'gesture-handler-styles';
      style.textContent = `
        [data-draggable="true"],
        .captcha-slider-button,
        .captcha-click-marker {
          touch-action: none;
          user-select: none;
          -webkit-user-select: none;
          -webkit-touch-callout: none;
        }

        .dragging {
          opacity: 0.8;
          transform: scale(1.1);
          z-index: 1000;
        }

        @media (hover: none) and (pointer: coarse) {
          [data-draggable="true"],
          .captcha-slider-button,
          .captcha-click-marker {
            cursor: grab;
          }

          [data-draggable="true"]:active,
          .captcha-slider-button:active,
          .captcha-click-marker:active {
            cursor: grabbing;
          }
        }

        .swipe-indicator {
          position: fixed;
          pointer-events: none;
          opacity: 0;
          transition: opacity 0.3s ease;
          z-index: 9999;
        }

        .swipe-indicator.show {
          opacity: 1;
        }
      `;

      document.head.appendChild(style);
    }
  }

  const gestureHandler = new GestureHandler();

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => gestureHandler.init());
  } else {
    gestureHandler.init();
  }

  window.GestureHandler = gestureHandler;
})();
