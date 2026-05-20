/**
 * HJTPX 滑块验证码组件
 * 支持移动端触摸操作、轨迹记录、平滑动画
 */

class SliderCaptcha {
    constructor(container, options = {}) {
        this.container = container;
        this.options = {
            apiBase: '/api/v1',
            onSuccess: null,
            onError: null,
            onRefresh: null,
            enableAnimation: true,
            enableEncryption: true,
            tolerance: 10,
            ...options
        };

        this.sliderState = {
            isDragging: false,
            startX: 0,
            currentX: 0,
            maxX: 0,
            puzzleY: 0,
            targetX: 0,
            puzzleStyle: 0
        };

        this.trajectoryData = [];
        this.speedData = {
            points: [],
            startTime: 0,
            endTime: 0,
            distance: 0,
            maxSpeed: 0
        };
        
        this.lastTrajectoryTime = 0;
        this.minTrajectoryInterval = 8;
        this.trajectoryVersion = '2.0';

        this.sessionId = null;
        this.canvas = null;
        this.ctx = null;

        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.refresh();
    }

    render() {
        this.container.innerHTML = `
            <div class="slider-captcha">
                <div class="slider-image-wrapper">
                    <canvas class="slider-canvas" width="360" height="220"></canvas>
                    <div class="slider-puzzle"></div>
                    <button class="captcha-refresh-btn" id="slider-refresh-btn" aria-label="刷新验证码">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                    <div class="slider-skeleton"></div>
                </div>
                <div class="slider-container" role="slider" aria-label="拖动滑块完成验证"
                     aria-valuemin="0" aria-valuemax="100" aria-valuenow="0" tabindex="0">
                    <div class="slider-track"></div>
                    <div class="slider-button" role="button" aria-label="滑块按钮">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="9 18 15 12 9 6"></polyline>
                        </svg>
                    </div>
                    <div class="slider-text">拖动滑块完成验证</div>
                </div>
            </div>
        `;

        this.canvas = this.container.querySelector('.slider-canvas');
        this.ctx = this.canvas.getContext('2d');
        this.elements = {
            puzzle: this.container.querySelector('.slider-puzzle'),
            container: this.container.querySelector('.slider-container'),
            track: this.container.querySelector('.slider-track'),
            button: this.container.querySelector('.slider-button'),
            text: this.container.querySelector('.slider-text'),
            refresh: this.container.querySelector('#slider-refresh-btn'),
            skeleton: this.container.querySelector('.slider-skeleton'),
            canvas: this.canvas
        };
    }

