/**
 * HJTPX 点选验证码组件 - 增强版
 * 支持图片点选验证、点击动画、状态反馈、时序分析、安全防护
 */

class PointCaptcha {
    constructor(container, options = {}) {
        this.container = container;
        this.options = {
            apiBase: '/api/v1',
            onSuccess: null,
            onError: null,
            onRefresh: null,
            gridSize: 3,
            targetCount: 3,
            tolerance: 15,
            minClickInterval: 200,
            maxClickInterval: 5000,
            enableTimingAnalysis: true,
            enableSecurityCheck: true,
            ...options
        };

        this.state = {
            isLoaded: false,
            isVerifying: false,
            selectedPoints: [],
            targetPoints: [],
            sessionId: null,
            imageLoaded: false,
            clickHistory: [],
            startTime: null,
            mouseTrajectory: [],
            suspiciousScore: 0,
            totalClicks: 0
        };

        this.elements = {};
        this.imageWidth = 360;
        this.imageHeight = 220;
        this.targetObjects = [];
        this.currentTargetIndex = 0;
        this.animationId = null;
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.refresh();
    }

    render() {
        this.container.innerHTML = `
            <div class="point-captcha">
                <div class="captcha-header">
                    <span class="captcha-title">请按顺序点击图中的 <span class="target-word"></span></span>
                    <button class="captcha-refresh-btn" id="point-refresh-btn" aria-label="刷新验证码">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
                <div class="captcha-image-wrapper">
                    <canvas class="captcha-canvas" width="360" height="220"></canvas>
                    <div class="point-skeleton"></div>
                    <div class="click-ripple-container"></div>
                </div>
                <div class="captcha-progress">
                    <div class="progress-dots"></div>
                </div>
                <div class="captcha-feedback"></div>
                <div class="captcha-actions">
                    <button class="btn-reset" id="point-reset-btn">重置选择</button>
                    <button class="btn-verify" id="point-verify-btn">确认验证</button>
                </div>
            </div>
        `;

        this.elements = {
            canvas: this.container.querySelector('.captcha-canvas'),
            ctx: this.container.querySelector('.captcha-canvas').getContext('2d'),
            refresh: this.container.querySelector('#point-refresh-btn'),
            reset: this.container.querySelector('#point-reset-btn'),
            verify: this.container.querySelector('#point-verify-btn'),
            feedback: this.container.querySelector('.captcha-feedback'),
            title: this.container.querySelector('.target-word'),
            skeleton: this.container.querySelector('.point-skeleton'),
            progressDots: this.container.querySelector('.progress-dots'),
            rippleContainer: this.container.querySelector('.click-ripple-container')
        };
    }

    bindEvents() {
        this.elements.canvas.addEventListener('click', (e) => this.handleCanvasClick(e));
        this.elements.canvas.addEventListener('mousemove', (e) => this.handleMouseMove(e));
        this.elements.canvas.addEventListener('mouseleave', (e) => this.handleMouseLeave(e));
        this.elements.refresh.addEventListener('click', () => this.refresh());
        this.elements.reset.addEventListener('click', () => this.reset());
        this.elements.verify.addEventListener('click', () => this.verify());
    }

    handleMouseMove(e) {
        if (this.state.isVerifying || !this.state.isLoaded) return;
        
        const rect = this.elements.canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        
        this.state.mouseTrajectory.push({
            x: x,
            y: y,
            timestamp: Date.now()
        });
        
        if (this.state.mouseTrajectory.length > 50) {
            this.state.mouseTrajectory.shift();
        }
    }

    handleMouseLeave(e) {
        this.state.mouseTrajectory = [];
    }

    handleCanvasClick(e) {
        if (this.state.isVerifying || !this.state.isLoaded) return;
        
        if (!this.state.startTime) {
            this.state.startTime = Date.now();
        }
        
        const rect = this.elements.canvas.getBoundingClientRect();
        const scaleX = this.imageWidth / rect.width;
        const scaleY = this.imageHeight / rect.height;
        const x = (e.clientX - rect.left) * scaleX;
        const y = (e.clientY - rect.top) * scaleY;
        
        this.processClick(x, y);
    }

