class AdaptiveEcosystem {
    constructor(options = {}) {
        this.options = {
            apiBase: '/api/v1/captcha/adaptive-ecosystem',
            userId: this.generateUserId(),
            enableAnalytics: true,
            autoAdapt: true,
            ...options
        };

        this.sessionData = null;
        this.userProfile = null;
        this.metrics = null;
        this.listeners = {};
    }

    generateUserId() {
        return 'user_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }

    async init() {
        await this.loadMetrics();
        this.setupEventListeners();
        return this;
    }

    async loadMetrics() {
        try {
            const response = await fetch(`${this.options.apiBase}/metrics`);
            const result = await response.json();
            if (result.code === 0 && result.data) {
                this.metrics = result.data;
                this.emit('metricsLoaded', this.metrics);
            }
        } catch (error) {
            console.error('Failed to load metrics:', error);
        }
    }

    setupEventListeners() {
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.emit('pageHidden');
            } else {
                this.emit('pageVisible');
            }
        });
    }

    async generateCaptcha(context = {}) {
        try {
            const response = await fetch(`${this.options.apiBase}/create`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    user_id: this.options.userId,
                    ip_address: await this.getClientIP(),
                    user_agent: navigator.userAgent,
                    fingerprint: await this.getFingerprint(),
                    context: context,
                    preferred_type: this.userProfile?.preferred_types?.[0] || 'slider',
                    request_purpose: 'captcha_verification'
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                this.sessionData = result.data;
                this.userProfile = result.data.user_profile;
                this.renderCaptcha();
                return result.data;
            }
            return null;
        } catch (error) {
            console.error('Generate error:', error);
            return null;
        }
    }

    async verifyCaptcha(answer, responseTime, behaviorData = {}) {
        if (!this.sessionData) {
            throw new Error('No active session');
        }

        try {
            const enhancedBehaviorData = {
                ...behaviorData,
                mouse_trajectory: this.captureMouseTrajectory(),
                keystroke_pattern: this.captureKeystrokePattern(),
                touch_data: this.captureTouchData(),
                device_orientation: this.getDeviceOrientation(),
                battery_level: await this.getBatteryLevel(),
                connection_type: this.getConnectionType()
            };

            const response = await fetch(`${this.options.apiBase}/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    session_id: this.sessionData.session_id,
                    user_id: this.options.userId,
                    answer: answer,
                    response_time: responseTime,
                    behavior_data: enhancedBehaviorData,
                    environment_data: {
                        user_agent: navigator.userAgent,
                        language: navigator.language,
                        platform: navigator.platform,
                        screen_resolution: `${screen.width}x${screen.height}`,
                        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
                    }
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                if (result.data.learning_update) {
                    this.applyLearningUpdate(result.data.learning_update);
                }
                return result.data;
            }
            return null;
        } catch (error) {
            console.error('Verify error:', error);
            return null;
        }
    }

    captureMouseTrajectory() {
        const trajectory = [];
        const now = Date.now();
        
        const moveHandler = (e) => {
            trajectory.push({
                x: e.clientX,
                y: e.clientY,
                t: Date.now() - now
            });
        };

        document.addEventListener('mousemove', moveHandler);
        
        setTimeout(() => {
            document.removeEventListener('mousemove', moveHandler);
        }, 1000);

        return trajectory;
    }

    captureKeystrokePattern() {
        const pattern = {
            keyDownTimes: [],
            keyUpTimes: [],
            keystrokeDurations: []
        };

        let lastKeyDown = null;

        const keyDownHandler = (e) => {
            lastKeyDown = Date.now();
            pattern.keyDownTimes.push(lastKeyDown);
        };

        const keyUpHandler = (e) => {
            if (lastKeyDown) {
                pattern.keyUpTimes.push(Date.now());
                pattern.keystrokeDurations.push(Date.now() - lastKeyDown);
            }
        };

        document.addEventListener('keydown', keyDownHandler);
        document.addEventListener('keyup', keyUpHandler);

        setTimeout(() => {
            document.removeEventListener('keydown', keyDownHandler);
            document.removeEventListener('keyup', keyUpHandler);
        }, 5000);

        return pattern;
    }

    captureTouchData() {
        const touches = [];
        
        const touchStartHandler = (e) => {
            for (let touch of e.changedTouches) {
                touches.push({
                    type: 'start',
                    x: touch.clientX,
                    y: touch.clientY,
                    t: Date.now()
                });
            }
        };

        const touchMoveHandler = (e) => {
            for (let touch of e.changedTouches) {
                touches.push({
                    type: 'move',
                    x: touch.clientX,
                    y: touch.clientY,
                    t: Date.now()
                });
            }
        };

        const touchEndHandler = (e) => {
            for (let touch of e.changedTouches) {
                touches.push({
                    type: 'end',
                    x: touch.clientX,
                    y: touch.clientY,
                    t: Date.now()
                });
            }
        };

        document.addEventListener('touchstart', touchStartHandler);
        document.addEventListener('touchmove', touchMoveHandler);
        document.addEventListener('touchend', touchEndHandler);

        setTimeout(() => {
            document.removeEventListener('touchstart', touchStartHandler);
            document.removeEventListener('touchmove', touchMoveHandler);
            document.removeEventListener('touchend', touchEndHandler);
        }, 5000);

        return touches;
    }

    getDeviceOrientation() {
        if (window.DeviceOrientationEvent) {
            return {
                alpha: window.DeviceOrientationEvent.alpha,
                beta: window.DeviceOrientationEvent.beta,
                gamma: window.DeviceOrientationEvent.gamma
            };
        }
        return null;
    }

    async getBatteryLevel() {
        if ('getBattery' in navigator) {
            try {
                const battery = await navigator.getBattery();
                return battery.level;
            } catch {
                return null;
            }
        }
        return null;
    }

    getConnectionType() {
        const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
        if (connection) {
            return connection.effectiveType;
        }
        return 'unknown';
    }

    async getClientIP() {
        try {
            const response = await fetch('https://api.ipify.org?format=json');
            const data = await response.json();
            return data.ip;
        } catch {
            return 'unknown';
        }
    }

    async getFingerprint() {
        const components = [
            navigator.userAgent,
            navigator.language,
            screen.width,
            screen.height,
            screen.colorDepth,
            new Date().getTimezoneOffset(),
            navigator.hardwareConcurrency || 'unknown',
            navigator.platform
        ];

        const fingerprint = components.join('|');
        return await this.hashString(fingerprint);
    }

    async hashString(str) {
        const encoder = new TextEncoder();
        const data = encoder.encode(str);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = Array.from(new Uint8Array(hashBuffer));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }

    applyLearningUpdate(update) {
        if (!this.userProfile) return;

        this.userProfile.adaptation_level = (this.userProfile.adaptation_level || 0) + (update.confidence_delta || 0);

        this.emit('learningUpdate', update);
    }

    renderCaptcha() {
        const container = document.getElementById('ecosystem-container');
        if (!container) return;

        const config = this.sessionData.captcha_config;
        const captchaData = this.sessionData.captcha_data;
        const hints = this.sessionData.adaptive_hints || [];
        const risk = this.sessionData.risk_assessment;

        container.innerHTML = `
            <div class="ecosystem-wrapper">
                <div class="ecosystem-header">
                    <div class="status-indicator ${this.sessionData.ecosystem_status}">
                        <span class="status-dot"></span>
                        <span class="status-text">${this.getStatusText(this.sessionData.ecosystem_status)}</span>
                    </div>
                    <div class="difficulty-badge ${config.difficulty_level}">
                        ${this.getDifficultyLabel(config.difficulty_level)}
                    </div>
                </div>

                <div class="captcha-area" id="captcha-area">
                    ${this.renderCaptchaByType(config.captcha_type, captchaData)}
                </div>

                ${hints.length > 0 ? `
                    <div class="adaptive-hints">
                        ${hints.map(hint => `<div class="hint-item">${hint}</div>`).join('')}
                    </div>
                ` : ''}

                ${risk ? `
                    <div class="risk-indicator ${risk.risk_level}">
                        <span class="risk-label">风险等级: ${risk.risk_level}</span>
                        <span class="risk-score">${(risk.risk_score * 100).toFixed(1)}%</span>
                    </div>
                ` : ''}

                <div class="ecosystem-actions">
                    <button id="submit-btn" class="btn btn-primary" disabled>验证</button>
                    <button id="skip-btn" class="btn btn-secondary">跳过</button>
                </div>

                <div class="ecosystem-footer">
                    <div class="model-info">
                        <span class="model-version">${this.sessionData.model_version}</span>
                        <span class="expires-in">${this.sessionData.expires_in}s</span>
                    </div>
                </div>
            </div>
        `;

        this.setupCaptchaInteraction(config.captcha_type, captchaData);
    }

    renderCaptchaByType(type, data) {
        switch (type) {
            case 'slider':
                return this.renderSliderCaptcha(data);
            case 'emoji':
                return this.renderEmojiCaptcha(data);
            case '3d':
                return this.render3DCaptcha(data);
            default:
                return this.renderSliderCaptcha(data);
        }
    }

    renderSliderCaptcha(data) {
        const tolerance = data.tolerance || 5;
        return `
            <div class="slider-captcha">
                <div class="slider-track">
                    <div class="slider-background">
                        <div class="slider-gap" style="left: 60%;"></div>
                    </div>
                    <div class="slider-handle" id="slider-handle"></div>
                </div>
                <input type="range" id="slider-input" min="0" max="300" value="0" />
                <p class="captcha-instruction">拖动滑块完成拼图 (容差: ${tolerance}px)</p>
            </div>
        `;
    }

    renderEmojiCaptcha(data) {
        const emojiCount = data.emoji_count || 8;
        const emojis = ['😊', '😂', '😍', '🥳', '😎', '🤔', '😴', '🥺', '😤', '😱', '🎉', '🔥'];
        const selected = emojis.slice(0, emojiCount);
        const target = selected[Math.floor(Math.random() * selected.length)];

        return `
            <div class="emoji-captcha">
                <p class="target-emoji">请选择: <strong>${target}</strong></p>
                <div class="emoji-grid">
                    ${selected.map(emoji => `
                        <button class="emoji-btn" data-emoji="${emoji}">${emoji}</button>
                    `).join('')}
                </div>
                <p class="captcha-instruction">点击与示例相同的表情</p>
            </div>
        `;
    }

    render3DCaptcha(data) {
        return `
            <div class="3d-captcha">
                <div class="3d-preview">
                    <div class="cube">
                        <div class="face front">前</div>
                        <div class="face back">后</div>
                        <div class="face left">左</div>
                        <div class="face right">右</div>
                        <div class="face top">上</div>
                        <div class="face bottom">下</div>
                    </div>
                </div>
                <div class="rotation-control">
                    <input type="range" id="rotation-input" min="0" max="360" value="0" />
                </div>
                <p class="captcha-instruction">旋转到指定角度 (${data.rotation_steps || 360}°)</p>
            </div>
        `;
    }

    setupCaptchaInteraction(type, data) {
        switch (type) {
            case 'slider':
                this.setupSliderInteraction();
                break;
            case 'emoji':
                this.setupEmojiInteraction();
                break;
            case '3d':
                this.setup3DInteraction();
                break;
        }
    }

    setupSliderInteraction() {
        const slider = document.getElementById('slider-input');
        const handle = document.getElementById('slider-handle');
        const submitBtn = document.getElementById('submit-btn');

        if (slider && handle) {
            slider.addEventListener('input', (e) => {
                handle.style.left = `${e.target.value}px`;
                submitBtn.disabled = false;
            });
        }

        if (submitBtn) {
            submitBtn.addEventListener('click', () => this.handleSliderSubmit());
        }
    }

    setupEmojiInteraction() {
        const emojiBtns = document.querySelectorAll('.emoji-btn');
        const submitBtn = document.getElementById('submit-btn');

        emojiBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                emojiBtns.forEach(b => b.classList.remove('selected'));
                btn.classList.add('selected');
                submitBtn.disabled = false;
            });
        });

        if (submitBtn) {
            submitBtn.addEventListener('click', () => this.handleEmojiSubmit());
        }
    }

    setup3DInteraction() {
        const rotationInput = document.getElementById('rotation-input');
        const cube = document.querySelector('.cube');
        const submitBtn = document.getElementById('submit-btn');

        if (rotationInput && cube) {
            rotationInput.addEventListener('input', (e) => {
                cube.style.transform = `rotateY(${e.target.value}deg)`;
                submitBtn.disabled = false;
            });
        }

        if (submitBtn) {
            submitBtn.addEventListener('click', () => this.handle3DSubmit());
        }
    }

    handleSliderSubmit() {
        const slider = document.getElementById('slider-input');
        const answer = slider ? parseInt(slider.value) : 0;
        const responseTime = Date.now() - (this.startTime || Date.now());

        this.verifyCaptcha(answer, responseTime).then(result => {
            this.showResult(result);
        });
    }

    handleEmojiSubmit() {
        const selectedBtn = document.querySelector('.emoji-btn.selected');
        const answer = selectedBtn ? selectedBtn.dataset.emoji : null;
        const responseTime = Date.now() - (this.startTime || Date.now());

        this.verifyCaptcha(answer, responseTime).then(result => {
            this.showResult(result);
        });
    }

    handle3DSubmit() {
        const rotationInput = document.getElementById('rotation-input');
        const answer = rotationInput ? parseInt(rotationInput.value) : 0;
        const responseTime = Date.now() - (this.startTime || Date.now());

        this.verifyCaptcha(answer, responseTime).then(result => {
            this.showResult(result);
        });
    }

    showResult(result) {
        const container = document.getElementById('ecosystem-container');
        if (!container) return;

        const resultDiv = document.createElement('div');
        resultDiv.className = `result-display ${result.success ? 'success' : 'error'}`;

        if (result.success) {
            resultDiv.innerHTML = `
                <div class="result-icon success">
                    <i class="fas fa-check-circle"></i>
                </div>
                <div class="result-message">${result.message}</div>
                <div class="result-score">得分: ${(result.score * 100).toFixed(1)}%</div>
                ${result.next_difficulty ? `<div class="next-difficulty">下一难度: ${result.next_difficulty}</div>` : ''}
            `;
        } else {
            resultDiv.innerHTML = `
                <div class="result-icon error">
                    <i class="fas fa-times-circle"></i>
                </div>
                <div class="result-message">${result.message}</div>
                <button class="btn btn-primary retry-btn">重试</button>
            `;

            resultDiv.querySelector('.retry-btn')?.addEventListener('click', () => {
                this.generateCaptcha();
            });
        }

        container.appendChild(resultDiv);
    }

    getStatusText(status) {
        const statusMap = {
            'initializing': '初始化中',
            'active': '活跃',
            'evolving': '进化中',
            'optimizing': '优化中',
            'degraded': '性能下降'
        };
        return statusMap[status] || status;
    }

    getDifficultyLabel(level) {
        const labelMap = {
            'easy': '简单',
            'medium': '中等',
            'hard': '困难',
            'expert': '专家'
        };
        return labelMap[level] || level;
    }

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(callback => callback(data));
        }
    }

    getSessionData() {
        return this.sessionData;
    }

    getUserProfile() {
        return this.userProfile;
    }

    getMetrics() {
        return this.metrics;
    }

    reset() {
        this.sessionData = null;
        this.startTime = Date.now();
    }

    destroy() {
        this.listeners = {};
        this.sessionData = null;
        this.userProfile = null;
        this.metrics = null;
    }
}

class EcosystemAnalytics {
    constructor(service) {
        this.service = service;
        this.metrics = {
            captchaAttempts: 0,
            successfulAttempts: 0,
            failedAttempts: 0,
            avgResponseTime: 0,
            difficultyDistribution: {},
            typeDistribution: {}
        };
    }

    recordAttempt(result) {
        this.metrics.captchaAttempts++;

        if (result.success) {
            this.metrics.successfulAttempts++;
        } else {
            this.metrics.failedAttempts++;
        }

        const responseTime = result.feedback?.time_taken || 0;
        if (responseTime > 0) {
            const totalTime = this.metrics.avgResponseTime * (this.metrics.captchaAttempts - 1);
            this.metrics.avgResponseTime = (totalTime + responseTime) / this.metrics.captchaAttempts;
        }

        const type = result.captcha_type;
        this.metrics.typeDistribution[type] = (this.metrics.typeDistribution[type] || 0) + 1;

        const difficulty = result.feedback?.difficulty_hit;
        if (difficulty) {
            this.metrics.difficultyDistribution[difficulty] = (this.metrics.difficultyDistribution[difficulty] || 0) + 1;
        }
    }

    getSuccessRate() {
        if (this.metrics.captchaAttempts === 0) return 0;
        return this.metrics.successfulAttempts / this.metrics.captchaAttempts;
    }

    getReport() {
        return {
            total_attempts: this.metrics.captchaAttempts,
            success_rate: this.getSuccessRate(),
            avg_response_time: this.metrics.avgResponseTime,
            type_distribution: this.metrics.typeDistribution,
            difficulty_distribution: this.metrics.difficultyDistribution
        };
    }

    reset() {
        this.metrics = {
            captchaAttempts: 0,
            successfulAttempts: 0,
            failedAttempts: 0,
            avgResponseTime: 0,
            difficultyDistribution: {},
            typeDistribution: {}
        };
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { AdaptiveEcosystem, EcosystemAnalytics };
}
