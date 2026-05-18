// 表情验证码前端交互
class EmojiCaptcha {
    constructor() {
        this.sessionId = null;
        this.targetEmojis = [];
        this.shuffledEmojis = [];
        this.selectedEmojis = [];
        this.clickTimes = [];
        this.startTime = null;
        this.isVerifying = false;
        
        this.init();
    }
    
    init() {
        this.bindEvents();
        this.fetchCaptcha();
        this.initTheme();
    }
    
    initTheme() {
        const themeToggle = document.getElementById('themeToggle');
        const themeIcon = document.getElementById('themeIcon');
        
        const savedTheme = localStorage.getItem('theme') || 'light';
        document.documentElement.setAttribute('data-bs-theme', savedTheme);
        this.updateThemeIcon(themeIcon, savedTheme);
        
        themeToggle.addEventListener('click', () => {
            const currentTheme = document.documentElement.getAttribute('data-bs-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-bs-theme', newTheme);
            localStorage.setItem('theme', newTheme);
            this.updateThemeIcon(themeIcon, newTheme);
        });
    }
    
    updateThemeIcon(icon, theme) {
        icon.className = theme === 'dark' ? 'fas fa-sun' : 'fas fa-moon';
    }
    
    bindEvents() {
        document.getElementById('resetBtn').addEventListener('click', () => this.resetSelection());
        document.getElementById('verifyBtn').addEventListener('click', () => this.verify());
        document.getElementById('refreshBtn').addEventListener('click', () => this.fetchCaptcha());
    }
    
    async fetchCaptcha() {
        this.showLoading();
        this.hideResult();
        
        try {
            const response = await fetch('/api/v1/captcha/emoji/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            
            const data = await response.json();
            
            if (data.code === 0) {
                this.sessionId = data.data.sessionId;
                this.targetEmojis = data.data.targetEmojis;
                this.shuffledEmojis = data.data.shuffledEmojis;
                
                this.renderCaptcha();
                this.showCaptcha();
            } else {
                this.showError(data.message || '获取验证码失败');
            }
        } catch (error) {
            console.error('获取验证码失败:', error);
            this.showError('网络错误，请稍后重试');
        }
    }
    
    renderCaptcha() {
        this.renderTargetEmojis();
        this.renderShuffledEmojis();
        this.resetSelection();
    }
    
    renderTargetEmojis() {
        const container = document.getElementById('targetEmojis');
        container.innerHTML = '';
        
        this.targetEmojis.forEach((emoji, index) => {
            if (index > 0) {
                const arrow = document.createElement('span');
                arrow.className = 'target-arrow';
                arrow.textContent = '→';
                container.appendChild(arrow);
            }
            
            const emojiEl = document.createElement('span');
            emojiEl.className = 'target-emoji';
            emojiEl.textContent = emoji;
            emojiEl.style.animationDelay = `${index * 0.1}s`;
            container.appendChild(emojiEl);
        });
    }
    
    renderShuffledEmojis() {
        const container = document.getElementById('shuffledEmojis');
        container.innerHTML = '';
        
        this.shuffledEmojis.forEach((emoji, index) => {
            const btn = document.createElement('button');
            btn.className = 'emoji-button';
            btn.textContent = emoji;
            btn.dataset.index = index;
            btn.addEventListener('click', () => this.onEmojiClick(emoji, btn));
            container.appendChild(btn);
        });
    }
    
    onEmojiClick(emoji, btn) {
        if (this.isVerifying) return;
        if (this.selectedEmojis.length >= this.targetEmojis.length) return;
        
        if (!this.startTime) {
            this.startTime = Date.now();
        }
        
        this.clickTimes.push(Date.now());
        this.selectedEmojis.push(emoji);
        btn.classList.add('selected');
        btn.disabled = true;
        
        this.renderSelectedEmojis();
        this.updateVerifyButton();
    }
    
    renderSelectedEmojis() {
        const container = document.getElementById('selectedEmojis');
        container.innerHTML = '';
        
        if (this.selectedEmojis.length === 0) {
            const placeholder = document.createElement('span');
            placeholder.className = 'text-muted small';
            placeholder.textContent = '点击上方表情开始选择';
            container.appendChild(placeholder);
            return;
        }
        
        this.selectedEmojis.forEach((emoji, index) => {
            const emojiEl = document.createElement('span');
            emojiEl.className = 'selected-emoji removeable';
            emojiEl.textContent = emoji;
            emojiEl.addEventListener('click', () => this.removeSelection(index));
            container.appendChild(emojiEl);
        });
    }
    
    removeSelection(index) {
        if (this.isVerifying) return;
        
        const removedEmoji = this.selectedEmojis[index];
        this.selectedEmojis.splice(index, 1);
        this.clickTimes.splice(index, 1);
        
        const shuffledButtons = document.querySelectorAll('#shuffledEmojis .emoji-button');
        for (let btn of shuffledButtons) {
            if (btn.textContent === removedEmoji && btn.classList.contains('selected')) {
                btn.classList.remove('selected');
                btn.disabled = false;
                break;
            }
        }
        
        this.renderSelectedEmojis();
        this.updateVerifyButton();
    }
    
    resetSelection() {
        this.selectedEmojis = [];
        this.clickTimes = [];
        this.startTime = null;
        
        const shuffledButtons = document.querySelectorAll('#shuffledEmojis .emoji-button');
        shuffledButtons.forEach(btn => {
            btn.classList.remove('selected');
            btn.disabled = false;
        });
        
        this.renderSelectedEmojis();
        this.updateVerifyButton();
        this.hideResult();
    }
    
    updateVerifyButton() {
        const verifyBtn = document.getElementById('verifyBtn');
        verifyBtn.disabled = this.selectedEmojis.length !== this.targetEmojis.length;
    }
    
    async verify() {
        if (this.isVerifying) return;
        if (this.selectedEmojis.length !== this.targetEmojis.length) return;
        
        this.isVerifying = true;
        const verifyBtn = document.getElementById('verifyBtn');
        verifyBtn.disabled = true;
        verifyBtn.innerHTML = '<i class="fas fa-spinner fa-spin me-1"></i> 验证中...';
        
        const clickIntervals = [];
        for (let i = 1; i < this.clickTimes.length; i++) {
            clickIntervals.push(this.clickTimes[i] - this.clickTimes[i - 1]);
        }
        const totalTime = this.clickTimes.length > 0 
            ? this.clickTimes[this.clickTimes.length - 1] - this.startTime 
            : 0;
        
        const behaviorData = {
            clickTimes: this.clickTimes,
            clickIntervals: clickIntervals,
            totalTime: totalTime,
            isMobile: /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
        };
        
        try {
            const response = await fetch('/api/v1/captcha/emoji/verify', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    sessionId: this.sessionId,
                    selectedEmojis: this.selectedEmojis,
                    behaviorData: behaviorData
                })
            });
            