    bindEvents() {
        const { button, container, refresh } = this.elements;

        const startDrag = (e) => {
            if (this.sliderState.isDragging || this.isLoading) return;

            this.sliderState.isDragging = true;
            const clientX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            this.sliderState.startX = clientX;
            this.sliderState.currentX = 0;
            this.sliderState.maxX = container.offsetWidth - button.offsetWidth - 4;

            this.speedData = {
                points: [],
                startTime: performance.now(),
                endTime: 0,
                distance: 0,
                maxSpeed: 0
            };
            this.trajectoryData = [];
            this.lastTrajectoryTime = 0;
            
            let extraData = {};
            if (e.type === 'touchstart' && e.touches[0]) {
                const touch = e.touches[0];
                if (touch.force !== undefined) {
                    extraData.pressure = touch.force;
                }
                if (touch.tiltX !== undefined) {
                    extraData.tiltX = touch.tiltX;
                    extraData.tiltY = touch.tiltY;
                }
            }
            
            this.addTrajectoryPoint(0, this.sliderState.puzzleY, 'start', extraData);

            button.classList.add('dragging');
            container.classList.add('is-dragging');
            this.elements.text.textContent = '拖动中...';
        };

        const drag = (e) => {
            if (!this.sliderState.isDragging) return;
            e.preventDefault();

            const clientX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX;
            let deltaX = clientX - this.sliderState.startX;
            deltaX = Math.max(0, Math.min(deltaX, this.sliderState.maxX));
            
            const prevX = this.sliderState.currentX;
            this.sliderState.currentX = deltaX;

            const currentTime = performance.now();
            const dt = currentTime - (this.speedData.points.length > 0 ?
                this.speedData.points[this.speedData.points.length - 1].time : this.speedData.startTime);
            const dx = deltaX - prevX;
            const speed = dt > 0 ? dx / (dt / 1000) : 0;

            this.speedData.points.push({ x: deltaX, y: this.sliderState.puzzleY, time: currentTime, speed });
            this.speedData.distance += Math.abs(dx);
            if (Math.abs(speed) > this.speedData.maxSpeed) {
                this.speedData.maxSpeed = Math.abs(speed);
            }

            let extraData = {};
            if (e.type === 'touchmove' && e.touches[0]) {
                const touch = e.touches[0];
                if (touch.force !== undefined) {
                    extraData.pressure = touch.force;
                }
                if (touch.tiltX !== undefined) {
                    extraData.tiltX = touch.tiltX;
                    extraData.tiltY = touch.tiltY;
                }
            }

            this.addTrajectoryPoint(deltaX, this.sliderState.puzzleY, 'move', extraData);
            this.animateSliderPosition(deltaX);
            this.updateAccessibility();
        };

        const endDrag = (e) => {
            if (!this.sliderState.isDragging) return;

            this.sliderState.isDragging = false;
            this.speedData.endTime = performance.now();
            button.classList.remove('dragging');
            container.classList.remove('is-dragging');

            this.addTrajectoryPoint(this.sliderState.currentX, this.sliderState.puzzleY, 'end');

            if (this.sliderState.currentX > 10) {
                this.verify();
            } else {
                this.reset();
            }
        };

        button.addEventListener('mousedown', startDrag);
        button.addEventListener('touchstart', startDrag, { passive: false });
        
        document.addEventListener('mousemove', drag);
        document.addEventListener('touchmove', drag, { passive: false });
        
        document.addEventListener('mouseup', endDrag);
        document.addEventListener('touchend', endDrag);

        refresh.addEventListener('click', () => this.refresh());

        container.addEventListener('keydown', (e) => {
            if (this.isLoading || this.sliderState.isDragging) return;
            switch (e.key) {
                case 'ArrowRight':
                case 'ArrowUp':
                    e.preventDefault();
                    this.simulateSliderDrag(20);
                    break;
                case 'ArrowLeft':
                case 'ArrowDown':
                    e.preventDefault();
                    this.simulateSliderDrag(-20);
                    break;
                case 'Enter':
                case ' ':
                    e.preventDefault();
                    if (this.sliderState.currentX > 10) this.verify();
                    break;
            }
        });
    }

    addTrajectoryPoint(x, y, event, extraData = {}) {
        const currentTime = performance.now();
        
        if (event !== 'start' && this.lastTrajectoryTime > 0) {
            const interval = currentTime - this.lastTrajectoryTime;
            if (interval < this.minTrajectoryInterval && event === 'move') {
                return;
            }
        }
        
        this.lastTrajectoryTime = currentTime;
        
        const point = {
            x: Math.round(x * 100) / 100,
            y: Math.round(y * 100) / 100,
            timestamp: Math.round(currentTime),
            event: event,
            version: this.trajectoryVersion
        };
        
        if (this.trajectoryData.length > 0) {
            const lastPoint = this.trajectoryData[this.trajectoryData.length - 1];
            const dx = point.x - lastPoint.x;
            const dy = point.y - lastPoint.y;
            const dt = point.timestamp - lastPoint.timestamp;
            
            if (dt > 0) {
                point.velocity_x = Math.round((dx / dt) * 1000 * 100) / 100;
                point.velocity_y = Math.round((dy / dt) * 1000 * 100) / 100;
                point.velocity_magnitude = Math.round(Math.sqrt(dx * dx + dy * dy) / dt * 1000 * 100) / 100;
            }
        }
        
        if (event === 'start') {
            point.touch_capable = 'ontouchstart' in window;
            point.pointers = navigator.maxTouchPoints || 0;
        }
        
        if (extraData.pressure !== undefined) {
            point.pressure = extraData.pressure;
        }
        
        if (extraData.tiltX !== undefined) {
            point.tilt_x = extraData.tiltX;
            point.tilt_y = extraData.tiltY;
        }
        
        this.trajectoryData.push(point);
        
        if (this.trajectoryData.length > 500) {
            this.trajectoryData = this.trajectoryData.filter((_, index) => {
                return index % 2 === 0 || index < 50 || index > this.trajectoryData.length - 50;
            });
        }
    }

