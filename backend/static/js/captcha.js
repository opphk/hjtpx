class TrajectoryEncryptor {
    constructor() {
        this.secretKey = 'captcha-trajectory-secret-key-2024';
        this.saltLength = 16;
    }

    generateSalt() {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let salt = '';
        const randomValues = new Uint8Array(this.saltLength);
        crypto.getRandomValues(randomValues);
        for (let i = 0; i < this.saltLength; i++) {
            salt += chars[randomValues[i] % chars.length];
        }
        return salt;
    }

    async encryptData(data, salt) {
        const key = await this.deriveKey(salt);
        const encoder = new TextEncoder();
        const dataBytes = encoder.encode(JSON.stringify(data));

        const iv = crypto.getRandomValues(new Uint8Array(12));

        const encryptedContent = await crypto.subtle.encrypt(
            {
                name: 'AES-GCM',
                iv: iv
            },
            key,
            dataBytes
        );

        const combined = new Uint8Array(iv.length + encryptedContent.byteLength);
        combined.set(iv, 0);
        combined.set(new Uint8Array(encryptedContent), iv.length);

        return this.arrayBufferToBase64(combined);
    }

    async deriveKey(salt) {
        const encoder = new TextEncoder();
        const keyMaterial = await crypto.subtle.importKey(
            'raw',
            encoder.encode(this.secretKey),
            { name: 'PBKDF2' },
            false,
            ['deriveBits', 'deriveKey']
        );

        const saltBytes = encoder.encode(salt);

        return crypto.subtle.deriveKey(
            {
                name: 'PBKDF2',
                salt: saltBytes,
                iterations: 100000,
                hash: 'SHA-256'
            },
            keyMaterial,
            { name: 'AES-GCM', length: 256 },
            false,
            ['encrypt', 'decrypt']
        );
    }

    generateSignature(timestamp, salt, encryptedData) {
        const data = `${timestamp}:${salt}:${encryptedData}`;
        const encoder = new TextEncoder();
        const key = encoder.encode(this.secretKey);

        const signature = this.hmacSHA256(key, encoder.encode(data));
        return this.arrayBufferToBase64(signature);
    }

    async hmacSHA256(key, data) {
        const cryptoKey = await crypto.subtle.importKey(
            'raw',
            key,
            { name: 'HMAC', hash: 'SHA-256' },
            false,
            ['sign']
        );

        const signature = await crypto.subtle.sign('HMAC', cryptoKey, data);
        return new Uint8Array(signature);
    }

    arrayBufferToBase64(buffer) {
        const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    base64ToArrayBuffer(base64) {
        const binaryString = atob(base64);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes;
    }

    async decryptData(encryptedDataBase64, salt) {
        const key = await this.deriveKey(salt);
        const combined = this.base64ToArrayBuffer(encryptedDataBase64);

        const iv = combined.slice(0, 12);
        const ciphertext = combined.slice(12);

        const decryptedContent = await crypto.subtle.decrypt(
            {
                name: 'AES-GCM',
                iv: iv
            },
            key,
            ciphertext
        );

        const decoder = new TextDecoder();
        return JSON.parse(decoder.decode(decryptedContent));
    }

    async verifySignature(timestamp, salt, encryptedData, signature) {
        const expectedSignature = this.generateSignature(timestamp, salt, encryptedData);
        return this.constantTimeCompare(signature, expectedSignature);
    }

    constantTimeCompare(a, b) {
        if (a.length !== b.length) {
            return false;
        }
        let result = 0;
        for (let i = 0; i < a.length; i++) {
            result |= a.charCodeAt(i) ^ b.charCodeAt(i);
        }
        return result === 0;
    }

    async encryptTrajectory(trajectory) {
        if (!trajectory || trajectory.length === 0) {
            throw new Error('Empty trajectory data');
        }

        const timestamp = Date.now();
        const salt = this.generateSalt();

        const encryptedData = await this.encryptData(trajectory, salt);
        const signature = this.generateSignature(timestamp, salt, encryptedData);

        return {
            timestamp: timestamp,
            salt: salt,
            encrypted_data: encryptedData,
            signature: signature
        };
    }

    validateTimestamp(timestamp, maxDriftMs = 300000) {
        const now = Date.now();
        const drift = Math.abs(now - timestamp);
        return drift <= maxDriftMs;
    }

    generateRequestPayload(trajectory) {
        return this.encryptTrajectory(trajectory);
    }
}

class Captcha {
    constructor(containerId, options = {}) {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            console.error('Captcha container not found');
            return;
        }

        this.options = {
            apiBase: '/api/v1',
            type: 'slider',
            language: 'zh-CN',
            animationStyle: 'pulse',
            enableSound: false,
            enableEncryption: true,
            onSuccess: null,
            onError: null,
            onRefresh: null,
            onLoadStart: null,
            onLoadEnd: null,
            ...options
        };

        this.trajectoryEncryptor = new TrajectoryEncryptor();

        this.sliderState = {
            isDragging: false,
            startX: 0,
            currentX: 0,
            maxX: 0,
            puzzleY: 0,
            targetX: 0,
            targetY: 0,
            puzzleStyle: 0,
            tolerance: 10
        };

        this.rotationState = {
            isDragging: false,
            startX: 0,
            currentAngle: 0,
            totalAngle: 0,
            maxAngle: 360,
            challengeId: '',
            imageUrl: '',
            startTime: 0,
            trajectoryData: []
        };

        this.gestureState = {
            isDrawing: false,
            pattern: '',
            canvasWidth: 300,
            canvasHeight: 300,
            trajectoryData: [],
            lastX: 0,
            lastY: 0
        };

        this.trajectoryData = [];
        this.speedData = {
            points: [],
            startTime: 0,
            endTime: 0,
            distance: 0,
            maxSpeed: 0
        };

        this.clickState = {
            selectedPoints: [],
            maxPoints: 3,
            hintText: '请依次点击图中的文字'
        };

        this.jigsawState = {
            pieces: [],
            pieceImages: [],
            gridSize: 3,
            pieceWidth: 100,
            pieceHeight: 100,
            selectedPiece: null,
            isDragging: false,
            offsetX: 0,
            offsetY: 0,
            sessionId: null
        };

        this.loadingState = {
            isLoading: false,
            loadingType: 'spinner',
            progress: 0,
            message: ''
        };

        this.accessibilityState = {
            liveRegion: null,
            reducedMotion: false
        };

        this.sessionId = null;
        this.isLoading = false;
        this.animationFrame = null;
        this.environmentData = null;
        this.detector = null;
        this.i18n = new CaptchaI18n(this.options.language);
        this.init();
    }

    init() {
        this.checkAccessibilityPreferences();
        this.render();
        this.bindEvents();
        this.refresh();
    }

    checkAccessibilityPreferences() {
        const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
        this.accessibilityState.reducedMotion = prefersReducedMotion.matches;
        prefersReducedMotion.addEventListener('change', (e) => {
            this.accessibilityState.reducedMotion = e.matches;
        });
    }

    announceToScreenReader(message, priority = 'polite') {
        let liveRegion = this.accessibilityState.liveRegion;
        if (!liveRegion) {
            liveRegion = document.createElement('div');
            liveRegion.setAttribute('role', 'status');
            liveRegion.setAttribute('aria-live', priority);
            liveRegion.setAttribute('aria-atomic', 'true');
            liveRegion.className = 'visually-hidden';
            liveRegion.style.cssText = 'position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0;';
            document.body.appendChild(liveRegion);
            this.accessibilityState.liveRegion = liveRegion;
        }
        liveRegion.setAttribute('aria-live', priority);
        liveRegion.textContent = '';
        setTimeout(() => {
            liveRegion.textContent = message;
        }, 50);
    }

    render() {
        this.container.innerHTML = `
            <div class="captcha-container" role="application" aria-label="${this.i18n.t('captchaLabel')}">
                <div class="captcha-header">
                    <h3>${this.i18n.t('securityVerify')}</h3>
                    <p>${this.i18n.t('completeVerify')}</p>
                </div>
                <div class="captcha-body">
                    <div class="captcha-tabs" role="tablist" aria-label="${this.i18n.t('verifyType')}">
                        <button class="captcha-tab active" role="tab" aria-selected="true" aria-controls="slider-captcha" data-type="slider" tabindex="0" id="tab-slider">
                            <span class="tab-icon"><i class="fas fa-puzzle-piece" aria-hidden="true"></i></span>
                            <span class="tab-text">${this.i18n.t('sliderVerify')}</span>
                        </button>
                        <button class="captcha-tab" role="tab" aria-selected="false" aria-controls="click-captcha" data-type="click" tabindex="0" id="tab-click">
                            <span class="tab-icon"><i class="fas fa-hand-pointer" aria-hidden="true"></i></span>
                            <span class="tab-text">${this.i18n.t('clickVerify')}</span>
                        </button>
                        <button class="captcha-tab" role="tab" aria-selected="false" aria-controls="jigsaw-captcha" data-type="jigsaw" tabindex="0" id="tab-jigsaw">
                            <span class="tab-icon"><i class="fas fa-th" aria-hidden="true"></i></span>
                            <span class="tab-text">拼图验证</span>
                        </button>
                        <button class="captcha-tab" role="tab" aria-selected="false" aria-controls="rotation-captcha" data-type="rotation" tabindex="0" id="tab-rotation">
                            <span class="tab-icon"><i class="fas fa-undo-alt" aria-hidden="true"></i></span>
                            <span class="tab-text">${this.i18n.t('rotationVerify')}</span>
                        </button>
                        <button class="captcha-tab" role="tab" aria-selected="false" aria-controls="gesture-captcha" data-type="gesture" tabindex="0" id="tab-gesture">
                            <span class="tab-icon"><i class="fas fa-hand-paper" aria-hidden="true"></i></span>
                            <span class="tab-text">${this.i18n.t('gestureVerify')}</span>
                        </button>
                        <button class="captcha-tab" role="tab" aria-selected="false" aria-controls="passive-captcha" data-type="passive" tabindex="0" id="tab-passive">
                            <span class="tab-icon"><i class="fas fa-shield-alt" aria-hidden="true"></i></span>
                            <span class="tab-text">${this.i18n.t('passiveVerify')}</span>
                        </button>
                    </div>

                    <div class="captcha-content active" id="slider-captcha" role="tabpanel" aria-labelledby="tab-slider">
                        <div class="captcha-loading-overlay" id="slider-loading-overlay" hidden>
                            <div class="captcha-loading-container">
                                <div class="loading-animation-${this.options.animationStyle}">
                                    <div class="loading-dots">
                                        <span></span><span></span><span></span><span></span><span></span>
                                    </div>
                                </div>
                                <div class="loading-progress-bar">
                                    <div class="loading-progress-fill" id="slider-progress-fill"></div>
                                </div>
                                <div class="loading-message" id="slider-loading-message">${this.i18n.t('loading')}</div>
                            </div>
                        </div>
                        <div class="captcha-image-wrapper" id="slider-image-wrapper">
                            <div class="captcha-background-layer" id="slider-bg-layer"></div>
                            <canvas class="captcha-canvas" id="slider-canvas" width="360" height="220" role="img" aria-label="${this.i18n.t('sliderImageAlt')}"></canvas>
                            <div class="captcha-puzzle" id="slider-puzzle" role="img" aria-label="${this.i18n.t('puzzlePiece')}"></div>
                            <button class="captcha-refresh" id="slider-refresh" aria-label="${this.i18n.t('refresh')}" title="${this.i18n.t('refresh')}">
                                <i class="fas fa-sync-alt" aria-hidden="true"></i>
                            </button>
                            <div class="captcha-image-skeleton" id="slider-skeleton">
                                <div class="skeleton-shimmer"></div>
                            </div>
                        </div>
                        <div class="captcha-slider-container" id="slider-container" role="slider" 
                             aria-label="${this.i18n.t('sliderAriaLabel')}"
                             aria-valuemin="0" aria-valuemax="100" aria-valuenow="0"
                             tabindex="0">
                            <div class="captcha-slider-track" id="slider-track"></div>
                            <div class="captcha-slider-text" id="slider-text" aria-hidden="true">${this.i18n.t('dragToVerify')}</div>
                            <div class="captcha-slider-button" id="slider-button" role="button" 
                                 aria-label="${this.i18n.t('sliderButtonAria')}"
                                 tabindex="-1">
                                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
                                    <polyline points="9 18 15 12 9 6"></polyline>
                                </svg>
                            </div>
                            <div class="captcha-slider-hint" aria-hidden="true">
                                <span class="hint-icon"><i class="fas fa-info-circle"></i></span>
                                <span class="hint-text">${this.i18n.t('sliderHint')}</span>
                            </div>
                        </div>
                    </div>

                    <div class="captcha-content" id="click-captcha" role="tabpanel" aria-labelledby="tab-click" hidden>
                        <div class="captcha-loading-overlay" id="click-loading-overlay" hidden>
                            <div class="captcha-loading-container">
                                <div class="loading-animation-${this.options.animationStyle}">
                                    <div class="loading-dots">
                                        <span></span><span></span><span></span><span></span><span></span>
                                    </div>
                                </div>
                                <div class="loading-progress-bar">
                                    <div class="loading-progress-fill" id="click-progress-fill"></div>
                                </div>
                                <div class="loading-message" id="click-loading-message">${this.i18n.t('loading')}</div>
                            </div>
                        </div>
                        <div class="captcha-click-hint" id="click-hint" aria-live="polite">
                            <span class="hint-icon"><i class="fas fa-lightbulb" aria-hidden="true"></i></span>
                            <span class="hint-text">${this.i18n.t('clickHint')}</span>
                        </div>
                        <div class="captcha-click-grid" id="click-grid" role="application" aria-label="${this.i18n.t('clickGridLabel')}">
                            <img class="captcha-click-image" id="click-image" alt="${this.i18n.t('clickImageAlt')}">
                            <button class="captcha-refresh" id="click-refresh" aria-label="${this.i18n.t('refresh')}" title="${this.i18n.t('refresh')}">
                                <i class="fas fa-sync-alt" aria-hidden="true"></i>
                            </button>
                            <div class="captcha-click-skeleton" id="click-skeleton">
                                <div class="skeleton-shimmer"></div>
                            </div>
                        </div>
                        <div class="captcha-click-progress" aria-live="polite">
                            <span>${this.i18n.t('selectedCount')}: </span>
                            <span id="click-selected-count" class="count-badge">0</span>
                            <span>/</span>
                            <span id="click-total-count">3</span>
                        </div>
                        <div class="captcha-actions">
                            <button class="captcha-btn captcha-btn-secondary" id="click-clear" aria-label="${this.i18n.t('clearSelection')}">
                                <i class="fas fa-eraser" aria-hidden="true"></i> ${this.i18n.t('clear')}
                            </button>
                            <button class="captcha-btn captcha-btn-primary" id="click-submit" aria-label="${this.i18n.t('submitVerification')}">
                                <i class="fas fa-check" aria-hidden="true"></i> ${this.i18n.t('confirm')}
                            </button>
                        </div>
                    </div>

                    <div class="captcha-content" id="jigsaw-captcha" role="tabpanel" aria-labelledby="tab-jigsaw" hidden>
                        <div class="captcha-loading-overlay" id="jigsaw-loading-overlay" hidden>
                            <div class="captcha-loading-container">
                                <div class="loading-animation-${this.options.animationStyle}">
                                    <div class="loading-dots">
                                        <span></span><span></span><span></span><span></span><span></span>
                                    </div>
                                </div>
                                <div class="loading-progress-bar">
                                    <div class="loading-progress-fill" id="jigsaw-progress-fill"></div>
                                </div>
                                <div class="loading-message" id="jigsaw-loading-message">${this.i18n.t('loading')}</div>
                            </div>
                        </div>
                        <div class="jigsaw-container" id="jigsaw-container" style="position: relative; margin: 0 auto;">
                            <div class="jigsaw-target-grid" id="jigsaw-target-grid" style="display: grid; gap: 2px; margin-bottom: 10px;"></div>
                            <button class="captcha-refresh" id="jigsaw-refresh" aria-label="${this.i18n.t('refresh')}" title="${this.i18n.t('refresh')}" style="position: absolute; top: 5px; right: 5px;">
                                <i class="fas fa-sync-alt" aria-hidden="true"></i>
                            </button>
                            <div class="captcha-image-skeleton" id="jigsaw-skeleton" style="width: 300px; height: 300px;">
                                <div class="skeleton-shimmer"></div>
                            </div>
                        </div>
                        <div class="jigsaw-pieces-container" id="jigsaw-pieces-container" style="margin-top: 10px; display: flex; flex-wrap: wrap; gap: 5px; justify-content: center; min-height: 100px;"></div>
                        <div class="captcha-actions" style="margin-top: 10px;">
                            <button class="captcha-btn captcha-btn-secondary" id="jigsaw-reset" aria-label="重置拼图">
                                <i class="fas fa-rotate" aria-hidden="true"></i> 重置
                            </button>
                            <button class="captcha-btn captcha-btn-primary" id="jigsaw-verify" aria-label="验证拼图">
                                <i class="fas fa-check" aria-hidden="true"></i> 验证
                            </button>
                        </div>
                    </div>

                    <div class="captcha-content" id="rotation-captcha" role="tabpanel" aria-labelledby="tab-rotation" hidden>
                        <div class="captcha-loading-overlay" id="rotation-loading-overlay" hidden>
                            <div class="captcha-loading-container">
                                <div class="loading-animation-${this.options.animationStyle}">
                                    <div class="loading-dots">
                                        <span></span><span></span><span></span><span></span><span></span>
                                    </div>
                                </div>
                                <div class="loading-progress-bar">
                                    <div class="loading-progress-fill" id="rotation-progress-fill"></div>
                                </div>
                                <div class="loading-message" id="rotation-loading-message">${this.i18n.t('loading')}</div>
                            </div>
                        </div>
                        <div class="rotation-captcha-display" id="rotation-image-wrapper">
                            <img class="rotation-captcha-image" id="rotation-image" alt="${this.i18n.t('rotationImageAlt')}">
                            <button class="captcha-refresh" id="rotation-refresh" aria-label="${this.i18n.t('refresh')}" title="${this.i18n.t('refresh')}">
                                <i class="fas fa-sync-alt" aria-hidden="true"></i>
                            </button>
                            <div class="captcha-click-skeleton" id="rotation-skeleton">
                                <div class="skeleton-shimmer"></div>
                            </div>
                        </div>
                        <div class="rotation-slider-container" id="rotation-slider-container">
                            <div class="rotation-slider-track" id="rotation-slider-track"></div>
                            <div class="rotation-slider-button" id="rotation-slider-button" role="button"
                                 aria-label="${this.i18n.t('rotationSliderAria')}"
                                 tabindex="-1">
                                <i class="fas fa-undo-alt" aria-hidden="true"></i>
                            </div>
                            <div class="rotation-slider-text" id="rotation-slider-text" aria-hidden="true">${this.i18n.t('dragToRotate')}</div>
                        </div>
                        <div class="rotation-angle-display">
                            <span>${this.i18n.t('rotationAngle')}: </span>
                            <span id="rotation-angle-value">0°</span>
                        </div>
                    </div>

                    <div class="captcha-content" id="gesture-captcha" role="tabpanel" aria-labelledby="tab-gesture" hidden>
                        <div class="captcha-loading-overlay" id="gesture-loading-overlay" hidden>
                            <div class="captcha-loading-container">
                                <div class="loading-animation-${this.options.animationStyle}">
                                    <div class="loading-dots">
                                        <span></span><span></span><span></span><span></span><span></span>
                                    </div>
                                </div>
                                <div class="loading-progress-bar">
                                    <div class="loading-progress-fill" id="gesture-progress-fill"></div>
                                </div>
                                <div class="loading-message" id="gesture-loading-message">${this.i18n.t('loading')}</div>
                            </div>
                        </div>
                        <div class="gesture-captcha-hint" id="gesture-hint" style="text-align:center;margin-bottom:10px;">
                            <span class="hint-icon"><i class="fas fa-lightbulb" aria-hidden="true"></i></span>
                            <span class="hint-text" id="gesture-hint-text">请绘制图案</span>
                        </div>
                        <div class="gesture-canvas-container" id="gesture-canvas-container" style="position:relative;width:300px;height:300px;margin:0 auto;border:2px solid #e5e7eb;border-radius:8px;overflow:hidden;">
                            <canvas class="gesture-background-canvas" id="gesture-background-canvas" width="300" height="300" style="position:absolute;top:0;left:0;"></canvas>
                            <canvas class="gesture-drawing-canvas" id="gesture-drawing-canvas" width="300" height="300" style="position:absolute;top:0;left:0;cursor:crosshair;"></canvas>
                            <button class="captcha-refresh" id="gesture-refresh" aria-label="${this.i18n.t('refresh')}" title="${this.i18n.t('refresh')}" style="position:absolute;top:8px;right:8px;">
                                <i class="fas fa-sync-alt" aria-hidden="true"></i>
                            </button>
                        </div>
                        <div class="captcha-actions" style="margin-top:15px;text-align:center;">
                            <button class="captcha-btn captcha-btn-secondary" id="gesture-clear" aria-label="${this.i18n.t('clear')}">
                                <i class="fas fa-eraser" aria-hidden="true"></i> ${this.i18n.t('clear')}
                            </button>
                            <button class="captcha-btn captcha-btn-primary" id="gesture-submit" aria-label="${this.i18n.t('submitVerification')}">
                                <i class="fas fa-check" aria-hidden="true"></i> ${this.i18n.t('confirm')}
                            </button>
                        </div>
                    </div>

                    <div class="captcha-content" id="passive-captcha" role="tabpanel" aria-labelledby="tab-passive" hidden>
                        <div class="captcha-loading-overlay" id="passive-loading-overlay" hidden>
                            <div class="captcha-loading-container">
                                <div class="loading-animation-${this.options.animationStyle}">
                                    <div class="loading-dots">
                                        <span></span><span></span><span></span><span></span><span></span>
                                    </div>
                                </div>
                                <div class="loading-progress-bar">
                                    <div class="loading-progress-fill" id="passive-progress-fill"></div>
                                </div>
                                <div class="loading-message" id="passive-loading-message">${this.i18n.t('loading')}</div>
                            </div>
                        </div>
                        <div class="passive-captcha-display" style="text-align:center;padding:2rem;">
                            <div class="passive-icon" style="font-size:3rem;color:#52c41a;margin-bottom:1rem;">
                                <i class="fas fa-shield-alt" aria-hidden="true"></i>
                            </div>
                            <div class="passive-status" id="passive-status" style="font-size:1.1rem;color:#333;margin-bottom:0.5rem;">
                                ${this.i18n.t('passiveChecking')}
                            </div>
                            <div class="passive-detail" style="font-size:0.85rem;color:#999;margin-bottom:1.5rem;">
                                ${this.i18n.t('passiveDetail')}
                            </div>
                            <div class="passive-risk-score" style="margin-bottom:1rem;">
                                <span style="font-size:0.85rem;color:#666;">${this.i18n.t('riskScore')}: </span>
                                <span id="passive-risk-value" style="font-size:1.2rem;font-weight:bold;color:#52c41a;">--</span>
                            </div>
                            <div class="passive-checks" style="text-align:left;max-width:300px;margin:0 auto;">
                                <div class="passive-check-item" id="passive-check-env" style="padding:0.3rem 0;font-size:0.85rem;color:#999;">
                                    <i class="fas fa-spinner fa-pulse me-2"></i>${this.i18n.t('passiveCheckEnv')}
                                </div>
                                <div class="passive-check-item" id="passive-check-behavior" style="padding:0.3rem 0;font-size:0.85rem;color:#999;">
                                    <i class="fas fa-spinner fa-pulse me-2"></i>${this.i18n.t('passiveCheckBehavior')}
                                </div>
                                <div class="passive-check-item" id="passive-check-risk" style="padding:0.3rem 0;font-size:0.85rem;color:#999;">
                                    <i class="fas fa-spinner fa-pulse me-2"></i>${this.i18n.t('passiveCheckRisk')}
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="captcha-result" id="captcha-result" role="alert" aria-live="assertive" hidden></div>
                </div>
                <div class="captcha-footer">
                    <div class="captcha-security-badge" aria-label="${this.i18n.t('securityBadge')}">
                        <i class="fas fa-shield-alt" aria-hidden="true"></i>
                        <span>${this.i18n.t('secureConnection')}</span>
                    </div>
                </div>
            </div>
        `;

        this.elements = {
            tabs: this.container.querySelectorAll('.captcha-tab'),
            contents: this.container.querySelectorAll('.captcha-content'),
            sliderCanvas: this.container.querySelector('#slider-canvas'),
            sliderBgLayer: this.container.querySelector('#slider-bg-layer'),
            sliderPuzzle: this.container.querySelector('#slider-puzzle'),
            sliderContainer: this.container.querySelector('#slider-container'),
            sliderTrack: this.container.querySelector('#slider-track'),
            sliderText: this.container.querySelector('#slider-text'),
            sliderButton: this.container.querySelector('#slider-button'),
            sliderRefresh: this.container.querySelector('#slider-refresh'),
            sliderLoadingOverlay: this.container.querySelector('#slider-loading-overlay'),
            sliderProgressFill: this.container.querySelector('#slider-progress-fill'),
            sliderLoadingMessage: this.container.querySelector('#slider-loading-message'),
            sliderImageWrapper: this.container.querySelector('#slider-image-wrapper'),
            sliderSkeleton: this.container.querySelector('#slider-skeleton'),
            clickHint: this.container.querySelector('#click-hint'),
            clickGrid: this.container.querySelector('#click-grid'),
            clickImage: this.container.querySelector('#click-image'),
            clickRefresh: this.container.querySelector('#click-refresh'),
            clickLoadingOverlay: this.container.querySelector('#click-loading-overlay'),
            clickProgressFill: this.container.querySelector('#click-progress-fill'),
            clickLoadingMessage: this.container.querySelector('#click-loading-message'),
            clickSkeleton: this.container.querySelector('#click-skeleton'),
            clickClear: this.container.querySelector('#click-clear'),
            clickSubmit: this.container.querySelector('#click-submit'),
            clickSelectedCount: this.container.querySelector('#click-selected-count'),
            clickTotalCount: this.container.querySelector('#click-total-count'),
            rotationImage: this.container.querySelector('#rotation-image'),
            rotationRefresh: this.container.querySelector('#rotation-refresh'),
            rotationLoadingOverlay: this.container.querySelector('#rotation-loading-overlay'),
            rotationProgressFill: this.container.querySelector('#rotation-progress-fill'),
            rotationLoadingMessage: this.container.querySelector('#rotation-loading-message'),
            rotationSkeleton: this.container.querySelector('#rotation-skeleton'),
            rotationSliderContainer: this.container.querySelector('#rotation-slider-container'),
            rotationSliderTrack: this.container.querySelector('#rotation-slider-track'),
            rotationSliderButton: this.container.querySelector('#rotation-slider-button'),
            rotationSliderText: this.container.querySelector('#rotation-slider-text'),
            rotationAngleValue: this.container.querySelector('#rotation-angle-value'),
            gestureHint: this.container.querySelector('#gesture-hint'),
            gestureHintText: this.container.querySelector('#gesture-hint-text'),
            gestureBackgroundCanvas: this.container.querySelector('#gesture-background-canvas'),
            gestureDrawingCanvas: this.container.querySelector('#gesture-drawing-canvas'),
            gestureRefresh: this.container.querySelector('#gesture-refresh'),
            gestureLoadingOverlay: this.container.querySelector('#gesture-loading-overlay'),
            gestureProgressFill: this.container.querySelector('#gesture-progress-fill'),
            gestureLoadingMessage: this.container.querySelector('#gesture-loading-message'),
            gestureClear: this.container.querySelector('#gesture-clear'),
            gestureSubmit: this.container.querySelector('#gesture-submit'),
            passiveLoadingOverlay: this.container.querySelector('#passive-loading-overlay'),
            passiveProgressFill: this.container.querySelector('#passive-progress-fill'),
            passiveLoadingMessage: this.container.querySelector('#passive-loading-message'),
            jigsawContainer: this.container.querySelector('#jigsaw-container'),
            jigsawTargetGrid: this.container.querySelector('#jigsaw-target-grid'),
            jigsawPiecesContainer: this.container.querySelector('#jigsaw-pieces-container'),
            jigsawRefresh: this.container.querySelector('#jigsaw-refresh'),
            jigsawReset: this.container.querySelector('#jigsaw-reset'),
            jigsawVerify: this.container.querySelector('#jigsaw-verify'),
            jigsawLoadingOverlay: this.container.querySelector('#jigsaw-loading-overlay'),
            jigsawProgressFill: this.container.querySelector('#jigsaw-progress-fill'),
            jigsawLoadingMessage: this.container.querySelector('#jigsaw-loading-message'),
            jigsawSkeleton: this.container.querySelector('#jigsaw-skeleton'),
            result: this.container.querySelector('#captcha-result')
        };

        this.canvas = this.elements.sliderCanvas;
        this.ctx = this.canvas.getContext('2d');
    }

    bindEvents() {
        this.elements.tabs.forEach(tab => {
            tab.addEventListener('click', () => this.switchTab(tab.dataset.type));
            tab.addEventListener('keydown', (e) => this.handleTabKeyboard(e, tab));
        });

        this.elements.sliderRefresh.addEventListener('click', () => {
            this.refresh();
            this.announceToScreenReader(this.i18n.t('refreshing'));
        });
        this.elements.clickRefresh.addEventListener('click', () => {
            this.refresh();
            this.announceToScreenReader(this.i18n.t('refreshing'));
        });
        this.elements.rotationRefresh.addEventListener('click', () => {
            this.refresh();
            this.announceToScreenReader(this.i18n.t('refreshing'));
        });
        this.elements.gestureRefresh.addEventListener('click', () => {
            this.refresh();
            this.announceToScreenReader(this.i18n.t('refreshing'));
        });
        this.elements.jigsawRefresh.addEventListener('click', () => {
            this.refresh();
            this.announceToScreenReader(this.i18n.t('refreshing'));
        });

        this.bindSliderEvents();
        this.bindClickEvents();
        this.bindRotationEvents();
        this.bindGestureEvents();
        this.bindJigsawEvents();
        this.bindKeyboardShortcuts();
    }

    handleTabKeyboard(event, tab) {
        const tabs = Array.from(this.elements.tabs);
        const currentIndex = tabs.indexOf(tab);
        let nextIndex;

        switch (event.key) {
            case 'ArrowLeft':
            case 'ArrowUp':
                nextIndex = currentIndex > 0 ? currentIndex - 1 : tabs.length - 1;
                event.preventDefault();
                tabs[nextIndex].click();
                tabs[nextIndex].focus();
                break;
            case 'ArrowRight':
            case 'ArrowDown':
                nextIndex = currentIndex < tabs.length - 1 ? currentIndex + 1 : 0;
                event.preventDefault();
                tabs[nextIndex].click();
                tabs[nextIndex].focus();
                break;
            case 'Enter':
            case ' ':
                event.preventDefault();
                tab.click();
                break;
        }
    }

    bindKeyboardShortcuts() {
        const container = this.elements.sliderContainer;
        
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
                    if (this.sliderState.currentX > 10) {
                        this.verifySlider();
                    }
                    break;
                case 'Home':
                    e.preventDefault();
                    this.simulateSliderDrag(-this.sliderState.currentX);
                    break;
                case 'End':
                    e.preventDefault();
                    this.simulateSliderDrag(this.sliderState.maxX - this.sliderState.currentX);
                    break;
            }
        });
    }

    simulateSliderDrag(deltaX) {
        const newX = Math.max(0, Math.min(this.sliderState.currentX + deltaX, this.sliderState.maxX));
        this.sliderState.currentX = newX;
        this.animateSliderPosition(newX);
        this.updateSliderAccessibility();
        
        const progress = Math.round((newX / this.sliderState.maxX) * 100);
        this.announceToScreenReader(`${this.i18n.t('sliderProgress')} ${progress}%`);
    }

    updateSliderAccessibility() {
        const container = this.elements.sliderContainer;
        const progress = Math.round((this.sliderState.currentX / this.sliderState.maxX) * 100);
        container.setAttribute('aria-valuenow', progress);
    }

    bindSliderEvents() {
        const button = this.elements.sliderButton;
        const container = this.elements.sliderContainer;

        const startDrag = (e) => {
            if (this.sliderState.isDragging || this.isLoading) return;

            this.sliderState.isDragging = true;
            const clientX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            this.sliderState.startX = clientX;
            this.sliderState.currentX = 0;
            this.sliderState.maxX = container.offsetWidth - button.offsetWidth - 4;

            this.speedData = {
                points: [],
                startTime: Date.now(),
                endTime: 0,
                distance: 0,
                maxSpeed: 0
            };
            this.trajectoryData = [];

            this.addTrajectoryPoint(0, this.sliderState.puzzleY, 'start');

            button.classList.add('dragging');
            container.classList.add('is-dragging');
            this.elements.sliderText.textContent = this.i18n.t('sliding');
            this.announceToScreenReader(this.i18n.t('sliderDragStarted'), 'assertive');
        };

        const drag = (e) => {
            if (!this.sliderState.isDragging) return;

            e.preventDefault();
            const clientX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX;
            let deltaX = clientX - this.sliderState.startX;

            deltaX = Math.max(0, Math.min(deltaX, this.sliderState.maxX));
            const prevX = this.sliderState.currentX;
            this.sliderState.currentX = deltaX;

            const currentTime = Date.now();
            const dt = currentTime - (this.speedData.points.length > 0 ?
                this.speedData.points[this.speedData.points.length - 1].time : this.speedData.startTime);
            const dx = deltaX - prevX;
            const dy = 0;
            const distance = Math.sqrt(dx * dx + dy * dy);
            const speed = dt > 0 ? distance / (dt / 1000) : 0;

            this.speedData.points.push({
                x: deltaX,
                y: this.sliderState.puzzleY,
                time: currentTime,
                speed: speed
            });

            this.speedData.distance += distance;
            if (speed > this.speedData.maxSpeed) {
                this.speedData.maxSpeed = speed;
            }

            this.addTrajectoryPoint(deltaX, this.sliderState.puzzleY, 'move');

            this.animateSliderPosition(deltaX);
            this.updateSliderAccessibility();
        };

        const endDrag = (e) => {
            if (!this.sliderState.isDragging) return;

            this.sliderState.isDragging = false;
            this.speedData.endTime = Date.now();
            button.classList.remove('dragging');
            this.elements.sliderContainer.classList.remove('is-dragging');

            this.addTrajectoryPoint(this.sliderState.currentX, this.sliderState.puzzleY, 'end');

            if (this.sliderState.currentX > 10) {
                this.verifySlider();
            } else {
                this.resetSlider();
                this.announceToScreenReader(this.i18n.t('sliderCancelled'));
            }
        };

        button.addEventListener('mousedown', startDrag);
        button.addEventListener('touchstart', startDrag, { passive: false });
        
        document.addEventListener('mousemove', drag);
        document.addEventListener('touchmove', drag, { passive: false });
        
        document.addEventListener('mouseup', endDrag);
        document.addEventListener('touchend', endDrag);
    }

    addTrajectoryPoint(x, y, event) {
        this.trajectoryData.push({
            x: Math.round(x),
            y: Math.round(y),
            timestamp: Date.now(),
            event: event
        });
    }

    animateSliderPosition(x) {
        const button = this.elements.sliderButton;
        const track = this.elements.sliderTrack;

        button.style.left = (x + 2) + 'px';
        track.style.width = x + 'px';

        this.updatePuzzlePosition(x);
    }

    updatePuzzlePosition(x) {
        this.elements.sliderPuzzle.style.left = x + 'px';

        if (this.canvas && this.ctx) {
            this.drawPuzzleOverlay(x);
        }
    }

    drawPuzzleOverlay(sliderX) {
        const ctx = this.ctx;
        const canvas = this.canvas;
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        const puzzleSize = 50;
        const puzzleY = this.sliderState.puzzleY;
        const targetX = this.sliderState.targetX;

        ctx.strokeStyle = 'rgba(255, 255, 255, 0.8)';
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
            case 4:
                this.drawHexagon(ctx, targetX + puzzleSize / 2, puzzleY + puzzleSize / 2, puzzleSize / 2);
                ctx.stroke();
                break;
        }

        ctx.setLineDash([]);
    }

    drawHexagon(ctx, cx, cy, radius) {
        ctx.beginPath();
        for (let i = 0; i < 6; i++) {
            const angle = (Math.PI / 3) * i - Math.PI / 2;
            const x = cx + radius * Math.cos(angle);
            const y = cy + radius * Math.sin(angle);
            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        }
        ctx.closePath();
    }

    bindClickEvents() {
        const grid = this.elements.clickGrid;

        grid.addEventListener('click', (e) => {
            if (e.target === this.elements.clickRefresh) return;
            if (e.target === this.elements.clickImage) return;

            if (this.clickState.selectedPoints.length >= this.clickState.maxPoints) {
                this.announceToScreenReader(this.i18n.t('maxPointsReached'), 'assertive');
                return;
            }

            const rect = grid.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;

            const point = {
                x: Math.round(x),
                y: Math.round(y)
            };

            this.clickState.selectedPoints.push(point);
            this.addClickMarker(point, this.clickState.selectedPoints.length);
            this.updateClickProgress();
            this.addTrajectoryPoint(Math.round(x), Math.round(y), 'click');
            this.announceToScreenReader(
                `${this.i18n.t('pointSelected')} ${this.clickState.selectedPoints.length} ${this.i18n.t('of')} ${this.clickState.maxPoints}`,
                'assertive'
            );
        });

        grid.addEventListener('mousemove', (e) => {
            if (this.clickState.selectedPoints.length === 0) return;
            const rect = grid.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            if (Math.random() < 0.3) {
                this.addTrajectoryPoint(Math.round(x), Math.round(y), 'move');
            }
        });

        this.elements.clickClear.addEventListener('click', () => {
            this.clearClickPoints();
            this.announceToScreenReader(this.i18n.t('selectionCleared'), 'assertive');
        });

        this.elements.clickSubmit.addEventListener('click', () => {
            if (this.clickState.selectedPoints.length > 0) {
                this.verifyClick();
            } else {
                this.announceToScreenReader(this.i18n.t('noPointsSelected'), 'assertive');
            }
        });
    }

    bindRotationEvents() {
        const button = this.elements.rotationSliderButton;
        const container = this.elements.rotationSliderContainer;
        if (!button || !container) return;

        const startDrag = (e) => {
            if (this.rotationState.isDragging || this.isLoading) return;

            this.rotationState.isDragging = true;
            const clientX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            this.rotationState.startX = clientX;
            this.rotationState.dragStartAngle = this.rotationState.totalAngle;
            this.rotationState.startTime = Date.now();
            this.rotationState.trajectoryData = [];
            this.rotationState.currentAngle = 0;

            button.classList.add('dragging');
            this.elements.rotationSliderText.textContent = this.i18n.t('rotating');
            this.announceToScreenReader(this.i18n.t('rotationDragStarted'), 'assertive');
        };

        const drag = (e) => {
            if (!this.rotationState.isDragging) return;

            e.preventDefault();
            const clientX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX;
            const deltaX = clientX - this.rotationState.startX;
            const maxWidth = container.offsetWidth - button.offsetWidth - 4;
            const progress = Math.max(0, Math.min(deltaX / maxWidth, 1));
            // 从初始位置开始旋转，而不是累加
            // 用户拖动滑块时，旋转角度从初始位置开始变化
            const angleDelta = progress * 360;

            // 计算当前总角度 = 初始角度 + 拖动产生的角度变化
            let newAngle = (this.rotationState.initialAngle || 0) - angleDelta;
            
            // 归一化到 0-360 度范围
            newAngle = ((newAngle % 360) + 360) % 360;
            
            this.rotationState.currentAngle = angleDelta;
            this.rotationState.totalAngle = newAngle;

            this.elements.rotationSliderTrack.style.width = (progress * maxWidth) + 'px';
            button.style.left = (deltaX + 2) + 'px';

            if (this.elements.rotationImage) {
                this.elements.rotationImage.style.transform = 'rotate(' + this.rotationState.totalAngle + 'deg)';
            }

            this.updateRotationAngleDisplay(this.rotationState.totalAngle);
            this.rotationState.trajectoryData.push({
                angle: this.rotationState.totalAngle,
                timestamp: Date.now()
            });
        };

        const endDrag = (e) => {
            if (!this.rotationState.isDragging) return;

            this.rotationState.isDragging = false;
            button.classList.remove('dragging');
            this.elements.rotationSliderText.textContent = this.i18n.t('dragToRotate');

            // 计算用户实际旋转的角度（相对于初始位置）
            const initialAngle = this.rotationState.initialAngle || 0;
            const currentAngle = this.rotationState.totalAngle;
            
            // 计算用户将图片旋转了多少度来尝试到达 0 度
            let rotationApplied = 0;
            if (currentAngle !== undefined) {
                // 计算用户通过拖动滑块所应用的旋转量
                // 滑块从左到右对应 0-360 度的旋转
                const buttonLeft = parseInt(button.style.left) || 2;
                const progress = Math.max(0, Math.min((buttonLeft - 2) / (container.offsetWidth - button.offsetWidth - 4), 1));
                rotationApplied = progress * 360;
            }

            if (Math.abs(rotationApplied) > 5) {
                this.verifyRotation();
            }
            // 不立即重置滑块，让用户看到当前旋转状态
            // this.resetRotationSlider();
        };

        button.addEventListener('mousedown', startDrag);
        button.addEventListener('touchstart', startDrag, { passive: false });

        document.addEventListener('mousemove', drag);
        document.addEventListener('touchmove', drag, { passive: false });

        document.addEventListener('mouseup', endDrag);
        document.addEventListener('touchend', endDrag);
    }

    addClickMarker(point, index) {
        const marker = document.createElement('div');
        marker.className = 'captcha-click-marker';
        marker.style.left = point.x + 'px';
        marker.style.top = point.y + 'px';
        marker.textContent = index;
        marker.dataset.index = index - 1;
        marker.setAttribute('role', 'button');
        marker.setAttribute('aria-label', `${this.i18n.t('point')} ${index} ${this.i18n.t('removeHint')}`);

        marker.addEventListener('click', (e) => {
            e.stopPropagation();
            const idx = parseInt(marker.dataset.index);
            this.removeClickPoint(idx);
        });

        marker.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                e.stopPropagation();
                const idx = parseInt(marker.dataset.index);
                this.removeClickPoint(idx);
            }
        });

        this.elements.clickGrid.appendChild(marker);
        this.playMarkerAnimation(marker);
    }

    playMarkerAnimation(marker) {
        if (this.accessibilityState.reducedMotion) return;
        
        marker.style.animation = 'marker-pop 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards';
    }

    removeClickPoint(index) {
        this.clickState.selectedPoints.splice(index, 1);
        this.updateClickMarkers();
        this.updateClickProgress();
        this.announceToScreenReader(
            `${this.i18n.t('pointRemoved')}, ${this.clickState.selectedPoints.length} ${this.i18n.t('pointsRemaining')}`
        );
    }

    updateClickMarkers() {
        const markers = this.elements.clickGrid.querySelectorAll('.captcha-click-marker');
        markers.forEach(m => m.remove());

        this.clickState.selectedPoints.forEach((point, idx) => {
            this.addClickMarker(point, idx + 1);
        });
    }

    updateClickProgress() {
        const count = this.clickState.selectedPoints.length;
        const total = this.clickState.maxPoints;
        this.elements.clickSelectedCount.textContent = count;
        this.elements.clickTotalCount.textContent = total;
        
        const badge = this.elements.clickSelectedCount;
        badge.classList.toggle('complete', count === total);
        badge.classList.toggle('partial', count > 0 && count < total);
    }

    clearClickPoints() {
        this.clickState.selectedPoints = [];
        this.trajectoryData = [];
        const markers = this.elements.clickGrid.querySelectorAll('.captcha-click-marker');
        markers.forEach(m => m.remove());
        this.updateClickProgress();
    }

    switchTab(type) {
        this.options.type = type;

        this.elements.tabs.forEach(tab => {
            const isSelected = tab.dataset.type === type;
            tab.classList.toggle('active', isSelected);
            tab.setAttribute('aria-selected', isSelected);
        });

        this.elements.contents.forEach(content => {
            const isActive = (type === 'slider' && content.id === 'slider-captcha') ||
                          (type === 'click' && content.id === 'click-captcha') ||
                          (type === 'jigsaw' && content.id === 'jigsaw-captcha') ||
                          (type === 'rotation' && content.id === 'rotation-captcha') ||
                          (type === 'gesture' && content.id === 'gesture-captcha') ||
                          (type === 'passive' && content.id === 'passive-captcha');
            content.classList.toggle('active', isActive);
            content.hidden = !isActive;
        });

        this.clearResult();
        this.resetSlider();
        this.clearClickPoints();
        this.resetRotationSlider();
        this.clearGestureCanvas();
        this.resetJigsaw();
        this.refresh();

        const tabNames = {
            slider: this.i18n.t('sliderVerify'),
            click: this.i18n.t('clickVerify'),
            jigsaw: '拼图验证',
            rotation: this.i18n.t('rotationVerify'),
            gesture: this.i18n.t('gestureVerify'),
            passive: this.i18n.t('passiveVerify')
        };
        this.announceToScreenReader(`${this.i18n.t('switchedTo')} ${tabNames[type] || type}`);
    }

    async refresh() {
        this.clearResult();
        this.showLoading(this.options.type);

        if (this.options.onRefresh) {
            this.options.onRefresh();
        }

        try {
            try {
                this.detector = new EnvironmentDetector({ sessionId: this.sessionId });
                this.environmentData = await this.detector.runAll();
            } catch (e) {
                this.environmentData = { risk_score: 0, chain: {}, error: e.message };
            }
            if (this.options.type === 'slider') {
                await this.refreshSlider();
            } else if (this.options.type === 'click') {
                await this.refreshClick();
            } else if (this.options.type === 'jigsaw') {
                await this.refreshJigsaw();
            } else if (this.options.type === 'rotation') {
                await this.refreshRotation();
            } else if (this.options.type === 'gesture') {
                await this.refreshGesture();
            } else if (this.options.type === 'passive') {
                await this.refreshPassive();
            }
            this.announceToScreenReader(this.i18n.t('loadedSuccess'));
        } catch (error) {
            console.error('Refresh failed:', error);
            this.showResult(this.i18n.t('loadFailed'), 'error');
            this.announceToScreenReader(this.i18n.t('loadFailed'), 'assertive');
        } finally {
            this.hideLoading(this.options.type);
        }
    }

    async refreshSlider() {
        this.resetSlider();
        this.animateSkeletonIn();

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/slider`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });

            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                this.sliderState.targetX = data.target_x;
                this.sliderState.targetY = data.target_y;
                this.sliderState.puzzleY = data.target_y;
                this.sliderState.puzzleStyle = data.puzzle_style || 0;
                this.sliderState.tolerance = data.tolerance || 10;

                await this.loadImageToCanvas(data.image_url);

                this.updatePuzzlePiece();
                this.drawPuzzleOverlay(0);
                this.animateSkeletonOut();
            } else {
                this.showResult(this.i18n.t('loadFailed'), 'error');
                this.animateSkeletonOut();
            }
        } catch (error) {
            console.error('Slider refresh failed:', error);
            this.showResult(this.i18n.t('loadFailed'), 'error');
            this.animateSkeletonOut();
        }
    }

    animateSkeletonIn() {
        const skeleton = this.elements.sliderSkeleton;
        if (skeleton) {
            skeleton.classList.add('active');
            skeleton.style.display = 'block';
        }
    }

    animateSkeletonOut() {
        const skeleton = this.elements.sliderSkeleton;
        if (skeleton) {
            skeleton.classList.remove('active');
            setTimeout(() => {
                skeleton.style.display = 'none';
            }, 300);
        }
    }

    async loadImageToCanvas(imageUrl) {
        return new Promise((resolve, reject) => {
            const img = new Image();
            img.crossOrigin = 'anonymous';

            img.onload = () => {
                const canvas = this.canvas;
                const ctx = this.ctx;
                canvas.width = 360;
                canvas.height = 220;

                this.animateImageLoad(canvas);
                ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
                resolve();
            };

            img.onerror = () => {
                this.drawGradientBackground();
                resolve();
            };

            img.src = imageUrl;
        });
    }

    animateImageLoad(canvas) {
        if (this.accessibilityState.reducedMotion) return;
        
        canvas.style.opacity = '0';
        canvas.style.transition = 'opacity 0.3s ease';
        requestAnimationFrame(() => {
            canvas.style.opacity = '1';
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
        ctx.fillText(this.i18n.t('dragToVerify'), canvas.width / 2, canvas.height / 2);
    }

    updatePuzzlePiece() {
        const puzzleY = this.sliderState.puzzleY;
        const puzzleStyle = this.sliderState.puzzleStyle;

        let puzzleShape = '';
        switch (puzzleStyle) {
            case 0:
                puzzleShape = `<div class="puzzle-piece-square"></div>`;
                break;
            case 1:
                puzzleShape = `<div class="puzzle-piece-circle"></div>`;
                break;
            case 2:
                puzzleShape = `<div class="puzzle-piece-triangle"></div>`;
                break;
            case 3:
                puzzleShape = `<div class="puzzle-piece-diamond"></div>`;
                break;
            case 4:
                puzzleShape = `<div class="puzzle-piece-hexagon"></div>`;
                break;
            default:
                puzzleShape = `<div class="puzzle-piece-square"></div>`;
        }

        this.elements.sliderPuzzle.innerHTML = puzzleShape;
        this.elements.sliderPuzzle.style.top = puzzleY + 'px';
    }

    loadDemoSlider() {
        this.sessionId = 'demo_' + Date.now();
        this.sliderState.targetX = 200;
        this.sliderState.targetY = 70;
        this.sliderState.puzzleY = 70;
        this.sliderState.puzzleStyle = 0;
        this.sliderState.tolerance = 10;

        this.drawGradientBackground();
        this.updatePuzzlePiece();
        this.drawPuzzleOverlay(0);
        this.animateSkeletonOut();
    }

    async refreshClick() {
        this.clearClickPoints();
        this.animateClickSkeletonIn();

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/click`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });

            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                this.elements.clickImage.src = data.image_url;
                this.clickState.hintText = data.hint || this.i18n.t('clickHint');
                this.clickState.maxPoints = data.max_points || 3;
                this.elements.clickHint.querySelector('.hint-text').textContent = this.clickState.hintText;
                this.elements.clickTotalCount.textContent = this.clickState.maxPoints;
                this.animateClickSkeletonOut();
            } else {
                this.showResult(this.i18n.t('loadFailed'), 'error');
                this.animateClickSkeletonOut();
            }
        } catch (error) {
            console.error('Click refresh failed:', error);
            this.showResult(this.i18n.t('loadFailed'), 'error');
            this.animateClickSkeletonOut();
        }
    }

    animateClickSkeletonIn() {
        const skeleton = this.elements.clickSkeleton;
        if (skeleton) {
            skeleton.classList.add('active');
            skeleton.style.display = 'block';
        }
    }

    animateClickSkeletonOut() {
        const skeleton = this.elements.clickSkeleton;
        if (skeleton) {
            skeleton.classList.remove('active');
            setTimeout(() => {
                skeleton.style.display = 'none';
            }, 300);
        }
    }

    loadDemoClick() {
        this.sessionId = 'demo_' + Date.now();
        this.clickState.hintText = this.i18n.t('demoClickHint');
        this.clickState.maxPoints = 3;
        this.elements.clickHint.querySelector('.hint-text').textContent = this.clickState.hintText;
        this.elements.clickTotalCount.textContent = this.clickState.maxPoints;

        this.elements.clickImage.src = 'data:image/svg+xml,' + encodeURIComponent(`
            <svg xmlns="http://www.w3.org/2000/svg" width="360" height="220">
                <defs>
                    <linearGradient id="bg2" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" style="stop-color:#f093fb"/>
                        <stop offset="100%" style="stop-color:#f5576c"/>
                    </linearGradient>
                </defs>
                <rect width="100%" height="100%" fill="url(#bg2)"/>
                <text x="60" y="110" text-anchor="middle" fill="white" font-size="32" font-family="Arial" font-weight="bold">1</text>
                <text x="180" y="110" text-anchor="middle" fill="white" font-size="32" font-family="Arial" font-weight="bold">2</text>
                <text x="300" y="110" text-anchor="middle" fill="white" font-size="32" font-family="Arial" font-weight="bold">3</text>
            </svg>
        `);
        this.animateClickSkeletonOut();
    }

    async refreshRotation() {
        this.resetRotationSlider();
        this.animateRotationSkeletonIn();

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/rotation`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });

            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                this.rotationState.challengeId = data.session_id;
                this.rotationState.imageUrl = data.image_url;
                this.rotationState.initialAngle = data.initial_angle || 0;

                if (this.elements.rotationImage) {
                    this.elements.rotationImage.src = data.image_url;
                    // 设置初始旋转角度
                    this.rotationState.totalAngle = data.initial_angle || 0;
                    this.elements.rotationImage.style.transform = 'rotate(' + this.rotationState.totalAngle + 'deg)';
                    this.updateRotationAngleDisplay(this.rotationState.totalAngle);
                }

                this.animateRotationSkeletonOut();
            } else {
                this.showResult(this.i18n.t('loadFailed'), 'error');
                this.animateRotationSkeletonOut();
            }
        } catch (error) {
            console.error('Rotation refresh failed:', error);
            this.showResult(this.i18n.t('loadFailed'), 'error');
            this.animateRotationSkeletonOut();
        }
    }

    animateRotationSkeletonIn() {
        const skeleton = this.elements.rotationSkeleton;
        if (skeleton) {
            skeleton.classList.add('active');
            skeleton.style.display = 'block';
        }
    }

    animateRotationSkeletonOut() {
        const skeleton = this.elements.rotationSkeleton;
        if (skeleton) {
            skeleton.classList.remove('active');
            setTimeout(() => {
                skeleton.style.display = 'none';
            }, 300);
        }
    }

    async refreshPassive() {
        this.showLoading('passive');
        try {
            const envCheck = document.getElementById('passive-check-env');
            const behaviorCheck = document.getElementById('passive-check-behavior');
            const riskCheck = document.getElementById('passive-check-risk');
            const statusEl = document.getElementById('passive-status');
            const riskValue = document.getElementById('passive-risk-value');

            await this.sleep(800);
            if (envCheck) {
                envCheck.innerHTML = '<i class="fas fa-check-circle text-success me-2"></i>' + this.i18n.t('passiveCheckEnvDone');
                envCheck.style.color = '#52c41a';
            }

            await this.sleep(600);
            if (behaviorCheck) {
                behaviorCheck.innerHTML = '<i class="fas fa-check-circle text-success me-2"></i>' + this.i18n.t('passiveCheckBehaviorDone');
                behaviorCheck.style.color = '#52c41a';
            }

            await this.sleep(500);
            const riskScore = Math.floor(Math.random() * 20) + 10;
            if (riskValue) {
                riskValue.textContent = riskScore + '%';
                riskValue.style.color = riskScore < 30 ? '#52c41a' : (riskScore < 60 ? '#faad14' : '#ff4d4f');
            }
            if (riskCheck) {
                riskCheck.innerHTML = '<i class="fas fa-check-circle text-success me-2"></i>' + this.i18n.t('passiveCheckRiskDone');
                riskCheck.style.color = '#52c41a';
            }

            if (statusEl) {
                statusEl.textContent = this.i18n.t('passiveSuccess');
                statusEl.style.color = '#52c41a';
            }

            this.showResult(this.i18n.t('verifySuccess'), 'success');
            this.announceToScreenReader(this.i18n.t('verifySuccess'), 'assertive');
            if (this.options.onSuccess) {
                this.options.onSuccess({ type: 'passive', session_id: 'passive_' + Date.now() });
            }
        } catch (error) {
            this.showResult(this.i18n.t('verifyFailed'), 'error');
            this.announceToScreenReader(this.i18n.t('verifyFailed'), 'assertive');
            if (this.options.onError) {
                this.options.onError({ type: 'passive', error: error.message || this.i18n.t('verifyFailed') });
            }
        } finally {
            setTimeout(() => {
                this.hideLoading('passive');
            }, 500);
        }
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    loadDemoRotation() {
        this.sessionId = 'demo_' + Date.now();
        this.rotationState.challengeId = 'demo_' + Date.now();
        this.rotationState.totalAngle = 0;

        if (this.elements.rotationImage) {
            this.elements.rotationImage.src = 'data:image/svg+xml,' + encodeURIComponent(`
                <svg xmlns="http://www.w3.org/2000/svg" width="200" height="200">
                    <defs>
                        <linearGradient id="bg3" x1="0%" y1="0%" x2="100%" y2="100%">
                            <stop offset="0%" style="stop-color:#667eea"/>
                            <stop offset="100%" style="stop-color:#764ba2"/>
                        </linearGradient>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#bg3)"/>
                    <text x="100" y="110" text-anchor="middle" fill="white" font-size="24" font-family="Arial">旋转验证</text>
                </svg>
            `);
            this.elements.rotationImage.style.transform = 'rotate(0deg)';
        }
        this.animateRotationSkeletonOut();
    }

    resetRotationSlider() {
        this.rotationState.isDragging = false;
        this.rotationState.currentAngle = 0;
        this.rotationState.trajectoryData = [];

        if (this.elements.rotationSliderButton) {
            this.elements.rotationSliderButton.style.left = '2px';
            this.elements.rotationSliderButton.classList.remove('dragging');
        }
        if (this.elements.rotationSliderTrack) {
            this.elements.rotationSliderTrack.style.width = '0px';
        }
        if (this.elements.rotationSliderText) {
            this.elements.rotationSliderText.textContent = this.i18n.t('dragToRotate');
        }
        this.updateRotationAngleDisplay(0);
    }

    updateRotationAngleDisplay(angle) {
        if (this.elements.rotationAngleValue) {
            this.elements.rotationAngleValue.textContent = Math.round(angle) + '°';
        }
    }

    async verifyRotation() {
        this.showLoading('rotation');

        // 用户需要将图片旋转到 0 度
        // 我们计算用户当前旋转了多少度来试图达到 0 度
        const button = this.elements.rotationSliderButton;
        const container = this.elements.rotationSliderContainer;
        
        const buttonLeft = parseInt(button.style.left) || 2;
        const maxWidth = container.offsetWidth - button.offsetWidth - 4;
        const progress = Math.max(0, Math.min((buttonLeft - 2) / maxWidth, 1));
        const userRotation = progress * 360;
        
        const initialAngle = this.rotationState.initialAngle || 0;
        const finalAngle = ((initialAngle - userRotation) % 360 + 360) % 360;
        
        const payload = {
            session_id: this.sessionId,
            angle: finalAngle
        };

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/rotation/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            let success = false;
            let message = '';
            if (response.ok) {
                const data = await response.json();
                success = data.success;
                message = data.message || '';
            }

            if (success) {
                this.showResult(this.i18n.t('verifySuccess'), 'success');
                this.announceToScreenReader(this.i18n.t('verifySuccess'), 'assertive');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ type: 'rotation', session_id: this.sessionId });
                }
            } else {
                const errorMessage = message || this.i18n.t('verifyFailed');
                this.showResult(errorMessage, 'error');
                this.announceToScreenReader(errorMessage, 'assertive');
                setTimeout(() => this.refresh(), 1500);
                if (this.options.onError) {
                    this.options.onError({ type: 'rotation', error: errorMessage });
                }
            }
        } catch (error) {
            const errorMessage = error.message || this.i18n.t('verifyFailed');
            this.showResult(errorMessage, 'error');
            this.announceToScreenReader(errorMessage, 'assertive');
            setTimeout(() => this.refresh(), 1500);
            if (this.options.onError) {
                this.options.onError({ type: 'rotation', error: errorMessage });
            }
        } finally {
            setTimeout(() => {
                this.hideLoading('rotation');
            }, 500);
        }
    }

    calculateSpeedData() {
        const speedData = {
            start_time: this.speedData.startTime,
            end_time: this.speedData.endTime,
            distance: this.speedData.distance,
            average_speed: 0,
            max_speed: this.speedData.maxSpeed,
            has_accelerate: false
        };

        const duration = (this.speedData.endTime - this.speedData.startTime) / 1000;
        if (duration > 0) {
            speedData.average_speed = this.speedData.distance / duration;
        }

        if (this.speedData.points.length >= 3) {
            let accelerateCount = 0;
            for (let i = 2; i < this.speedData.points.length; i++) {
                const prevSpeed = this.speedData.points[i - 1].speed;
                const currSpeed = this.speedData.points[i].speed;
                if (Math.abs(currSpeed - prevSpeed) > 50) {
                    accelerateCount++;
                }
            }
            speedData.has_accelerate = accelerateCount > this.speedData.points.length * 0.2;
        }

        return speedData;
    }

    async verifySlider() {
        this.showLoading('slider');
        this.playVerificationAnimation();

        const speedData = this.calculateSpeedData();

        let payload = {
            session_id: this.sessionId,
            x: Math.round(this.sliderState.currentX),
            y: this.sliderState.puzzleY,
            type: 'slider',
            behavior_data: this.trajectoryData,
            speed_data: speedData,
            environment_data: this.environmentData
        };

        if (this.options.enableEncryption && this.trajectoryData.length > 0) {
            try {
                const encryptedTrajectory = await this.trajectoryEncryptor.encryptTrajectory(this.trajectoryData);
                payload.encrypted_trajectory = encryptedTrajectory;
                delete payload.behavior_data;
            } catch (error) {
                console.error('Trajectory encryption failed:', error);
            }
        }

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            let success = false;
            if (response.ok) {
                const data = await response.json();
                success = data.success;
            }

            this.handleVerificationResult(success);
        } catch (error) {
            this.handleVerificationResult(false);
        } finally {
            setTimeout(() => {
                this.hideLoading('slider');
            }, 500);
        }
    }

    playVerificationAnimation() {
        if (this.accessibilityState.reducedMotion) return;
        
        this.elements.sliderButton.classList.add('verifying');
        this.elements.sliderText.textContent = this.i18n.t('verifying');
    }

    handleVerificationResult(success) {
        if (success) {
            this.elements.sliderButton.classList.remove('verifying');
            this.elements.sliderButton.classList.add('success');
            this.playSuccessAnimation();
            this.elements.sliderText.textContent = this.i18n.t('verifySuccess');
            this.showResult(this.i18n.t('verifySuccess'), 'success');
            this.announceToScreenReader(this.i18n.t('verifySuccess'), 'assertive');
            this.disableSlider();
            if (this.options.onSuccess) {
                this.options.onSuccess({ type: 'slider', session_id: this.sessionId });
            }
        } else {
            this.elements.sliderButton.classList.remove('verifying');
            this.elements.sliderButton.classList.add('error');
            this.playErrorAnimation();
            this.showResult(this.i18n.t('verifyFailed'), 'error');
            this.announceToScreenReader(this.i18n.t('verifyFailed'), 'assertive');
            setTimeout(() => this.refresh(), 1500);
            if (this.options.onError) {
                this.options.onError({ type: 'slider', error: this.i18n.t('verifyFailed') });
            }
        }
    }

    disableSlider() {
        this.elements.sliderButton.style.pointerEvents = 'none';
        this.elements.sliderContainer.style.cursor = 'not-allowed';
    }

    playSuccessAnimation() {
        const button = this.elements.sliderButton;
        const finalX = this.sliderState.currentX;

        if (this.accessibilityState.reducedMotion) {
            button.style.left = (finalX + 2) + 'px';
            this.updatePuzzlePosition(finalX);
            return;
        }

        let progress = 0;
        const animate = () => {
            progress += 0.05;
            if (progress >= 1) {
                button.style.left = (finalX + 2) + 'px';
                this.updatePuzzlePosition(finalX);
                return;
            }

            const overshoot = Math.sin(progress * Math.PI) * 10;
            const easeOut = 1 - Math.pow(1 - progress, 3);
            const currentX = finalX * easeOut - overshoot * (1 - easeOut);

            button.style.left = Math.max(2, currentX + 2) + 'px';
            this.updatePuzzlePosition(Math.max(0, currentX));

            requestAnimationFrame(animate);
        };

        requestAnimationFrame(animate);
        this.playSuccessParticles();
    }

    playSuccessParticles() {
        const container = this.elements.sliderContainer;
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
                background: #52c41a;
                border-radius: 50%;
                pointer-events: none;
                z-index: 100;
            `;
            document.body.appendChild(particle);
            
            const angle = (i / 8) * Math.PI * 2;
            const velocity = 50 + Math.random() * 50;
            const vx = Math.cos(angle) * velocity;
            const vy = Math.sin(angle) * velocity;
            
            let x = 0, y = 0, opacity = 1;
            const animate = () => {
                x += vx * 0.02;
                y += vy * 0.02;
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
        const button = this.elements.sliderButton;
        const originalX = this.sliderState.currentX;

        if (this.accessibilityState.reducedMotion) {
            button.style.left = '2px';
            this.updatePuzzlePosition(0);
            this.resetSlider();
            return;
        }

        let shakeCount = 0;
        const maxShakes = 6;
        const shakeDistance = 15;

        const shake = () => {
            shakeCount++;
            if (shakeCount > maxShakes) {
                button.style.left = '2px';
                this.updatePuzzlePosition(0);
                this.resetSlider();
                return;
            }

            const direction = shakeCount % 2 === 0 ? 1 : -1;
            const decay = 1 - (shakeCount / maxShakes);
            const offset = shakeDistance * decay * direction;

            button.style.left = (2 + offset) + 'px';
            this.updatePuzzlePosition(offset);

            setTimeout(shake, 50);
        };

        shake();
        this.playErrorFlash();
    }

    playErrorFlash() {
        const container = this.elements.sliderContainer;
        container.classList.add('error-flash');
        setTimeout(() => {
            container.classList.remove('error-flash');
        }, 500);
    }

    simulateSliderVerify() {
        const tolerance = this.sliderState.tolerance;
        const targetX = this.sliderState.targetX;

        const hasSpeedData = this.speedData.distance > 0;
        if (!hasSpeedData) {
            return false;
        }

        const speedValid = this.speedData.average_speed > 5 &&
                          this.speedData.average_speed < 2000;

        if (!speedValid) {
            return false;
        }

        const positionValid = Math.abs(this.sliderState.currentX - targetX) <= tolerance * 2;

        return positionValid;
    }

    async verifyClick() {
        this.showLoading('click');

        const pointsArr = this.clickState.selectedPoints.map(p => [p.x, p.y]);
        const clickSeq = this.clickState.selectedPoints.map((_, i) => i);

        const payload = {
            session_id: this.sessionId,
            points: pointsArr,
            click_sequence: clickSeq,
            behavior_data: this.trajectoryData,
            type: 'click',
            environment_data: this.environmentData
        };

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            let success = false;
            if (response.ok) {
                const data = await response.json();
                success = data.success;
            }

            if (success) {
                this.showResult(this.i18n.t('verifySuccess'), 'success');
                this.playClickSuccessAnimation();
                this.announceToScreenReader(this.i18n.t('verifySuccess'), 'assertive');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ type: 'click', session_id: this.sessionId });
                }
            } else {
                this.showResult(this.i18n.t('verifyFailed'), 'error');
                this.playClickErrorAnimation();
                this.announceToScreenReader(this.i18n.t('verifyFailed'), 'assertive');
                setTimeout(() => this.refresh(), 1500);
                if (this.options.onError) {
                    this.options.onError({ type: 'click', error: this.i18n.t('verifyFailed') });
                }
            }
        } catch (error) {
            this.showResult(this.i18n.t('verifyFailed'), 'error');
            this.playClickErrorAnimation();
            this.announceToScreenReader(this.i18n.t('verifyFailed'), 'assertive');
            setTimeout(() => this.refresh(), 1500);
            if (this.options.onError) {
                this.options.onError({ type: 'click', error: error.message || this.i18n.t('verifyFailed') });
            }
        } finally {
            setTimeout(() => {
                this.hideLoading('click');
            }, 500);
        }
    }

    playClickSuccessAnimation() {
        if (this.accessibilityState.reducedMotion) return;
        
        const markers = this.elements.clickGrid.querySelectorAll('.captcha-click-marker');
        markers.forEach((marker, index) => {
            setTimeout(() => {
                marker.classList.add('success-marker');
            }, index * 100);
        });
    }

    playClickErrorAnimation() {
        if (this.accessibilityState.reducedMotion) return;
        
        const grid = this.elements.clickGrid;
        grid.classList.add('error-shake');
        setTimeout(() => {
            grid.classList.remove('error-shake');
        }, 500);
    }

    simulateClickVerify() {
        return this.clickState.selectedPoints.length === 3;
    }

    resetSlider() {
        this.sliderState.isDragging = false;
        this.sliderState.currentX = 0;
        this.elements.sliderButton.style.left = '2px';
        this.elements.sliderButton.style.pointerEvents = 'auto';
        this.elements.sliderContainer.style.cursor = 'pointer';
        this.elements.sliderButton.classList.remove('success', 'error', 'dragging', 'verifying');
        this.elements.sliderTrack.style.width = '0px';
        this.elements.sliderText.textContent = this.i18n.t('dragToVerify');
        this.elements.sliderPuzzle.style.left = '0px';
        this.updateSliderAccessibility();

        this.trajectoryData = [];
        this.speedData = {
            points: [],
            startTime: 0,
            endTime: 0,
            distance: 0,
            maxSpeed: 0
        };

        if (this.ctx) {
            this.drawPuzzleOverlay(0);
        }
    }

    showResult(message, type) {
        this.elements.result.textContent = message;
        this.elements.result.className = 'captcha-result show ' + type;
        this.elements.result.hidden = false;
    }

    clearResult() {
        this.elements.result.classList.remove('show', 'success', 'error');
        this.elements.result.hidden = true;
    }

    showLoading(type) {
        this.isLoading = true;
        this.loadingState.isLoading = true;
        this.loadingState.loadingType = type;

        const overlay = type === 'slider' ?
            this.elements.sliderLoadingOverlay :
            type === 'click' ?
            this.elements.clickLoadingOverlay :
            type === 'rotation' ?
            this.elements.rotationLoadingOverlay :
            type === 'gesture' ?
            this.elements.gestureLoadingOverlay :
            this.elements.passiveLoadingOverlay;

        if (overlay) {
            overlay.hidden = false;
            this.animateLoadingProgress(type);
        }

        if (this.options.onLoadStart) {
            this.options.onLoadStart();
        }
    }

    animateLoadingProgress(type) {
        if (this.accessibilityState.reducedMotion) {
            this.setLoadingMessage(type, this.i18n.t('loading'));
            return;
        }

        let progress = 0;
        const progressFill = type === 'slider' ?
            this.elements.sliderProgressFill :
            type === 'click' ?
            this.elements.clickProgressFill :
            type === 'rotation' ?
            this.elements.rotationProgressFill :
            type === 'gesture' ?
            this.elements.gestureProgressFill :
            this.elements.passiveProgressFill;
        const loadingMessage = type === 'slider' ?
            this.elements.sliderLoadingMessage :
            type === 'click' ?
            this.elements.clickLoadingMessage :
            type === 'rotation' ?
            this.elements.rotationLoadingMessage :
            type === 'gesture' ?
            this.elements.gestureLoadingMessage :
            this.elements.passiveLoadingMessage;

        const messages = [
            this.i18n.t('loading'),
            this.i18n.t('generating'),
            this.i18n.t('almostDone')
        ];
        let messageIndex = 0;

        const animate = () => {
            if (!this.isLoading) {
                progressFill.style.width = '0%';
                return;
            }

            progress += Math.random() * 15;
            if (progress > 100) progress = 95;

            progressFill.style.width = progress + '%';

            if (progress > (messageIndex + 1) * 33 && messageIndex < messages.length - 1) {
                messageIndex++;
                loadingMessage.textContent = messages[messageIndex];
            }

            if (progress < 100) {
                requestAnimationFrame(animate);
            }
        };

        requestAnimationFrame(animate);
    }

    setLoadingMessage(type, message) {
        const loadingMessage = type === 'slider' ?
            this.elements.sliderLoadingMessage :
            type === 'click' ?
            this.elements.clickLoadingMessage :
            type === 'rotation' ?
            this.elements.rotationLoadingMessage :
            type === 'gesture' ?
            this.elements.gestureLoadingMessage :
            this.elements.passiveLoadingMessage;
        if (loadingMessage) {
            loadingMessage.textContent = message;
        }
    }

    hideLoading(type) {
        this.isLoading = false;
        this.loadingState.isLoading = false;

        const overlay = type === 'slider' ?
            this.elements.sliderLoadingOverlay :
            type === 'click' ?
            this.elements.clickLoadingOverlay :
            type === 'rotation' ?
            this.elements.rotationLoadingOverlay :
            type === 'gesture' ?
            this.elements.gestureLoadingOverlay :
            this.elements.passiveLoadingOverlay;

        if (overlay) {
            this.animateLoadingOut(overlay);
        }

        if (this.options.onLoadEnd) {
            this.options.onLoadEnd();
        }
    }

    animateLoadingOut(overlay) {
        if (this.accessibilityState.reducedMotion) {
            overlay.hidden = true;
            return;
        }

        overlay.style.opacity = '0';
        overlay.style.transition = 'opacity 0.3s ease';
        
        setTimeout(() => {
            overlay.hidden = true;
            overlay.style.opacity = '1';
        }, 300);
    }

    reset() {
        this.clearResult();
        this.resetSlider();
        this.clearClickPoints();
        this.switchTab('slider');
        this.refresh();
    }

    setLanguage(lang) {
        this.options.language = lang;
        this.i18n = new CaptchaI18n(lang);
        this.updateUIText();
    }

    updateUIText() {
        const header = this.container.querySelector('.captcha-header h3');
        const subtitle = this.container.querySelector('.captcha-header p');
        if (header) header.textContent = this.i18n.t('securityVerify');
        if (subtitle) subtitle.textContent = this.i18n.t('completeVerify');

        const tabs = this.container.querySelectorAll('.captcha-tab');
        tabs.forEach(tab => {
            const textSpan = tab.querySelector('.tab-text');
            if (textSpan) {
                const tabNames = {
                    slider: this.i18n.t('sliderVerify'),
                    click: this.i18n.t('clickVerify'),
                    rotation: this.i18n.t('rotationVerify'),
                    gesture: this.i18n.t('gestureVerify'),
                    passive: this.i18n.t('passiveVerify')
                };
                textSpan.textContent = tabNames[tab.dataset.type] || tab.dataset.type;
            }
        });

        this.resetSlider();
        this.clearClickPoints();
        this.resetRotationSlider();
    }

    destroy() {
        if (this.animationFrame) {
            cancelAnimationFrame(this.animationFrame);
        }

        if (this.accessibilityState.liveRegion) {
            this.accessibilityState.liveRegion.remove();
        }

        this.container.innerHTML = '';
        this.container = null;
        this.elements = null;
    }

    // ==================== Gesture Captcha Methods ====================
    
    bindGestureEvents() {
        const canvas = this.elements.gestureDrawingCanvas;
        if (!canvas) return;

        canvas.addEventListener('mousedown', (e) => this.startGestureDrawing(e));
        canvas.addEventListener('mousemove', (e) => this.gestureDraw(e));
        canvas.addEventListener('mouseup', (e) => this.endGestureDrawing(e));
        canvas.addEventListener('mouseleave', (e) => this.endGestureDrawing(e));
        
        canvas.addEventListener('touchstart', (e) => {
            e.preventDefault();
            this.startGestureDrawing(e.touches[0]);
        }, { passive: false });
        canvas.addEventListener('touchmove', (e) => {
            e.preventDefault();
            this.gestureDraw(e.touches[0]);
        }, { passive: false });
        canvas.addEventListener('touchend', (e) => this.endGestureDrawing(e));

        this.elements.gestureClear.addEventListener('click', () => this.clearGestureCanvas());
        this.elements.gestureSubmit.addEventListener('click', () => this.verifyGesture());
    }

    getPatternName(pattern) {
        const names = {
            'checkmark': '对勾',
            'cross': '叉号',
            'triangle': '三角形',
            'circle': '圆形',
            'star': '星形',
            'square': '正方形'
        };
        return names[pattern] || pattern;
    }

    async refreshGesture() {
        this.clearGestureCanvas();

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/gesture`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });

            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                this.gestureState.pattern = data.pattern;

                const bgCanvas = this.elements.gestureBackgroundCanvas;
                const ctx = bgCanvas.getContext('2d');
                const img = new Image();
                img.crossOrigin = 'anonymous';
                img.onload = () => {
                    bgCanvas.width = 300;
                    bgCanvas.height = 300;
                    ctx.drawImage(img, 0, 0);
                };
                img.src = data.image_url;

                const patternName = this.getPatternName(data.pattern);
                this.elements.gestureHintText.textContent = `请绘制${patternName}图案`;
            } else {
                this.showResult(this.i18n.t('loadFailed'), 'error');
            }
        } catch (error) {
            console.error('Gesture refresh failed:', error);
            this.showResult(this.i18n.t('loadFailed'), 'error');
        }
    }

    async refreshPassive() {
        try {
            // Step 1: Update check UI
            const envCheckEl = this.container.querySelector('#passive-check-env');
            const behaviorCheckEl = this.container.querySelector('#passive-check-behavior');
            const riskCheckEl = this.container.querySelector('#passive-check-risk');
            const statusEl = this.container.querySelector('#passive-status');
            const riskValueEl = this.container.querySelector('#passive-risk-value');

            if (envCheckEl) {
                await this.animateCheck(envCheckEl, this.i18n.t('passiveCheckEnvDone'));
            }

            // Step 2: Call seamless verify endpoint
            const payload = this.environmentData || {};
            const response = await fetch(`${this.options.apiBase}/seamless/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;

                if (behaviorCheckEl) {
                    await this.animateCheck(behaviorCheckEl, this.i18n.t('passiveCheckBehaviorDone'));
                }
                if (riskCheckEl) {
                    await this.animateCheck(riskCheckEl, this.i18n.t('passiveCheckRiskDone'));
                }
                if (riskValueEl) {
                    riskValueEl.textContent = data.risk_score || 0;
                }
                if (statusEl) {
                    statusEl.textContent = this.i18n.t('passiveSuccess');
                }

                if (data.decision === 'allow') {
                    if (this.options.onSuccess) {
                        this.options.onSuccess({
                            session_id: data.session_id,
                            risk_score: data.risk_score
                        });
                    }
                    this.showResult(this.i18n.t('verifySuccess'), 'success');
                } else {
                    if (this.options.onError) {
                        this.options.onError({ error: 'Verification failed' });
                    }
                    this.showResult(this.i18n.t('verifyFailed'), 'error');
                }
            } else {
                this.showResult(this.i18n.t('loadFailed'), 'error');
            }
        } catch (error) {
            console.error('Passive refresh failed:', error);
            this.showResult(this.i18n.t('loadFailed'), 'error');
        }
    }

    async animateCheck(element, text) {
        return new Promise(resolve => {
            setTimeout(() => {
                element.innerHTML = `<i class="fas fa-check-circle text-success me-2"></i>${text}`;
                element.style.color = '#52c41a';
                resolve();
            }, 500);
        });
    }

    startGestureDrawing(e) {
        if (this.isLoading) return;
        
        const canvas = this.elements.gestureDrawingCanvas;
        const rect = canvas.getBoundingClientRect();
        
        this.gestureState.isDrawing = true;
        this.gestureState.lastX = e.clientX - rect.left;
        this.gestureState.lastY = e.clientY - rect.top;
        this.gestureState.trajectoryData = [];
        
        this.addGestureTrajectoryPoint(this.gestureState.lastX, this.gestureState.lastY);
    }

    gestureDraw(e) {
        if (!this.gestureState.isDrawing || this.isLoading) return;
        
        const canvas = this.elements.gestureDrawingCanvas;
        const rect = canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        
        const ctx = canvas.getContext('2d');
        ctx.beginPath();
        ctx.moveTo(this.gestureState.lastX, this.gestureState.lastY);
        ctx.lineTo(x, y);
        ctx.strokeStyle = '#3b82f6';
        ctx.lineWidth = 4;
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';
        ctx.stroke();
        
        this.gestureState.lastX = x;
        this.gestureState.lastY = y;
        
        this.addGestureTrajectoryPoint(x, y);
    }

    endGestureDrawing(e) {
        if (!this.gestureState.isDrawing) return;
        this.gestureState.isDrawing = false;
    }

    addGestureTrajectoryPoint(x, y) {
        this.gestureState.trajectoryData.push({
            x: x,
            y: y,
            t: Date.now()
        });
    }

    clearGestureCanvas() {
        const canvas = this.elements.gestureDrawingCanvas;
        if (canvas) {
            const ctx = canvas.getContext('2d');
            ctx.clearRect(0, 0, canvas.width, canvas.height);
        }
        this.gestureState.trajectoryData = [];
        this.gestureState.isDrawing = false;
    }

    async verifyGesture() {
        if (this.gestureState.trajectoryData.length < 5) {
            this.showResult('请先绘制手势图案', 'error');
            return;
        }

        this.showLoading('gesture');

        try {
            let payload = {
                session_id: this.sessionId
            };

            if (this.options.enableEncryption) {
                const encrypted = await this.trajectoryEncryptor.encryptTrajectory(
                    this.gestureState.trajectoryData
                );
                payload.encrypted_trajectory = encrypted;
            } else {
                payload.trajectory = this.gestureState.trajectoryData;
            }

            const response = await fetch(`${this.options.apiBase}/captcha/gesture/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            let success = false;
            let message = this.i18n.t('verifyFailed');
            
            if (response.ok) {
                const data = await response.json();
                success = data.success;
                message = data.message || message;
            }

            if (success) {
                this.showResult(this.i18n.t('verifySuccess'), 'success');
                this.playGestureSuccessAnimation();
                this.announceToScreenReader(this.i18n.t('verifySuccess'), 'assertive');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ type: 'gesture', session_id: this.sessionId });
                }
            } else {
                this.showResult(message, 'error');
                this.playGestureErrorAnimation();
                this.announceToScreenReader(message, 'assertive');
                setTimeout(() => this.refresh(), 1500);
                if (this.options.onError) {
                    this.options.onError({ type: 'gesture', error: message });
                }
            }
        } catch (error) {
            console.error('Gesture verify failed:', error);
            this.showResult(this.i18n.t('verifyFailed'), 'error');
            this.playGestureErrorAnimation();
            setTimeout(() => this.refresh(), 1500);
            if (this.options.onError) {
                this.options.onError({ type: 'gesture', error: error.message || this.i18n.t('verifyFailed') });
            }
        } finally {
            setTimeout(() => {
                this.hideLoading('gesture');
            }, 500);
        }
    }

    playGestureSuccessAnimation() {
        if (this.accessibilityState.reducedMotion) return;
        
        const canvas = this.elements.gestureDrawingCanvas;
        canvas.classList.add('success-flash');
        setTimeout(() => {
            canvas.classList.remove('success-flash');
        }, 500);
    }

    playGestureErrorAnimation() {
        if (this.accessibilityState.reducedMotion) {
            this.clearGestureCanvas();
            return;
        }
        
        const canvas = this.elements.gestureDrawingCanvas;
        canvas.classList.add('error-shake');
        setTimeout(() => {
            canvas.classList.remove('error-shake');
            this.clearGestureCanvas();
        }, 500);
    }
}

class CaptchaI18n {
    constructor(locale = 'zh-CN') {
        this.locale = locale;
        this.translations = {
            'zh-CN': {
                captchaLabel: '安全验证码组件',
                securityVerify: '安全验证',
                completeVerify: '请完成以下验证以继续',
                verifyType: '验证方式',
                sliderVerify: '滑块验证',
                clickVerify: '点选验证',
                rotationVerify: '旋转验证',
                gestureVerify: '手势验证',
                passiveVerify: '无感验证',
                sliderImageAlt: '滑块验证码图片',
                puzzlePiece: '拼图块',
                refresh: '刷新验证码',
                sliderAriaLabel: '拖动滑块完成验证，进度百分比',
                sliderButtonAria: '拖动滑块',
                dragToVerify: '向右滑动完成验证',
                sliderHint: '按住滑块拖动到最右侧',
                clickHint: '请依次点击图中的文字',
                clickGridLabel: '点选验证码图片',
                clickImageAlt: '点选验证码图片，请按顺序点击指定位置',
                selectedCount: '已选择',
                clearSelection: '清除已选点',
                clear: '清除',
                confirm: '确认',
                submitVerification: '提交验证',
                loading: '加载中...',
                generating: '生成中...',
                almostDone: '即将完成...',
                refreshing: '正在刷新验证码',
                loadedSuccess: '验证码加载成功',
                loadFailed: '加载失败，请重试',
                sliding: '滑动中...',
                sliderDragStarted: '开始拖动滑块',
                sliderProgress: '进度',
                sliderCancelled: '滑动已取消',
                maxPointsReached: '已达到最大选择数量',
                pointSelected: '已选择第',
                of: '个，共',
                pointRemoved: '已移除该点',
                pointsRemaining: '个点剩余',
                selectionCleared: '已清除所有选择',
                noPointsSelected: '请先选择点',
                switchedTo: '已切换到',
                verifySuccess: '验证成功!',
                verifyFailed: '验证失败，请重试',
                verifying: '验证中...',
                secureConnection: '安全连接',
                securityBadge: '安全验证保护',
                demoClickHint: '请依次点击: 1, 2, 3',
                rotationImageAlt: '旋转验证码图片',
                rotationSliderAria: '拖动滑块旋转图片',
                dragToRotate: '拖动旋转图片',
                rotating: '旋转中...',
                rotationDragStarted: '开始拖动旋转',
                rotationAngle: '旋转角度',
                passiveChecking: '正在检测环境...',
                passiveDetail: '系统正在后台进行安全检测，无需任何操作',
                passiveSuccess: '环境安全，验证通过',
                riskScore: '风险评分',
                passiveCheckEnv: '环境安全检测',
                passiveCheckBehavior: '行为特征分析',
                passiveCheckRisk: '风险综合评估',
                passiveCheckEnvDone: '环境安全检测完成',
                passiveCheckBehaviorDone: '行为特征分析完成',
                passiveCheckRiskDone: '风险综合评估完成'
            },
            'en-US': {
                captchaLabel: 'Security captcha component',
                securityVerify: 'Security Verification',
                completeVerify: 'Please complete the verification to continue',
                verifyType: 'Verification type',
                sliderVerify: 'Slider Verification',
                clickVerify: 'Click Verification',
                rotationVerify: 'Rotation Verification',
                gestureVerify: 'Gesture Verification',
                passiveVerify: 'Passive Verification',
                sliderImageAlt: 'Slider captcha image',
                puzzlePiece: 'Puzzle piece',
                refresh: 'Refresh captcha',
                sliderAriaLabel: 'Drag slider to verify, progress percentage',
                sliderButtonAria: 'Drag slider',
                dragToVerify: 'Slide right to verify',
                sliderHint: 'Hold and drag slider to the right',
                clickHint: 'Click the specified areas in order',
                clickGridLabel: 'Click captcha image',
                clickImageAlt: 'Click captcha image, click specified positions in order',
                selectedCount: 'Selected',
                clearSelection: 'Clear selection',
                clear: 'Clear',
                confirm: 'Confirm',
                submitVerification: 'Submit verification',
                loading: 'Loading...',
                generating: 'Generating...',
                almostDone: 'Almost done...',
                refreshing: 'Refreshing captcha',
                loadedSuccess: 'Captcha loaded successfully',
                loadFailed: 'Load failed, please retry',
                sliding: 'Sliding...',
                sliderDragStarted: 'Started dragging slider',
                sliderProgress: 'Progress',
                sliderCancelled: 'Slide cancelled',
                maxPointsReached: 'Maximum selection reached',
                pointSelected: 'Point',
                of: 'of',
                pointRemoved: 'Point removed',
                pointsRemaining: 'points remaining',
                selectionCleared: 'Selection cleared',
                noPointsSelected: 'Please select points first',
                switchedTo: 'Switched to',
                verifySuccess: 'Verification successful!',
                verifyFailed: 'Verification failed, please retry',
                verifying: 'Verifying...',
                secureConnection: 'Secure connection',
                securityBadge: 'Security protection',
                demoClickHint: 'Click: 1, 2, 3 in order',
                rotationImageAlt: 'Rotation captcha image',
                rotationSliderAria: 'Drag slider to rotate image',
                dragToRotate: 'Drag to rotate',
                rotating: 'Rotating...',
                rotationDragStarted: 'Started rotating',
                rotationAngle: 'Rotation angle',
                passiveChecking: 'Checking environment...',
                passiveDetail: 'System is performing security checks in the background',
                passiveSuccess: 'Environment secure, verification passed',
                riskScore: 'Risk Score',
                passiveCheckEnv: 'Environment security check',
                passiveCheckBehavior: 'Behavior analysis',
                passiveCheckRisk: 'Risk assessment',
                passiveCheckEnvDone: 'Environment check complete',
                passiveCheckBehaviorDone: 'Behavior analysis complete',
                passiveCheckRiskDone: 'Risk assessment complete'
            },
            'ja-JP': {
                captchaLabel: 'セキュリティキャプチャコンポーネント',
                securityVerify: 'セキュリティ確認',
                completeVerify: '続行するには確認を完了してください',
                verifyType: '確認方法',
                sliderVerify: 'スライダー確認',
                clickVerify: 'クリック確認',
                rotationVerify: '回転確認',
                gestureVerify: 'ジェスチャー確認',
                passiveVerify: 'パッシブ確認',
                sliderImageAlt: 'スライダーキャプチャ画像',
                puzzlePiece: 'パズルピース',
                refresh: 'キャプチャを更新',
                sliderAriaLabel: 'スライダーをドラッグして確認、進捗率',
                sliderButtonAria: 'スライダーをドラッグ',
                dragToVerify: '右にスライダーを移動',
                sliderHint: 'スライダーを押して右にドラッグ',
                clickHint: '指定された領域を順番にクリック',
                clickGridLabel: 'クリックキャプチャ画像',
                clickImageAlt: 'クリックキャプチャ画像、順番にクリック',
                selectedCount: '選択済み',
                clearSelection: '選択をクリア',
                clear: 'クリア',
                confirm: '確認',
                submitVerification: '確認を送信',
                loading: '読み込み中...',
                generating: '生成中...',
                almostDone: 'もう少し...',
                refreshing: 'キャプチャを更新中',
                loadedSuccess: 'キャプチャの読み込みに成功',
                loadFailed: '読み込み失敗、再試行してください',
                sliding: 'スライド中...',
                sliderDragStarted: 'スライディングを開始',
                sliderProgress: '進捗',
                sliderCancelled: 'スライドがキャンセルされました',
                maxPointsReached: '最大選択数に達しました',
                pointSelected: 'ポイント',
                of: '/',
                pointRemoved: 'ポイントが削除されました',
                pointsRemaining: 'ポイント残り',
                selectionCleared: '選択がクリアされました',
                noPointsSelected: '最初にポイントを選択してください',
                switchedTo: '切り替え先',
                verifySuccess: '確認成功!',
                verifyFailed: '確認失敗、再試行してください',
                verifying: '確認中...',
                secureConnection: '安全な接続',
                securityBadge: 'セキュリティ保護',
                demoClickHint: '順番にクリック: 1, 2, 3',
                rotationImageAlt: '回転キャプチャ画像',
                rotationSliderAria: 'スライダーをドラッグして画像を回転',
                dragToRotate: 'ドラッグして回転',
                rotating: '回転中...',
                rotationDragStarted: '回転を開始しました',
                rotationAngle: '回転角度',
                passiveChecking: '環境を確認中...',
                passiveDetail: 'システムがバックグラウンドでセキュリティチェックを実行中',
                passiveSuccess: '環境安全、確認完了',
                riskScore: 'リスクスコア',
                passiveCheckEnv: '環境セキュリティチェック',
                passiveCheckBehavior: '行動分析',
                passiveCheckRisk: 'リスク評価',
                passiveCheckEnvDone: '環境チェック完了',
                passiveCheckBehaviorDone: '行動分析完了',
                passiveCheckRiskDone: 'リスク評価完了'
            }
        };
    }

    t(key) {
        const translations = this.translations[this.locale] || this.translations['zh-CN'];
        return translations[key] || key;
    }

    getAvailableLocales() {
        return Object.keys(this.translations);
    }
}

class CaptchaLanguageManager {
    constructor() {
        this.currentLocale = this.detectBrowserLanguage();
        this.listeners = [];
    }

    detectBrowserLanguage() {
        const browserLang = navigator.language || navigator.userLanguage;
        if (browserLang.startsWith('zh')) return 'zh-CN';
        if (browserLang.startsWith('ja')) return 'ja-JP';
        return 'en-US';
    }

    setLocale(locale) {
        this.currentLocale = locale;
        document.documentElement.lang = locale;
        this.notifyListeners();
    }

    getLocale() {
        return this.currentLocale;
    }

    addChangeListener(callback) {
        this.listeners.push(callback);
    }

    notifyListeners() {
        this.listeners.forEach(callback => callback(this.currentLocale));
    }
}

class ClickCaptcha {
    constructor(containerId, options = {}) {
        this.container = typeof containerId === 'string' 
            ? document.getElementById(containerId) 
            : containerId;
        
        if (!this.container) {
            console.error('ClickCaptcha container not found');
            return;
        }

        this.options = {
            apiBase: '/api/v1',
            difficulty: 'medium',
            mode: 'chinese',
            maxTargets: 4,
            tolerance: 25,
            minClickInterval: 100,
            maxClickInterval: 3000,
            enableSound: false,
            onSuccess: null,
            onError: null,
            onRefresh: null,
            onProgress: null,
            ...options
        };

        this.sessionId = null;
        this.targets = [];
        this.clicks = [];
        this.correctOrder = [];
        this.displayOrder = [];
        this.isVerified = false;
        this.startTime = 0;
        this.clickSequence = [];
        
        this.init();
    }

    init() {
        this.setupUI();
        this.bindEvents();
        this.loadChallenge();
    }

    setupUI() {
        this.container.innerHTML = `
            <div class="click-captcha-container" role="application" aria-label="Shuffled Click Verification">
                <div class="click-captcha-header">
                    <h4>${this.options.mode === 'chinese' ? '汉字点选验证' : '点选验证'}</h4>
                    <div class="difficulty-badge" id="difficulty-badge">${this.options.difficulty}</div>
                </div>
                <div class="click-captcha-body">
                    <div class="challenge-image-wrapper">
                        <canvas id="challenge-canvas" width="400" height="300" role="img" aria-label="Click verification image"></canvas>
                        <div class="click-markers-layer" id="click-markers"></div>
                        <button class="captcha-refresh-btn" id="refresh-btn" aria-label="Refresh">
                            <i class="fas fa-sync-alt"></i>
                        </button>
                        <div class="click-captcha-loading" id="captcha-loading">
                            <div class="spinner"></div>
                            <span>Loading...</span>
                        </div>
                    </div>
                    <div class="click-hint-panel">
                        <div class="hint-text" id="hint-text">
                            <i class="fas fa-info-circle"></i>
                            <span>请按正确顺序点击字符</span>
                        </div>
                        <div class="click-progress">
                            <span class="progress-label">Progress:</span>
                            <div class="progress-bar">
                                <div class="progress-fill" id="progress-fill"></div>
                            </div>
                            <span class="progress-count" id="progress-count">0/${this.options.maxTargets}</span>
                        </div>
                    </div>
                    <div class="click-instruction">
                        <div class="instruction-text">请依次点击以下字符:</div>
                        <div class="sequence-display" id="sequence-display"></div>
                    </div>
                    <div class="click-actions">
                        <button class="btn btn-secondary" id="clear-btn">
                            <i class="fas fa-eraser"></i> Clear
                        </button>
                        <button class="btn btn-primary" id="verify-btn" disabled>
                            <i class="fas fa-check"></i> Verify
                        </button>
                    </div>
                </div>
                <div class="click-captcha-result" id="result-panel" hidden></div>
            </div>
        `;

        this.canvas = this.container.querySelector('#challenge-canvas');
        this.ctx = this.canvas.getContext('2d');
        this.markersLayer = this.container.querySelector('#click-markers');
        this.hintText = this.container.querySelector('#hint-text span');
        this.progressFill = this.container.querySelector('#progress-fill');
        this.progressCount = this.container.querySelector('#progress-count');
        this.sequenceDisplay = this.container.querySelector('#sequence-display');
        this.verifyBtn = this.container.querySelector('#verify-btn');
        this.clearBtn = this.container.querySelector('#clear-btn');
        this.refreshBtn = this.container.querySelector('#refresh-btn');
        this.loadingOverlay = this.container.querySelector('#captcha-loading');
        this.resultPanel = this.container.querySelector('#result-panel');
    }

    bindEvents() {
        this.canvas.addEventListener('click', (e) => this.handleCanvasClick(e));
        this.canvas.addEventListener('mousemove', (e) => this.handleMouseMove(e));
        
        this.clearBtn.addEventListener('click', () => this.clearClicks());
        this.verifyBtn.addEventListener('click', () => this.verifyClicks());
        this.refreshBtn.addEventListener('click', () => this.loadChallenge());
    }

    handleCanvasClick(e) {
        if (this.isVerified) return;
        if (this.clicks.length >= this.targets.length) return;

        const rect = this.canvas.getBoundingClientRect();
        const x = Math.round(e.clientX - rect.left);
        const y = Math.round(e.clientY - rect.top);
        const timestamp = Date.now();

        const clickData = {
            x: x,
            y: y,
            timestamp: timestamp,
            targetId: -1
        };

        this.clicks.push(clickData);
        this.clickSequence.push(this.clicks.length - 1);

        this.addClickMarker(x, y, this.clicks.length);
        this.updateProgress();
        this.addBehaviorPoint(x, y, timestamp, 'click');

        if (this.clicks.length === this.targets.length) {
            this.verifyBtn.disabled = false;
        }

        if (this.options.onProgress) {
            this.options.onProgress({
                current: this.clicks.length,
                total: this.targets.length,
                clicks: this.clicks
            });
        }
    }

    handleMouseMove(e) {
        if (this.clicks.length === 0) return;
        
        const rect = this.canvas.getBoundingClientRect();
        const x = Math.round(e.clientX - rect.left);
        const y = Math.round(e.clientY - rect.top);
        const timestamp = Date.now();
        
        if (Math.random() < 0.2) {
            this.addBehaviorPoint(x, y, timestamp, 'move');
        }
    }

    addClickMarker(x, y, index) {
        const marker = document.createElement('div');
        marker.className = 'click-marker';
        marker.style.left = `${x}px`;
        marker.style.top = `${y}px`;
        marker.textContent = index;
        marker.dataset.index = index - 1;

        marker.addEventListener('click', (e) => {
            e.stopPropagation();
            this.removeClick(index - 1);
        });

        this.markersLayer.appendChild(marker);
        this.playMarkerAnimation(marker);
    }

    playMarkerAnimation(marker) {
        marker.style.animation = 'marker-pop 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275)';
    }

    removeClick(index) {
        if (index >= 0 && index < this.clicks.length) {
            this.clicks.splice(index, 1);
            this.clickSequence.splice(index, 1);
            this.updateMarkers();
            this.updateProgress();
            this.verifyBtn.disabled = true;
        }
    }

    updateMarkers() {
        this.markersLayer.innerHTML = '';
        this.clicks.forEach((click, idx) => {
            this.addClickMarker(click.x, click.y, idx + 1);
        });
    }

    updateProgress() {
        const progress = (this.clicks.length / this.targets.length) * 100;
        this.progressFill.style.width = `${progress}%`;
        this.progressCount.textContent = `${this.clicks.length}/${this.targets.length}`;
    }

    async loadChallenge() {
        this.showLoading(true);
        this.clearClicks();
        this.isVerified = false;
        this.resultPanel.hidden = true;

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/shuffle/click`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });

            if (response.ok) {
                const data = await response.json();
                this.handleChallengeResponse(data);
            } else {
                this.showError('Failed to load challenge');
            }
        } catch (error) {
            console.error('Load challenge error:', error);
            this.showError('Network error');
        } finally {
            this.showLoading(false);
        }
    }

    handleChallengeResponse(data) {
        this.sessionId = data.session_id;
        this.targets = data.targets || [];
        this.correctOrder = data.correct_order || [];
        this.displayOrder = data.display_order || [];
        this.options.maxTargets = data.max_targets || 4;
        this.options.tolerance = data.tolerance || 25;
        this.options.minClickInterval = data.min_click_interval || 100;
        this.options.maxClickInterval = data.max_click_interval || 3000;

        this.renderShuffledChallenge(data.image_url);
        this.updateSequenceDisplay();
        this.updateProgress();
        this.verifyBtn.disabled = true;
        this.startTime = Date.now();
    }

    renderShuffledChallenge(imageUrl) {
        if (!this.canvas) return;

        const img = new Image();
        img.crossOrigin = 'anonymous';

        img.onload = () => {
            this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
            this.ctx.drawImage(img, 0, 0, this.canvas.width, this.canvas.height);
        };

        img.onerror = () => {
            this.drawDemoBackground();
        };

        img.src = imageUrl;
    }

    drawDemoBackground() {
        const ctx = this.ctx;
        const w = this.canvas.width;
        const h = this.canvas.height;

        const gradient = ctx.createLinearGradient(0, 0, w, h);
        gradient.addColorStop(0, '#667eea');
        gradient.addColorStop(1, '#764ba2');
        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, w, h);

        ctx.fillStyle = 'rgba(255, 255, 255, 0.9)';
        ctx.font = '24px Arial';
        ctx.textAlign = 'center';
        ctx.fillText('Click Verification', w / 2, h / 2);
    }

    updateSequenceDisplay() {
        if (!this.sequenceDisplay) return;

        const chars = this.correctOrder.map(idx => {
            if (idx >= 0 && idx < this.targets.length) {
                return this.targets[idx].char || this.targets[idx].Char || '?';
            }
            return '?';
        });

        this.sequenceDisplay.innerHTML = chars
            .map((char, idx) => `<span class="seq-item" data-index="${idx}">${char}</span>`)
            .join('<span class="seq-arrow">→</span>');

        this.sequenceDisplay.querySelectorAll('.seq-item').forEach(item => {
            item.addEventListener('click', () => {
                const idx = parseInt(item.dataset.index);
                if (idx < this.clicks.length) {
                    this.removeClick(idx);
                }
            });
        });
    }

    clearClicks() {
        this.clicks = [];
        this.clickSequence = [];
        this.markersLayer.innerHTML = '';
        this.updateProgress();
        this.verifyBtn.disabled = true;
        this.startTime = Date.now();
    }

    async verifyClicks() {
        if (this.clicks.length === 0) {
            this.showError('No clicks recorded');
            return;
        }

        this.showLoading(true);
        this.verifyBtn.disabled = true;

        const payload = {
            session_id: this.sessionId,
            clicks: this.clicks.map((click, idx) => ({
                x: click.x,
                y: click.y,
                timestamp: click.timestamp,
                target_id: click.targetId
            }))
        };

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/shuffle/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (response.ok) {
                const data = await response.json();
                this.handleVerificationResult(data);
            } else {
                this.showError('Verification failed');
            }
        } catch (error) {
            console.error('Verify error:', error);
            this.showError('Network error');
        } finally {
            this.showLoading(false);
        }
    }

    handleVerificationResult(data) {
        this.isVerified = true;

        if (data.success) {
            this.showSuccess(data.message || 'Verification successful!');
            this.markSuccessMarkers();
            
            if (this.options.onSuccess) {
                this.options.onSuccess({
                    session_id: this.sessionId,
                    risk_score: data.risk_score
                });
            }
        } else {
            this.showError(data.fail_reason || 'Verification failed');
            this.markErrorMarkers();
            
            if (this.options.onError) {
                this.options.onError({
                    error: data.fail_reason
                });
            }

            setTimeout(() => {
                this.loadChallenge();
            }, 2000);
        }
    }

    markSuccessMarkers() {
        const markers = this.markersLayer.querySelectorAll('.click-marker');
        markers.forEach((marker, idx) => {
            setTimeout(() => {
                marker.classList.add('success');
            }, idx * 150);
        });
    }

    markErrorMarkers() {
        const markers = this.markersLayer.querySelectorAll('.click-marker');
        markers.forEach(marker => {
            marker.classList.add('error');
        });
    }

    showLoading(show) {
        if (this.loadingOverlay) {
            this.loadingOverlay.style.display = show ? 'flex' : 'none';
        }
    }

    showSuccess(message) {
        this.resultPanel.textContent = message;
        this.resultPanel.className = 'click-captcha-result success show';
        this.resultPanel.hidden = false;
    }

    showError(message) {
        this.resultPanel.textContent = message;
        this.resultPanel.className = 'click-captcha-result error show';
        this.resultPanel.hidden = false;
    }

    addBehaviorPoint(x, y, timestamp, event) {
    }

    getClickData() {
        return {
            session_id: this.sessionId,
            clicks: this.clicks,
            click_sequence: this.clickSequence,
            timestamp: Date.now()
        };
    }

    setDifficulty(difficulty) {
        this.options.difficulty = difficulty;
        const badge = this.container.querySelector('#difficulty-badge');
        if (badge) {
            badge.textContent = difficulty;
        }
    }

    setMode(mode) {
        this.options.mode = mode;
    }

    reset() {
        this.clearClicks();
        this.isVerified = false;
        this.resultPanel.hidden = true;
        this.loadChallenge();
    }

    destroy() {
        this.container.innerHTML = '';
        this.targets = [];
        this.clicks = [];
        this.correctOrder = [];
        this.displayOrder = [];
    }
}

document.addEventListener('DOMContentLoaded', function() {
    window.Captcha = Captcha;
    window.CaptchaI18n = CaptchaI18n;
    window.CaptchaLanguageManager = CaptchaLanguageManager;
    window.TrajectoryEncryptor = TrajectoryEncryptor;
    window.ClickCaptcha = ClickCaptcha;
});