            const data = await response.json();
            
            if (data.code === 0) {
                if (data.data.success) {
                    this.showSuccess('验证成功！');
                    setTimeout(() => {
                        alert('验证成功！');
                        this.fetchCaptcha();
                    }, 1000);
                } else {
                    this.showError(data.data.message || '验证失败，请重试');
                }
            } else {
                this.showError(data.message || '验证失败，请重试');
            }
        } catch (error) {
            console.error('验证失败:', error);
            this.showError('网络错误，请稍后重试');
        } finally {
            this.isVerifying = false;
            verifyBtn.disabled = false;
            verifyBtn.innerHTML = '<i class="fas fa-check me-1"></i> 验证';
        }
    }
    
    showLoading() {
        document.getElementById('loadingState').style.display = 'block';
        document.getElementById('captchaState').style.display = 'none';
    }
    
    showCaptcha() {
        document.getElementById('loadingState').style.display = 'none';
        document.getElementById('captchaState').style.display = 'block';
    }
    
    showSuccess(message) {
        const banner = document.getElementById('resultBanner');
        const icon = document.getElementById('resultIcon');
        const text = document.getElementById('resultText');
        
        banner.className = 'result-banner success show';
        icon.className = 'result-icon fas fa-check-circle';
        text.textContent = message;
    }
    
    showError(message) {
        const banner = document.getElementById('resultBanner');
        const icon = document.getElementById('resultIcon');
        const text = document.getElementById('resultText');
        
        banner.className = 'result-banner error show';
        icon.className = 'result-icon fas fa-times-circle';
        text.textContent = message;
    }
    
    hideResult() {
        document.getElementById('resultBanner').classList.remove('show');
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new EmojiCaptcha();
});
