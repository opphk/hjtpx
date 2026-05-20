class MultisensoryCaptcha {
    constructor(options = {}) {
        this.options = {
            apiBase: '/api/v1/captcha/multisensory',
            types: ['visual', 'audio', 'tactile'],
            visualType: 'slider',
            language: 'zh-CN',
            requireAll: true,
            ...options
        };
        
        this.sessionData = null;
        this.answers = {};
        this.verified = {};
        this.currentType = null;
        this.isPlaying = false;
    }

    async init() {
        await this.generateCaptcha();
        this.setupEventListeners();
    }

    async generateCaptcha() {
        try {
            const response = await fetch(`${this.options.apiBase}/create`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    types: this.options.types,
                    visual_type: this.options.visualType,
                    language: this.options.language
                })
            });

            const result = await response.json();
            
            if (result.code === 0 && result.data) {
                this.sessionData = result.data;
                this.answers = {};
                this.verified = {};
                this.renderCaptcha();
            } else {
                this.showError('生成验证码失败');
            }
        } catch (error) {
            console.error('生成验证码错误:', error);
            this.showError('网络错误，请重试');
        }
    }

    renderCaptcha() {
        const container = document.getElementById('multisensory-container');
        if (!container) return;

        container.innerHTML = `
            <div class="multisensory-wrapper">
                <div class="captcha-tabs">
                    ${this.sessionData.types.map(type => `
                        <button class="tab-button" data-type="${type}" id="tab-${type}">
                            ${this.getTypeIcon(type)} ${this.getTypeName(type)}
                            <span class="status-icon" id="status-${type}"></span>
                        </button>
                    `).join('')}
                </div>
                
                <div class="captcha-content" id="captcha-content"></div>
                
                <div class="captcha-actions">
                    <button id="refresh-btn" class="btn btn-secondary">
                        <i class="fas fa-sync-alt"></i> 重新生成
                    </button>
                    <button id="verify-btn" class="btn btn-primary" disabled>
                        <i class="fas fa-check"></i> 验证
                    </button>
                </div>
                
                <div class="result-message" id="result-message"></div>
            </div>
        `;

        if (this.sessionData.types.length > 0) {
            this.switchType(this.sessionData.types[0]);
        }
    }

    switchType(type) {
        this.currentType = type;
        
        document.querySelectorAll('.tab-button').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.type === type);
        });

        const content = document.getElementById('captcha-content');
        
        switch (type) {
            case 'visual':
                this.renderVisualCaptcha(content);
                break;
            case 'audio':
                this.renderAudioCaptcha(content);
                break;
            case 'tactile':
                this.renderTactileCaptcha(content);
                break;
        }

        this.updateStatusIcons();
    }

    renderVisualCaptcha(container) {
        const visual = this.sessionData.visual;
        
        if (visual.type === 'slider') {
            container.innerHTML = `
                <div class="visual-captcha slider-captcha">
                    <div class="slider-image-container">
                        <img src="${visual.background_url}" class="background-image" id="slider-background">
                        <img src="${visual.slider_url}" class="slider-piece" id="slider-piece">
                    </div>
                    <div class="slider-control">
                        <input type="range" id="slider-input" min="0" max="300" value="0">
                    </div>
                    <p class="instruction">拖动滑块完成拼图</p>
                </div>
            `;
            this.setupSliderInteraction();
        } else if (visual.type === 'emoji') {
            container.innerHTML = `
                <div class="visual-captcha emoji-captcha">
                    <p class="target-emoji">请选择: ${visual.target_emoji}</p>
                    <div class="emoji-grid">
                        ${visual.emojis.map(emoji => `
                            <button class="emoji-button" data-emoji="${emoji}">${emoji}</button>
                        `).join('')}
                    </div>
                </div>
            `;
            this.setupEmojiInteraction();
        }
    }

    renderAudioCaptcha(container) {
        const audio = this.sessionData.audio;
        
        container.innerHTML = `
            <div class="audio-captcha">
                <div class="audio-player">
                    <button id="play-audio-btn" class="play-button">
                        <i class="fas fa-play"></i>
                    </button>
                    <p class="instruction">点击播放音频验证码</p>
                </div>
                <div class="audio-input-container">
                    <input type="text" id="audio-input" class="form-control" 
                           placeholder="请输入听到的数字" maxlength="4">
                </div>
            </div>
        `;
        
        this.setupAudioInteraction();
    }

    renderTactileCaptcha(container) {
        const tactile = this.sessionData.tactile;
        
        container.innerHTML = `
            <div class="tactile-captcha">
                <div class="vibration-control">
                    <button id="vibrate-btn" class="vibrate-button">
                        <i class="fas fa-mobile-alt"></i>
                    </button>
                    <p class="instruction">点击感受震动模式</p>
                    <p class="hint">（短震=0-4，长震=5-9）</p>
                </div>
                <div class="vibration-display" id="vibration-display"></div>
                <div class="tactile-input-container">
                    <input type="text" id="tactile-input" class="form-control" 
                           placeholder="请输入感受到的数字" maxlength="4">
                </div>
            </div>
        `;
        
        this.setupTactileInteraction();
    }

    setupSliderInteraction() {
        const sliderInput = document.getElementById('slider-input');
        const sliderPiece = document.getElementById('slider-piece');
        
        if (sliderInput && sliderPiece) {
            sliderInput.addEventListener('input', (e) => {
                const value = e.target.value;
                sliderPiece.style.left = `${value}px`;
                this.answers.visual = `${value},0`;
                this.updateVerifyButton();
            });
        }
    }

    setupEmojiInteraction() {
        const emojiButtons = document.querySelectorAll('.emoji-button');
        
        emojiButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                emojiButtons.forEach(b => b.classList.remove('selected'));
                btn.classList.add('selected');
                this.answers.visual = btn.dataset.emoji;
                this.updateVerifyButton();
            });
        });
    }

    setupAudioInteraction() {
        const playBtn = document.getElementById('play-audio-btn');
        const audioInput = document.getElementById('audio-input');
        
        if (playBtn && this.sessionData.audio) {
            playBtn.addEventListener('click', () => this.playAudio());
        }
        
        if (audioInput) {
            audioInput.addEventListener('input', (e) => {
                this.answers.audio = e.target.value;
                this.updateVerifyButton();
            });
        }
    }

    setupTactileInteraction() {
        const vibrateBtn = document.getElementById('vibrate-btn');
        const tactileInput = document.getElementById('tactile-input');
        
        if (vibrateBtn) {
            vibrateBtn.addEventListener('click', () => this.triggerVibration());
        }
        
        if (tactileInput) {
            tactileInput.addEventListener('input', (e) => {
                this.answers.tactile = e.target.value;
                this.updateVerifyButton();
            });
        }
    }

    playAudio() {
        if (this.isPlaying || !this.sessionData.audio) return;
        
        try {
            this.isPlaying = true;
            const playBtn = document.getElementById('play-audio-btn');
            if (playBtn) {
                playBtn.innerHTML = '<i class="fas fa-stop"></i>';
                playBtn.classList.add('playing');
            }
            
            const audioData = atob(this.sessionData.audio.voice_data);
            const bytes = new Uint8Array(audioData.length);
            for (let i = 0; i < audioData.length; i++) {
                bytes[i] = audioData.charCodeAt(i);
            }
            const blob = new Blob([bytes], { type: 'audio/wav' });
            const url = URL.createObjectURL(blob);
            
            const audio = new Audio(url);
            audio.onended = () => {
                this.isPlaying = false;
                if (playBtn) {
                    playBtn.innerHTML = '<i class="fas fa-play"></i>';
                    playBtn.classList.remove('playing');
                }
                URL.revokeObjectURL(url);
            };
            audio.play();
        } catch (error) {
            console.error('播放音频失败:', error);
            this.isPlaying = false;
        }
    }

    triggerVibration() {
        if (!navigator.vibrate || !this.sessionData.tactile) {
            this.showError('您的设备不支持震动功能');
            return;
        }
        
        const pattern = this.sessionData.tactile.pattern;
        const display = document.getElementById('vibration-display');
        
        if (display) {
            let html = '';
            for (let i = 0; i < pattern.length; i += 2) {
                const duration = pattern[i];
                const type = duration < 200 ? 'short' : 'long';
                html += `<span class="vibration-pulse ${type}"></span>`;
            }
            display.innerHTML = html;
        }
        
        navigator.vibrate(pattern);
    }

    setupEventListeners() {
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('tab-button')) {
                this.switchType(e.target.dataset.type);
            }
            
            if (e.target.id === 'refresh-btn') {
                this.generateCaptcha();
            }
            
            if (e.target.id === 'verify-btn') {
                this.verify();
            }
        });
    }

    updateVerifyButton() {
        const verifyBtn = document.getElementById('verify-btn');
        if (!verifyBtn) return;
        
        const hasAnswers = this.sessionData.types.some(type => this.answers[type]);
        verifyBtn.disabled = !hasAnswers;
    }

    updateStatusIcons() {
        for (const type of this.sessionData.types) {
            const statusIcon = document.getElementById(`status-${type}`);
            if (statusIcon) {
                if (this.verified[type]) {
                    statusIcon.innerHTML = '<i class="fas fa-check-circle"></i>';
                    statusIcon.className = 'status-icon verified';
                } else {
                    statusIcon.innerHTML = '';
                    statusIcon.className = 'status-icon';
                }
            }
        }
    }

    async verify() {
        try {
            const response = await fetch(`${this.options.apiBase}/verify`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    session_id: this.sessionData.session_id,
                    answers: this.answers,
                    require_all: this.options.requireAll
                })
            });

            const result = await response.json();
            
            if (result.code === 0 && result.data) {
                this.verified = result.data.verified || {};
                this.updateStatusIcons();
                
                if (result.data.success) {
                    this.showSuccess(result.data.message || '验证成功！');
                    if (this.options.onSuccess) {
                        this.options.onSuccess(result.data);
                    }
                } else {
                    this.showError(result.data.message || '验证失败');
                }
            } else {
                this.showError('验证失败，请重试');
            }
        } catch (error) {
            console.error('验证错误:', error);
            this.showError('网络错误，请重试');
        }
    }

    showSuccess(message) {
        const resultDiv = document.getElementById('result-message');
        if (resultDiv) {
            resultDiv.className = 'result-message success show';
            resultDiv.innerHTML = `<i class="fas fa-check-circle"></i> ${message}`;
        }
    }

    showError(message) {
        const resultDiv = document.getElementById('result-message');
        if (resultDiv) {
            resultDiv.className = 'result-message error show';
            resultDiv.innerHTML = `<i class="fas fa-exclamation-circle"></i> ${message}`;
        }
    }

    getTypeIcon(type) {
        const icons = {
            visual: '👁️',
            audio: '🔊',
            tactile: '📱'
        };
        return icons[type] || '❓';
    }

    getTypeName(type) {
        const names = {
            visual: '视觉',
            audio: '音频',
            tactile: '触觉'
        };
        return names[type] || type;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = MultisensoryCaptcha;
}
