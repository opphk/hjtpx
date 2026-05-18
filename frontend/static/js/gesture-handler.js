(function() {
  'use strict';

  const GESTURE_CONFIG = {
    minSwipeDistance: 30,
    maxSwipeTime: 1000,
    rotationSensitivity: 0.1,
    pinchThreshold: 0.02,
    dragThreshold: 5,
    velocityThreshold: 0.5,
    longPressDelay: 500,
    doubleTapDelay: 250,
    tapRadius: 30,
    tripleTapDelay: 400,
    threeFingerThreshold: 3
  };

  class GestureHandler {
    constructor() {
      this.gestureStartPoints = new Map();
      this.activeGestures = new Map();
      this.gestureHandlers = new Map();
      this.pinchStartDistance = 0;
      this.pinchStartScale = 1;
      this.pinchStartAngle = 0;
      this.rotationStartAngle = 0;
      this.lastTapTime = 0;
      this.tapCount = 0;
      this.tapTimeout = null;
      this.lastTapPos = { x: 0, y: 0 };
      this.longPressTimer = null;
      this.currentScale = 1;
      this.currentRotation = 0;
      this.isDragging = false;
      this.isPinching = false;
      this.lastDragX = null;
      this.lastDragY = null;
      this.velocityX = 0;
      this.velocityY = 0;
      this.lastMoveTime = 0;
      this.touchHistory = [];
    }

    init() {
      console.log('[Gesture] 初始化手势处理模块 v2');
      this.setupTouchGestures();
      this.setupSwipeGestures();
      this.setupPinchZoom();
      this.setupDragAndDrop();
      this.setupDoubleTap();
      this.setupTripleTap();
      this.setupLongPress();
      this.setupRotation();
      this.setupThreeFingerGestures();
      this.injectGestureStyles();
    }

    setupTouchGestures() {
      document.addEventListener('touchstart', (e) => this.handleGestureStart(e), { passive: false });
      document.addEventListener('touchmove', (e) => this.handleGestureMove(e), { passive: false });
      document.addEventListener('touchend', (e) => this.handleGestureEnd(e), { passive: false });
      document.addEventListener('touchcancel', (e) => this.handleGestureCancel(e), { passive: false });
    }

    handleGestureStart(e) {
      const now = Date.now();
      
      this.touchHistory.push({
        time: now,
        touches: Array.from(e.touches).map(t => ({ x: t.clientX, y: t.clientY, id: t.identifier }))
      });

      if (this.touchHistory.length > 50) {
        this.touchHistory.shift();
      }

      e.touches.forEach(touch => {
        const pointId = touch.identifier || 'default';
        this.gestureStartPoints.set(pointId, {
          x: touch.clientX,
          y: touch.clientY,
          time: now,
          target: e.currentTarget,
          initialDistance: this.gestureStartPoints.size > 0 ? this.calculateDistance() : 0
        });
      });

      if (e.touches.length === 2) {
        this.handlePinchStart(e);
        this.handleRotationStart(e);
      }

      if (e.touches.length === 3) {
        this.handleThreeFingerStart(e);
      }

      this.startLongPressDetection(e);
    }

    handleGestureMove(e) {
      const now = Date.now();
      const deltaTime = this.lastMoveTime ? now - this.lastMoveTime : 16;
      this.lastMoveTime = now;

      const primaryTouch = e.touches[0];
      if (this.lastDragX !== null && this.lastDragY !== null) {
        this.velocityX = (primaryTouch.clientX - this.lastDragX) / deltaTime;
        this.velocityY = (primaryTouch.clientY - this.lastDragY) / deltaTime;
      }
      this.lastDragX = primaryTouch.clientX;
      this.lastDragY = primaryTouch.clientY;

      if (this.longPressTimer) {
        const startPoint = this.gestureStartPoints.get(primaryTouch.identifier);
        if (startPoint) {
          const deltaX = Math.abs(primaryTouch.clientX - startPoint.x);
          const deltaY = Math.abs(primaryTouch.clientY - startPoint.y);
          if (deltaX > 15 || deltaY > 15) {
            this.cancelLongPress();
          }
        }
      }

      if (e.touches.length === 2) {
        this.handlePinchMove(e);
        this.handleRotationMove(e);
      }

      if (e.touches.length === 3) {
        this.handleThreeFingerMove(e);
      }

      if (this.isDragging && e.touches.length === 1) {
        this.dispatchContinuousDragEvent(e);
      }
    }

    handleGestureEnd(e) {
      const now = Date.now();
      e.changedTouches.forEach(touch => {
        const pointId = touch.identifier || 'default';
        const startPoint = this.gestureStartPoints.get(pointId);

        if (startPoint) {
          const deltaX = touch.clientX - startPoint.x;
          const deltaY = touch.clientY - startPoint.y;
          const deltaTime = now - startPoint.time;

          if (deltaTime < 250 && Math.abs(deltaX) < GESTURE_CONFIG.tapRadius && Math.abs(deltaY) < GESTURE_CONFIG.tapRadius) {
            this.handleTap(touch);
          }

          if (Math.abs(deltaX) > GESTURE_CONFIG.dragThreshold || Math.abs(deltaY) > GESTURE_CONFIG.dragThreshold) {
            this.dispatchGestureEvent('dragend', startPoint.target, {
              deltaX: deltaX,
              deltaY: deltaY,
              velocityX: this.velocityX,
              velocityY: this.velocityY,
              duration: deltaTime
            });
          }

          this.gestureStartPoints.delete(pointId);
        }
      });

      this.cancelLongPress();

      if (e.touches.length < 2) {
        this.pinchStartDistance = 0;
        this.isPinching = false;
        this.dispatchGestureEvent('pinchend', document.body, {
          scale: this.currentScale,
          rotation: this.currentRotation
        });
      }

      if (e.touches.length < 3) {
        this.activeGestures.delete('threeFinger');
      }

      this.lastDragX = null;
      this.lastDragY = null;
      this.isDragging = false;
    }

    handleGestureCancel(e) {
      this.gestureStartPoints.clear();
      this.pinchStartDistance = 0;
      this.activeGestures.clear();
      this.cancelLongPress();
      this.isDragging = false;
    }

    calculateDistance() {
      const points = Array.from(this.gestureStartPoints.values());
      if (points.length < 2) return 0;
      const dx = points[0].x - points[1].x;
      const dy = points[0].y - points[1].y;
      return Math.sqrt(dx * dx + dy * dy);
    }

    handlePinchStart(e) {
      if (e.touches.length === 2) {
        const dx = e.touches[0].clientX - e.touches[1].clientX;
        const dy = e.touches[0].clientY - e.touches[1].clientY;
        this.pinchStartDistance = Math.sqrt(dx * dx + dy * dy);
        this.pinchStartScale = this.currentScale;
        
        const angle = Math.atan2(e.touches[1].clientY - e.touches[0].clientY, 
                                  e.touches[1].clientX - e.touches[0].clientX);
        this.pinchStartAngle = angle;
        this.isPinching = true;

        this.dispatchGestureEvent('pinchstart', document.body, {
          startDistance: this.pinchStartDistance,
          center: this.getCenterPoint(e.touches)
        });
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
          this.currentScale = newScale;

          const center = this.getCenterPoint(e.touches);

          this.dispatchGestureEvent('pinch', document.body, {
            scale: newScale,
            scaleChange: scaleChange,
            center: center,
            velocity: scaleChange / (Date.now() - this.lastMoveTime)
          });

          this.pinchStartScale = newScale;
        }
      }
    }

    handleRotationStart(e) {
      if (e.touches.length === 2) {
        const angle = Math.atan2(e.touches[1].clientY - e.touches[0].clientY,
                                  e.touches[1].clientX - e.touches[0].clientX);
        this.rotationStartAngle = angle;
        this.currentRotation = 0;
      }
    }

    handleRotationMove(e) {
      if (e.touches.length === 2 && this.pinchStartDistance > 0) {
        const currentAngle = Math.atan2(e.touches[1].clientY - e.touches[0].clientY,
                                         e.touches[1].clientX - e.touches[0].clientX);
        let rotationDelta = currentAngle - this.rotationStartAngle;
        
        rotationDelta = ((rotationDelta + Math.PI) % (Math.PI * 2)) - Math.PI;
        
        this.currentRotation += rotationDelta * (180 / Math.PI);

        this.dispatchGestureEvent('rotate', document.body, {
          rotation: this.currentRotation,
          rotationDelta: rotationDelta * (180 / Math.PI),
          center: this.getCenterPoint(e.touches)
        });

        this.rotationStartAngle = currentAngle;
      }
    }

    getCenterPoint(touches) {
      if (touches.length >= 2) {
        return {
          x: (touches[0].clientX + touches[1].clientX) / 2,
          y: (touches[0].clientY + touches[1].clientY) / 2
        };
      }
      return { x: window.innerWidth / 2, y: window.innerHeight / 2 };
    }

    handleThreeFingerStart(e) {
      if (e.touches.length === 3) {
        this.activeGestures.set('threeFinger', {
          startTime: Date.now(),
          initialPoints: Array.from(e.touches).map(t => ({ x: t.clientX, y: t.clientY }))
        });
        this.dispatchGestureEvent('threefingerstart', document.body, {
          points: Array.from(e.touches).map(t => ({ x: t.clientX, y: t.clientY }))
        });
      }
    }

    handleThreeFingerMove(e) {
      if (e.touches.length === 3) {
        const gesture = this.activeGestures.get('threeFinger');
        if (gesture) {
          const center = {
            x: (e.touches[0].clientX + e.touches[1].clientX + e.touches[2].clientX) / 3,
            y: (e.touches[0].clientY + e.touches[1].clientY + e.touches[2].clientY) / 3
          };

          const avgMoveX = (e.touches[0].clientX + e.touches[1].clientX + e.touches[2].clientX) / 3 -
                          (gesture.initialPoints[0].x + gesture.initialPoints[1].x + gesture.initialPoints[2].x) / 3;
          const avgMoveY = (e.touches[0].clientY + e.touches[1].clientY + e.touches[2].clientY) / 3 -
                          (gesture.initialPoints[0].y + gesture.initialPoints[1].y + gesture.initialPoints[2].y) / 3;

          this.dispatchGestureEvent('threefingermove', document.body, {
            center: center,
            deltaX: avgMoveX,
            deltaY: avgMoveY,
            duration: Date.now() - gesture.startTime
          });
        }
      }
    }

    startLongPressDetection(e) {
      const touch = e.touches[0];
      const target = e.currentTarget;

      this.longPressTimer = setTimeout(() => {
        if (this.gestureStartPoints.has(touch.identifier)) {
          target.classList.add('long-press-active');
          this.triggerHapticFeedback('medium');
          this.dispatchGestureEvent('longpress', target, {
            x: touch.clientX,
            y: touch.clientY,
            duration: GESTURE_CONFIG.longPressDelay
          });
        }
      }, GESTURE_CONFIG.longPressDelay);
    }

    cancelLongPress() {
      if (this.longPressTimer) {
        clearTimeout(this.longPressTimer);
        this.longPressTimer = null;
      }
      document.querySelectorAll('.long-press-active').forEach(el => el.classList.remove('long-press-active'));
    }

    handleTap(touch) {
      const now = Date.now();
      const deltaTime = now - this.lastTapTime;
      const deltaX = Math.abs(touch.clientX - this.lastTapPos.x);
      const deltaY = Math.abs(touch.clientY - this.lastTapPos.y);

      if (deltaTime < GESTURE_CONFIG.doubleTapDelay && deltaX < GESTURE_CONFIG.tapRadius && deltaY < GESTURE_CONFIG.tapRadius) {
        this.tapCount++;
        
        if (this.tapCount === 2) {
          if (this.tapTimeout) {
            clearTimeout(this.tapTimeout);
            this.tapTimeout = null;
          }
          
          this.dispatchGestureEvent('doubletap', document.body, {
            x: touch.clientX,
            y: touch.clientY,
            timeBetweenTaps: deltaTime
          });
          this.triggerHapticFeedback('medium');
          
          this.tapTimeout = setTimeout(() => {
            this.tapCount = 0;
            this.tapTimeout = null;
          }, GESTURE_CONFIG.tripleTapDelay);
        } else if (this.tapCount === 3) {
          this.dispatchGestureEvent('tripletap', document.body, {
            x: touch.clientX,
            y: touch.clientY,
            timeBetweenTaps: deltaTime
          });
          this.triggerHapticFeedback('heavy');
          this.tapCount = 0;
        }
      } else {
        this.tapCount = 1;
        if (this.tapTimeout) {
          clearTimeout(this.tapTimeout);
        }
        this.tapTimeout = setTimeout(() => {
          this.dispatchGestureEvent('singletap', document.body, {
            x: touch.clientX,
            y: touch.clientY
          });
          this.triggerHapticFeedback('light');
          this.tapCount = 0;
          this.tapTimeout = null;
        }, GESTURE_CONFIG.doubleTapDelay);
      }

      this.lastTapTime = now;
      this.lastTapPos = { x: touch.clientX, y: touch.clientY };
    }

    setupSwipeGestures() {
      let swipeStartX = 0;
      let swipeStartY = 0;
      let swipeStartTime = 0;
      let isSwipe = false;
      let startTarget = null;

      document.addEventListener('touchstart', (e) => {
        if (e.touches.length === 1) {
          swipeStartX = e.touches[0].clientX;
          swipeStartY = e.touches[0].clientY;
          swipeStartTime = Date.now();
          isSwipe = true;
          startTarget = e.target;
        }
      }, { passive: true });

      document.addEventListener('touchmove', (e) => {
        if (isSwipe && e.touches.length === 1) {
          const deltaX = Math.abs(e.touches[0].clientX - swipeStartX);
          const deltaY = Math.abs(e.touches[0].clientY - swipeStartY);

          if (deltaX < 5 && deltaY < 5) {
            return;
          }

          const velocityX = deltaX / (Date.now() - swipeStartTime);
          const velocityY = deltaY / (Date.now() - swipeStartTime);

          if (velocityX > GESTURE_CONFIG.velocityThreshold || velocityY > GESTURE_CONFIG.velocityThreshold) {
            const direction = deltaX > deltaY ? 
              (e.touches[0].clientX > swipeStartX ? 'right' : 'left') :
              (e.touches[0].clientY > swipeStartY ? 'down' : 'up');

            this.dispatchGestureEvent('swipeprogress', startTarget, {
              direction: direction,
              distance: Math.max(deltaX, deltaY),
              velocity: Math.max(velocityX, velocityY)
            });
          }
        }
      }, { passive: true });

      document.addEventListener('touchend', (e) => {
        if (isSwipe && e.changedTouches.length === 1) {
          const swipeEndX = e.changedTouches[0].clientX;
          const swipeEndY = e.changedTouches[0].clientY;
          const swipeEndTime = Date.now();

          const deltaX = swipeEndX - swipeStartX;
          const deltaY = swipeEndY - swipeStartY;
          const deltaTime = swipeEndTime - swipeStartTime;

          const velocityX = Math.abs(deltaX) / deltaTime;
          const velocityY = Math.abs(deltaY) / deltaTime;

          if (deltaTime < GESTURE_CONFIG.maxSwipeTime) {
            if (Math.abs(deltaX) > GESTURE_CONFIG.minSwipeDistance && velocityX > GESTURE_CONFIG.velocityThreshold / 2) {
              const direction = deltaX > 0 ? 'right' : 'left';
              this.dispatchGestureEvent('swipe', startTarget, {
                direction: direction,
                distance: Math.abs(deltaX),
                velocity: velocityX,
                duration: deltaTime,
                x: swipeEndX,
                y: swipeEndY
              });
              this.triggerHapticFeedback('light');
            }

            if (Math.abs(deltaY) > GESTURE_CONFIG.minSwipeDistance && velocityY > GESTURE_CONFIG.velocityThreshold / 2) {
              const direction = deltaY > 0 ? 'down' : 'up';
              this.dispatchGestureEvent('swipe', startTarget, {
                direction: direction,
                distance: Math.abs(deltaY),
                velocity: velocityY,
                duration: deltaTime,
                x: swipeEndX,
                y: swipeEndY
              });
              this.triggerHapticFeedback('light');
            }
          }

          isSwipe = false;
          startTarget = null;
        }
      }, { passive: true });
    }

    setupPinchZoom() {
      document.addEventListener('gesturestart', (e) => {
        this.currentScale = e.scale;
        this.isPinching = true;
        this.dispatchGestureEvent('pinchstart', document.body, {
          scale: e.scale,
          rotation: e.rotation,
          center: { x: e.center?.x || window.innerWidth / 2, y: e.center?.y || window.innerHeight / 2 }
        });
      });

      document.addEventListener('gesturechange', (e) => {
        if (this.isPinching) {
          const scale = Math.max(0.5, Math.min(3, e.scale));
          this.currentScale = scale;

          this.dispatchGestureEvent('pinch', document.body, {
            scale: scale,
            rotation: e.rotation,
            center: { x: e.center?.x || window.innerWidth / 2, y: e.center?.y || window.innerHeight / 2 },
            scaleChange: e.scale - this.currentScale
          });
        }
      });

      document.addEventListener('gestureend', () => {
        this.isPinching = false;
        this.dispatchGestureEvent('pinchend', document.body, {
          scale: this.currentScale,
          rotation: this.currentRotation
        });
      });
    }

    setupDragAndDrop() {
      const draggableElements = document.querySelectorAll('[data-draggable="true"], .captcha-slider-button, .captcha-click-marker, [draggable="true"]');

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
      this.isDragging = true;

      const rect = target.getBoundingClientRect();
      const offsetX = touch.clientX - rect.left;
      const offsetY = touch.clientY - rect.top;

      this.dispatchGestureEvent('dragstart', target, {
        x: touch.clientX,
        y: touch.clientY,
        element: target,
        offsetX: offsetX,
        offsetY: offsetY,
        startTime: Date.now()
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
        deltaY: this.lastDragY ? touch.clientY - this.lastDragY : 0,
        velocityX: this.velocityX,
        velocityY: this.velocityY
      });

      this.lastDragX = touch.clientX;
      this.lastDragY = touch.clientY;
    }

    dispatchContinuousDragEvent(e) {
      const touch = e.touches[0];
      const target = document.elementFromPoint(touch.clientX, touch.clientY);
      
      if (target) {
        this.dispatchGestureEvent('dragcontinuous', target, {
          x: touch.clientX,
          y: touch.clientY,
          velocityX: this.velocityX,
          velocityY: this.velocityY
        });
      }
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
        element: target,
        velocityX: this.velocityX,
        velocityY: this.velocityY
      });

      this.isDragging = false;
      this.lastDragX = null;
      this.lastDragY = null;
    }

    setupDoubleTap() {
      const doubleTapTargets = document.querySelectorAll('.captcha-click-image, .captcha-canvas, img, [data-doubletap]');

      doubleTapTargets.forEach(el => {
        el.addEventListener('touchend', (e) => {
          const now = Date.now();
          const touch = e.changedTouches[0];

          if (now - this.lastTapTime < GESTURE_CONFIG.doubleTapDelay) {
            const deltaX = Math.abs(touch.clientX - this.lastTapPos.x);
            const deltaY = Math.abs(touch.clientY - this.lastTapPos.y);

            if (deltaX < GESTURE_CONFIG.tapRadius && deltaY < GESTURE_CONFIG.tapRadius) {
              this.handleDoubleTapZoom(el, touch);
              this.lastTapTime = 0;
              return;
            }
          }

          this.lastTapTime = now;
          this.lastTapPos = { x: touch.clientX, y: touch.clientY };
        });
      });
    }

    setupTripleTap() {
      document.addEventListener('tripletap', (e) => {
        const target = e.target;
        if (target.classList.contains('captcha-canvas') || target.classList.contains('captcha-click-image')) {
          target.style.transform = 'scale(1)';
          this.currentScale = 1;
          this.dispatchGestureEvent('zoomreset', target, {});
        }
      });
    }

    handleDoubleTapZoom(target, touch) {
      const currentTransform = target.style.transform || '';
      const scaleMatch = currentTransform.match(/scale\(([\d.]+)\)/);
      const currentScale = scaleMatch ? parseFloat(scaleMatch[1]) : 1;

      const newScale = currentScale === 1 ? 2 : (currentScale === 2 ? 3 : 1);

      target.style.transition = 'transform 0.3s cubic-bezier(0.4, 0, 0.2, 1)';
      target.style.transform = `scale(${newScale})`;

      this.currentScale = newScale;

      this.dispatchGestureEvent('doubletap', target, {
        x: touch.clientX,
        y: touch.clientY,
        scale: newScale,
        target: target
      });

      setTimeout(() => {
        target.style.transition = '';
      }, 300);
    }

    setupLongPress() {
      const longPressTargets = document.querySelectorAll('[data-longpress], .captcha-slider-button, .captcha-click-marker, .btn');

      longPressTargets.forEach(el => {
        el.addEventListener('longpress', (e) => {
          console.log('[Gesture] 长按事件:', e.detail);
        });
      });
    }

    setupRotation() {
      document.addEventListener('rotate', (e) => {
        const target = document.querySelector('.captcha-canvas, .captcha-image');
        if (target && e.detail && e.detail.rotation !== undefined) {
          target.style.transform = `rotate(${e.detail.rotation}deg) scale(${this.currentScale})`;
        }
      });
    }

    dispatchGestureEvent(eventType, target, detail) {
      const event = new CustomEvent(eventType, {
        bubbles: true,
        cancelable: true,
        detail: detail
      });
      target.dispatchEvent(event);

      const handler = this.gestureHandlers.get(eventType);
      if (handler) {
        handler(event);
      }
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
          opacity: 0.85;
          transform: scale(1.05);
          z-index: 1000;
          cursor: grabbing;
          transition: transform 0.1s ease;
        }

        .dragging:active {
          cursor: grabbing;
        }

        .long-press-active {
          background-color: rgba(201, 169, 110, 0.3) !important;
          transform: scale(0.98);
        }

        .touch-highlight {
          box-shadow: 0 0 20px rgba(201, 169, 110, 0.4);
        }

        @media (hover: none) and (pointer: coarse) {
          [data-draggable="true"],
          .captcha-slider-button,
          .captcha-click-marker {
            cursor: grab;
            min-height: 44px;
            min-width: 44px;
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
          border-radius: 50%;
          background: rgba(201, 169, 110, 0.3);
        }

        .swipe-indicator.show {
          opacity: 1;
        }

        .gesture-feedback {
          position: fixed;
          pointer-events: none;
          border-radius: 50%;
          background: rgba(201, 169, 110, 0.2);
          animation: gestureRipple 0.4s ease-out forwards;
          z-index: 9999;
        }

        @keyframes gestureRipple {
          0% {
            transform: scale(0);
            opacity: 1;
          }
          100% {
            transform: scale(2);
            opacity: 0;
          }
        }

        .scale-animation {
          transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .captcha-canvas,
        .captcha-click-image {
          transform-origin: center center;
        }

        .three-finger-gesture {
          animation: pulse 1s infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.6; }
        }
      `;

      document.head.appendChild(style);
    }

    getGestureState() {
      return {
        isDragging: this.isDragging,
        isPinching: this.isPinching,
        currentScale: this.currentScale,
        currentRotation: this.currentRotation,
        activeGestures: Array.from(this.activeGestures.keys()),
        velocityX: this.velocityX,
        velocityY: this.velocityY
      };
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