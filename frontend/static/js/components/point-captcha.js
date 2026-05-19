/**
 * HJTPX 点选验证码组件
 * 支持图片点选验证、点击动画、状态反馈
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
            ...options
        };

        this.state = {
            isLoaded: false,
            isVerifying: false,
            selectedPoints: [],
            targetPoints: [],
            sessionId: null,
            imageLoaded: false
        };

        this.elements = {};
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
                    <span class="captcha-title">请点击图中所有的 <span class="target-word"></span></span>
                    <button class="captcha-refresh-btn" id="point-refresh-btn" aria-label="刷新验证码">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
                <div class="captcha-image-wrapper">
                    <img class="captcha-image" alt="验证码图片" />
                    <canvas class="point-layer" width="360" height="220"></canvas>
                    <div class="point-skeleton"></div>
                </div>
                <div class="captcha-feedback"></div>
                <div class="captcha-actions">
                    <button class="btn-reset" id="point-reset-btn">重置选择</button>
                    <button class="btn-verify" id="point-verify-btn">确认验证</button>
                </div>
            </div>
        `;

        this.elements = {
            image: this.container.querySelector('.captcha-image'),
            canvas: this.container.querySelector('.point-layer'),
            ctx: this.container.querySelector('.point-layer').getContext('2d'),
            refresh: this.container.querySelector('#point-refresh-btn'),
            reset: this.container.querySelector('#point-reset-btn'),
            verify: this.container.querySelector('#point-verify-btn'),
            feedback: this.container.querySelector('.captcha-feedback'),
            title: this.container.querySelector('.target-word'),
            skeleton: this.container.querySelector('.point-skeleton')
        };
    }

    bindEvents() {
        this.elements.image.addEventListener('click', (e) => this.handleImageClick(e));
        this.elements.canvas.addEventListener('click', (e) => this.handleCanvasClick(e));
        this.elements.refresh.addEventListener('click', () => this.refresh());
        this.elements.reset.addEventListener('click', () => this.reset());
        this.elements.verify.addEventListener('click', () => this.verify());
    }

    handleImageClick(e) {
        if (this.state.isVerifying || !this.state.isLoaded) return;
        const rect = this.elements.canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        this.addPoint(x, y);
    }

    handleCanvasClick(e) {
        if (this.state.isVerifying || !this.state.isLoaded) return;
        const rect = this.elements.canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        this.addPoint(x, y);
    }

    addPoint(x, y) {
        const exists = this.state.selectedPoints.some(p => 
            Math.abs(p.x - x) < 20 && Math.abs(p.y - y) < 20
        );

        if (!exists) {
            const point = { x, y, index: this.state.selectedPoints.length };
            this.state.selectedPoints.push(point);
            this.drawPoint(point);
            this.playClickAnimation(x, y);
            this.updateFeedback();
        }
    }

    drawPoint(point) {
        const ctx = this.elements.ctx;
        ctx.save();
        
        ctx.beginPath();
        ctx.arc(point.x, point.y, 12, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(40, 167, 69, 0.3)';
        ctx.fill();
        ctx.strokeStyle = '#28a745';
        ctx.lineWidth = 2;
        ctx.stroke();
        
        ctx.beginPath();
        ctx.arc(point.x, point.y, 6, 0, Math.PI * 2);
        ctx.fillStyle = '#28a745';
        ctx.fill();
        
        ctx.fillStyle = '#fff';
        ctx.font = 'bold 12px Arial';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(point.index + 1, point.x, point.y);
        
        ctx.restore();
    }

    playClickAnimation(x, y) {
        const ripple = document.createElement('div');
        ripple.className = 'click-ripple';
        ripple.style.cssText = `
            position: absolute;
            left: ${x}px;
            top: ${y}px;
            width: 20px;
            height: 20px;
            border-radius: 50%;
            background: rgba(40, 167, 69, 0.4);
            transform: translate(-50%, -50%) scale(0);
            animation: ripple 0.6s ease-out forwards;
            pointer-events: none;
        `;
        this.elements.canvas.parentElement.appendChild(ripple);
        
        setTimeout(() => ripple.remove(), 600);
    }

    updateFeedback() {
        const count = this.state.selectedPoints.length;
        const target = this.state.targetPoints.length;
        
        if (count === 0) {
            this.elements.feedback.textContent = '';
        } else {
            this.elements.feedback.textContent = `已选择 ${count} 个点${target > 0 ? `，目标 ${target} 个` : ''}`;
        }
    }

    async refresh() {
        this.state.isLoaded = false;
        this.state.isVerifying = false;
        this.state.selectedPoints = [];
        this.showSkeleton();
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/point`);
            if (response.ok) {
                const data = await response.json();
                this.state.sessionId = data.session_id;
                this.state.targetPoints = data.target_points || [];
                this.elements.title.textContent = data.target_word || '目标对象';
                
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
            if (this.options.onRefresh) this.options.onRefresh();
        }
    }

    loadDemoData() {
        this.state.sessionId = 'demo_' + Date.now();
        this.state.targetPoints = [
            { x: 80, y: 80 },
            { x: 180, y: 120 },
            { x: 280, y: 90 }
        ];
        this.elements.title.textContent = '猫';
        
        this.drawDemoImage();
    }

    async loadImage(imageUrl) {
        return new Promise((resolve) => {
            const img = new Image();
            img.crossOrigin = 'anonymous';
            img.onload = () => {
                this.elements.image.src = img.src;
                this.state.imageLoaded = true;
                resolve();
            };
            img.onerror = () => {
                this.drawDemoImage();
                resolve();
            };
            img.src = imageUrl;
        });
    }

    drawDemoImage() {
        const canvas = this.elements.image;
        const ctx = document.createElement('canvas').getContext('2d');
        const tempCanvas = document.createElement('canvas');
        tempCanvas.width = 360;
        tempCanvas.height = 220;
        const tempCtx = tempCanvas.getContext('2d');

        const gradient = tempCtx.createLinearGradient(0, 0, 360, 220);
        gradient.addColorStop(0, '#f093fb');
        gradient.addColorStop(1, '#f5576c');
        tempCtx.fillStyle = gradient;
        tempCtx.fillRect(0, 0, 360, 220);

        tempCtx.fillStyle = 'rgba(255, 255, 255, 0.9)';
        tempCtx.font = 'bold 20px Arial';
        tempCtx.textAlign = 'center';
        tempCtx.fillText('点选验证演示', 180, 110);

        this.elements.image.src = tempCanvas.toDataURL();
        this.state.imageLoaded = true;
    }

    clearCanvas() {
        const ctx = this.elements.ctx;
        ctx.clearRect(0, 0, 360, 220);
    }

    async verify() {
        if (this.state.isVerifying || this.state.selectedPoints.length === 0) return;
        
        this.state.isVerifying = true;
        this.elements.verify.disabled = true;
        this.elements.verify.textContent = '验证中...';
        
        const payload = {
            session_id: this.state.sessionId,
            selected_points: this.state.selectedPoints,
            target_points: this.state.targetPoints.length > 0 ? this.state.targetPoints : null
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
        
        this.createSuccessRays();
    }

    createSuccessRays() {
        const canvas = this.elements.canvas;
        const ctx = this.elements.ctx;
        
        ctx.save();
        ctx.globalCompositeOperation = 'lighter';
        
        this.state.selectedPoints.forEach((point, index) => {
            setTimeout(() => {
                for (let i = 0; i < 6; i++) {
                    const angle = (i / 6) * Math.PI * 2;
                    ctx.beginPath();
                    ctx.moveTo(point.x, point.y);
                    ctx.lineTo(
                        point.x + Math.cos(angle) * 40,
                        point.y + Math.sin(angle) * 40
                    );
                    ctx.strokeStyle = `rgba(40, 167, 69, ${0.8 - index * 0.2})`;
                    ctx.lineWidth = 2;
                    ctx.stroke();
                }
            }, index * 100);
        });
        
        ctx.restore();
    }

    playErrorAnimation() {
        const canvas = this.elements.canvas;
        let shakeCount = 0;
        
        const shake = () => {
            if (shakeCount >= 3) {
                canvas.style.transform = '';
                return;
            }
            
            canvas.style.transform = shakeCount % 2 === 0 ? 'translateX(-5px)' : 'translateX(5px)';
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
        this.clearCanvas();
        this.elements.feedback.textContent = '';
        this.elements.feedback.className = 'captcha-feedback';
        this.elements.canvas.classList.remove('success');
        this.elements.canvas.style.borderColor = '';
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

if (typeof module !== 'undefined' && module.exports) {
    module.exports = PointCaptcha;
}