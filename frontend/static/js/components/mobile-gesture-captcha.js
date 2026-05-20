/**
 * HJTPX 移动端手势验证码组件
 * 专为移动设备优化，支持触摸手势、压力感应、多点触控
 */

class MobileGestureCaptcha {
    constructor(container, options = {}) {
        this.container = container;
        this.options = {
            apiBase: '/api/v1',
            onSuccess: null,
            onError: null,
            onRefresh: null,
            enableHapticFeedback: true,
            enableTouchVisualization: true,
            gestureTimeout: 10000,
            minPoints: 3,
            maxPoints: 6,
            touchPointSize: 40,
            lineWidth: 3,
            ...options
        };

        this.state = {
            isLoading: false,
            isDrawing: false,
            sessionId: null,
            pattern: [],
            drawnPath: [],
            startTime: null,
            touchData: null,
            deviceInfo: null,
        };

        this.canvas = null;
        this.ctx = null;
        this.touchStartHandler = null;
        this.touchMoveHandler = null;
        this.touchEndHandler = null;

        this.init();
    }

    init() {
        this.detectDevice();
        this.render();
        this.bindEvents();
        this.refresh();
    }

    detectDevice() {
        this.state.deviceInfo = {
            isMobile: /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent),
            touchCapable: 'ontouchstart' in window,
            maxTouchPoints: navigator.maxTouchPoints || 0,
            platform: navigator.platform,
            userAgent: navigator.userAgent,
            screenWidth: window.screen.width,
            screenHeight: window.screen.height,
        };
    }

    render() {
        const isMobile = this.state.deviceInfo.isMobile;
        const size = isMobile ? '100%' : '360px';
        const height = isMobile ? '300px' : '300px';

        this.container.innerHTML = `
            <div class="mobile-gesture-captcha" style="width: ${size}; max-width: ${size}; height: ${height};">
                <div class="captcha-header">
                    <span class="captcha-title">${isMobile ? '滑动连接圆点' : 'Connect the dots'}</span>
                    <button class="refresh-btn" id="gesture-refresh-btn">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
                <div class="gesture-canvas-container">
                    <canvas class="gesture-canvas" width="300" height="220"></canvas>
                    <div class="gesture-loading" style="display: none;">
                        <div class="loading-spinner"></div>
                    </div>
                </div>
                <div class="gesture-hint">
                    ${isMobile ? '按顺序滑动圆点完成验证' : 'Slide through dots in order'}
                </div>
                <div class="gesture-progress">
                    <div class="progress-bar"></div>
                </div>
            </div>
        `;

        this.canvas = this.container.querySelector('.gesture-canvas');
        this.ctx = this.canvas.getContext('2d');
        this.elements = {
            title: this.container.querySelector('.captcha-title'),
            refresh: this.container.querySelector('#gesture-refresh-btn'),
            hint: this.container.querySelector('.gesture-hint'),
            progress: this.container.querySelector('.progress-bar'),
            loading: this.container.querySelector('.gesture-loading'),
        };
    }

    bindEvents() {
        const canvas = this.canvas;

        this.touchStartHandler = this.handleTouchStart.bind(this);
        this.touchMoveHandler = this.handleTouchMove.bind(this);
        this.touchEndHandler = this.handleTouchEnd.bind(this);

        canvas.addEventListener('touchstart', this.touchStartHandler, { passive: false });
        canvas.addEventListener('touchmove', this.touchMoveHandler, { passive: false });
        canvas.addEventListener('touchend', this.touchEndHandler, { passive: false });

        canvas.addEventListener('mousedown', this.handleMouseDown.bind(this));
        canvas.addEventListener('mousemove', this.handleMouseMove.bind(this));
        canvas.addEventListener('mouseup', this.handleMouseUp.bind(this));

        this.elements.refresh.addEventListener('click', () => this.refresh());

        canvas.style.touchAction = 'none';
    }

    handleTouchStart(e) {
        e.preventDefault();
        if (this.state.isLoading || this.state.isDrawing) return;

        this.state.isDrawing = true;
        this.state.startTime = Date.now();
        this.state.drawnPath = [];
        this.state.touchData = this.initTouchData();

        const touch = e.touches[0];
        const point = this.getCanvasPoint(touch.clientX, touch.clientY);
        
        this.checkAndAddPoint(point);
        this.state.drawnPath.push({
            x: point.x,
            y: point.y,
            timestamp: Date.now(),
            pressure: touch.force || 0.5,
        });

        this.touchData.totalTouches++;
        this.touchData.touchDuration = 0;

        this.updateProgress();

        if (this.options.enableHapticFeedback) {
            this.triggerHaptic('light');
        }
    }

    handleTouchMove(e) {
        e.preventDefault();
        if (!this.state.isDrawing) return;

        const touch = e.touches[0];
        const point = this.getCanvasPoint(touch.clientX, touch.clientY);
        
        const lastPoint = this.state.drawnPath[this.state.drawnPath.length - 1];
        const now = Date.now();
        const dt = now - (lastPoint?.timestamp || this.state.startTime);

        this.state.drawnPath.push({
            x: point.x,
            y: point.y,
            timestamp: now,
            pressure: touch.force || 0.5,
        });

        if (dt > 0) {
            const dx = point.x - lastPoint.x;
            const dy = point.y - lastPoint.y;
            const distance = Math.sqrt(dx * dx + dy * dy);
            const velocity = distance / dt * 1000;
            this.touchData.velocityProfile.push(velocity);
        }

        this.touchData.touchDuration = now - this.state.startTime;

        if (touch.force !== undefined) {
            this.touchData.touchPressure = touch.force;
        }

        if (touch.touchType === 'direct') {
            this.touchData.touchArea = 20 * touch.radiusX * touch.radiusY;
        }

        this.checkAndAddPoint(point);
        this.draw();
    }

    handleTouchEnd(e) {
        e.preventDefault();
        if (!this.state.isDrawing) return;

        this.state.isDrawing = false;
        this.touchData.touchDuration = Date.now() - this.state.startTime;

        if (this.state.drawnPath.length > 1) {
            this.verify();
        } else {
            this.reset();
        }

        if (this.options.enableHapticFeedback) {
            this.triggerHaptic('medium');
        }
    }

    handleMouseDown(e) {
        if (this.state.isLoading || this.state.isDrawing) return;

        this.state.isDrawing = true;
        this.state.startTime = Date.now();
        this.state.drawnPath = [];
        this.state.touchData = this.initTouchData();

        const point = this.getCanvasPoint(e.clientX, e.clientY);
        this.checkAndAddPoint(point);
        this.state.drawnPath.push({
            x: point.x,
            y: point.y,
            timestamp: Date.now(),
            pressure: 0.5,
        });

        this.touchData.totalTouches++;
        this.updateProgress();
    }

    handleMouseMove(e) {
        if (!this.state.isDrawing) return;

        const point = this.getCanvasPoint(e.clientX, e.clientY);
        
        const lastPoint = this.state.drawnPath[this.state.drawnPath.length - 1];
        const now = Date.now();
        const dt = now - (lastPoint?.timestamp || this.state.startTime);

        this.state.drawnPath.push({
            x: point.x,
            y: point.y,
            timestamp: now,
            pressure: 0.5,
        });

        if (dt > 0) {
            const dx = point.x - lastPoint.x;
            const dy = point.y - lastPoint.y;
            const distance = Math.sqrt(dx * dx + dy * dy);
            const velocity = distance / dt * 1000;
            this.touchData.velocityProfile.push(velocity);
        }

        this.touchData.touchDuration = now - this.state.startTime;

        this.checkAndAddPoint(point);
        this.draw();
    }

    handleMouseUp(e) {
        if (!this.state.isDrawing) return;

        this.state.isDrawing = false;
        this.touchData.touchDuration = Date.now() - this.state.startTime;

        if (this.state.drawnPath.length > 1) {
            this.verify();
        } else {
            this.reset();
        }
    }

    initTouchData() {
        return {
            totalTouches: 0,
            touchPressure: 0.5,
            touchDuration: 0,
            touchArea: 20,
            velocityProfile: [],
            isMultiTouch: false,
        };
    }

    getCanvasPoint(clientX, clientY) {
        const rect = this.canvas.getBoundingClientRect();
        const scaleX = this.canvas.width / rect.width;
        const scaleY = this.canvas.height / rect.height;

        return {
            x: (clientX - rect.left) * scaleX,
            y: (clientY - rect.top) * scaleY,
        };
    }

    checkAndAddPoint(point) {
        const pointSize = this.options.touchPointSize;
        const pattern = this.state.pattern;

        for (let i = 0; i < pattern.length; i++) {
            const patternPoint = pattern[i];
            const dx = point.x - patternPoint.x;
            const dy = point.y - patternPoint.y;
            const distance = Math.sqrt(dx * dx + dy * dy);

            if (distance <= pointSize) {
                const alreadyAdded = this.state.drawnPath.some(
                    p => p.pointIndex === i
                );

                if (!alreadyAdded) {
                    const lastPoint = this.state.drawnPath[this.state.drawnPath.length - 1];
                    if (lastPoint) {
                        lastPoint.pointIndex = i;
                        lastPoint.patternIndex = this.state.pattern.indexOf(patternPoint);
                    }

                    this.state.drawnPath[this.state.drawnPath.length - 1].pointIndex = i;

                    if (this.options.enableHapticFeedback) {
                        this.triggerHaptic('selection');
                    }

                    this.updateProgress();
                    return;
                }
            }
        }
    }

    updateProgress() {
        const selectedPoints = this.state.drawnPath.filter(p => p.pointIndex !== undefined).length;
        const totalPoints = this.state.pattern.length;
        const progress = (selectedPoints / totalPoints) * 100;

        this.elements.progress.style.width = `${progress}%`;

        if (progress === 100) {
            this.elements.hint.textContent = this.state.deviceInfo.isMobile ? 
                '手势完成，验证中...' : 'Gesture complete, verifying...';
        }
    }

    triggerHaptic(type) {
        if (!this.options.enableHapticFeedback || !navigator.vibrate) return;

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
            case 'selection':
                navigator.vibrate(5);
                break;
            case 'success':
                navigator.vibrate([10, 50, 10]);
                break;
            case 'error':
                navigator.vibrate([50, 30, 50]);
                break;
        }
    }

    async refresh() {
        this.state.isLoading = true;
        this.elements.loading.style.display = 'flex';

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/gesture`);
            if (response.ok) {
                const data = await response.json();
                this.state.sessionId = data.session_id;
                this.parsePattern(data.pattern);
            } else {
                this.loadDemoData();
            }
        } catch (error) {
            this.loadDemoData();
        } finally {
            this.state.isLoading = false;
            this.elements.loading.style.display = 'none';
            this.reset();
            if (this.options.onRefresh) this.options.onRefresh();
        }
    }

    loadDemoData() {
        this.state.sessionId = 'demo_' + Date.now();
        const patternLength = this.options.minPoints + 
            Math.floor(Math.random() * (this.options.maxPoints - this.options.minPoints + 1));
        
        const points = [
            {x: 50, y: 50}, {x: 150, y: 50}, {x: 250, y: 50},
            {x: 50, y: 110}, {x: 150, y: 110}, {x: 250, y: 110},
            {x: 50, y: 170}, {x: 150, y: 170}, {x: 250, y: 170},
        ];

        const shuffled = [...points].sort(() => Math.random() - 0.5);
        this.state.pattern = shuffled.slice(0, patternLength);

        const patternNumbers = shuffled.slice(0, patternLength).map((_, i) => i + 1);
        this.state.patternString = patternNumbers.join('-');
    }

    parsePattern(patternString) {
        const pointMap = {
            1: {x: 50, y: 50}, 2: {x: 150, y: 50}, 3: {x: 250, y: 50},
            4: {x: 50, y: 110}, 5: {x: 150, y: 110}, 6: {x: 250, y: 110},
            7: {x: 50, y: 170}, 8: {x: 150, y: 170}, 9: {x: 250, y: 170},
        };

        const numbers = patternString.split('-').map(n => parseInt(n));
        this.state.pattern = numbers.map(n => pointMap[n]);
        this.state.patternString = patternString;
    }

    draw() {
        if (!this.ctx) return;

        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);

        this.drawPatternPoints();
        this.drawDrawnPath();
    }

    drawPatternPoints() {
        const ctx = this.ctx;
        const pointSize = this.options.touchPointSize;

        this.state.pattern.forEach((point, index) => {
            const selectedPoint = this.state.drawnPath.find(p => p.pointIndex === index);

            ctx.beginPath();
            ctx.arc(point.x, point.y, pointSize / 2, 0, Math.PI * 2);

            if (selectedPoint) {
                const gradient = ctx.createRadialGradient(
                    point.x, point.y, 0,
                    point.x, point.y, pointSize / 2
                );
                gradient.addColorStop(0, '#4CAF50');
                gradient.addColorStop(1, '#45a049');
                ctx.fillStyle = gradient;
            } else {
                const gradient = ctx.createRadialGradient(
                    point.x, point.y, 0,
                    point.x, point.y, pointSize / 2
                );
                gradient.addColorStop(0, '#667eea');
                gradient.addColorStop(1, '#764ba2');
                ctx.fillStyle = gradient;
            }

            ctx.fill();

            ctx.strokeStyle = selectedPoint ? '#2e7d32' : '#5a5aaa';
            ctx.lineWidth = 2;
            ctx.stroke();

            ctx.fillStyle = '#fff';
            ctx.font = 'bold 14px Arial';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(index + 1, point.x, point.y);
        });
    }

    drawDrawnPath() {
        if (this.state.drawnPath.length < 2) return;

        const ctx = this.ctx;

        ctx.beginPath();
        ctx.moveTo(this.state.drawnPath[0].x, this.state.drawnPath[0].y);

        for (let i = 1; i < this.state.drawnPath.length; i++) {
            ctx.lineTo(this.state.drawnPath[i].x, this.state.drawnPath[i].y);
        }

        ctx.strokeStyle = 'rgba(102, 126, 234, 0.8)';
        ctx.lineWidth = this.options.lineWidth;
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';
        ctx.stroke();

        if (this.options.enableTouchVisualization) {
            this.drawVelocityIndicator();
        }
    }

    drawVelocityIndicator() {
        if (this.touchData.velocityProfile.length === 0) return;

        const ctx = this.ctx;
        const lastVelocity = this.touchData.velocityProfile[this.touchData.velocityProfile.length - 1];

        if (lastVelocity > 500) {
            ctx.strokeStyle = 'rgba(244, 67, 54, 0.5)';
            ctx.lineWidth = 1;
            ctx.stroke();
        } else if (lastVelocity > 200) {
            ctx.strokeStyle = 'rgba(255, 193, 7, 0.5)';
            ctx.lineWidth = 1;
            ctx.stroke();
        }
    }

    async verify() {
        this.state.isLoading = true;
        this.elements.loading.style.display = 'flex';

        const selectedPattern = this.state.drawnPath
            .filter(p => p.pointIndex !== undefined)
            .map(p => p.pointIndex + 1)
            .join('-');

        const payload = {
            id: this.state.sessionId,
            pattern: selectedPattern,
            touch_data: this.touchData,
            device_info: this.state.deviceInfo,
        };

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/gesture/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });

            const data = await response.json();

            if (data.success || data.data?.success) {
                this.handleSuccess();
            } else {
                this.handleError(data.message || '验证失败');
            }
        } catch (error) {
            this.handleError('网络错误，请重试');
        } finally {
            this.state.isLoading = false;
            this.elements.loading.style.display = 'none';
        }
    }

    handleSuccess() {
        this.elements.title.textContent = this.state.deviceInfo.isMobile ? 
            '验证成功' : 'Verification successful';
        
        this.elements.progress.style.backgroundColor = '#4CAF50';
        
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        this.drawPatternPoints();
        
        this.playSuccessAnimation();

        if (this.options.enableHapticFeedback) {
            this.triggerHaptic('success');
        }

        if (this.options.onSuccess) {
            this.options.onSuccess({
                type: 'gesture',
                session_id: this.state.sessionId,
            });
        }
    }

    handleError(message) {
        this.elements.title.textContent = message;
        
        this.elements.progress.style.backgroundColor = '#f44336';
        
        if (this.options.enableHapticFeedback) {
            this.triggerHaptic('error');
        }

        this.playErrorAnimation();

        setTimeout(() => {
            this.reset();
        }, 1500);

        if (this.options.onError) {
            this.options.onError({
                type: 'gesture',
                error: message,
            });
        }
    }

    playSuccessAnimation() {
        const points = this.state.pattern;
        
        points.forEach((point, index) => {
            setTimeout(() => {
                this.ctx.beginPath();
                this.ctx.arc(point.x, point.y, 30, 0, Math.PI * 2);
                this.ctx.strokeStyle = 'rgba(76, 175, 80, 0.8)';
                this.ctx.lineWidth = 3;
                this.ctx.stroke();
            }, index * 100);
        });
    }

    playErrorAnimation() {
        let shakeCount = 0;
        const maxShakes = 4;

        const shake = () => {
            if (shakeCount >= maxShakes) {
                this.canvas.style.transform = '';
                return;
            }

            const direction = shakeCount % 2 === 0 ? -1 : 1;
            this.canvas.style.transform = `translateX(${direction * 10}px)`;
            shakeCount++;

            setTimeout(() => {
                this.canvas.style.transform = '';
                setTimeout(shake, 50);
            }, 50);
        };

        shake();
    }

    reset() {
        this.state.drawnPath = [];
        this.state.isDrawing = false;
        this.state.startTime = null;
        this.state.touchData = this.initTouchData();

        this.elements.progress.style.width = '0%';
        this.elements.progress.style.backgroundColor = '#667eea';
        this.elements.title.textContent = this.state.deviceInfo.isMobile ? 
            '滑动连接圆点' : 'Connect the dots';
        this.elements.hint.textContent = this.state.deviceInfo.isMobile ? 
            '按顺序滑动圆点完成验证' : 'Slide through dots in order';

        this.draw();
    }

    destroy() {
        if (this.canvas) {
            this.canvas.removeEventListener('touchstart', this.touchStartHandler);
            this.canvas.removeEventListener('touchmove', this.touchMoveHandler);
            this.canvas.removeEventListener('touchend', this.touchEndHandler);
        }

        this.container.innerHTML = '';
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = MobileGestureCaptcha;
}

if (typeof window !== 'undefined') {
    window.MobileGestureCaptcha = MobileGestureCaptcha;
}