    animateSliderPosition(x) {
        this.elements.button.style.left = (x + 2) + 'px';
        this.elements.track.style.width = x + 'px';
        this.elements.puzzle.style.left = x + 'px';
        this.drawPuzzleOverlay(x);
    }

    drawPuzzleOverlay(sliderX) {
        if (!this.ctx) return;
        
        const ctx = this.ctx;
        const canvas = this.canvas;
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        const puzzleSize = 50;
        const puzzleY = this.sliderState.puzzleY;
        const targetX = this.sliderState.targetX;

        ctx.strokeStyle = 'rgba(255, 255, 255, 0.9)';
        ctx.lineWidth = 2;
        ctx.setLineDash([5, 3]);

        switch (this.sliderState.puzzleStyle) {
            case 0:
                ctx.strokeRect(targetX, puzzleY, puzzleSize, puzzleSize);
                break;
            case 1:
                ctx.beginPath();
                ctx.arc(targetX + puzzleSize / 2, puzzleY + puzzleSize / 2, puzzleSize / 2, 0, Math.PI * 2);
                ctx.stroke();
                break;
            case 2:
                ctx.beginPath();
                ctx.moveTo(targetX + puzzleSize / 2, puzzleY);
                ctx.lineTo(targetX + puzzleSize, puzzleY + puzzleSize);
                ctx.lineTo(targetX, puzzleY + puzzleSize);
                ctx.closePath();
                ctx.stroke();
                break;
            case 3:
                ctx.beginPath();
                ctx.moveTo(targetX + puzzleSize / 2, puzzleY);
                ctx.lineTo(targetX + puzzleSize, puzzleY + puzzleSize / 2);
                ctx.lineTo(targetX + puzzleSize / 2, puzzleY + puzzleSize);
                ctx.lineTo(targetX, puzzleY + puzzleSize / 2);
                ctx.closePath();
                ctx.stroke();
                break;
        }
        ctx.setLineDash([]);
    }

    simulateSliderDrag(deltaX) {
        const newX = Math.max(0, Math.min(this.sliderState.currentX + deltaX, this.sliderState.maxX));
        this.sliderState.currentX = newX;
        this.animateSliderPosition(newX);
        this.updateAccessibility();
    }

    updateAccessibility() {
        const progress = Math.round((this.sliderState.currentX / this.sliderState.maxX) * 100);
        this.elements.container.setAttribute('aria-valuenow', progress);
    }

