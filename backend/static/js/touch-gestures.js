// 移动端触摸优化模块 - 增强版
const TouchOptimization = {
    config: {
        enableDoubleTap: true,
        enablePinch: true,
        enableSwipe: true,
        enableLongPress: true,
        enable3DTouch: true,
        touchFeedbackTime: 100,
        swipeThreshold: 50,
        pinchThreshold: 0.1,
        longPressDelay: 500,
        tapThreshold: 10,
        doubleTapDelay: 300,
        enableHapticFeedback: true,
        enableVisualFeedback: true,
        enableSoundFeedback: false,
        preventDefaultTouch: true,
        enablePassiveListeners: true
    },
    
    activeTouches: new Map(),
    gestureState: {
        pinching: false,
        swiping: false,
        longPressing: false
    },
    
    lastTap: { time: 0, x: 0, y: 0 },
    tapCount: 0,
    
    initialized: false,
    
    init: function(config) {
        if (this.initialized) {
            console.warn('TouchOptimization already initialized');
            return;
        }
        
        if (config) {
            this.config = { ...this.config, ...config };
        }
        
        this.setupEventListeners();
        this.setupPerformanceMonitoring();
        this.detectDeviceCapabilities();
        this.initialized = true;
        
        document.dispatchEvent(new CustomEvent('touchOptimizationReady'));
    },
    
    setupEventListeners: function() {
        const options = this.config.enablePassiveListeners ? 
            { passive: true, capture: true } : 
            { passive: false, capture: true };
        
        document.addEventListener('touchstart', this.handleTouchStart.bind(this), options);
        document.addEventListener('touchmove', this.handleTouchMove.bind(this), options);
        document.addEventListener('touchend', this.handleTouchEnd.bind(this), options);
        document.addEventListener('touchcancel', this.handleTouchCancel.bind(this), options);
        
        if (this.config.enable3DTouch && 'ontouchforce' in window) {
            document.addEventListener('touchforcechange', this.handleForceChange.bind(this), options);
        }
    },
    
    detectDeviceCapabilities: function() {
        this.deviceCapabilities = {
            hasTouch: 'ontouchstart' in window,
            hasPointer: navigator.maxTouchPoints > 0,
            has3DTouch: 'ontouchforce' in window,
            hasHaptic: 'vibrate' in navigator,
            supportsPointerEvents: window.PointerEvent !== undefined,
            supportsTouchEvents: 'ontouchstart' in window,
            touchPoints: navigator.maxTouchPoints || 0
        };
        
        if (this.deviceCapabilities.touchPoints > 1 && this.config.enablePinch) {
            this.setupPinchZoom();
        }
    },
    
    handleTouchStart: function(e) {
        const touch = e.touches[0];
        const touchId = this.getTouchIdentifier(e);
        
        this.activeTouches.set(touchId, {
            startX: touch.clientX,
            startY: touch.clientY,
            currentX: touch.clientX,
            currentY: touch.clientY,
            startTime: Date.now(),
            target: e.target,
            touchType: this.getTouchType(e.touches.length)
        });
        
        if (this.activeTouches.size === 1) {
            this.startGestureDetection(touchId);
        }
        
        this.provideVisualFeedback(touch, 'start');
        
        if (this.config.preventDefaultTouch) {
            e.preventDefault();
        }
    },
    
    handleTouchMove: function(e) {
        const touch = e.touches[0];
        const touchId = this.getTouchIdentifier(e);
        const touchData = this.activeTouches.get(touchId);
        
        if (!touchData) return;
        
        touchData.currentX = touch.clientX;
        touchData.currentY = touch.clientY;
        touchData.moveTime = Date.now();
        
        this.updateGestureState(touchId, e);
        
        this.provideVisualFeedback(touch, 'move');
        
        if (this.config.preventDefaultTouch) {
            e.preventDefault();
        }
    },
    
    handleTouchEnd: function(e) {
        const touchId = this.getTouchIdentifier(e);
        const touchData = this.activeTouches.get(touchId);
        
        if (!touchData) return;
        
        const duration = Date.now() - touchData.startTime;
        const distance = this.calculateDistance(
            touchData.startX, touchData.startY,
            touchData.currentX, touchData.currentY
        );
        
        this.detectGesture(touchId, touchData, duration, distance);
        
        this.provideVisualFeedback(touchData, 'end');
        
        this.activeTouches.delete(touchId);
        
        if (this.activeTouches.size === 0) {
            this.resetGestureState();
        }
        
        if (this.config.preventDefaultTouch) {
            e.preventDefault();
        }
    },
    
    handleTouchCancel: function(e) {
        this.activeTouches.clear();
        this.resetGestureState();
    },
    
    handleForceChange: function(e) {
        const force = e.touches[0].force;
        const maxForce = 1.0;
        
        if (force > 0.5 && force < maxForce) {
            this.dispatchCustomEvent('touchForceMedium', { force, touch: e.touches[0] });
        } else if (force >= maxForce) {
            this.dispatchCustomEvent('touchForceHeavy', { force, touch: e.touches[0] });
        }
    },
    
    startGestureDetection: function(touchId) {
        if (this.config.enableLongPress) {
            this.longPressTimer = setTimeout(() => {
                const touchData = this.activeTouches.get(touchId);
                if (touchData) {
                    this.gestureState.longPressing = true;
                    this.dispatchCustomEvent('longPress', {
                        x: touchData.startX,
                        y: touchData.startY,
                        target: touchData.target
                    });
                    this.provideHapticFeedback('heavy');
                }
            }, this.config.longPressDelay);
        }
    },
    
    updateGestureState: function(touchId, e) {
        const touchData = this.activeTouches.get(touchId);
        if (!touchData) return;
        
        if (this.activeTouches.size === 2 && this.config.enablePinch) {
            const touches = Array.from(this.activeTouches.values());
            const distance = this.calculateDistance(
                touches[0].currentX, touches[0].currentY,
                touches[1].currentX, touches[1].currentY
            );
            const startDistance = this.calculateDistance(
                touches[0].startX, touches[0].startY,
                touches[1].startX, touches[1].startY
            );
            
            const scale = distance / startDistance;
            
            if (Math.abs(1 - scale) > this.config.pinchThreshold) {
                this.gestureState.pinching = true;
                this.dispatchCustomEvent('pinch', {
                    scale: scale,
                    direction: scale > 1 ? 'in' : 'out'
                });
            }
        }
        
        const deltaX = touchData.currentX - touchData.startX;
        const deltaY = touchData.currentY - touchData.startY;
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        
        if (distance > this.config.swipeThreshold) {
            this.gestureState.swiping = true;
        }
    },
    
    detectGesture: function(touchId, touchData, duration, distance) {
        if (this.gestureState.longPressing) {
            this.gestureState.longPressing = false;
            return;
        }
        
        if (this.gestureState.pinching) {
            this.gestureState.pinching = false;
            return;
        }
        
        if (distance < this.config.tapThreshold) {
            this.detectTap(touchData, duration);
        } else if (this.gestureState.swiping) {
            this.detectSwipe(touchData);
        }
        
        this.gestureState.swiping = false;
    },
    
    detectTap: function(touchData, duration) {
        if (this.config.enableDoubleTap) {
            const now = Date.now();
            const timeSinceLastTap = now - this.lastTap.time;
            const distanceSinceLastTap = this.calculateDistance(
                this.lastTap.x, this.lastTap.y,
                touchData.startX, touchData.startY
            );
            
            if (timeSinceLastTap < this.config.doubleTapDelay && 
                distanceSinceLastTap < this.config.tapThreshold) {
                this.tapCount++;
                
                if (this.tapCount === 2) {
                    this.dispatchCustomEvent('doubleTap', {
                        x: touchData.startX,
                        y: touchData.startY,
                        target: touchData.target
                    });
                    this.provideHapticFeedback('light');
                    this.tapCount = 0;
                }
            } else {
                this.tapCount = 1;
            }
            
            this.lastTap = { time: now, x: touchData.startX, y: touchData.startY };
        }
        
        this.dispatchCustomEvent('tap', {
            x: touchData.startX,
            y: touchData.startY,
            target: touchData.target,
            duration: duration
        });
    },
    
    detectSwipe: function(touchData) {
        const deltaX = touchData.currentX - touchData.startX;
        const deltaY = touchData.currentY - touchData.startY;
        const absDeltaX = Math.abs(deltaX);
        const absDeltaY = Math.abs(deltaY);
        
        let direction;
        if (absDeltaX > absDeltaY) {
            direction = deltaX > 0 ? 'right' : 'left';
        } else {
            direction = deltaY > 0 ? 'down' : 'up';
        }
        
        const velocity = this.calculateVelocity(touchData);
        
        this.dispatchCustomEvent('swipe', {
            direction: direction,
            distanceX: deltaX,
            distanceY: deltaY,
            velocity: velocity,
            target: touchData.target
        });
        
        this.provideHapticFeedback('light');
    },
    
    calculateVelocity: function(touchData) {
        const duration = (touchData.moveTime || Date.now()) - touchData.startTime;
        if (duration === 0) return 0;
        
        const distance = this.calculateDistance(
            touchData.startX, touchData.startY,
            touchData.currentX, touchData.currentY
        );
        
        return distance / (duration / 1000);
    },
    
    calculateDistance: function(x1, y1, x2, y2) {
        return Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2));
    },
    
    getTouchIdentifier: function(e) {
        return e.changedTouches ? e.changedTouches[0].identifier : 'mouse';
    },
    
    getTouchType: function(touchCount) {
        switch (touchCount) {
            case 1: return 'single';
            case 2: return 'double';
            case 3: return 'triple';
            default: return 'multi';
        }
    },
    
    resetGestureState: function() {
        this.gestureState = {
            pinching: false,
            swiping: false,
            longPressing: false
        };
        
        if (this.longPressTimer) {
            clearTimeout(this.longPressTimer);
            this.longPressTimer = null;
        }
    },
    
    provideVisualFeedback: function(touchData, type) {
        if (!this.config.enableVisualFeedback) return;
        
        const element = touchData.target || (touchData.startX ? null : null);
        
        switch (type) {
            case 'start':
                if (element) {
                    element.style.transform = 'scale(0.95)';
                    element.style.transition = 'transform 0.1s ease';
                }
                break;
            case 'end':
                if (element) {
                    element.style.transform = 'scale(1)';
                }
                break;
            case 'move':
                break;
        }
    },
    
    provideHapticFeedback: function(type) {
        if (!this.config.enableHapticFeedback || !this.deviceCapabilities.hasHaptic) return;
        
        switch (type) {
            case 'light':
                navigator.vibrate(10);
                break;
            case 'medium':
                navigator.vibrate(20);
                break;
            case 'heavy':
                navigator.vibrate(30);
                break;
            case 'success':
                navigator.vibrate([10, 50, 10]);
                break;
            case 'error':
                navigator.vibrate([30, 50, 30, 50, 30]);
                break;
        }
    },
    
    provideSoundFeedback: function(type) {
        if (!this.config.enableSoundFeedback) return;
        
        const audioContext = TouchOptimization.audioContext || 
            (TouchOptimization.audioContext = new (window.AudioContext || window.webkitAudioContext)());
        
        const oscillator = audioContext.createOscillator();
        const gainNode = audioContext.createGain();
        
        oscillator.connect(gainNode);
        gainNode.connect(audioContext.destination);
        
        switch (type) {
            case 'tap':
                oscillator.frequency.value = 800;
                gainNode.gain.value = 0.1;
                break;
            case 'swipe':
                oscillator.frequency.value = 600;
                gainNode.gain.value = 0.15;
                break;
            case 'success':
                oscillator.frequency.value = 1000;
                gainNode.gain.value = 0.2;
                break;
        }
        
        oscillator.start();
        oscillator.stop(audioContext.currentTime + 0.1);
    },
    
    setupPinchZoom: function() {
        let initialDistance = 0;
        let initialScale = 1;
        
        document.addEventListener('touchstart', (e) => {
            if (e.touches.length === 2) {
                const dx = e.touches[0].clientX - e.touches[1].clientX;
                const dy = e.touches[0].clientY - e.touches[1].clientY;
                initialDistance = Math.sqrt(dx * dx + dy * dy);
            }
        }, { passive: true });
        
        document.addEventListener('touchmove', (e) => {
            if (e.touches.length === 2 && initialDistance > 0) {
                const dx = e.touches[0].clientX - e.touches[1].clientX;
                const dy = e.touches[0].clientY - e.touches[1].clientY;
                const currentDistance = Math.sqrt(dx * dx + dy * dy);
                
                const scale = currentDistance / initialDistance * initialScale;
                
                this.dispatchCustomEvent('pinchZoom', {
                    scale: scale,
                    currentDistance: currentDistance,
                    initialDistance: initialDistance
                });
            }
        }, { passive: true });
        
        document.addEventListener('touchend', (e) => {
            if (e.touches.length < 2) {
                initialDistance = 0;
            }
        }, { passive: true });
    },
    
    dispatchCustomEvent: function(eventName, detail) {
        const event = new CustomEvent(`touch${eventName.charAt(0).toUpperCase() + eventName.slice(1)}`, {
            detail: detail,
            bubbles: true,
            cancelable: true
        });
        document.dispatchEvent(event);
    },
    
    setupPerformanceMonitoring: function() {
        if ('PerformanceObserver' in window) {
            try {
                const observer = new PerformanceObserver((list) => {
                    for (const entry of list.getEntries()) {
                        if (entry.entryType === 'event') {
                            if (entry.duration > 100) {
                                console.warn(`Touch event took ${entry.duration}ms:`, entry.name);
                            }
                        }
                    }
                });
                
                observer.observe({ entryTypes: ['event'] });
            } catch (e) {
                console.warn('Performance monitoring not available');
            }
        }
    },
    
    getTouchStats: function() {
        return {
            activeTouches: this.activeTouches.size,
            gestureState: { ...this.gestureState },
            deviceCapabilities: { ...this.deviceCapabilities },
            tapCount: this.tapCount,
            config: { ...this.config }
        };
    },
    
    enableFeature: function(feature) {
        if (feature in this.config) {
            this.config[feature] = true;
        }
    },
    
    disableFeature: function(feature) {
        if (feature in this.config) {
            this.config[feature] = false;
        }
    },
    
    destroy: function() {
        document.removeEventListener('touchstart', this.handleTouchStart);
        document.removeEventListener('touchmove', this.handleTouchMove);
        document.removeEventListener('touchend', this.handleTouchEnd);
        document.removeEventListener('touchcancel', this.handleTouchCancel);
        
        if (this.longPressTimer) {
            clearTimeout(this.longPressTimer);
        }
        
        this.activeTouches.clear();
        this.resetGestureState();
        this.initialized = false;
    }
};

if (typeof module !== 'undefined' && module.exports) {
    module.exports = TouchOptimization;
}
