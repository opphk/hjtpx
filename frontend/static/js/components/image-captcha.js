/**
 * HJTPX 图形验证码组件
 * 支持字母/数字组合验证码、刷新功能
 */

class ImageCaptcha {
    constructor(container, options = {}) {
        this.container = container;
        this.options = {
            apiBase: '/api/v1',
            onSuccess: null,
            onError: null,
            onRefresh: null,
            codeLength: 4,
            ...options
        };

        this.state = {
            sessionId: null,
            isVerifying: false,
            isLoaded: false
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
            <div class="image-captcha">
                <div class="captcha-group">
                    <div class="captcha-image-wrapper">
                        <img class="captcha-img" alt="验证码图片" />
                        <div class="captcha-skeleton"></div>
                    </div>
                    <button class="captcha-refresh-btn" id="image-refresh-btn" aria-label="刷新验证码">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
                <div class="input-group">
                    <input type="text" 
                           class="captcha-input" 
                           id="captcha-input"
                           maxlength="4"
                           placeholder="请输入验证码"
                           autocomplete="off"
                           aria-label="验证码输入框" />
                </div>
                <div class="captcha-feedback"></div>
                <button class="btn-verify" id="image-verify-btn">确认验证</button>
            </div>
        `;

        this.elements = {
            image: this.container.querySelector('.captcha-img'),
            refresh: this.container.querySelector('#image-refresh-btn'),
            input: this.container.querySelector('#captcha-input'),
            verify: this.container.querySelector('#image-verify-btn'),
            feedback: this.container.querySelector('.captcha-feedback'),
            skeleton: this.container.querySelector('.captcha-skeleton')
        };
    }

    bindEvents() {
        this.elements.refresh.addEventListener('click', () => this.refresh());
        this.elements.verify.addEventListener('click', () => this.verify());
        this.elements.input.addEventListener('keyup', (e) => {
            if (e.key === 'Enter') {
                this.verify();
            }
            this.clearFeedback();
        });
    }

    async refresh() {
        this.state.isVerifying = false;
        this.state.isLoaded = false;
        this.elements.input.value = '';
        this.showSkeleton();
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/image`);
            if (response.ok) {
                const data = await response.json();
                this.state.sessionId = data.session_id;
                this.elements.image.src = data.image_url;
            } else {
                this.generateDemoCode();
            }
        } catch (error) {
            this.generateDemoCode();
        } finally {
            this.hideSkeleton();
            this.state.isLoaded = true;
            if (this.options.onRefresh) this.options.onRefresh();
        }
    }

    generateDemoCode() {
        this.state.sessionId = 'demo_' + Date.now();
        const code = this.generateRandomCode();
        this.drawCodeImage(code);
    }

    generateRandomCode() {
        const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789';
        let code = '';
        for (let i = 0; i < this.options.codeLength; i++) {
            code += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return code;
    }

    drawCodeImage(code) {
        const canvas = document.createElement('canvas');
        canvas.width = 120;
        canvas.height = 40;
        const ctx = canvas.getContext('2d');

        const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
        gradient.addColorStop(0, '#667eea');
        gradient.addColorStop(1, '#764ba2');
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, canvas.width, canvas.height);

        for (let i = 0; i < 4; i++) {
            ctx.beginPath();
            ctx.strokeStyle = `rgba(255, 255, 255, ${0.3 + Math.random() * 0.3})`;
            ctx.lineWidth = 1;
            ctx.moveTo(Math.random() * canvas.width, Math.random() * canvas.height);
            ctx.lineTo(Math.random() * canvas.width, Math.random() * canvas.height);
            ctx.stroke();
        }

        const fonts = ['Arial', 'Helvetica', 'Times New Roman', 'Georgia', 'Verdana'];
        code.split('').forEach((char, index) => {
            ctx.save();
            ctx.font = `${20 + Math.random() * 8}px ${fonts[Math.floor(Math.random() * fonts.length)]}`;
            ctx.fillStyle = `rgba(255, 255, 255, ${0.8 + Math.random() * 0.2})`;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            
            const x = 20 + index * 25 + (Math.random() - 0.5) * 4;
            const y = canvas.height / 2 + (Math.random() - 0.5) * 8;
            const rotate = (Math.random() - 0.5) * 0.3;
            
            ctx.translate(x, y);
            ctx.rotate(rotate);
            ctx.fillText(char, 0, 0);
            ctx.restore();
        });

        this.elements.image.src = canvas.toDataURL();
    }

    clearFeedback() {
        this.elements.feedback.textContent = '';
        this.elements.feedback.className = 'captcha-feedback';
    }

    async verify() {
        const code = this.elements.input.value.trim();
        
        if (!code) {
            this.elements.feedback.textContent = '请输入验证码';
            this.elements.feedback.className = 'captcha-feedback error';
            return;
        }

        if (this.state.isVerifying || !this.state.isLoaded) return;
        
        this.state.isVerifying = true;
        this.elements.verify.disabled = true;
        this.elements.verify.textContent = '验证中...';
        
        const payload = {
            session_id: this.state.sessionId,
            code: code
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
                this.handleError(data.message || '验证码错误');
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
        this.elements.input.disabled = true;
        
        this.playSuccessAnimation();
        
        if (this.options.onSuccess) {
            this.options.onSuccess({ type: 'image', session_id: this.state.sessionId });
        }
    }

    handleError(message) {
        this.elements.feedback.textContent = message;
        this.elements.feedback.className = 'captcha-feedback error';
        this.playErrorAnimation();
        
        if (this.options.onError) {
            this.options.onError({ type: 'image', error: message });
        }
        
        setTimeout(() => this.refresh(), 2000);
    }

    playSuccessAnimation() {
        const image = this.elements.image;
        image.style.transition = 'transform 0.3s ease';
        image.style.transform = 'scale(1.1)';
        
        setTimeout(() => {
            image.style.transform = 'scale(1)';
        }, 300);
    }

    playErrorAnimation() {
        const input = this.elements.input;
        input.style.boxShadow = '0 0 0 2px #dc3545';
        
        setTimeout(() => {
            input.style.boxShadow = '';
        }, 500);
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
    module.exports = ImageCaptcha;
}