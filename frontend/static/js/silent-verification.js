(function() {
    'use strict';

    const SilentVerification = {
        config: {
            apiBase: '/api/v1',
            fingerprint: null,
            sessionId: null,
            trustScore: 50,
            riskScore: 0,
            currentLevel: 0,
            verificationData: {},
            cacheTimeout: 24 * 60 * 60 * 1000,
            checkInterval: 30000,
            enableProgressive: true,
            levels: {
                silent: 0,
                light: 1,
                moderate: 2,
                strict: 3
            }
        },

        storage: {
            prefix: '_sv_',

            get(key) {
                try {
                    const item = localStorage.getItem(this.prefix + key);
                    if (!item) return null;
                    const data = JSON.parse(item);
                    if (data.expires && Date.now() > data.expires) {
                        localStorage.removeItem(this.prefix + key);
                        return null;
                    }
                    return data.value;
                } catch (e) {
                    return null;
                }
            },

            set(key, value, ttl = null) {
                try {
                    const data = {
                        value: value,
                        expires: ttl ? Date.now() + ttl : null
                    };
                    localStorage.setItem(this.prefix + key, JSON.stringify(data));
                } catch (e) {
                    console.warn('Storage write failed:', e);
                }
            },

            remove(key) {
                localStorage.removeItem(this.prefix + key);
            },

            clear() {
                const keys = Object.keys(localStorage);
                keys.forEach(key => {
                    if (key.startsWith(this.prefix)) {
                        localStorage.removeItem(key);
                    }
                });
            }
        },

        fingerprint: {
            components: {},
            hash: null,

            async collect() {
                this.components = {
                    userAgent: navigator.userAgent,
                    platform: navigator.platform,
                    language: navigator.language,
                    languages: navigator.languages ? navigator.languages.join(',') : '',
                    colorDepth: screen.colorDepth,
                    pixelDepth: screen.pixelDepth,
                    deviceMemory: navigator.deviceMemory || 'unknown',
                    hardwareConcurrency: navigator.hardwareConcurrency || 'unknown',
                    screenWidth: screen.width,
                    screenHeight: screen.height,
                    screenAvailWidth: screen.availWidth,
                    screenAvailHeight: screen.availHeight,
                    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                    timezoneOffset: new Date().getTimezoneOffset(),
                    touchSupport: 'ontouchstart' in window || navigator.maxTouchPoints > 0,
                    maxTouchPoints: navigator.maxTouchPoints || 0,
                    doNotTrack: navigator.doNotTrack || window.doNotTrack,
                    cookiesEnabled: navigator.cookieEnabled,
                    indexedDB: 'indexedDB' in window,
                    localStorage: 'localStorage' in window,
                    sessionStorage: 'sessionStorage' in window,
                    webdriver: navigator.webdriver || false,
                    pdfViewerEnabled: navigator.pdfViewerEnabled
                };

                await this.collectCanvas();
                await this.collectWebGL();
                await this.collectAudio();
                await this.collectFonts();

                this.hash = await this.calculateHash();
                SilentVerification.config.fingerprint = this.hash;
                return this.hash;
            },

            async collectCanvas() {
                try {
                    const canvas = document.createElement('canvas');
                    canvas.width = 280;
                    canvas.height = 60;
                    const ctx = canvas.getContext('2d');

                    ctx.textBaseline = 'alphabetic';
                    ctx.fillStyle = '#f60';
                    ctx.fillRect(125, 1, 62, 20);
                    ctx.fillStyle = '#069';
                    ctx.font = '11pt Arial';
                    ctx.fillText('Cwm fjordbank glyphs vext quiz', 2, 15);
                    ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
                    ctx.font = '18pt Arial';
                    ctx.fillText('Cwm fjordbank glyphs vext quiz', 4, 45);

                    const dataUrl = canvas.toDataURL();
                    this.components.canvasHash = await this.hashString(dataUrl);
                } catch (e) {
                    this.components.canvasHash = 'unavailable';
                }
            },

            async collectWebGL() {
                try {
                    const canvas = document.createElement('canvas');
                    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');

                    if (gl) {
                        const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                        if (debugInfo) {
                            this.components.webglVendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL);
                            this.components.webglRenderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                        }
                        this.components.webglVersion = gl.getParameter(gl.VERSION);
                        this.components.webglShadingLanguage = gl.getParameter(gl.SHADING_LANGUAGE_VERSION);
                    } else {
                        this.components.webglVendor = 'not_supported';
                        this.components.webglRenderer = 'not_supported';
                    }
                } catch (e) {
                    this.components.webglVendor = 'error';
                    this.components.webglRenderer = 'error';
                }
            },

            async collectAudio() {
                try {
                    const AudioContext = window.OfflineAudioContext || window.webkitOfflineAudioContext;
                    if (AudioContext) {
                        const context = new AudioContext(1, 44100, 44100);
                        const oscillator = context.createOscillator();
                        const compressor = context.createDynamicsCompressor();

                        oscillator.type = 'triangle';
                        oscillator.frequency.setValueAtTime(10000, context.currentTime);

                        compressor.threshold.setValueAtTime(-50, context.currentTime);
                        compressor.knee.setValueAtTime(40, context.currentTime);
                        compressor.ratio.setValueAtTime(12, context.currentTime);
                        compressor.attack.setValueAtTime(0, context.currentTime);
                        compressor.release.setValueAtTime(0.25, context.currentTime);

                        oscillator.connect(compressor);
                        compressor.connect(context.destination);
                        oscillator.start(0);

                        this.components.audioFingerprint = 'captured';
                    } else {
                        this.components.audioFingerprint = 'unavailable';
                    }
                } catch (e) {
                    this.components.audioFingerprint = 'error';
                }
            },

            async collectFonts() {
                const baseFonts = ['monospace', 'sans-serif', 'serif'];
                const testFonts = [
                    'Arial', 'Helvetica', 'Times New Roman', 'Courier New', 'Verdana',
                    'Georgia', 'Comic Sans MS', 'Impact', 'Lucida Console', 'Tahoma'
                ];

                const span = document.createElement('span');
                span.style.cssText = 'position:absolute;left:-9999px;top:-9999px;visibility:hidden;white-space:nowrap;font-size:72px';
                document.body.appendChild(span);

                const baseWidths = {};
                baseFonts.forEach(font => {
                    span.style.fontFamily = font;
                    baseWidths[font] = span.offsetWidth;
                });

                const detectedFonts = [];
                testFonts.forEach(font => {
                    for (const base of baseFonts) {
                        span.style.fontFamily = `'${font}', ${base}`;
                        if (span.offsetWidth !== baseWidths[base]) {
                            detectedFonts.push(font);
                            break;
                        }
                    }
                });

                document.body.removeChild(span);
                this.components.fonts = detectedFonts.join(',');
                this.components.fontCount = detectedFonts.length;
            },

            async calculateHash() {
                const data = JSON.stringify(this.components);
                const encoder = new TextEncoder();
                const dataBuffer = encoder.encode(data);
                const hashBuffer = await crypto.subtle.digest('SHA-256', dataBuffer);
                const hashArray = Array.from(new Uint8Array(hashBuffer));
                return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
            },

            async hashString(str) {
                const encoder = new TextEncoder();
                const data = encoder.encode(str);
                const hashBuffer = await crypto.subtle.digest('SHA-256', data);
                const hashArray = Array.from(new Uint8Array(hashBuffer));
                return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
            },

            async getHash() {
                if (!this.hash) {
                    await this.collect();
                }
                return this.hash;
            }
        },

        async startVerification(options = {}) {
            Object.assign(this.config, options);

            const cached = this.storage.get('verification_cache');
            if (cached && cached.fingerprint === this.config.fingerprint) {
                return cached;
            }

            await this.fingerprint.collect();

            try {
                const response = await fetch(`${this.config.apiBase}/progressive/start`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        fingerprint: this.config.fingerprint,
                        data: this.fingerprint.components,
                        required_level: options.requiredLevel || 0
                    })
                });

                const result = await response.json();

                if (result.success) {
                    this.config.sessionId = result.session_id;
                    this.config.trustScore = result.trust_score;
                    this.config.riskScore = result.risk_score;
                    this.config.currentLevel = result.level;

                    const cacheData = {
                        fingerprint: this.config.fingerprint,
                        session_id: result.session_id,
                        trust_score: result.trust_score,
                        risk_score: result.risk_score,
                        level: result.level,
                        skip_verification: result.skip_verification,
                        timestamp: Date.now()
                    };
                    this.storage.set('verification_cache', cacheData, this.config.cacheTimeout);

                    if (result.skip_verification) {
                        return { passed: true, skipped: true, ...cacheData };
                    }

                    return cacheData;
                }

                return { passed: false, error: result.message };
            } catch (error) {
                console.error('Verification start failed:', error);
                return { passed: false, error: error.message };
            }
        },

        async completeChallenge(challengeType, result) {
            if (!this.config.sessionId) {
                return { success: false, error: 'No active session' };
            }

            try {
                const response = await fetch(`${this.config.apiBase}/progressive/challenge`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        session_id: this.config.sessionId,
                        challenge_token: result.token,
                        result: result.data
                    })
                });

                const data = await response.json();

                if (data.success) {
                    this.config.trustScore = data.trust_score;
                    this.config.riskScore = data.risk_score;

                    if (data.passed) {
                        this.storage.set('verification_cache', {
                            fingerprint: this.config.fingerprint,
                            trust_score: data.trust_score,
                            risk_score: data.risk_score,
                            passed: true,
                            timestamp: Date.now()
                        }, this.config.cacheTimeout);
                    }
                }

                return data;
            } catch (error) {
                return { success: false, error: error.message };
            }
        },

        async getSessionStatus() {
            if (!this.config.sessionId) {
                return null;
            }

            try {
                const response = await fetch(
                    `${this.config.apiBase}/progressive/status/${this.config.sessionId}`
                );
                return await response.json();
            } catch (error) {
                console.error('Get session status failed:', error);
                return null;
            }
        },

        async evaluateTrust() {
            await this.fingerprint.collect();

            try {
                const response = await fetch(`${this.config.apiBase}/trust/evaluate`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        fingerprint: this.config.fingerprint,
                        data: this.fingerprint.components
                    })
                });

                const result = await response.json();
                if (result.success) {
                    return result.data;
                }
                return null;
            } catch (error) {
                console.error('Evaluate trust failed:', error);
                return null;
            }
        },

        async verifyDevice(duration = 168) {
            try {
                const response = await fetch(`${this.config.apiBase}/trust/verify`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        fingerprint: this.config.fingerprint,
                        duration: duration
                    })
                });

                const result = await response.json();
                if (result.success) {
                    this.storage.remove('verification_cache');
                }
                return result;
            } catch (error) {
                return { success: false, error: error.message };
            }
        },

        async recordEvent(eventType, data = {}) {
            try {
                return await fetch(`${this.config.apiBase}/trust/event`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        fingerprint: this.config.fingerprint,
                        event: eventType,
                        ...data
                    })
                });
            } catch (error) {
                console.error('Record event failed:', error);
            }
        },

        getTrustScore() {
            return this.config.trustScore;
        },

        getRiskScore() {
            return this.config.riskScore;
        },

        getVerificationLevel() {
            return this.config.currentLevel;
        },

        shouldShowVerification() {
            return this.config.currentLevel > 0;
        },

        clearCache() {
            this.storage.clear();
            this.config.sessionId = null;
        }
    };

    const ProgressiveUI = {
        container: null,
        overlay: null,
        currentChallenge: null,
        sliderTrack: null,
        sliderHandle: null,
        isDragging: false,
        startX: 0,
        targetX: 0,
        currentX: 0,
        threshold: 0.8,

        createOverlay() {
            this.overlay = document.createElement('div');
            this.overlay.id = 'progressive-verification-overlay';
            Object.assign(this.overlay.style, {
                position: 'fixed',
                top: '0',
                left: '0',
                width: '100%',
                height: '100%',
                background: 'rgba(0, 0, 0, 0.7)',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                zIndex: '999999',
                opacity: '0',
                transition: 'opacity 0.3s ease'
            });
            document.body.appendChild(this.overlay);
            setTimeout(() => {
                this.overlay.style.opacity = '1';
            }, 10);
        },

        createContainer() {
            this.container = document.createElement('div');
            this.container.id = 'progressive-verification';
            Object.assign(this.container.style, {
                background: '#ffffff',
                borderRadius: '12px',
                boxShadow: '0 10px 40px rgba(0, 0, 0, 0.2)',
                padding: '24px',
                width: '360px',
                maxWidth: '90vw',
                fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
            });
            this.overlay.appendChild(this.container);
        },

        showChallenge(challenge, onComplete) {
            this.createOverlay();
            this.createContainer();
            this.currentChallenge = challenge;
            this.onComplete = onComplete;

            this.renderChallenge(challenge);
            this.animateIn();
        },

        renderChallenge(challenge) {
            this.container.innerHTML = '';

            const header = document.createElement('div');
            header.style.cssText = 'text-align: center; margin-bottom: 20px;';
            header.innerHTML = `
                <h3 style="margin: 0 0 8px 0; color: #333; font-size: 18px;">安全验证</h3>
                <p style="margin: 0; color: #666; font-size: 14px;">请完成验证以继续</p>
            `;
            this.container.appendChild(header);

            const content = document.createElement('div');
            content.style.cssText = 'margin-bottom: 20px;';

            switch (challenge.type) {
                case 'slider_captcha':
                    this.renderSliderChallenge(content, challenge);
                    break;
                case 'image_select':
                    this.renderImageSelectChallenge(content, challenge);
                    break;
                case 'simple_interaction':
                    this.renderSimpleInteraction(content, challenge);
                    break;
                default:
                    this.renderDefaultChallenge(content, challenge);
            }
            this.container.appendChild(content);

            const footer = document.createElement('div');
            footer.style.cssText = 'text-align: center; font-size: 12px; color: #999;';
            footer.textContent = '静默验证系统 - 保护您的账户安全';
            this.container.appendChild(footer);
        },

        renderSliderChallenge(container, challenge) {
            const sliderContainer = document.createElement('div');
            sliderContainer.style.cssText = 'position: relative; height: 40px; margin-bottom: 10px;';

            const track = document.createElement('div');
            track.style.cssText = `
                width: 100%;
                height: 40px;
                background: linear-gradient(to right, #e8e8e8, #f5f5f5);
                border-radius: 20px;
                border: 2px solid #ddd;
                position: relative;
                overflow: hidden;
            `;
            this.sliderTrack = track;

            const targetIndicator = document.createElement('div');
            targetIndicator.style.cssText = `
                position: absolute;
                right: 30px;
                top: 50%;
                transform: translateY(-50%);
                width: 40px;
                height: 40px;
                background: #fff;
                border: 2px solid #4CAF50;
                border-radius: 50%;
                display: flex;
                align-items: center;
                justify-content: center;
            `;
            targetIndicator.innerHTML = '→';
            targetIndicator.style.color = '#4CAF50';
            targetIndicator.style.fontSize = '20px';
            track.appendChild(targetIndicator);
            sliderContainer.appendChild(track);

            const handle = document.createElement('div');
            handle.id = 'slider-handle';
            handle.style.cssText = `
                position: absolute;
                left: 0;
                top: 0;
                width: 44px;
                height: 44px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                border-radius: 50%;
                cursor: grab;
                display: flex;
                align-items: center;
                justify-content: center;
                box-shadow: 0 2px 10px rgba(102, 126, 234, 0.4);
                user-select: none;
                z-index: 10;
            `;
            handle.innerHTML = '⋮⋮';
            handle.style.color = '#fff';
            handle.style.fontSize = '14px';
            handle.style.letterSpacing = '2px';
            this.sliderHandle = handle;
            sliderContainer.appendChild(handle);

            const instruction = document.createElement('p');
            instruction.style.cssText = 'text-align: center; color: #666; font-size: 13px; margin: 0 0 15px 0;';
            instruction.textContent = '拖动滑块完成拼图';

            container.appendChild(sliderContainer);
            container.appendChild(instruction);

            this.targetX = track.offsetWidth - handle.offsetWidth - 30;
            this.setupSliderEvents();
        },

        setupSliderEvents() {
            const handle = this.sliderHandle;

            const onStart = (e) => {
                this.isDragging = true;
                handle.style.cursor = 'grabbing';
                e.preventDefault();
            };

            const onMove = (e) => {
                if (!this.isDragging) return;

                let clientX;
                if (e.type === 'touchmove') {
                    clientX = e.touches[0].clientX;
                } else {
                    clientX = e.clientX;
                }

                const trackRect = this.sliderTrack.getBoundingClientRect();
                let newX = clientX - trackRect.left - handle.offsetWidth / 2;
                newX = Math.max(0, Math.min(newX, trackRect.width - handle.offsetWidth));

                this.currentX = newX;
                handle.style.left = newX + 'px';

                const progress = newX / this.targetX;
                this.updateSliderProgress(progress);
            };

            const onEnd = () => {
                if (!this.isDragging) return;
                this.isDragging = false;
                handle.style.cursor = 'grab';

                const progress = this.currentX / this.targetX;
                if (progress >= this.threshold) {
                    this.handleSliderComplete(true);
                } else {
                    this.handleSliderComplete(false);
                }
            };

            handle.addEventListener('mousedown', onStart);
            handle.addEventListener('touchstart', onStart, { passive: false });

            document.addEventListener('mousemove', onMove);
            document.addEventListener('touchmove', onMove, { passive: false });

            document.addEventListener('mouseup', onEnd);
            document.addEventListener('touchend', onEnd);
        },

        updateSliderProgress(progress) {
            const track = this.sliderTrack;
            track.style.background = `linear-gradient(to right,
                #667eea 0%,
                #667eea ${progress * 100}%,
                #e8e8e8 ${progress * 100}%,
                #e8e8e8 100%)`;
        },

        handleSliderComplete(success) {
            const handle = this.sliderHandle;

            if (success) {
                handle.style.background = 'linear-gradient(135deg, #4CAF50 0%, #45a049 100%)';
                handle.innerHTML = '✓';
                handle.style.color = '#fff';

                setTimeout(() => {
                    this.completeChallenge(success);
                }, 300);
            } else {
                handle.style.transition = 'left 0.3s ease';
                handle.style.left = '0';
                this.updateSliderProgress(0);
                setTimeout(() => {
                    handle.style.transition = 'none';
                    this.renderSliderChallenge(
                        this.container.querySelector('div'),
                        this.currentChallenge
                    );
                }, 500);
            }
        },

        renderImageSelectChallenge(container, challenge) {
            const grid = document.createElement('div');
            grid.style.cssText = `
                display: grid;
                grid-template-columns: repeat(3, 1fr);
                gap: 8px;
                margin-bottom: 15px;
            `;

            for (let i = 0; i < 9; i++) {
                const item = document.createElement('div');
                item.style.cssText = `
                    aspect-ratio: 1;
                    background: linear-gradient(135deg, #f5f5f5 0%, #e8e8e8 100%);
                    border: 2px solid #ddd;
                    border-radius: 8px;
                    cursor: pointer;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    font-size: 24px;
                    color: #999;
                    transition: all 0.2s ease;
                `;
                item.textContent = '?';

                const isTarget = i === 2 || i === 5 || i === 8;
                item.dataset.index = i;
                item.dataset.target = isTarget;

                item.addEventListener('click', () => {
                    if (isTarget) {
                        item.style.background = 'linear-gradient(135deg, #4CAF50 0%, #45a049 100%)';
                        item.style.borderColor = '#4CAF50';
                        item.style.color = '#fff';
                        item.textContent = '✓';
                    } else {
                        item.style.background = 'linear-gradient(135deg, #f44336 0%, #e53935 100%)';
                        item.style.borderColor = '#f44336';
                        item.style.color = '#fff';
                        item.textContent = '✗';
                        setTimeout(() => {
                            this.renderImageSelectChallenge(container, challenge);
                        }, 500);
                        return;
                    }

                    setTimeout(() => {
                        this.completeChallenge(true);
                    }, 300);
                });

                item.addEventListener('mouseenter', () => {
                    if (item.textContent === '?') {
                        item.style.borderColor = '#667eea';
                        item.style.transform = 'scale(1.05)';
                    }
                });

                item.addEventListener('mouseleave', () => {
                    if (item.textContent === '?') {
                        item.style.borderColor = '#ddd';
                        item.style.transform = 'scale(1)';
                    }
                });

                grid.appendChild(item);
            }

            container.appendChild(grid);

            const instruction = document.createElement('p');
            instruction.style.cssText = 'text-align: center; color: #666; font-size: 13px; margin: 0;';
            instruction.textContent = '点击所有带有绿色标记的图片';
            container.appendChild(instruction);
        },

        renderSimpleInteraction(container, challenge) {
            const button = document.createElement('button');
            button.style.cssText = `
                width: 100%;
                padding: 15px 20px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                border: none;
                border-radius: 8px;
                color: #fff;
                font-size: 16px;
                cursor: pointer;
                transition: transform 0.2s ease, box-shadow 0.2s ease;
            `;
            button.textContent = '点击验证';

            button.addEventListener('click', () => {
                button.textContent = '验证中...';
                button.disabled = true;

                setTimeout(() => {
                    button.textContent = '✓ 验证成功';
                    button.style.background = 'linear-gradient(135deg, #4CAF50 0%, #45a049 100%)';
                    setTimeout(() => {
                        this.completeChallenge(true);
                    }, 500);
                }, 800);
            });

            button.addEventListener('mouseenter', () => {
                button.style.transform = 'translateY(-2px)';
                button.style.boxShadow = '0 4px 15px rgba(102, 126, 234, 0.4)';
            });

            button.addEventListener('mouseleave', () => {
                button.style.transform = 'translateY(0)';
                button.style.boxShadow = 'none';
            });

            container.appendChild(button);
        },

        renderDefaultChallenge(container, challenge) {
            const message = document.createElement('div');
            message.style.cssText = 'text-align: center; padding: 30px 0;';
            message.innerHTML = `
                <div style="font-size: 48px; margin-bottom: 15px;">🛡️</div>
                <p style="color: #666; font-size: 14px;">正在验证您的设备信息...</p>
            `;
            container.appendChild(message);

            setTimeout(() => {
                this.completeChallenge(true);
            }, 1000);
        },

        async completeChallenge(success) {
            if (this.onComplete) {
                await this.onComplete(success, this.currentChallenge);
            }
            this.hide();
        },

        animateIn() {
            this.container.style.transform = 'scale(0.9) translateY(20px)';
            this.container.style.opacity = '0';
            this.container.style.transition = 'all 0.3s ease';

            requestAnimationFrame(() => {
                this.container.style.transform = 'scale(1) translateY(0)';
                this.container.style.opacity = '1';
            });
        },

        hide() {
            if (this.overlay) {
                this.overlay.style.opacity = '0';
                setTimeout(() => {
                    if (this.overlay && this.overlay.parentNode) {
                        this.overlay.parentNode.removeChild(this.overlay);
                    }
                    this.overlay = null;
                    this.container = null;
                }, 300);
            }
        }
    };

    const WhitelistManager = {
        apiBase: '/api/v1',

        async check(fingerprint) {
            try {
                const response = await fetch(
                    `${this.apiBase}/whitelist/check?target=${fingerprint}&type=fingerprint`
                );
                const result = await response.json();
                return result.data || { is_whitelisted: false };
            } catch (error) {
                console.error('Check whitelist failed:', error);
                return { is_whitelisted: false };
            }
        },

        async add(target, type, reason) {
            try {
                const response = await fetch(`${this.apiBase}/whitelist`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ target, type, reason })
                });
                return await response.json();
            } catch (error) {
                return { success: false, error: error.message };
            }
        },

        async remove(target, type) {
            try {
                const response = await fetch(
                    `${this.apiBase}/whitelist/${target}?type=${type}`,
                    { method: 'DELETE' }
                );
                return await response.json();
            } catch (error) {
                return { success: false, error: error.message };
            }
        },

        async list(type = 'fingerprint', page = 1, pageSize = 20) {
            try {
                const response = await fetch(
                    `${this.apiBase}/whitelist?type=${type}&page=${page}&page_size=${pageSize}`
                );
                return await response.json();
            } catch (error) {
                return { success: false, error: error.message };
            }
        },

        async getStats() {
            try {
                const response = await fetch(`${this.apiBase}/whitelist/stats`);
                return await response.json();
            } catch (error) {
                return { success: false, error: error.message };
            }
        },

        async export(type = 'fingerprint') {
            try {
                const response = await fetch(`${this.apiBase}/whitelist/export?type=${type}`);
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `whitelist_${type}_${Date.now()}.json`;
                a.click();
                window.URL.revokeObjectURL(url);
                return { success: true };
            } catch (error) {
                return { success: false, error: error.message };
            }
        }
    };

    const SilentVerificationUI = {
        async init(options = {}) {
            Object.assign(SilentVerification.config, options);

            await SilentVerification.fingerprint.collect();

            const result = await SilentVerification.startVerification(options);

            if (result.skipped) {
                return { verified: true, skipped: true };
            }

            if (result.level > 0 && ProgressiveUI) {
                return new Promise((resolve) => {
                    this.showProgressiveUI(result, resolve);
                });
            }

            return result;
        },

        async showProgressiveUI(verificationData, resolve) {
            const challenges = [];

            switch (verificationData.level) {
                case 1:
                    challenges.push({ type: 'simple_interaction', token: 'token_light_1' });
                    break;
                case 2:
                    challenges.push({ type: 'slider_captcha', token: 'token_mod_1' });
                    challenges.push({ type: 'image_select', token: 'token_mod_2' });
                    break;
                case 3:
                    challenges.push({ type: 'slider_captcha', token: 'token_strict_1' });
                    challenges.push({ type: 'image_select', token: 'token_strict_2' });
                    challenges.push({ type: 'simple_interaction', token: 'token_strict_3' });
                    break;
            }

            let currentIndex = 0;

            const showNextChallenge = () => {
                if (currentIndex >= challenges.length) {
                    ProgressiveUI.hide();
                    resolve({ verified: true, completed: true });
                    return;
                }

                const challenge = challenges[currentIndex];

                ProgressiveUI.showChallenge(challenge, async (success) => {
                    if (success) {
                        await SilentVerification.completeChallenge(challenge.type, {
                            token: challenge.token,
                            data: 'completed'
                        });

                        currentIndex++;
                        setTimeout(showNextChallenge, 300);
                    }
                });
            };

            showNextChallenge();
        },

        getFingerprint() {
            return SilentVerification.config.fingerprint;
        },

        getTrustScore() {
            return SilentVerification.getTrustScore();
        },

        getRiskScore() {
            return SilentVerification.getRiskScore();
        },

        evaluate() {
            return SilentVerification.evaluateTrust();
        },

        verify(duration) {
            return SilentVerification.verifyDevice(duration);
        },

        record(event) {
            return SilentVerification.recordEvent(event);
        },

        clear() {
            SilentVerification.clearCache();
        }
    };

    window.SilentVerification = SilentVerification;
    window.ProgressiveUI = ProgressiveUI;
    window.WhitelistManager = WhitelistManager;
    window.SilentVerificationUI = SilentVerificationUI;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = {
            SilentVerification,
            ProgressiveUI,
            WhitelistManager,
            SilentVerificationUI
        };
    }
})();