    processClick(x, y) {
        const now = Date.now();
        const lastClick = this.state.clickHistory[this.state.clickHistory.length - 1];
        const interval = lastClick ? now - lastClick.timestamp : 0;
        
        this.state.clickHistory.push({
            x: x,
            y: y,
            timestamp: now,
            interval: interval
        });
        
        this.state.totalClicks++;
        
        if (this.options.enableSecurityCheck) {
            this.performSecurityCheck(x, y, interval);
        }
        
        const targetPoint = this.state.targetPoints[this.currentTargetIndex];
        if (!targetPoint) {
            this.handleError('验证数据异常');
            return;
        }
        
        const distance = this.calculateDistance(x, y, targetPoint.x, targetPoint.y);
        const tolerance = this.options.tolerance * (targetPoint.radius || 1);
        
        if (distance <= tolerance) {
            this.handleCorrectClick(x, y, targetPoint);
        } else if (distance <= tolerance * 2) {
            this.playNearMissAnimation(x, y);
            this.updateFeedback('接近了，再试一次', 'warning');
        } else {
            this.playMissAnimation(x, y);
            if (this.state.suspiciousScore > 50) {
                this.handleError('检测到异常行为');
            }
        }
    }

    handleCorrectClick(x, y, targetPoint) {
        const point = {
            x: x,
            y: y,
            index: this.state.selectedPoints.length,
            targetIndex: this.currentTargetIndex,
            timestamp: Date.now()
        };
        
        this.state.selectedPoints.push(point);
        this.drawCorrectPoint(point, targetPoint);
        this.playClickAnimation(x, y);
        this.currentTargetIndex++;
        
        this.updateProgressDots();
        this.updateFeedback();
        
        if (this.currentTargetIndex >= this.state.targetPoints.length) {
            this.elements.verify.disabled = false;
            this.updateFeedback('所有目标已找到，点击确认验证', 'success');
        }
    }

