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
            };

            this.activeTouches = new Map();
            this.swipeGestures = [];
            this.tapGestures = [];
            this.longPressTimers = new Map();
            this.lastTapTime = 0;
            this.hapticGenerator = null;

            this.init();
        }

        init() {
            if (typeof document === 'undefined') return;

            this.setupHapticFeedback();
            this.setupTouchListeners();
            this.setupGestureRecognition();
        }

        setupHapticFeedback() {
            if ('vibrate' in navigator) {
                this.hapticGenerator = {
                    light: () => navigator.vibrate(10),
                    medium: () => navigator.vibrate(20),
                    heavy: () => navigator.vibrate(30),
                    success: () => navigator.vibrate([10, 50, 10]),
                    error: () => navigator.vibrate([50, 30, 50]),
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
            document.addEventListener('touchstart', this.handleTouchStart.bind(this), { passive: false });
            document.addEventListener('touchmove', this.handleTouchMove.bind(this), { passive: false });
            document.addEventListener('touchend', this.handleTouchEnd.bind(this), { passive: false });
            document.addEventListener('touchcancel', this.handleTouchCancel.bind(this), { passive: false });

            if (this.config.enableSwipeGesture) {
                this.setupSwipeGestures();
            }
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
        }

        handleTouchMove(event) {
            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                const touchData = this.activeTouches.get(touch.identifier);

                if (touchData) {
                    touchData.currentX = touch.clientX;
                    touchData.currentY = touch.clientY;

                    const deltaX = touch.clientX - touchData.startX;
                    const deltaY = touch.clientY - touchData.startY;

                    if (Math.abs(deltaX) > this.config.swipeThreshold ||
                        Math.abs(deltaY) > this.config.swipeThreshold) {
                        this.clearLongPressTimer(touch.identifier);
                    }
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
                }

                this.clearLongPressTimer(touch.identifier);
            }
        }

        handleTouchCancel(event) {
            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                this.activeTouches.delete(touch.identifier);
                this.clearLongPressTimer(touch.identifier);
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
