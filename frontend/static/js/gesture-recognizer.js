const GestureRecognizer = (function() {
    'use strict';

    class GestureState {
        constructor(type) {
            this.type = type;
            this.startTime = 0;
            this.startX = 0;
            this.startY = 0;
            this.currentX = 0;
            this.currentY = 0;
            this.velocityX = 0;
            this.velocityY = 0;
            this.scale = 1;
            this.rotation = 0;
            this.isActive = false;
        }

        update(x, y, timestamp) {
            const dt = timestamp - this.startTime;
            if (dt > 0) {
                this.velocityX = (x - this.currentX) / dt;
                this.velocityY = (y - this.currentY) / dt;
            }

            this.currentX = x;
            this.currentY = y;
        }

        getDeltaX() {
            return this.currentX - this.startX;
        }

        getDeltaY() {
            return this.currentY - this.startY;
        }

        getDuration() {
            return Date.now() - this.startTime;
        }

        reset() {
            this.startTime = 0;
            this.startX = 0;
            this.startY = 0;
            this.currentX = 0;
            this.currentY = 0;
            this.velocityX = 0;
            this.velocityY = 0;
            this.scale = 1;
            this.rotation = 0;
            this.isActive = false;
        }
    }

    class MultiTouchGestureRecognizer {
        constructor(element, options = {}) {
            this.element = element;
            this.options = {
                minPinchScale: options.minPinchScale || 1.2,
                minRotation: options.minRotation || 15,
                doubleTapDelay: options.doubleTapDelay || 300,
                swipeThreshold: options.swipeThreshold || 50,
                swipeVelocityThreshold: options.swipeVelocityThreshold || 0.5,
            };

            this.pinchState = new GestureState('pinch');
            this.rotationState = new GestureState('rotation');
            this.doubleTapState = null;
            this.lastTapTime = 0;
            this.lastTapPosition = { x: 0, y: 0 };

            this.callbacks = {
                onPinchStart: null,
                onPinchMove: null,
                onPinchEnd: null,
                onRotationStart: null,
                onRotationMove: null,
                onRotationEnd: null,
                onDoubleTap: null,
            };

            this.touches = new Map();
            this.init();
        }

        init() {
            this.element.addEventListener('touchstart', this.handleTouchStart.bind(this), { passive: false });
            this.element.addEventListener('touchmove', this.handleTouchMove.bind(this), { passive: false });
            this.element.addEventListener('touchend', this.handleTouchEnd.bind(this), { passive: false });
            this.element.addEventListener('touchcancel', this.handleTouchCancel.bind(this), { passive: false });
        }

        handleTouchStart(event) {
            event.preventDefault();

            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                this.touches.set(touch.identifier, {
                    x: touch.clientX,
                    y: touch.clientY,
                    startX: touch.clientX,
                    startY: touch.clientY,
                });
            }

            if (this.touches.size === 2) {
                const touchArray = Array.from(this.touches.values());
                const distance = this.calculateDistance(
                    touchArray[0].x, touchArray[0].y,
                    touchArray[1].x, touchArray[1].y
                );

                this.pinchState.startX = touchArray[0].x;
                this.pinchState.startY = touchArray[0].y;
                this.pinchState.startTime = Date.now();
                this.pinchState.isActive = true;
                this.pinchState.scale = 1;

                if (this.callbacks.onPinchStart) {
                    this.callbacks.onPinchStart({ scale: 1, center: this.getTouchCenter() });
                }
            }
        }

        handleTouchMove(event) {
            event.preventDefault();

            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                if (this.touches.has(touch.identifier)) {
                    const touchData = this.touches.get(touch.identifier);
                    touchData.x = touch.clientX;
                    touchData.y = touch.clientY;
                }
            }

            if (this.touches.size === 2) {
                const touchArray = Array.from(this.touches.values());
                const currentDistance = this.calculateDistance(
                    touchArray[0].x, touchArray[0].y,
                    touchArray[1].x, touchArray[1].y
                );

                const previousDistance = this.calculateDistance(
                    touchArray[0].startX, touchArray[0].startY,
                    touchArray[1].startX, touchArray[1].startY
                );

                if (previousDistance > 0) {
                    this.pinchState.scale = currentDistance / previousDistance;

                    if (this.callbacks.onPinchMove) {
                        this.callbacks.onPinchMove({
                            scale: this.pinchState.scale,
                            center: this.getTouchCenter(),
                        });
                    }
                }
            }
        }

        handleTouchEnd(event) {
            event.preventDefault();

            for (let i = 0; i < event.changedTouches.length; i++) {
                const touch = event.changedTouches[i];
                this.touches.delete(touch.identifier);
            }

            if (this.touches.size < 2 && this.pinchState.isActive) {
                if (Math.abs(this.pinchState.scale - 1) > this.options.minPinchScale) {
                    if (this.callbacks.onPinchEnd) {
                        this.callbacks.onPinchEnd({ scale: this.pinchState.scale });
                    }
                }
                this.pinchState.reset();
            }

            if (this.touches.size === 0) {
                const touchArray = Array.from(this.touches.values());
                if (touchArray.length === 0) {
                    const now = Date.now();
                    const lastTouch = event.changedTouches[0];

                    if (lastTouch) {
                        const timeSinceLastTap = now - this.lastTapTime;
                        const distance = this.calculateDistance(
                            lastTouch.clientX, lastTouch.clientY,
                            this.lastTapPosition.x, this.lastTapPosition.y
                        );

                        if (timeSinceLastTap < this.options.doubleTapDelay && distance < 30) {
                            if (this.callbacks.onDoubleTap) {
                                this.callbacks.onDoubleTap({
                                    x: lastTouch.clientX,
                                    y: lastTouch.clientY,
                                });
                            }
                            this.lastTapTime = 0;
                        } else {
                            this.lastTapTime = now;
                            this.lastTapPosition.x = lastTouch.clientX;
                            this.lastTapPosition.y = lastTouch.clientY;
                        }
                    }
                }
            }
        }

        handleTouchCancel(event) {
            this.touches.clear();
            this.pinchState.reset();
        }

        calculateDistance(x1, y1, x2, y2) {
            return Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2));
        }

        getTouchCenter() {
            const touchArray = Array.from(this.touches.values());
            if (touchArray.length === 0) return { x: 0, y: 0 };

            const sumX = touchArray.reduce((sum, touch) => sum + touch.x, 0);
            const sumY = touchArray.reduce((sum, touch) => sum + touch.y, 0);

            return {
                x: sumX / touchArray.length,
                y: sumY / touchArray.length,
            };
        }

        on(event, callback) {
            if (event in this.callbacks) {
                this.callbacks[event] = callback;
            }
        }

        off(event) {
            if (event in this.callbacks) {
                this.callbacks[event] = null;
            }
        }

        destroy() {
            this.element.removeEventListener('touchstart', this.handleTouchStart.bind(this));
            this.element.removeEventListener('touchmove', this.handleTouchMove.bind(this));
            this.element.removeEventListener('touchend', this.handleTouchEnd.bind(this));
            this.element.removeEventListener('touchcancel', this.handleTouchCancel.bind(this));

            this.touches.clear();
            this.callbacks = {
                onPinchStart: null,
                onPinchMove: null,
                onPinchEnd: null,
                onRotationStart: null,
                onRotationMove: null,
                onRotationEnd: null,
                onDoubleTap: null,
            };
        }
    }

    return {
        MultiTouchGestureRecognizer: MultiTouchGestureRecognizer,
        GestureState: GestureState,
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = GestureRecognizer;
}

if (typeof window !== 'undefined') {
    window.GestureRecognizer = GestureRecognizer;
}