    calculateDistance(x1, y1, x2, y2) {
        return Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2));
    }

    drawCorrectPoint(point, targetPoint) {
        const ctx = this.elements.ctx;
        ctx.save();
        
        const gradient = ctx.createRadialGradient(
            point.x, point.y, 0,
            point.x, point.y, 15
        );
        gradient.addColorStop(0, 'rgba(40, 167, 69, 0.8)');
        gradient.addColorStop(1, 'rgba(40, 167, 69, 0)');
        
        ctx.beginPath();
        ctx.arc(point.x, point.y, 15, 0, Math.PI * 2);
        ctx.fillStyle = gradient;
        ctx.fill();
        
        ctx.beginPath();
        ctx.arc(point.x, point.y, 10, 0, Math.PI * 2);
        ctx.fillStyle = '#28a745';
        ctx.fill();
        
        ctx.beginPath();
        ctx.arc(point.x, point.y, 6, 0, Math.PI * 2);
        ctx.fillStyle = '#fff';
        ctx.fill();
        
        ctx.fillStyle = '#fff';
        ctx.font = 'bold 10px Arial';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(point.index + 1, point.x, point.y);
        
        ctx.restore();
    }

    playClickAnimation(x, y) {
        const container = this.elements.rippleContainer;
        const rect = this.elements.canvas.getBoundingClientRect();
        const scaleX = rect.width / this.imageWidth;
        const scaleY = rect.height / this.imageHeight;
        
        for (let i = 0; i < 3; i++) {
            setTimeout(() => {
                const ripple = document.createElement('div');
                ripple.className = 'click-ripple';
                ripple.style.cssText = `
                    position: absolute;
                    left: ${x * scaleX}px;
                    top: ${y * scaleY}px;
                    width: 8px;
                    height: 8px;
                    border-radius: 50%;
                    background: rgba(40, 167, 69, ${0.8 - i * 0.2});
                    transform: translate(-50%, -50%) scale(${1 + i * 0.5});
                    animation: ripple 0.5s ease-out forwards;
                    pointer-events: none;
                `;
                container.appendChild(ripple);
                
                setTimeout(() => ripple.remove(), 500);
            }, i * 100);
        }
    }

    playNearMissAnimation(x, y) {
        const ctx = this.elements.ctx;
        const rect = this.elements.canvas.getBoundingClientRect();
        const scaleX = rect.width / this.imageWidth;
        const scaleY = rect.height / this.imageHeight;
        
        const ripple = document.createElement('div');
        ripple.style.cssText = `
            position: absolute;
            left: ${x * scaleX}px;
            top: ${y * scaleY}px;
            width: 30px;
            height: 30px;
            border-radius: 50%;
            border: 2px solid rgba(255, 193, 7, 0.8);
            transform: translate(-50%, -50%) scale(0);
            animation: nearMiss 0.6s ease-out forwards;
            pointer-events: none;
        `;
        this.elements.rippleContainer.appendChild(ripple);
        
        setTimeout(() => ripple.remove(), 600);
    }

    playMissAnimation(x, y) {
        const rect = this.elements.canvas.getBoundingClientRect();
        const scaleX = rect.width / this.imageWidth;
        const scaleY = rect.height / this.imageHeight;
        
        const ripple = document.createElement('div');
        ripple.style.cssText = `
            position: absolute;
            left: ${x * scaleX}px;
            top: ${y * scaleY}px;
            width: 20px;
            height: 20px;
            border-radius: 50%;
            background: rgba(220, 53, 69, 0.4);
            transform: translate(-50%, -50%) scale(0);
            animation: miss 0.4s ease-out forwards;
            pointer-events: none;
        `;
        this.elements.rippleContainer.appendChild(ripple);
        
        setTimeout(() => ripple.remove(), 400);
    }

    updateProgressDots() {
        const dots = this.elements.progressDots;
        dots.innerHTML = '';
        
        this.state.targetPoints.forEach((_, index) => {
            const dot = document.createElement('span');
            dot.className = 'progress-dot';
            if (index < this.currentTargetIndex) {
                dot.classList.add('completed');
            } else if (index === this.currentTargetIndex) {
                dot.classList.add('current');
            }
            dots.appendChild(dot);
        });
    }

    updateFeedback(message, type) {
        if (message) {
            this.elements.feedback.textContent = message;
            this.elements.feedback.className = `captcha-feedback ${type || ''}`;
        } else {
            const count = this.state.selectedPoints.length;
            const target = this.state.targetPoints.length;
            
            if (count === 0) {
                this.elements.feedback.textContent = '';
            } else {
                this.elements.feedback.textContent = `已选择 ${count}/${target} 个目标`;
                this.elements.feedback.className = 'captcha-feedback';
            }
        }
    }

    async refresh() {
        this.state.isLoaded = false;
        this.state.isVerifying = false;
        this.state.selectedPoints = [];
        this.state.clickHistory = [];
        this.state.startTime = null;
        this.state.mouseTrajectory = [];
        this.state.suspiciousScore = 0;
        this.state.totalClicks = 0;
        this.currentTargetIndex = 0;
        this.showSkeleton();
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/point`);
            if (response.ok) {
                const data = await response.json();
                this.state.sessionId = data.session_id;
                this.state.targetPoints = data.target_points || [];
                this.elements.title.textContent = data.target_word || '目标对象';
                this.imageWidth = data.width || 360;
                this.imageHeight = data.height || 220;
                
                this.elements.canvas.width = this.imageWidth;
                this.elements.canvas.height = this.imageHeight;
                
                await this.loadImage(data.image_url);
            } else {
                this.loadDemoData();
            }
        } catch (error) {
            this.loadDemoData();
        } finally {
            this.hideSkeleton();
            this.state.isLoaded = true;
            this.clearCanvas();
            this.drawGeneratedImage();
            this.updateProgressDots();
            this.elements.verify.disabled = true;
            if (this.options.onRefresh) this.options.onRefresh();
        }
    }

    loadDemoData() {
        this.state.sessionId = 'demo_' + Date.now();
        this.elements.title.textContent = '圆形';
        this.generateRandomTargets();
        this.generateRandomBackground();
    }

    generateRandomTargets() {
        const targetCount = this.options.targetCount || 3;
        const padding = 30;
        const targets = [];
        const shapes = ['circle', 'square', 'triangle', 'star'];
        const colors = ['#e74c3c', '#3498db', '#9b59b6', '#1abc9c', '#f39c12'];
        
        for (let i = 0; i < targetCount; i++) {
            let attempts = 0;
            let newTarget;
            
            do {
                newTarget = {
                    x: padding + Math.random() * (this.imageWidth - padding * 2),
                    y: padding + Math.random() * (this.imageHeight - padding * 2),
                    radius: 15 + Math.random() * 10,
                    color: colors[i % colors.length],
                    shape: shapes[i % shapes.length],
                    rotation: Math.random() * Math.PI * 2
                };
                attempts++;
            } while (this.checkOverlap(newTarget, targets) && attempts < 50);
            
            if (attempts < 50) {
                targets.push(newTarget);
            }
        }
        
        this.state.targetPoints = targets;
    }

    checkOverlap(newTarget, existingTargets) {
        const minDistance = 50;
        
        for (const target of existingTargets) {
            const distance = this.calculateDistance(newTarget.x, newTarget.y, target.x, target.y);
            if (distance < minDistance) {
                return true;
            }
        }
        
        return false;
    }

    generateRandomBackground() {
        this.backgroundType = Math.floor(Math.random() * 5);
        this.backgroundColors = this.generateColorPalette();
    }

    generateColorPalette() {
        const baseHue = Math.random() * 360;
        const palette = [];
        
        for (let i = 0; i < 5; i++) {
            palette.push(`hsl(${baseHue + i * 15}, ${30 + Math.random() * 40}%, ${50 + Math.random() * 30}%)`);
        }
        
        return palette;
    }

    drawGeneratedImage() {
        const ctx = this.elements.ctx;
        ctx.clearRect(0, 0, this.imageWidth, this.imageHeight);
        
        switch (this.backgroundType) {
            case 0:
                this.drawGradientBackground(ctx);
                break;
            case 1:
                this.drawGridBackground(ctx);
                break;
            case 2:
                this.drawNoiseBackground(ctx);
                break;
            case 3:
                this.drawPatternBackground(ctx);
                break;
            case 4:
                this.drawGeometricBackground(ctx);
                break;
        }
        
        this.drawTargetObjects(ctx);
        this.drawHints(ctx);
    }

    drawGradientBackground(ctx) {
        const gradient = ctx.createLinearGradient(
            0, 0,
            this.imageWidth, this.imageHeight
        );
        gradient.addColorStop(0, this.backgroundColors[0]);
        gradient.addColorStop(0.5, this.backgroundColors[2]);
        gradient.addColorStop(1, this.backgroundColors[4]);
        
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, this.imageWidth, this.imageHeight);
        
        ctx.globalAlpha = 0.1;
        for (let i = 0; i < 20; i++) {
            ctx.beginPath();
            const x = Math.random() * this.imageWidth;
            const y = Math.random() * this.imageHeight;
            const r = 20 + Math.random() * 60;
            ctx.arc(x, y, r, 0, Math.PI * 2);
            ctx.fillStyle = this.backgroundColors[i % 5];
            ctx.fill();
        }
        ctx.globalAlpha = 1;
    }

    drawGridBackground(ctx) {
        ctx.fillStyle = this.backgroundColors[0];
        ctx.fillRect(0, 0, this.imageWidth, this.imageHeight);
        
        ctx.strokeStyle = this.backgroundColors[2];
        ctx.lineWidth = 0.5;
        
        const gridSize = 30;
        
        for (let x = 0; x <= this.imageWidth; x += gridSize) {
            ctx.beginPath();
            ctx.moveTo(x, 0);
            ctx.lineTo(x, this.imageHeight);
            ctx.stroke();
        }
        
        for (let y = 0; y <= this.imageHeight; y += gridSize) {
            ctx.beginPath();
            ctx.moveTo(0, y);
            ctx.lineTo(this.imageWidth, y);
            ctx.stroke();
        }
        
        for (let i = 0; i < 30; i++) {
            ctx.beginPath();
            const x = Math.random() * this.imageWidth;
            const y = Math.random() * this.imageHeight;
            const r = 2 + Math.random() * 5;
            ctx.arc(x, y, r, 0, Math.PI * 2);
            ctx.fillStyle = this.backgroundColors[3];
            ctx.fill();
        }
    }

    drawNoiseBackground(ctx) {
        ctx.fillStyle = this.backgroundColors[0];
        ctx.fillRect(0, 0, this.imageWidth, this.imageHeight);
        
        const imageData = ctx.getImageData(0, 0, this.imageWidth, this.imageHeight);
        const data = imageData.data;
        
        for (let i = 0; i < data.length; i += 4) {
            const noise = (Math.random() - 0.5) * 30;
            data[i] = Math.max(0, Math.min(255, data[i] + noise));
            data[i + 1] = Math.max(0, Math.min(255, data[i + 1] + noise));
            data[i + 2] = Math.max(0, Math.min(255, data[i + 2] + noise));
        }
        
        ctx.putImageData(imageData, 0, 0);
    }

    drawPatternBackground(ctx) {
        ctx.fillStyle = this.backgroundColors[0];
        ctx.fillRect(0, 0, this.imageWidth, this.imageHeight);
        
        const patternSize = 20;
        
        for (let x = 0; x < this.imageWidth; x += patternSize) {
            for (let y = 0; y < this.imageHeight; y += patternSize) {
                if ((Math.floor(x / patternSize) + Math.floor(y / patternSize)) % 2 === 0) {
                    ctx.fillStyle = this.backgroundColors[1];
                    ctx.fillRect(x, y, patternSize, patternSize);
                }
            }
        }
        
        ctx.globalAlpha = 0.3;
        for (let i = 0; i < 15; i++) {
            ctx.beginPath();
            const x = Math.random() * this.imageWidth;
            const y = Math.random() * this.imageHeight;
            const r = 5 + Math.random() * 15;
            ctx.arc(x, y, r, 0, Math.PI * 2);
            ctx.fillStyle = this.backgroundColors[4];
            ctx.fill();
        }
        ctx.globalAlpha = 1;
    }

    drawGeometricBackground(ctx) {
        const gradient = ctx.createRadialGradient(
            this.imageWidth / 2, this.imageHeight / 2, 0,
            this.imageWidth / 2, this.imageHeight / 2, this.imageWidth / 2
        );
        gradient.addColorStop(0, this.backgroundColors[0]);
        gradient.addColorStop(1, this.backgroundColors[2]);
        
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, this.imageWidth, this.imageHeight);
        
        for (let i = 0; i < 8; i++) {
            ctx.save();
            ctx.translate(
                this.imageWidth * (0.2 + Math.random() * 0.6),
                this.imageHeight * (0.2 + Math.random() * 0.6)
            );
            ctx.rotate(Math.random() * Math.PI);
            
            ctx.beginPath();
            const size = 20 + Math.random() * 40;
            
            if (i % 3 === 0) {
                ctx.rect(-size / 2, -size / 2, size, size);
            } else if (i % 3 === 1) {
                ctx.arc(0, 0, size / 2, 0, Math.PI * 2);
            } else {
                ctx.moveTo(0, -size / 2);
                ctx.lineTo(size / 2, size / 2);
                ctx.lineTo(-size / 2, size / 2);
                ctx.closePath();
            }
            
            ctx.fillStyle = this.backgroundColors[(i + 2) % 5];
            ctx.globalAlpha = 0.2;
            ctx.fill();
            ctx.restore();
        }
        
        ctx.globalAlpha = 1;
    }

    drawTargetObjects(ctx) {
        this.state.targetPoints.forEach((target, index) => {
            ctx.save();
            ctx.translate(target.x, target.y);
            ctx.rotate(target.rotation);
            
            ctx.shadowColor = 'rgba(0, 0, 0, 0.3)';
            ctx.shadowBlur = 10;
            ctx.shadowOffsetX = 2;
            ctx.shadowOffsetY = 2;
            
            switch (target.shape) {
                case 'circle':
                    this.drawCircle(ctx, target);
                    break;
                case 'square':
                    this.drawSquare(ctx, target);
                    break;
                case 'triangle':
                    this.drawTriangle(ctx, target);
                    break;
                case 'star':
                    this.drawStar(ctx, target);
                    break;
            }
            
            ctx.restore();
        });
    }

    drawCircle(ctx, target) {
        const gradient = ctx.createRadialGradient(0, 0, 0, 0, 0, target.radius);
        gradient.addColorStop(0, this.lightenColor(target.color, 30));
        gradient.addColorStop(1, target.color);
        
        ctx.beginPath();
        ctx.arc(0, 0, target.radius, 0, Math.PI * 2);
        ctx.fillStyle = gradient;
        ctx.fill();
        
        ctx.strokeStyle = this.darkenColor(target.color, 20);
        ctx.lineWidth = 2;
        ctx.stroke();
        
        ctx.beginPath();
        ctx.arc(-target.radius * 0.3, -target.radius * 0.3, target.radius * 0.2, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
        ctx.fill();
    }

    drawSquare(ctx, target) {
        const gradient = ctx.createLinearGradient(-target.radius, -target.radius, target.radius, target.radius);
        gradient.addColorStop(0, this.lightenColor(target.color, 30));
        gradient.addColorStop(1, target.color);
        
        ctx.beginPath();
        ctx.rect(-target.radius, -target.radius, target.radius * 2, target.radius * 2);
        ctx.fillStyle = gradient;
        ctx.fill();
        
        ctx.strokeStyle = this.darkenColor(target.color, 20);
        ctx.lineWidth = 2;
        ctx.stroke();
        
        ctx.fillStyle = 'rgba(255, 255, 255, 0.3)';
        ctx.fillRect(-target.radius * 0.7, -target.radius * 0.7, target.radius * 0.4, target.radius * 0.4);
    }

    drawTriangle(ctx, target) {
        const gradient = ctx.createLinearGradient(0, -target.radius, 0, target.radius);
        gradient.addColorStop(0, this.lightenColor(target.color, 30));
        gradient.addColorStop(1, target.color);
        
        ctx.beginPath();
        ctx.moveTo(0, -target.radius);
        ctx.lineTo(target.radius, target.radius);
        ctx.lineTo(-target.radius, target.radius);
        ctx.closePath();
        ctx.fillStyle = gradient;
        ctx.fill();
        
        ctx.strokeStyle = this.darkenColor(target.color, 20);
        ctx.lineWidth = 2;
        ctx.stroke();
    }

    drawStar(ctx, target) {
        const gradient = ctx.createRadialGradient(0, 0, 0, 0, 0, target.radius);
        gradient.addColorStop(0, this.lightenColor(target.color, 40));
        gradient.addColorStop(1, target.color);
        
        ctx.beginPath();
        for (let i = 0; i < 5; i++) {
            const angle = (i * 4 * Math.PI) / 5 - Math.PI / 2;
            const x = Math.cos(angle) * target.radius;
            const y = Math.sin(angle) * target.radius;
            
            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        }
        ctx.closePath();
        ctx.fillStyle = gradient;
        ctx.fill();
        
        ctx.strokeStyle = this.darkenColor(target.color, 20);
        ctx.lineWidth = 2;
        ctx.stroke();
    }

    lightenColor(color, percent) {
        const num = parseInt(color.replace('#', ''), 16);
        const amt = Math.round(2.55 * percent);
        const R = Math.min(255, (num >> 16) + amt);
        const G = Math.min(255, ((num >> 8) & 0x00FF) + amt);
        const B = Math.min(255, (num & 0x0000FF) + amt);
        return `rgb(${R}, ${G}, ${B})`;
    }

    darkenColor(color, percent) {
        const num = parseInt(color.replace('#', ''), 16);
        const amt = Math.round(2.55 * percent);
        const R = Math.max(0, (num >> 16) - amt);
        const G = Math.max(0, ((num >> 8) & 0x00FF) - amt);
        const B = Math.max(0, (num & 0x0000FF) - amt);
        return `rgb(${R}, ${G}, ${B})`;
    }

    drawHints(ctx) {
        ctx.font = '12px Arial';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        
        ctx.fillStyle = 'rgba(0, 0, 0, 0.1)';
        ctx.fillRect(5, 5, 80, 20);
        
        ctx.fillStyle = '#fff';
        ctx.fillText('点击圆形', 10, 15);
    }

    async loadImage(imageUrl) {
        return new Promise((resolve) => {
            const img = new Image();
            img.crossOrigin = 'anonymous';
            img.onload = () => {
                this.elements.canvas.width = img.width;
                this.elements.canvas.height = img.height;
                this.imageWidth = img.width;
                this.imageHeight = img.height;
                
                this.elements.ctx.drawImage(img, 0, 0);
                this.state.imageLoaded = true;
                resolve();
            };
            img.onerror = () => {
                this.drawGeneratedImage();
                resolve();
            };
            img.src = imageUrl;
        });
    }

    clearCanvas() {
        const ctx = this.elements.ctx;
        ctx.clearRect(0, 0, this.imageWidth, this.imageHeight);
    }

    async verify() {
        if (this.state.isVerifying || this.state.selectedPoints.length === 0) return;
        
        if (this.options.enableTimingAnalysis) {
            const timingAnalysis = this.analyzeTiming();
            if (!timingAnalysis.isValid) {
                this.handleError('点击时序异常');
                return;
            }
        }
        
        if (this.options.enableSecurityCheck) {
            const securityCheck = this.performFinalSecurityCheck();
            if (!securityCheck.isValid) {
                this.handleError('安全检查未通过');
                return;
            }
        }
        
        this.state.isVerifying = true;
        this.elements.verify.disabled = true;
        this.elements.verify.textContent = '验证中...';
        
        const payload = {
            session_id: this.state.sessionId,
            selected_points: this.state.selectedPoints,
            target_points: this.state.targetPoints,
            timing_data: this.state.clickHistory,
            total_time: Date.now() - this.state.startTime,
            suspicious_score: this.state.suspiciousScore
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
            this.state.isVerifying = false;
            this.elements.verify.disabled = false;
            this.elements.verify.textContent = '确认验证';
        }
    }

    analyzeTiming() {
        const history = this.state.clickHistory;
        const totalTime = Date.now() - this.state.startTime;
        
        if (history.length < 2) {
            return { isValid: true, reason: '数据不足' };
        }
        
        let totalInterval = 0;
        let maxInterval = 0;
        let minInterval = Infinity;
        let suspiciousIntervals = 0;
        
        for (let i = 1; i < history.length; i++) {
            const interval = history[i].interval;
            totalInterval += interval;
            maxInterval = Math.max(maxInterval, interval);
            minInterval = Math.min(minInterval, interval);
            
            if (interval < this.options.minClickInterval) {
                suspiciousIntervals++;
            }
            
            if (interval > this.options.maxClickInterval) {
                suspiciousIntervals++;
            }
        }
        
        const avgInterval = totalInterval / (history.length - 1);
        const variance = history.slice(1).reduce((sum, click, i) => {
            return sum + Math.pow(click.interval - avgInterval, 2);
        }, 0) / (history.length - 1);
        
        const stdDev = Math.sqrt(variance);
        const coefficientOfVariation = avgInterval > 0 ? stdDev / avgInterval : 0;
        
        if (totalTime < 500) {
            return { isValid: false, reason: '完成时间过短', score: 20 };
        }
        
        if (suspiciousIntervals > history.length / 2) {
            return { isValid: false, reason: '点击间隔异常', score: 30 };
        }
        
        if (coefficientOfVariation < 0.1 && history.length > 3) {
            return { isValid: false, reason: '点击间隔过于规律', score: 25 };
        }
        
        return {
            isValid: true,
            avgInterval,
            maxInterval,
            minInterval,
            totalTime,
            coefficientOfVariation
        };
    }

    performSecurityCheck(x, y, interval) {
        if (this.state.totalClicks > 10) {
            this.state.suspiciousScore += 5;
        }
        
        if (interval < 50) {
            this.state.suspiciousScore += 3;
        }
        
        const mouseCoverage = this.calculateMouseCoverage();
        if (mouseCoverage < 0.1 && this.state.totalClicks > 2) {
            this.state.suspiciousScore += 2;
        }
        
        if (this.isClickTooAccurate(x, y)) {
            this.state.suspiciousScore += 1;
        }
        
        this.checkAutomationIndicators();
    }

    calculateMouseCoverage() {
        if (this.state.mouseTrajectory.length < 2) return 0;
        
        const xs = this.state.mouseTrajectory.map(p => p.x);
        const ys = this.state.mouseTrajectory.map(p => p.y);
        
        const minX = Math.min(...xs);
        const maxX = Math.max(...xs);
        const minY = Math.min(...ys);
        const maxY = Math.max(...ys);
        
        const coverage = ((maxX - minX) * (maxY - minY)) / (this.imageWidth * this.imageHeight);
        
        return coverage;
    }

    isClickTooAccurate(x, y) {
        for (const target of this.state.targetPoints) {
            const distance = this.calculateDistance(x, y, target.x, target.y);
            if (distance < 5) {
                return true;
            }
        }
        return false;
    }

    checkAutomationIndicators() {
        if (window.navigator.webdriver) {
            this.state.suspiciousScore += 20;
        }
        
        if (navigator.userAgent.indexOf('HeadlessChrome') !== -1) {
            this.state.suspiciousScore += 15;
        }
        
        if (!window.chrome || navigator.plugins.length === 0) {
            this.state.suspiciousScore += 5;
        }
    }

    performFinalSecurityCheck() {
        if (this.state.suspiciousScore > 50) {
            return { isValid: false, reason: '可疑分数过高', score: this.state.suspiciousScore };
        }
        
        if (this.state.totalClicks > this.state.targetPoints.length * 3) {
            return { isValid: false, reason: '点击次数过多', score: 10 };
        }
        
        const timingAnalysis = this.analyzeTiming();
        if (!timingAnalysis.isValid) {
            return { isValid: false, reason: timingAnalysis.reason, score: timingAnalysis.score || 10 };
        }
        
        return { isValid: true };
    }

    handleSuccess() {
        this.elements.canvas.classList.add('success');
        this.elements.feedback.textContent = '验证成功';
        this.elements.feedback.className = 'captcha-feedback success';
        
        this.playSuccessAnimation();
        
        if (this.options.onSuccess) {
            this.options.onSuccess({ type: 'point', session_id: this.state.sessionId });
        }
    }

    handleError(message) {
        this.elements.feedback.textContent = message;
        this.elements.feedback.className = 'captcha-feedback error';
        this.playErrorAnimation();
        
        if (this.options.onError) {
            this.options.onError({ type: 'point', error: message });
        }
        
        setTimeout(() => this.reset(), 2000);
    }

    playSuccessAnimation() {
        const canvas = this.elements.canvas;
        canvas.style.transition = 'border-color 0.5s ease';
        canvas.style.borderColor = '#28a745';
        
        this.state.selectedPoints.forEach((point, index) => {
            setTimeout(() => {
                this.createSuccessEffect(point);
            }, index * 150);
        });
    }

    createSuccessEffect(point) {
        const ctx = this.elements.ctx;
        ctx.save();
        ctx.globalCompositeOperation = 'lighter';
        
        for (let i = 0; i < 8; i++) {
            const angle = (i / 8) * Math.PI * 2;
            const endX = point.x + Math.cos(angle) * 50;
            const endY = point.y + Math.sin(angle) * 50;
            
            ctx.beginPath();
            ctx.moveTo(point.x, point.y);
            ctx.lineTo(endX, endY);
            ctx.strokeStyle = `rgba(40, 167, 69, ${0.8})`;
            ctx.lineWidth = 2;
            ctx.stroke();
        }
        
        ctx.beginPath();
        ctx.arc(point.x, point.y, 25, 0, Math.PI * 2);
        ctx.strokeStyle = 'rgba(40, 167, 69, 0.6)';
        ctx.lineWidth = 3;
        ctx.stroke();
        
        ctx.restore();
    }

    playErrorAnimation() {
        const canvas = this.elements.canvas;
        let shakeCount = 0;
        
        const shake = () => {
            if (shakeCount >= 4) {
                canvas.style.transform = '';
                return;
            }
            
            canvas.style.transform = shakeCount % 2 === 0 ? 'translateX(-8px)' : 'translateX(8px)';
            shakeCount++;
            
            setTimeout(() => {
                canvas.style.transform = '';
                setTimeout(shake, 100);
            }, 100);
        };
        shake();
    }

    reset() {
        this.state.selectedPoints = [];
        this.state.clickHistory = [];
        this.state.startTime = null;
        this.state.mouseTrajectory = [];
        this.state.suspiciousScore = 0;
        this.state.totalClicks = 0;
        this.currentTargetIndex = 0;
        
        this.clearCanvas();
        this.drawGeneratedImage();
        this.updateProgressDots();
        
        this.elements.feedback.textContent = '';
        this.elements.feedback.className = 'captcha-feedback';
        this.elements.canvas.classList.remove('success');
        this.elements.canvas.style.borderColor = '';
        this.elements.verify.disabled = true;
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
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
        }
        this.container.innerHTML = '';
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = PointCaptcha;
}