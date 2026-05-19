/**
 * HJTPX 旋转/手势/拼图验证码组件
 * 支持旋转验证、手势绘制、拼图匹配
 */

class RotateGestureCaptcha {
    constructor(container, options = {}) {
        this.container = container;
        this.options = {
            apiBase: '/api/v1',
            type: 'rotate',
            onSuccess: null,
            onError: null,
            onRefresh: null,
            ...options
        };

        this.state = {
            sessionId: null,
            isVerifying: false,
            isLoaded: false,
            rotation: 0,
            targetRotation: 0,
            startAngle: 0,
            isDragging: false,
            gesturePoints: [],
            puzzleOffset: 0,
            targetOffset: 0
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
        const type = this.options.type;
        
        if (type === 'rotate') {
            this.renderRotate();
        } else if (type === 'gesture') {
            this.renderGesture();
        } else if (type === 'puzzle') {
            this.renderPuzzle();
        }
    }

    renderRotate() {
        this.container.innerHTML = `
            <div class="rotate-captcha">
                <div class="captcha-header">
                    <span class="captcha-title">将图片旋转到正确方向</span>
                    <button class="captcha-refresh-btn" id="rotate-refresh-btn" aria-label="刷新验证码">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
                <div class="rotate-image-wrapper">
                    <div class="rotate-container">
                        <img class="rotate-image" alt="旋转验证图片" />
                        <div class="rotate-skeleton"></div>
                    </div>
                    <div class="rotation-indicator">
                        <span class="rotation-value">0°</span>
                    </div>
                </div>
                <div class="rotate-controls">
                    <button class="btn-rotate" id="rotate-left" aria-label="向左旋转">
                        <i class="fas fa-rotate-left"></i>
                    </button>
                    <input type="range" class="rotate-slider" id="rotate-slider" 
                           min="0" max="360" value="0" aria-label="旋转角度" />
                    <button class="btn-rotate" id="rotate-right" aria-label="向右旋转">
                        <i class="fas fa-rotate-right"></i>
                    </button>
                </div>
                <div class="captcha-feedback"></div>
                <button class="btn-verify" id="rotate-verify-btn">确认验证</button>
            </div>
        `;

        this.elements = {
            image: this.container.querySelector('.rotate-image'),
            container: this.container.querySelector('.rotate-container'),
            refresh: this.container.querySelector('#rotate-refresh-btn'),
            slider: this.container.querySelector('#rotate-slider'),
            leftBtn: this.container.querySelector('#rotate-left'),
            rightBtn: this.container.querySelector('#rotate-right'),
            indicator: this.container.querySelector('.rotation-value'),
            feedback: this.container.querySelector('.captcha-feedback'),
            verify: this.container.querySelector('#rotate-verify-btn'),
            skeleton: this.container.querySelector('.rotate-skeleton')
        };
    }

    renderGesture() {
        this.container.innerHTML = `
            <div class="gesture-captcha">
                <div class="captcha-header">
                    <span class="captcha-title">请在下方区域绘制手势</span>
                    <button class="captcha-refresh-btn" id="gesture-refresh-btn" aria-label="刷新验证码">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
                <div class="gesture-wrapper">
                    <canvas class="gesture-canvas" width="280" height="200"></canvas>
                    <div class="gesture-skeleton"></div>
                    <div class="gesture-hint">
                        <span>提示：请按指定轨迹绘制</span>
                    </div>
                </div>
                <div class="captcha-feedback"></div>
                <div class="captcha-actions">
                    <button class="btn-reset" id="gesture-reset-btn">清除重绘</button>
                    <button class="btn-verify" id="gesture-verify-btn">确认验证</button>
                </div>
            </div>
        `;

        this.elements = {
            canvas: this.container.querySelector('.gesture-canvas'),
            ctx: this.container.querySelector('.gesture-canvas').getContext('2d'),
            refresh: this.container.querySelector('#gesture-refresh-btn'),
            reset: this.container.querySelector('#gesture-reset-btn'),
            verify: this.container.querySelector('#gesture-verify-btn'),
            feedback: this.container.querySelector('.captcha-feedback'),
            skeleton: this.container.querySelector('.gesture-skeleton')
        };
    }

    renderPuzzle() {
        this.container.innerHTML = `
            <div class="puzzle-captcha">
                <div class="captcha-header">
                    <span class="captcha-title">将拼图拖动到正确位置</span>
                    <button class="captcha-refresh-btn" id="puzzle-refresh-btn" aria-label="刷新验证码">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
                <div class="puzzle-image-wrapper">
                    <div class="puzzle-main">
                        <img class="puzzle-bg-image" alt="拼图背景" />
                        <div class="puzzle-hole"></div>
                        <div class="puzzle-skeleton"></div>
                    </div>
                    <div class="puzzle-slider-container">
                        <div class="puzzle-track"></div>
                        <div class="puzzle-piece">
                            <img class="piece-image" alt="拼图块" />
                        </div>
                    </div>
                </div>
                <div class="captcha-feedback"></div>
                <button class="btn-verify" id="puzzle-verify-btn">确认验证</button>
            </div>
        `;

        this.elements = {
            bgImage: this.container.querySelector('.puzzle-bg-image'),
            pieceImage: this.container.querySelector('.piece-image'),
            hole: this.container.querySelector('.puzzle-hole'),
            piece: this.container.querySelector('.puzzle-piece'),
            container: this.container.querySelector('.puzzle-slider-container'),
            track: this.container.querySelector('.puzzle-track'),
            refresh: this.container.querySelector('#puzzle-refresh-btn'),
            verify: this.container.querySelector('#puzzle-verify-btn'),
            feedback: this.container.querySelector('.captcha-feedback'),
            skeleton: this.container.querySelector('.puzzle-skeleton')
        };
    }

    bindEvents() {
        const type = this.options.type;
        
        if (type === 'rotate') {
            this.bindRotateEvents();
        } else if (type === 'gesture') {
            this.bindGestureEvents();
        } else if (type === 'puzzle') {
            this.bindPuzzleEvents();
        }
    }

    bindRotateEvents() {
        const { refresh, slider, leftBtn, rightBtn, verify, container } = this.elements;
        
        refresh.addEventListener('click', () => this.refresh());
        verify.addEventListener('click', () => this.verify());
        
        slider.addEventListener('input', (e) => {
            this.state.rotation = parseInt(e.target.value);
            this.updateRotation();
        });
        
        leftBtn.addEventListener('click', () => {
            this.state.rotation = (this.state.rotation - 45 + 360) % 360;
            slider.value = this.state.rotation;
            this.updateRotation();
        });
        
        rightBtn.addEventListener('click', () => {
            this.state.rotation = (this.state.rotation + 45) % 360;
            slider.value = this.state.rotation;
            this.updateRotation();
        });

        let startY = 0;
        container.addEventListener('mousedown', (e) => {
            startY = e.clientY;
        });
        
        container.addEventListener('mousemove', (e) => {
            if (startY !== 0) {
                const delta = startY - e.clientY;
                this.state.rotation = (this.state.rotation + delta) % 360;
                slider.value = this.state.rotation;
                this.updateRotation();
                startY = e.clientY;
            }
        });
        
        container.addEventListener('mouseup', () => {
            startY = 0;
        });
    }

    bindGestureEvents() {
        const { refresh, reset, verify, canvas } = this.elements;
        let isDrawing = false;
        
        refresh.addEventListener('click', () => this.refresh());
        reset.addEventListener('click', () => this.resetGesture());
        verify.addEventListener('click', () => this.verify());
        
        const startDraw = (e) => {
            isDrawing = true;
            this.state.gesturePoints = [];
            const rect = canvas.getBoundingClientRect();
            const x = e.type === 'touchstart' ? e.touches[0].clientX : e.clientX;
            const y = e.type === 'touchstart' ? e.touches[0].clientY : e.clientY;
            this.addGesturePoint(x - rect.left, y - rect.top);
            this.elements.ctx.beginPath();
            this.elements.ctx.moveTo(x - rect.left, y - rect.top);
        };
        
        const draw = (e) => {
            if (!isDrawing) return;
            e.preventDefault();
            const rect = canvas.getBoundingClientRect();
            const x = e.type === 'touchmove' ? e.touches[0].clientX : e.clientX;
            const y = e.type === 'touchmove' ? e.touches[0].clientY : e.clientY;
            this.addGesturePoint(x - rect.left, y - rect.top);
            this.elements.ctx.lineTo(x - rect.left, y - rect.top);
            this.elements.ctx.stroke();
        };
        
        const endDraw = () => {
            isDrawing = false;
        };
        
        canvas.addEventListener('mousedown', startDraw);
        canvas.addEventListener('mousemove', draw);
        canvas.addEventListener('mouseup', endDraw);
        canvas.addEventListener('mouseleave', endDraw);
        
        canvas.addEventListener('touchstart', startDraw, { passive: false });
        canvas.addEventListener('touchmove', draw, { passive: false });
        canvas.addEventListener('touchend', endDraw);
    }

    bindPuzzleEvents() {
        const { piece, container, refresh, verify } = this.elements;
        let isDragging = false;
        let startX = 0;
        
        refresh.addEventListener('click', () => this.refresh());
        verify.addEventListener('click', () => this.verify());
        
        const startDrag = (e) => {
            if (this.state.isVerifying) return;
            isDragging = true;
            startX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            piece.classList.add('dragging');
        };
        
        const drag = (e) => {
            if (!isDragging) return;
            e.preventDefault();
            const clientX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX;
            let deltaX = clientX - startX;
            deltaX = Math.max(0, Math.min(deltaX, container.offsetWidth - piece.offsetWidth));
            this.state.puzzleOffset = deltaX;
            this.updatePuzzlePosition();
        };
        
        const endDrag = () => {
            isDragging = false;
            piece.classList.remove('dragging');
        };
        
        piece.addEventListener('mousedown', startDrag);
        piece.addEventListener('touchstart', startDrag, { passive: false });
        
        document.addEventListener('mousemove', drag);
        document.addEventListener('touchmove', drag, { passive: false });
        
        document.addEventListener('mouseup', endDrag);
        document.addEventListener('touchend', endDrag);
    }

    addGesturePoint(x, y) {
        this.state.gesturePoints.push({
            x: Math.round(x),
            y: Math.round(y),
            timestamp: Date.now()
        });
    }

    updateRotation() {
        this.elements.image.style.transform = `rotate(${this.state.rotation}deg)`;
        this.elements.indicator.textContent = `${this.state.rotation}°`;
    }

    updatePuzzlePosition() {
        this.elements.piece.style.left = this.state.puzzleOffset + 'px';
        this.elements.track.style.width = this.state.puzzleOffset + 'px';
    }

    async refresh() {
        this.state.isVerifying = false;
        this.state.isLoaded = false;
        this.showSkeleton();
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/${this.options.type}`);
            if (response.ok) {
                const data = await response.json();
                this.state.sessionId = data.session_id;
                
                if (this.options.type === 'rotate') {
                    this.state.targetRotation = data.target_rotation || 0;
                    await this.loadImage(this.elements.image, data.image_url);
                } else if (this.options.type === 'puzzle') {
                    this.state.targetOffset = data.target_offset || 0;
                    await this.loadImage(this.elements.bgImage, data.image_url);
                    await this.loadImage(this.elements.pieceImage, data.piece_url);
                    this.elements.hole.style.left = this.state.targetOffset + 'px';
                }
                
                this.state.rotation = 0;
                this.state.puzzleOffset = 0;
                this.state.gesturePoints = [];
            } else {
                this.loadDemoData();
            }
        } catch (error) {
            this.loadDemoData();
        } finally {
            this.hideSkeleton();
            this.state.isLoaded = true;
            if (this.options.type === 'rotate') {
                this.elements.slider.value = 0;
                this.updateRotation();
            } else if (this.options.type === 'gesture') {
                this.clearGestureCanvas();
            } else if (this.options.type === 'puzzle') {
                this.updatePuzzlePosition();
            }
            if (this.options.onRefresh) this.options.onRefresh();
        }
    }

    loadDemoData() {
        this.state.sessionId = 'demo_' + Date.now();
        
        if (this.options.type === 'rotate') {
            this.state.targetRotation = 90 + Math.floor(Math.random() * 180);
            this.drawRotateDemoImage();
        } else if (this.options.type === 'gesture') {
            this.state.gesturePoints = [];
        } else if (this.options.type === 'puzzle') {
            this.state.targetOffset = 100 + Math.floor(Math.random() * 150);
            this.drawPuzzleDemoImage();
        }
    }

    drawRotateDemoImage() {
        const canvas = document.createElement('canvas');
        canvas.width = 200;
        canvas.height = 200;
        const ctx = canvas.getContext('2d');
        
        const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
        gradient.addColorStop(0, '#4facfe');
        gradient.addColorStop(1, '#00f2fe');
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        
        ctx.fillStyle = '#fff';
        ctx.font = 'bold 30px Arial';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('↑', 100, 100);
        
        this.elements.image.src = canvas.toDataURL();
    }

    drawPuzzleDemoImage() {
        const canvas = document.createElement('canvas');
        canvas.width = 320;
        canvas.height = 160;
        const ctx = canvas.getContext('2d');
        
        const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
        gradient.addColorStop(0, '#667eea');
        gradient.addColorStop(1, '#764ba2');
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        
        ctx.fillStyle = '#fff';
        ctx.font = 'bold 24px Arial';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('拼图验证演示', 160, 80);
        
        this.elements.bgImage.src = canvas.toDataURL();
        this.elements.pieceImage.src = canvas.toDataURL();
        this.elements.hole.style.left = this.state.targetOffset + 'px';
    }

    async loadImage(imgElement, url) {
        return new Promise((resolve) => {
            const img = new Image();
            img.crossOrigin = 'anonymous';
            img.onload = () => {
                imgElement.src = img.src;
                resolve();
            };
            img.onerror = () => {
                resolve();
            };
            img.src = url;
        });
    }

    clearGestureCanvas() {
        const ctx = this.elements.ctx;
        ctx.clearRect(0, 0, this.elements.canvas.width, this.elements.canvas.height);
        ctx.strokeStyle = '#667eea';
        ctx.lineWidth = 4;
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';
    }

    resetGesture() {
        this.state.gesturePoints = [];
        this.clearGestureCanvas();
        this.elements.feedback.textContent = '';
        this.elements.feedback.className = 'captcha-feedback';
    }

    async verify() {
        if (this.state.isVerifying || !this.state.isLoaded) return;
        
        this.state.isVerifying = true;
        this.elements.verify.disabled = true;
        this.elements.verify.textContent = '验证中...';
        
        let payload = {
            session_id: this.state.sessionId
        };
        
        if (this.options.type === 'rotate') {
            payload.rotation = this.state.rotation;
        } else if (this.options.type === 'gesture') {
            payload.gesture_points = this.state.gesturePoints;
        } else if (this.options.type === 'puzzle') {
            payload.offset = this.state.puzzleOffset;
        }

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
        this.elements.feedback.textContent = '验证成功';
        this.elements.feedback.className = 'captcha-feedback success';
        
        this.playSuccessAnimation();
        
        if (this.options.onSuccess) {
            this.options.onSuccess({ type: this.options.type, session_id: this.state.sessionId });
        }
    }

    handleError(message) {
        this.elements.feedback.textContent = message;
        this.elements.feedback.className = 'captcha-feedback error';
        this.playErrorAnimation();
        
        if (this.options.onError) {
            this.options.onError({ type: this.options.type, error: message });
        }
        
        setTimeout(() => this.refresh(), 2000);
    }

    playSuccessAnimation() {
        if (this.options.type === 'rotate') {
            this.elements.image.style.transition = 'transform 0.5s ease';
            this.elements.image.style.transform = `rotate(${this.state.targetRotation}deg)`;
        } else if (this.options.type === 'puzzle') {
            this.elements.piece.style.transition = 'left 0.5s ease';
            this.elements.piece.style.left = this.state.targetOffset + 'px';
        }
    }

    playErrorAnimation() {
        if (this.options.type === 'rotate') {
            let shake = 0;
            const shakeInterval = setInterval(() => {
                this.elements.container.style.transform = shake % 2 === 0 ? 'translateX(-5px)' : 'translateX(5px)';
                shake++;
                if (shake >= 4) {
                    clearInterval(shakeInterval);
                    this.elements.container.style.transform = '';
                }
            }, 100);
        } else if (this.options.type === 'puzzle') {
            let shake = 0;
            const shakeInterval = setInterval(() => {
                this.elements.piece.style.transform = shake % 2 === 0 ? 'translateX(-5px)' : 'translateX(5px)';
                shake++;
                if (shake >= 4) {
                    clearInterval(shakeInterval);
                    this.elements.piece.style.transform = '';
                }
            }, 100);
        }
    }

    showSkeleton() {
        if (this.elements.skeleton) {
            this.elements.skeleton.style.display = 'block';
            this.elements.skeleton.classList.add('active');
        }
    }

    hideSkeleton() {
        if (this.elements.skeleton) {
            this.elements.skeleton.classList.remove('active');
            setTimeout(() => {
                this.elements.skeleton.style.display = 'none';
            }, 300);
        }
    }

    destroy() {
        this.container.innerHTML = '';
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = RotateGestureCaptcha;
}