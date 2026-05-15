(function(global) {
    'use strict';

    const AudioCaptcha = {
        version: '1.0.0',
        instances: new Map(),
        defaultInstance: null,
        config: {
            serverUrl: '',
            appId: '',
            timeout: 30000,
            retryAttempts: 3,
            playingText: '播放中...',
            playText: '播放验证码',
            pauseText: '暂停',
            replayText: '重新播放',
            loadingText: '加载中...',
            errorText: '加载失败',
            successText: '验证成功',
            failText: '验证失败',
            expireText: '验证码已过期',
            placeholderText: '请输入听到的验证码'
        }
    };

    function generateUUID() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            const r = Math.random() * 16 | 0;
            const v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }

    function extend(target, source) {
        for (const key in source) {
            if (Object.prototype.hasOwnProperty.call(source, key)) {
                target[key] = source[key];
            }
        }
        return target;
    }

    function getElement(element) {
        if (typeof element === 'string') {
            return document.querySelector(element);
        }
        return element || null;
    }

    function createElement(tag, className, attributes) {
        const element = document.createElement(tag);
        if (className) {
            element.className = className;
        }
        if (attributes) {
            for (const key in attributes) {
                if (key === 'style' && typeof attributes[key] === 'object') {
                    for (const styleKey in attributes[key]) {
                        element.style[styleKey] = attributes[key][styleKey];
                    }
                } else if (key === 'dataset') {
                    for (const dataKey in attributes[key]) {
                        element.dataset[dataKey] = attributes[key][dataKey];
                    }
                } else {
                    element.setAttribute(key, attributes[key]);
                }
            }
        }
        return element;
    }

    function addClass(element, className) {
        if (element && className) {
            const classes = className.split(' ').filter(c => c);
            element.classList.add(...classes);
        }
    }

    function removeClass(element, className) {
        if (element && className) {
            const classes = className.split(' ').filter(c => c);
            element.classList.remove(...classes);
        }
    }

    function show(element) {
        if (element) {
            element.style.display = '';
            removeClass(element, 'audio-captcha-hidden');
        }
    }

    function hide(element) {
        if (element) {
            element.style.display = 'none';
            addClass(element, 'audio-captcha-hidden');
        }
    }

    function announceToScreenReader(message, politeness = 'polite') {
        const liveRegion = document.createElement('div');
        liveRegion.setAttribute('role', 'status');
        liveRegion.setAttribute('aria-live', politeness);
        liveRegion.setAttribute('aria-atomic', 'true');
        liveRegion.className = 'audio-captcha-sr-announce';
        liveRegion.textContent = message;
        document.body.appendChild(liveRegion);
        setTimeout(() => {
            liveRegion.remove();
        }, 1000);
    }

    function getAbsoluteUrl(relativePath) {
        if (!relativePath) return '';
        if (relativePath.startsWith('http://') || relativePath.startsWith('https://')) {
            return relativePath;
        }
        const base = AudioCaptcha.config.serverUrl.replace(/\/$/, '');
        const path = relativePath.replace(/^\//, '');
        return `${base}/${path}`;
    }

    function request(url, options) {
        const config = extend({
            method: 'GET',
            headers: {
                'Content-Type': 'application/json'
            },
            body: null,
            timeout: AudioCaptcha.config.timeout
        }, options);

        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();

            xhr.open(config.method, url, true);
            xhr.setRequestHeader('Content-Type', 'application/json');

            for (const header in config.headers) {
                xhr.setRequestHeader(header, config.headers[header]);
            }

            xhr.timeout = config.timeout;

            xhr.onload = function() {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const response = JSON.parse(xhr.responseText);
                        resolve(response);
                    } catch (e) {
                        resolve(xhr.responseText);
                    }
                } else {
                    reject(new Error(`HTTP ${xhr.status}: ${xhr.statusText}`));
                }
            };

            xhr.onerror = function() {
                reject(new Error('Network error'));
            };

            xhr.ontimeout = function() {
                reject(new Error('Request timeout'));
            };

            if (config.body) {
                xhr.send(typeof config.body === 'string' ? config.body : JSON.stringify(config.body));
            } else {
                xhr.send();
            }
        });
    }

    AudioCaptcha.init = function(options) {
        return new Promise((resolve, reject) => {
            const config = extend(AudioCaptcha.config, options || {});

            if (!config.serverUrl) {
                const scripts = document.getElementsByTagName('script');
                for (let i = scripts.length - 1; i >= 0; i--) {
                    const src = scripts[i].getAttribute('src');
                    if (src && src.includes('audio')) {
                        config.serverUrl = src.replace(/\/[^/]+\.js$/, '');
                        break;
                    }
                }
                if (!config.serverUrl) {
                    config.serverUrl = window.location.origin;
                }
            }

            AudioCaptcha.config = config;
            resolve(AudioCaptcha);
        });
    };

    AudioCaptcha.create = function(options) {
        const instanceId = generateUUID();
        const container = getElement(options.container);

        if (!container) {
            throw new Error('[AudioCaptcha] Container element not found');
        }

        const instance = {
            id: instanceId,
            container: container,
            config: extend({}, AudioCaptcha.config, options || {}),
            state: {
                captchaId: null,
                verified: false,
                loading: false,
                playing: false,
                destroyed: false
            },
            elements: {},
            audioElement: null
        };

        container.innerHTML = '';
        addClass(container, 'audio-captcha-container');
        container.dataset.audioCaptchaId = instanceId;

        AudioCaptcha.instances.set(instanceId, instance);
        AudioCaptcha.defaultInstance = instance;

        renderWidget(instance);
        bindEvents(instance);

        return instance;
    };

    function renderWidget(instance) {
        const { container, config } = instance;

        const widget = createElement('div', 'audio-captcha-widget');
        widget.setAttribute('role', 'region');
        widget.setAttribute('aria-label', '音频验证码');

        const header = createElement('div', 'audio-captcha-header');
        const title = createElement('span', 'audio-captcha-title');
        title.textContent = '音频验证码';

        const body = createElement('div', 'audio-captcha-body');

        const playerContainer = createElement('div', 'audio-captcha-player-container');

        const playButton = createElement('button', 'audio-captcha-play-btn', {
            type: 'button',
            'aria-label': config.playText,
            title: config.playText
        });
        playButton.innerHTML = `
            <svg class="audio-captcha-play-icon" width="24" height="24" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M8 5v14l11-7z"/>
            </svg>
            <svg class="audio-captcha-pause-icon audio-captcha-hidden" width="24" height="24" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/>
            </svg>
        `;

        const progressContainer = createElement('div', 'audio-captcha-progress-container');
        const progressBar = createElement('div', 'audio-captcha-progress-bar');
        const progressFill = createElement('div', 'audio-captcha-progress-fill');
        const progressTime = createElement('div', 'audio-captcha-progress-time');
        progressTime.textContent = '0:00 / 0:00';

        progressBar.appendChild(progressFill);
        progressContainer.appendChild(progressBar);
        progressContainer.appendChild(progressTime);

        const volumeContainer = createElement('div', 'audio-captcha-volume-container');
        const volumeButton = createElement('button', 'audio-captcha-volume-btn', {
            type: 'button',
            'aria-label': '音量',
            title: '音量'
        });
        volumeButton.innerHTML = `
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
            </svg>
        `;
        const volumeSlider = createElement('input', 'audio-captcha-volume-slider', {
            type: 'range',
            min: '0',
            max: '100',
            value: '80',
            'aria-label': '音量调节'
        });
        volumeContainer.appendChild(volumeButton);
        volumeContainer.appendChild(volumeSlider);

        playerContainer.appendChild(playButton);
        playerContainer.appendChild(progressContainer);
        playerContainer.appendChild(volumeContainer);

        const replayButton = createElement('button', 'audio-captcha-replay-btn', {
            type: 'button',
            'aria-label': config.replayText
        });
        replayButton.innerHTML = `
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/>
            </svg>
            <span>重新播放</span>
        `;

        const inputContainer = createElement('div', 'audio-captcha-input-container');
        const inputLabel = createElement('label', 'audio-captcha-input-label');
        inputLabel.setAttribute('for', `audio-captcha-input-${instanceId}`);
        inputLabel.textContent = config.placeholderText;

        const input = createElement('input', 'audio-captcha-input', {
            type: 'text',
            id: `audio-captcha-input-${instanceId}`,
            placeholder: '输入验证码',
            autocomplete: 'off',
            maxlength: '8',
            'aria-describedby': `audio-captcha-hint-${instanceId}`
        });
        input.setAttribute('aria-label', config.placeholderText);

        const hint = createElement('span', 'audio-captcha-hint', {
            id: `audio-captcha-hint-${instanceId}`
        });
        hint.textContent = '请输入听到的4-6位字符';

        const verifyButton = createElement('button', 'audio-captcha-verify-btn', {
            type: 'button',
            disabled: 'disabled'
        });
        verifyButton.textContent = '验证';

        inputContainer.appendChild(inputLabel);
        inputContainer.appendChild(input);
        inputContainer.appendChild(hint);
        inputContainer.appendChild(verifyButton);

        const message = createElement('div', 'audio-captcha-message', {
            role: 'alert',
            'aria-live': 'assertive'
        });

        body.appendChild(playerContainer);
        body.appendChild(replayButton);
        body.appendChild(inputContainer);
        body.appendChild(message);

        header.appendChild(title);

        widget.appendChild(header);
        widget.appendChild(body);

        container.appendChild(widget);

        const liveRegion = document.createElement('div');
        liveRegion.setAttribute('role', 'status');
        liveRegion.setAttribute('aria-live', 'polite');
        liveRegion.className = 'audio-captcha-live-region';
        container.appendChild(liveRegion);

        instance.elements.widget = widget;
        instance.elements.header = header;
        instance.elements.title = title;
        instance.elements.body = body;
        instance.elements.playButton = playButton;
        instance.elements.playIcon = playButton.querySelector('.audio-captcha-play-icon');
        instance.elements.pauseIcon = playButton.querySelector('.audio-captcha-pause-icon');
        instance.elements.progressContainer = progressContainer;
        instance.elements.progressFill = progressFill;
        instance.elements.progressTime = progressTime;
        instance.elements.volumeButton = volumeButton;
        instance.elements.volumeSlider = volumeSlider;
        instance.elements.replayButton = replayButton;
        instance.elements.input = input;
        instance.elements.inputLabel = inputLabel;
        instance.elements.verifyButton = verifyButton;
        instance.elements.message = message;
        instance.elements.liveRegion = liveRegion;

        createAudioElement(instance);

        showLoading(instance);
        fetchCaptcha(instance).then(data => {
            instance.state.captchaId = data.id;
            instance.state.loading = false;
            hideLoading(instance);
            updateLiveRegion(instance, '音频验证码已加载完成');
        }).catch(err => {
            instance.state.loading = false;
            hideLoading(instance);
            renderError(instance, err.message || config.errorText);
            updateLiveRegion(instance, '音频验证码加载失败');
        });
    }

    function createAudioElement(instance) {
        if (instance.audioElement) {
            instance.audioElement.pause();
            instance.audioElement.src = '';
            instance.audioElement = null;
        }

        const audio = new Audio();
        audio.preload = 'auto';

        audio.addEventListener('timeupdate', () => {
            updateProgress(instance, audio);
        });

        audio.addEventListener('loadedmetadata', () => {
            updateDuration(instance, audio);
        });

        audio.addEventListener('ended', () => {
            instance.state.playing = false;
            updatePlayState(instance);
            updateLiveRegion(instance, '音频播放完成');
        });

        audio.addEventListener('error', (e) => {
            instance.state.playing = false;
            updatePlayState(instance);
            showError(instance, '音频加载失败');
            updateLiveRegion(instance, '音频加载失败');
        });

        audio.addEventListener('play', () => {
            instance.state.playing = true;
            updatePlayState(instance);
        });

        audio.addEventListener('pause', () => {
            if (!audio.ended) {
                instance.state.playing = false;
                updatePlayState(instance);
            }
        });

        instance.audioElement = audio;
    }

    function updateProgress(instance, audio) {
        if (!instance.elements.progressFill || !audio.duration) return;

        const progress = (audio.currentTime / audio.duration) * 100;
        instance.elements.progressFill.style.width = `${progress}%`;

        const currentTime = formatTime(audio.currentTime);
        const duration = formatTime(audio.duration);
        instance.elements.progressTime.textContent = `${currentTime} / ${duration}`;
    }

    function updateDuration(instance, audio) {
        const duration = formatTime(audio.duration);
        instance.elements.progressTime.textContent = `0:00 / ${duration}`;
    }

    function formatTime(seconds) {
        if (isNaN(seconds)) return '0:00';
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }

    function updatePlayState(instance) {
        const { playButton, playIcon, pauseIcon } = instance.elements;

        if (instance.state.playing) {
            if (playIcon) hide(playIcon);
            if (pauseIcon) show(pauseIcon);
            playButton.setAttribute('aria-label', instance.config.pauseText);
            addClass(playButton, 'audio-captcha-playing');
        } else {
            if (playIcon) show(playIcon);
            if (pauseIcon) hide(pauseIcon);
            playButton.setAttribute('aria-label', instance.config.playText);
            removeClass(playButton, 'audio-captcha-playing');
        }
    }

    function showLoading(instance) {
        const { body } = instance.elements;
        const loading = createElement('div', 'audio-captcha-loading', {
            role: 'status',
            'aria-label': instance.config.loadingText
        });
        loading.innerHTML = `
            <div class="audio-captcha-loading-spinner" aria-hidden="true">
                <div class="audio-captcha-spinner-ring"></div>
                <div class="audio-captcha-spinner-ring"></div>
                <div class="audio-captcha-spinner-ring"></div>
            </div>
            <span class="audio-captcha-loading-text">${instance.config.loadingText}</span>
        `;
        body.insertBefore(loading, body.firstChild);
        instance.elements.loading = loading;
    }

    function hideLoading(instance) {
        if (instance.elements.loading) {
            instance.elements.loading.remove();
            instance.elements.loading = null;
        }
    }

    function fetchCaptcha(instance) {
        const url = getAbsoluteUrl(`/api/captcha/audio?app_id=${instance.config.appId || ''}`);

        return request(url, { method: 'GET' }).then(data => {
            if (instance.audioElement) {
                instance.audioElement.src = `data:audio/wav;base64,${data.audio_b64}`;
            }
            instance.captchaDuration = data.duration || 0;
            return data;
        });
    }

    function updateLiveRegion(instance, message) {
        if (instance.elements.liveRegion) {
            instance.elements.liveRegion.textContent = message;
        }
    }

    function showError(instance, errorMessage) {
        const { message } = instance.elements;
        message.textContent = errorMessage;
        message.className = 'audio-captcha-message audio-captcha-message-show audio-captcha-message-error';
        show(message);
        announceToScreenReader(errorMessage);
    }

    function showSuccess(instance, successMessage) {
        const { message } = instance.elements;
        message.textContent = successMessage;
        message.className = 'audio-captcha-message audio-captcha-message-show audio-captcha-message-success';
        show(message);
        announceToScreenReader(successMessage);
    }

    function clearMessage(instance) {
        const { message } = instance.elements;
        hide(message);
        message.className = 'audio-captcha-message';
    }

    function renderError(instance, errorMessage) {
        const { body, liveRegion } = instance.elements;

        const error = createElement('div', 'audio-captcha-error', {
            role: 'alert'
        });
        error.innerHTML = `
            <div class="audio-captcha-error-icon" aria-hidden="true">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="#ff4d4f">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
                </svg>
            </div>
            <span class="audio-captcha-error-text">${errorMessage}</span>
            <button type="button" class="audio-captcha-retry-btn" aria-label="重新加载">重新加载</button>
        `;

        const retryBtn = error.querySelector('.audio-captcha-retry-btn');
        if (retryBtn) {
            retryBtn.addEventListener('click', () => {
                reload(instance);
            });
        }

        body.innerHTML = '';
        body.appendChild(error);

        if (liveRegion) {
            liveRegion.textContent = `错误: ${errorMessage}`;
        }
    }

    function bindEvents(instance) {
        const { playButton, replayButton, volumeButton, volumeSlider, input, verifyButton, progressContainer } = instance.elements;

        if (playButton) {
            playButton.addEventListener('click', () => {
                togglePlay(instance);
            });
        }

        if (replayButton) {
            replayButton.addEventListener('click', () => {
                replay(instance);
            });
        }

        if (volumeButton) {
            volumeButton.addEventListener('click', () => {
                toggleMute(instance);
            });
        }

        if (volumeSlider) {
            volumeSlider.addEventListener('input', (e) => {
                setVolume(instance, parseInt(e.target.value, 10) / 100);
            });
        }

        if (input) {
            input.addEventListener('input', () => {
                validateInput(instance);
            });

            input.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && !verifyButton.disabled) {
                    verify(instance);
                }
            });
        }

        if (verifyButton) {
            verifyButton.addEventListener('click', () => {
                verify(instance);
            });
        }

        if (progressContainer) {
            progressContainer.addEventListener('click', (e) => {
                seekAudio(instance, e);
            });
        }
    }

    function togglePlay(instance) {
        if (!instance.audioElement || !instance.audioElement.src) return;

        if (instance.state.playing) {
            instance.audioElement.pause();
        } else {
            instance.audioElement.play().catch(err => {
                console.error('[AudioCaptcha] Play error:', err);
                showError(instance, '播放失败');
            });
        }
    }

    function replay(instance) {
        if (!instance.audioElement) return;

        instance.audioElement.currentTime = 0;
        instance.audioElement.play().catch(err => {
            console.error('[AudioCaptcha] Replay error:', err);
            showError(instance, '播放失败');
        });

        updateLiveRegion(instance, '重新播放音频');
    }

    function toggleMute(instance) {
        if (!instance.audioElement) return;

        instance.audioElement.muted = !instance.audioElement.muted;

        const { volumeButton } = instance.elements;
        if (volumeButton) {
            if (instance.audioElement.muted) {
                addClass(volumeButton, 'audio-captcha-muted');
                volumeButton.setAttribute('aria-label', '取消静音');
            } else {
                removeClass(volumeButton, 'audio-captcha-muted');
                volumeButton.setAttribute('aria-label', '静音');
            }
        }
    }

    function setVolume(instance, volume) {
        if (!instance.audioElement) return;

        instance.audioElement.volume = Math.max(0, Math.min(1, volume));

        if (instance.elements.volumeSlider) {
            instance.elements.volumeSlider.value = volume * 100;
        }
    }

    function seekAudio(instance, event) {
        if (!instance.audioElement || !instance.audioElement.duration) return;

        const rect = event.currentTarget.getBoundingClientRect();
        const percent = (event.clientX - rect.left) / rect.width;
        instance.audioElement.currentTime = percent * instance.audioElement.duration;
    }

    function validateInput(instance) {
        const { input, verifyButton } = instance.elements;
        const value = input.value.trim();

        if (value.length >= 4 && value.length <= 8) {
            verifyButton.disabled = false;
            removeClass(verifyButton, 'audio-captcha-btn-disabled');
        } else {
            verifyButton.disabled = true;
            addClass(verifyButton, 'audio-captcha-btn-disabled');
        }
    }

    function verify(instance) {
        const { input, verifyButton, message } = instance.elements;
        const code = input.value.trim();

        if (!code || code.length < 4 || code.length > 8) {
            showError(instance, '请输入4-8位验证码');
            return;
        }

        verifyButton.disabled = true;
        verifyButton.textContent = '验证中...';
        clearMessage(instance);

        updateLiveRegion(instance, '验证中...');

        const verifyData = {
            captcha_id: instance.state.captchaId,
            code: code
        };

        request(getAbsoluteUrl('/api/captcha/audio/verify'), {
            method: 'POST',
            body: verifyData
        }).then(response => {
            if (response.success) {
                instance.state.verified = true;
                showSuccess(instance, instance.config.successText);
                handleSuccess(instance, response);
            } else {
                showError(instance, response.message || instance.config.failText);
                verifyButton.disabled = false;
                verifyButton.textContent = '验证';
                handleError(instance, response.message);
            }
        }).catch(err => {
            showError(instance, instance.config.errorText);
            verifyButton.disabled = false;
            verifyButton.textContent = '验证';
            handleError(instance, err.message);
        });
    }

    function handleSuccess(instance, response) {
        instance.state.verified = true;

        if (instance.config.onSuccess) {
            try {
                instance.config.onSuccess({
                    captchaId: instance.state.captchaId,
                    token: response.token || instance.state.captchaId,
                    response: response
                });
            } catch (e) {
                console.error('[AudioCaptcha] onSuccess callback error:', e);
            }
        }
    }

    function handleError(instance, errorMessage) {
        if (instance.config.onError) {
            try {
                instance.config.onError({
                    captchaId: instance.state.captchaId,
                    error: errorMessage
                });
            } catch (e) {
                console.error('[AudioCaptcha] onError callback error:', e);
            }
        }
    }

    function reload(instance) {
        instance.state.captchaId = null;
        instance.state.verified = false;
        instance.state.loading = true;

        const { input, verifyButton, progressFill, progressTime } = instance.elements;

        input.value = '';
        verifyButton.disabled = true;
        addClass(verifyButton, 'audio-captcha-btn-disabled');
        verifyButton.textContent = '验证';

        if (progressFill) progressFill.style.width = '0%';
        if (progressTime) progressTime.textContent = '0:00 / 0:00';

        clearMessage(instance);
        showLoading(instance);

        fetchCaptcha(instance).then(data => {
            instance.state.captchaId = data.id;
            instance.state.loading = false;
            hideLoading(instance);
            updateLiveRegion(instance, '音频验证码已重新加载');
        }).catch(err => {
            instance.state.loading = false;
            hideLoading(instance);
            renderError(instance, err.message || instance.config.errorText);
        });
    }

    function destroyInstance(instance) {
        instance.state.destroyed = true;
        AudioCaptcha.instances.delete(instance.id);

        if (AudioCaptcha.defaultInstance === instance) {
            const remaining = Array.from(AudioCaptcha.instances.values());
            AudioCaptcha.defaultInstance = remaining.length > 0 ? remaining[0] : null;
        }

        if (instance.audioElement) {
            instance.audioElement.pause();
            instance.audioElement.src = '';
            instance.audioElement = null;
        }

        if (instance.container) {
            instance.container.innerHTML = '';
            removeClass(instance.container, 'audio-captcha-container');
        }
    }

    AudioCaptcha.verify = function(options) {
        let container;

        if (options && options.container) {
            container = getElement(options.container);
        }

        if (!container) {
            container = createElement('div', 'audio-captcha-overlay');
            document.body.appendChild(container);
        }

        const instanceOptions = extend({}, options, { container: container });
        const instance = AudioCaptcha.create(instanceOptions);

        return {
            then: (resolve, reject) => {
                const originalOnSuccess = instance.config.onSuccess;
                instance.config.onSuccess = (result) => {
                    if (originalOnSuccess) {
                        try {
                            originalOnSuccess(result);
                        } catch (e) {
                            console.error('[AudioCaptcha] onSuccess error:', e);
                        }
                    }
                    resolve(result);
                };

                const originalOnError = instance.config.onError;
                instance.config.onError = (error) => {
                    if (originalOnError) {
                        try {
                            originalOnError(error);
                        } catch (e) {
                            console.error('[AudioCaptcha] onError error:', e);
                        }
                    }
                    if (reject) {
                        reject(error);
                    }
                };

                return instance;
            },
            destroy: () => destroyInstance(instance)
        };
    };

    AudioCaptcha.play = function(instanceId) {
        const instance = instanceId ? AudioCaptcha.instances.get(instanceId) : AudioCaptcha.defaultInstance;
        if (instance && instance.audioElement) {
            instance.audioElement.play();
        }
    };

    AudioCaptcha.pause = function(instanceId) {
        const instance = instanceId ? AudioCaptcha.instances.get(instanceId) : AudioCaptcha.defaultInstance;
        if (instance && instance.audioElement) {
            instance.audioElement.pause();
        }
    };

    AudioCaptcha.replay = function(instanceId) {
        const instance = instanceId ? AudioCaptcha.instances.get(instanceId) : AudioCaptcha.defaultInstance;
        if (instance) {
            replay(instance);
        }
    };

    AudioCaptcha.refresh = function(instanceId) {
        const instance = instanceId ? AudioCaptcha.instances.get(instanceId) : AudioCaptcha.defaultInstance;
        if (instance && !instance.state.destroyed) {
            reload(instance);
        }
    };

    AudioCaptcha.getInstance = function(instanceId) {
        if (instanceId) {
            return AudioCaptcha.instances.get(instanceId) || null;
        }
        return AudioCaptcha.defaultInstance;
    };

    AudioCaptcha.destroy = function(instanceId) {
        if (instanceId) {
            const instance = AudioCaptcha.instances.get(instanceId);
            if (instance) {
                destroyInstance(instance);
            }
        } else {
            AudioCaptcha.instances.forEach((instance) => {
                destroyInstance(instance);
            });
            AudioCaptcha.defaultInstance = null;
        }
    };

    const originalAudioCaptcha = global.AudioCaptcha;

    AudioCaptcha.noConflict = function() {
        global.AudioCaptcha = originalAudioCaptcha;
        return AudioCaptcha;
    };

    global.AudioCaptcha = AudioCaptcha;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = AudioCaptcha;
    }

})(typeof window !== 'undefined' ? window : this);
