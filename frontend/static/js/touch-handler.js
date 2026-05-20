const TouchHandler = (function() {
    'use strict';

    class TouchEventHandler {
        constructor(options = {}) {
            this.config = {
                enableHapticFeedback: options.enableHapticFeedback !== false,
                enableRippleEffect: options.enableRippleEffect !== false,
                touchTargetMinSize: options.touchTargetMinSize || 44,
                debounceDelay: options.debounceDelay || 150,
                preventDoubleTap: options.preventDoubleTap !== false,
                enableSwipeGesture: options.enableSwipeGesture !== false,
                swipeThreshold: options.swipeThreshold || 50,
                longPressDuration: options.longPressDuration || 500,
                enablePinchZoom: options.enablePinchZoom !== false,
                enableRotation: options.enableRotation !== false,
                touchSlop: options.touchSlop || 10,
                velocityThreshold: options.velocityThreshold || 0.3,
                enableMultiTouch: options.enableMultiTouch !== false,
            };

            this.activeTouches = new Map();
            this.swipeGestures = [];
            this.tapGestures = [];
            this.longPressTimers = new Map();
            this.lastTapTime = 0;
            this.hapticGenerator = null;
            this.pinchState = null;
            this.rotationState = null;
            this.touchHistory = [];
            this.maxHistoryLength = 50;
            this.lastInteractionTime = 0;
            this.isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
            this.supportLevel = this.detectTouchSupport();

            this.init();
        }

        detectTouchSupport() {
            const support = {
                touch: 'ontouchstart' in window,
                pointer: window.PointerEvent !== undefined,
                msPointer: window.MSPointerEvent !== undefined,
                maxTouchPoints: navigator.maxTouchPoints || 0,
                touchEvent: typeof TouchEvent !== 'undefined',
                force: false,
                tiltX: false,
                tiltY: false,
                pressure: false,
            };

            if (support.touchEvent) {
                const testTouch = new TouchEvent('touchstart', {
                    touches: [new Touch({ identifier: 0, target: document.body, clientX: 0, clientY: 0 })],
                    bubbles: true,
                });
                support.force = 'force' in testTouch.touches[0];
                support.tiltX = 'tiltX' in testTouch.touches[0];
                support.tiltY = 'tiltY' in testTouch.touches[0];
            }

            return support;
         }

        init() {
            if (typeof document === 'undefined') return;

            this.setupHapticFeedback();
            this.setupTouchListeners();
            this.setupGestureRecognition();
        }
            if ('vibrate' in navigator) {
                this.hapticGenerator = {
                    light: () => navigator.vibrate(10),
                    medium: () => navigator.vibrate(20),
                    heavy: () => navigator.vibrate(30),
                    success: () => navigator.vibrate([10, 50, 10]),
                    error: () => navigator.vibrate([50, 30, 50]),
                    selection: () => navigator.vibrate(5),
                    impact: (style) => {
                        switch (style) {
                            case 'light': navigator.vibrate(10); break;
                            case 'medium': navigator.vibrate(20); break;
                            case 'heavy': navigator.vibrate(30); break;
                            default: navigator.vibrate(15);
                        }
                    },
                };
            } else if ('webkit' in window) {
                this.hapticGenerator = {
                    light: () => {
                        const event = new Event('webkitHapticFeedback');
                        event.level = 0.1;
                        document.dispatchEvent(event);
                    },
                    medium: () => {
                        const event = new Event('webkitHapticFeedback');
                        event.level = 0.5;
                        document.dispatchEvent(event);
                    },
                    heavy: () => {
                        const event = new Event('webkitHapticFeedback');
                        event.level = 1.0;
                        document.dispatchEvent(event);
                    },
                };
            }
        }

        setupTouchListeners() {
            if (this.supportLevel.pointer) {
                this.setupPointerListeners();
            } else if (this.supportLevel.touch) {
                this.setupTouchEventListeners();
            }
            
            document.addEventListener('touchcancel', this.handleTouchCancel.bind(this), { passive: false });

            if (this.config.enableSwipeGesture) {
                this.setupSwipeGestures();
            }

            if (this.config.enablePinchZoom) {
                this.setupPinchZoom();
            }

            if (this.config.enableRotation) {
                this.setupRotation();
            }
        }

        setupPointerListeners() {
            const pointerDown = (e) => this.handlePointerDown(e);
            const pointerMove = (e) => this.handlePointerMove(e);
            const pointerUp = (e) => this.handlePointerUp(e);
            const pointerCancel = (e) => this.handlePointerCancel(e);

            document.addEventListener('pointerdown', pointerDown, { passive: false });
            document.addEventListener('pointermove', pointerMove, { passive: false });
            document.addEventListener('pointerup', pointerUp, { passive: false });
            document.addEventListener('pointercancel', pointerCancel, { passive: false });
        }

        setupTouchEventListeners() {
            document.addEventListener('touchstart', this.handleTouchStart.bind(this), { passive: false });
            document.addEventListener('touchmove', this.handleTouchMove.bind(this), { passive: false });
            document.addEventListener('touchend', this.handleTouchEnd.bind(this), { passive: false });
        }

        handlePointerDown(e) {
            const target = e.target;

            this.activeTouches.set(e.pointerId, {
                startX: e.clientX,
                startY: e.clientY,
                currentX: e.clientX,
                currentY: e.clientY,
                startTime: Date.now(),
                target: target,
                pressure: e.pressure || 0.5,
                tiltX: e.tiltX || 0,
                tiltY: e.tiltY || 0,
            });

            if (this.config.enableRippleEffect) {
                this.createRippleEffect(target, { clientX: e.clientX, clientY: e.clientY });
            }

            if (this.config.enableHapticFeedback && this.hapticGenerator) {
                this.hapticGenerator.light();
            }

            if (this.config.longPressDuration > 0) {
                const timerId = setTimeout(() => {
                    if (this.activeTouches.has(e.pointerId)) {
                        this.triggerLongPress({ clientX: e.clientX, clientY: e.clientY, target: target });
                    }
                }, this.config.longPressDuration);

                this.longPressTimers.set(e.pointerId, timerId);
            }

            this.recordInteraction('down', e);
        }

        handlePointerMove(e) {
            const touchData = this.activeTouches.get(e.pointerId);

            if (touchData) {
                const deltaX = e.clientX - touchData.startX;
                const deltaY = e.clientY - touchData.startY;

                touchData.currentX = e.clientX;
                touchData.currentY = e.clientY;
                touchData.pressure = e.pressure || 0.5;

                if (Math.abs(deltaX) > this.config.swipeThreshold ||
                    Math.abs(deltaY) > this.config.swipeThreshold) {
                    this.clearLongPressTimer(e.pointerId);
                }

                this.recordInteraction('move', e);
            }
        }

        handlePointerUp(e) {
            const touchData = this.activeTouches.get(e.pointerId);

            if (touchData) {
                const deltaX = e.clientX - touchData.startX;
                const deltaY = e.clientY - touchData.startY;
                const duration = Date.now() - touchData.startTime;

                if (Math.abs(deltaX) < 10 && Math.abs(deltaY) < 10 && duration < 300) {
                    if (this.config.preventDoubleTap) {
                        const now = Date.now();
                        if (now - this.lastTapTime < 300) {
                            this.triggerDoubleTap({ clientX: e.clientX, clientY: e.clientY, target: touchData.target }, touchData);
                        } else {
                            this.triggerTap({ clientX: e.clientX, clientY: e.clientY, target: touchData.target }, touchData);
                        }
                        this.lastTapTime = now;
                    } else {
                        this.triggerTap({ clientX: e.clientX, clientY: e.clientY, target: touchData.target }, touchData);
                    }
                }

                this.activeTouches.delete(e.pointerId);
                this.clearLongPressTimer(e.pointerId);
            }

            this.recordInteraction('up', e);
        }

        handlePointerCancel(e) {
            this.activeTouches.delete(e.pointerId);
            this.clearLongPressTimer(e.pointerId);
        }

        recordInteraction(type, event) {
            this.lastInteractionTime = Date.now();
            
            const record = {
                type: type,
                x: event.clientX,
                y: event.clientY,
                timestamp: Date.now(),
                pointerId: event.pointerId || event.identifier || 0,
            };

            this.touchHistory.push(record);
            
            if (this.touchHistory.length > this.maxHistoryLength) {
                this.touchHistory.shift();
            }
        }

        getInteractionAnalysis() {
            const now = Date.now();
            const recentInteractions = this.touchHistory.filter(
                r => now - r.timestamp < 1000
            );

            return {
                totalInteractions: this.touchHistory.length,
                recentInteractions: recentInteractions.length,
                interactionFrequency: recentInteractions.length,
                averageInterval: this.calculateAverageInterval(),
                isHuman: this.analyzeHumanBehavior(),
            };
        }

        calculateAverageInterval() {
            if (this.touchHistory.length < 2) return 0;

            let totalInterval = 0;
            for (let i = 1; i < this.touchHistory.length; i++) {
                totalInterval += this.touchHistory[i].timestamp - this.touchHistory[i - 1].timestamp;
            }

            return totalInterval / (this.touchHistory.length - 1);
        }

        analyzeHumanBehavior() {
            const analysis = this.getInteractionAnalysis();
            
            if (analysis.interactionFrequency > 100) {
                return false;
            }

            if (analysis.averageInterval > 0 && analysis.averageInterval < 5) {
                return false;
            }

            return true;
        }

        setupPinchZoom() {
            if (!this.config.enableMultiTouch) return;

            document.addEventListener('touchstart', (e) => {
                if (e.touches.length === 2) {
                    const touch1 = e.touches[0];
                    const touch2 = e.touches[1];
                    
                    this.pinchState = {
                        startDistance: this.getDistance(touch1.clientX, touch1.clientY, touch2.clientX, touch2.clientY),
                        startScale: 1,
                        currentScale: 1,
                    };
                }
            }, { passive: false });

            document.addEventListener('touchmove', (e) => {
                if (e.touches.length === 2 && this.pinchState) {
                    e.preventDefault();
                    
                    const touch1 = e.touches[0];
                    const touch2 = e.touches[1];
                    const currentDistance = this.getDistance(touch1.clientX, touch1.clientY, touch2.clientX, touch2.clientY);
                    
                    this.pinchState.currentScale = currentDistance / this.pinchState.startDistance;
                    
                    const pinchEvent = new CustomEvent('captcha:pinch', {
                        detail: {
                            scale: this.pinchState.currentScale,
                            startScale: this.pinchState.startScale,
                            center: this.getCenter(touch1.clientX, touch1.clientY, touch2.clientX, touch2.clientY),
                        },
                    });
                    document.dispatchEvent(pinchEvent);
                }
            }, { passive: false });

            document.addEventListener('touchend', (e) => {
                if (e.touches.length < 2 && this.pinchState) {
                    const pinchEndEvent = new CustomEvent('captcha:pinch-end', {
                        detail: {
                            scale: this.pinchState.currentScale,
                            startScale: this.pinchState.startScale,
                        },
                    });
                    document.dispatchEvent(pinchEndEvent);
                    this.pinchState = null;
                }
            });
        }

        setupRotation() {
            if (!this.config.enableMultiTouch) return;

            let initialRotation = 0;

            document.addEventListener('touchstart', (e) => {
                if (e.touches.length === 2) {
                    const touch1 = e.touches[0];
                    const touch2 = e.touches[1];
                    
                    initialRotation = this.getAngle(touch1.clientX, touch1.clientY, touch2.clientX, touch2.clientY);
                    
                    this.rotationState = {
                        startRotation: initialRotation,
                        currentRotation: 0,
                    };
                }
            }, { passive: false });

            document.addEventListener('touchmove', (e) => {
                if (e.touches.length === 2 && this.rotationState) {
                    e.preventDefault();
                    
                    const touch1 = e.touches[0];
                    const touch2 = e.touches[1];
                    const currentAngle = this.getAngle(touch1.clientX, touch1.clientY, touch2.clientX, touch2.clientY);
                    
                    this.rotationState.currentRotation = currentAngle - initialRotation;
                    
                    const rotationEvent = new CustomEvent('captcha:rotate', {
                        detail: {
                            rotation: this.rotationState.currentRotation,
                            startRotation: this.rotationState.startRotation,
                            center: this.getCenter(touch1.clientX, touch1.clientY, touch2.clientX, touch2.clientY),
                        },
                    });
                    document.dispatchEvent(rotationEvent);
                }
            }, { passive: false });

            document.addEventListener('touchend', (e) => {
                if (e.touches.length < 2 && this.rotationState) {
                    const rotationEndEvent = new CustomEvent('captcha:rotate-end', {
                        detail: {
                            rotation: this.rotationState.currentRotation,
                            startRotation: this.rotationState.startRotation,
                        },
                    });
                    document.dispatchEvent(rotationEndEvent);
                    this.rotationState = null;
                }
            });
        }

        getDistance(x1, y1, x2, y2) {
            return Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2));
        }

        getCenter(x1, y1, x2, y2) {
            return {
                x: (x1 + x2) / 2,
                y: (y1 + y2) / 2,
            };
        }

        getAngle(x1, y1, x2, y2) {
            return Math.atan2(y2 - y1, x2 - x1) * 180 / Math.PI;
        }

        handleTouchStart(event) {
            const touch = event.touches[0];
            const target = event.target;

            this.activeTouches.set(touch.identifier, {
                startX: touch.clientX,
                startY: touch.clientY,
                currentX: touch.clientX,
                currentY: touch.clientY,
                startTime: Date.now(),
                target: target,
                pressure: touch.force || 0.5,
                tiltX: touch.tiltX || 0,
                tiltY: touch.tiltY || 0,
            });

            if (this.config.enableRippleEffect) {
                this.createRippleEffect(target, touch);
            }

            if (this.config.enableHapticFeedback && this.hapticGenerator) {
                this.hapticGenerator.light();
            }

            if (this.config.longPressDuration > 0) {
                const timerId = setTimeout(() => {
                    if (this.activeTouches.has(touch.identifier)) {
                        this.triggerLongPress(touch);
                    }
                }, this.config.longPressDuration);

                this.longPressTimers.set(touch.identifier, timerId);
            }

            this.recordInteraction('down', touch);
        }

        handleTouchMove(event) {
            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                const touchData = this.activeTouches.get(touch.identifier);

                if (touchData) {
                    touchData.currentX = touch.clientX;
                    touchData.currentY = touch.clientY;
                    touchData.pressure = touch.force || 0.5;

                    const deltaX = touch.clientX - touchData.startX;
                    const deltaY = touch.clientY - touchData.startY;

                    if (Math.abs(deltaX) > this.config.swipeThreshold ||
                        Math.abs(deltaY) > this.config.swipeThreshold) {
                        this.clearLongPressTimer(touch.identifier);
                    }

                    this.recordInteraction('move', touch);
                }
            }
        }

        handleTouchEnd(event) {
            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                const touchData = this.activeTouches.get(touch.identifier);

                if (touchData) {
                    const deltaX = touch.clientX - touchData.startX;
                    const deltaY = touch.clientY - touchData.startY;
                    const duration = Date.now() - touchData.startTime;

                    if (Math.abs(deltaX) < 10 && Math.abs(deltaY) < 10 && duration < 300) {
                        if (this.config.preventDoubleTap) {
                            const now = Date.now();
                            if (now - this.lastTapTime < 300) {
                                this.triggerDoubleTap(touch, touchData);
                            } else {
                                this.triggerTap(touch, touchData);
                            }
                            this.lastTapTime = now;
                        } else {
                            this.triggerTap(touch, touchData);
                        }
                    }

                    this.activeTouches.delete(touch.identifier);
                    this.recordInteraction('up', touch);
                }

                this.clearLongPressTimer(touch.identifier);
            }
        }

        handleTouchCancel(event) {
            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                this.activeTouches.delete(touch.identifier);
                this.clearLongPressTimer(touch.identifier);
                this.recordInteraction('cancel', touch);
            }
        }

        clearLongPressTimer(touchId) {
            if (this.longPressTimers.has(touchId)) {
                clearTimeout(this.longPressTimers.get(touchId));
                this.longPressTimers.delete(touchId);
            }
        }

        triggerTap(touch, touchData) {
            const tapEvent = new CustomEvent('captcha:tap', {
                detail: {
                    x: touch.clientX,
                    y: touch.clientY,
                    target: touchData.target,
                    timestamp: Date.now(),
                },
            });
            document.dispatchEvent(tapEvent);

            if (this.config.enableHapticFeedback && this.hapticGenerator) {
                this.hapticGenerator.light();
            }
        }

        triggerDoubleTap(touch, touchData) {
            const doubleTapEvent = new CustomEvent('captcha:double-tap', {
                detail: {
                    x: touch.clientX,
                    y: touch.clientY,
                    target: touchData.target,
                    timestamp: Date.now(),
                },
            });
            document.dispatchEvent(doubleTapEvent);

            if (this.config.enableHapticFeedback && this.hapticGenerator) {
                this.hapticGenerator.medium();
            }
        }

        triggerLongPress(touch) {
            const longPressEvent = new CustomEvent('captcha:long-press', {
                detail: {
                    x: touch.clientX,
                    y: touch.clientY,
                    target: touch.target,
                    timestamp: Date.now(),
                },
            });
            document.dispatchEvent(longPressEvent);

            if (this.config.enableHapticFeedback && this.hapticGenerator) {
                this.hapticGenerator.heavy();
            }
        }

        createRippleEffect(element, touch) {
            if (!element || !element.getBoundingClientRect) return;

            const rect = element.getBoundingClientRect();
            const ripple = document.createElement('span');
            ripple.className = 'captcha-ripple';
            ripple.style.cssText = `
                position: absolute;
                border-radius: 50%;
                background: rgba(255, 255, 255, 0.3);
                transform: scale(0);
                animation: ripple-effect 0.6s linear;
                pointer-events: none;
                left: ${touch.clientX - rect.left}px;
                top: ${touch.clientY - rect.top}px;
                width: 100px;
                height: 100px;
                margin-left: -50px;
                margin-top: -50px;
            `;

            element.style.position = element.style.position || 'relative';
            element.style.overflow = 'hidden';
            element.appendChild(ripple);

            setTimeout(() => {
                ripple.remove();
            }, 600);
        }

        setupSwipeGestures() {
            const swipeEvent = new CustomEvent('captcha:swipe', {
                detail: { direction: 'none' },
            });

            document.addEventListener('touchend', (event) => {
                for (let i = 0; i < event.changedTouches.length; i++) {
                    const touch = event.changedTouches[i];
                    const touchData = this.activeTouches.get(touch.identifier);

                    if (touchData) {
                        const deltaX = touch.clientX - touchData.startX;
                        const deltaY = touch.clientY - touchData.startY;

                        let direction = 'none';
                        if (Math.abs(deltaX) > Math.abs(deltaY)) {
                            direction = deltaX > 0 ? 'right' : 'left';
                        } else {
                            direction = deltaY > 0 ? 'down' : 'up';
                        }

                        if (Math.abs(deltaX) > this.config.swipeThreshold ||
                            Math.abs(deltaY) > this.config.swipeThreshold) {
                            const swipeDetail = {
                                direction: direction,
                                deltaX: deltaX,
                                deltaY: deltaY,
                                velocity: Math.sqrt(deltaX * deltaX + deltaY * deltaY) /
                                         (Date.now() - touchData.startTime),
                            };

                            const swipeEvent = new CustomEvent('captcha:swipe', {
                                detail: swipeDetail,
                            });
                            document.dispatchEvent(swipeEvent);
                        }
                    }
                }
            });
        }

        setupGestureRecognition() {
            this.tapGestures = [];
        }

        addSwipeGesture(callback) {
            this.swipeGestures.push(callback);
        }

        removeSwipeGesture(callback) {
            const index = this.swipeGestures.indexOf(callback);
            if (index > -1) {
                this.swipeGestures.splice(index, 1);
            }
        }

        validateTouchTarget(element) {
            const rect = element.getBoundingClientRect();
            const minSize = this.config.touchTargetMinSize;

            return rect.width >= minSize && rect.height >= minSize;
        }

        adjustSmallTouchTargets() {
            const interactiveElements = document.querySelectorAll(
                'button, a, [role="button"], input[type="submit"], input[type="button"]'
            );

            interactiveElements.forEach(element => {
                if (!this.validateTouchTarget(element)) {
                    const rect = element.getBoundingClientRect();
                    if (rect.width < this.config.touchTargetMinSize) {
                        element.style.minWidth = this.config.touchTargetMinSize + 'px';
                    }
                    if (rect.height < this.config.touchTargetMinSize) {
                        element.style.minHeight = this.config.touchTargetMinSize + 'px';
                    }
                }
            });
        }

        destroy() {
            document.removeEventListener('touchstart', this.handleTouchStart.bind(this));
            document.removeEventListener('touchmove', this.handleTouchMove.bind(this));
            document.removeEventListener('touchend', this.handleTouchEnd.bind(this));
            document.removeEventListener('touchcancel', this.handleTouchCancel.bind(this));

            this.activeTouches.clear();
            this.longPressTimers.forEach(timer => clearTimeout(timer));
            this.longPressTimers.clear();
            this.swipeGestures = [];
            this.tapGestures = [];
        }
    }

    let instance = null;

    return {
        init: function(options) {
            if (!instance) {
                instance = new TouchEventHandler(options);
            }
            return instance;
        },

        getInstance: function() {
            return instance;
        },

        destroy: function() {
            if (instance) {
                instance.destroy();
                instance = null;
            }
        },
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = TouchHandler;
}

if (typeof window !== 'undefined') {
    window.TouchHandler = TouchHandler;
}