    async refresh() {
        this.isLoading = true;
        this.showSkeleton();
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/slider`);
            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                this.sliderState.targetX = data.target_x;
                this.sliderState.targetY = data.target_y;
                this.sliderState.puzzleY = data.target_y;
                this.sliderState.puzzleStyle = data.puzzle_style || 0;
                this.sliderState.tolerance = data.tolerance || this.options.tolerance;

                await this.loadImage(data.image_url);
                this.updatePuzzlePiece();
                this.drawPuzzleOverlay(0);
            } else {
                this.loadDemoData();
            }
        } catch (error) {
            this.loadDemoData();
        } finally {
            this.hideSkeleton();
            this.isLoading = false;
            if (this.options.onRefresh) this.options.onRefresh();
        }
    }

    loadDemoData() {
        this.sessionId = 'demo_' + Date.now();
        this.sliderState.targetX = 200;
        this.sliderState.targetY = 70;
        this.sliderState.puzzleY = 70;
        this.sliderState.puzzleStyle = 0;
        
        this.drawGradientBackground();
        this.updatePuzzlePiece();
        this.drawPuzzleOverlay(0);
    }

    async loadImage(imageUrl) {
        return new Promise((resolve) => {
            const img = new Image();
            img.crossOrigin = 'anonymous';
            img.onload = () => {
                this.canvas.width = 360;
                this.canvas.height = 220;
                this.ctx.drawImage(img, 0, 0, this.canvas.width, this.canvas.height);
                resolve();
            };
            img.onerror = () => {
                this.drawGradientBackground();
                resolve();
            };
            img.src = imageUrl;
        });
    }

    drawGradientBackground() {
        const canvas = this.canvas;
        const ctx = this.ctx;
        canvas.width = 360;
        canvas.height = 220;

        const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
        gradient.addColorStop(0, '#667eea');
        gradient.addColorStop(1, '#764ba2');

        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        ctx.fillStyle = 'rgba(255, 255, 255, 0.9)';
        ctx.font = '18px Arial';
        ctx.textAlign = 'center';
        ctx.fillText('拖动滑块完成验证', canvas.width / 2, canvas.height / 2);
    }

    updatePuzzlePiece() {
        const puzzleY = this.sliderState.puzzleY;
        const style = this.sliderState.puzzleStyle;

        let shape = '';
        switch (style) {
            case 0:
                shape = '<div class="puzzle-shape square"></div>';
                break;
            case 1:
                shape = '<div class="puzzle-shape circle"></div>';
                break;
            case 2:
                shape = '<div class="puzzle-shape triangle"></div>';
                break;
            case 3:
                shape = '<div class="puzzle-shape diamond"></div>';
                break;
            default:
                shape = '<div class="puzzle-shape square"></div>';
        }

        this.elements.puzzle.innerHTML = shape;
        this.elements.puzzle.style.top = puzzleY + 'px';
        this.elements.puzzle.style.left = '0px';
    }

    async verify() {
        this.isLoading = true;
        this.elements.text.textContent = '验证中...';
        
        const payload = {
            session_id: this.sessionId,
            x: Math.round(this.sliderState.currentX * 100) / 100,
            y: Math.round(this.sliderState.puzzleY * 100) / 100,
            behavior_data: this.trajectoryData,
            speed_data: this.calculateSpeedData(),
            trajectory_metadata: {
                version: this.trajectoryVersion,
                point_count: this.trajectoryData.length,
                start_time: this.speedData.startTime,
                end_time: this.speedData.endTime,
                duration: Math.round(this.speedData.endTime - this.speedData.startTime),
                sampling_quality: this.calculateSamplingQuality()
            },
            device_info: this.collectDeviceInfo()
        };

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            const data = await response.json();
            if (data.success) {
                this.handleSuccess();
            } else {
                this.handleError(data.message || '验证失败');
            }
        } catch (error) {
            this.handleError('网络错误，请重试');
        } finally {
            this.isLoading = false;
        }
    }

    calculateSpeedData() {
        const duration = (this.speedData.endTime - this.speedData.startTime) / 1000;
        const totalVelocity = this.speedData.points.reduce((sum, p) => sum + Math.abs(p.speed), 0);
        const avgVelocity = this.speedData.points.length > 0 ? totalVelocity / this.speedData.points.length : 0;
        
        return {
            start_time: Math.round(this.speedData.startTime),
            end_time: Math.round(this.speedData.endTime),
            distance: Math.round(this.speedData.distance * 100) / 100,
            average_speed: Math.round(avgVelocity * 100) / 100,
            max_speed: Math.round(this.speedData.maxSpeed * 100) / 100,
            point_count: this.speedData.points.length
        };
    }
    
    calculateSamplingQuality() {
        if (this.trajectoryData.length < 2) return 0;
        
        const duration = this.speedData.endTime - this.speedData.startTime;
        if (duration <= 0) return 0;
        
        const expectedRate = 60;
        const actualRate = (this.trajectoryData.length / duration) * 1000;
        
        return Math.min(1, actualRate / expectedRate);
    }
    
    collectDeviceInfo() {
        return {
            touch_capable: 'ontouchstart' in window,
            max_touch_points: navigator.maxTouchPoints || 0,
            device_pixel_ratio: window.devicePixelRatio || 1,
            screen_width: window.screen.width,
            screen_height: window.screen.height,
            color_depth: window.screen.colorDepth,
            language: navigator.language,
            platform: navigator.platform,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
        };
    }

    handleSuccess() {
        this.elements.button.classList.add('success');
        this.elements.text.textContent = '验证成功';
        this.playSuccessAnimation();
        
        if (this.options.onSuccess) {
            this.options.onSuccess({ type: 'slider', session_id: this.sessionId });
        }
    }

    handleError(message) {
        this.elements.button.classList.add('error');
        this.elements.text.textContent = message;
        this.playErrorAnimation();
        
        if (this.options.onError) {
            this.options.onError({ type: 'slider', error: message });
        }
        
        setTimeout(() => this.reset(), 1500);
    }

    playSuccessAnimation() {
        const button = this.elements.button;
        const finalX = this.sliderState.currentX;
        
        button.style.transition = 'transform 0.3s ease';
        button.style.transform = 'scale(1.1)';
        
        setTimeout(() => {
            button.style.transform = 'scale(1)';
            this.createSuccessParticles();
        }, 300);
    }

    createSuccessParticles() {
        const container = this.elements.container;
        const rect = container.getBoundingClientRect();
        
        for (let i = 0; i < 8; i++) {
            const particle = document.createElement('div');
            particle.className = 'success-particle';
            particle.style.cssText = `
                position: absolute;
                left: ${rect.left + this.sliderState.currentX}px;
                top: ${rect.top + 20}px;
                width: 8px;
                height: 8px;
                background: #28a745;
                border-radius: 50%;
                pointer-events: none;
                z-index: 100;
            `;
            document.body.appendChild(particle);
            
            const angle = (i / 8) * Math.PI * 2;
            const velocity = 50 + Math.random() * 50;
            let x = 0, y = 0, opacity = 1;
            
            const animate = () => {
                x += Math.cos(angle) * velocity * 0.02;
                y += Math.sin(angle) * velocity * 0.02;
                opacity -= 0.03;
                
                particle.style.transform = `translate(${x}px, ${y}px)`;
                particle.style.opacity = opacity;
                
                if (opacity > 0) {
                    requestAnimationFrame(animate);
                } else {
                    particle.remove();
                }
            };
            requestAnimationFrame(animate);
        }
    }

    playErrorAnimation() {
        const button = this.elements.button;
        let shakeCount = 0;
        const maxShakes = 4;
        
        const shake = () => {
            if (shakeCount >= maxShakes) {
                button.style.left = '2px';
                return;
            }
            
            const direction = shakeCount % 2 === 0 ? -1 : 1;
            button.style.transform = `translateX(${direction * 10}px)`;
            shakeCount++;
            
            setTimeout(() => {
                button.style.transform = '';
                setTimeout(shake, 50);
            }, 50);
        };
        shake();
    }

    reset() {
        this.sliderState.currentX = 0;
        this.elements.button.style.left = '2px';
        this.elements.button.classList.remove('success', 'error', 'dragging');
        this.elements.track.style.width = '0px';
        this.elements.text.textContent = '拖动滑块完成验证';
        this.elements.puzzle.style.left = '0px';
        this.trajectoryData = [];
        this.speedData = { points: [], startTime: 0, endTime: 0, distance: 0, maxSpeed: 0 };
        this.drawPuzzleOverlay(0);
    }

    showSkeleton() {
        this.elements.skeleton.style.display = 'block';
        this.elements.skeleton.classList.add('active');
    }

    hideSkeleton() {
        this.elements.skeleton.classList.remove('active');
        setTimeout(() => {
            this.elements.skeleton.style.display = 'none';
        }, 300);
    }

    destroy() {
        this.container.innerHTML = '';
    }
}

// 导出组件
if (typeof module !== 'undefined' && module.exports) {
    module.exports = SliderCaptcha;
}