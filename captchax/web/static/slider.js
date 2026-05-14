/**
 * CaptchaX Slider Module
 * @version 1.0.0
 * @license MIT
 */
(function(global) {
    'use strict';

    const SliderModule = {
        name: 'slider',
        version: '1.0.0'
    };

    function createSliderInteraction(instance, data) {
        const sliderImg = instance.elements.sliderImg;
        const sliderThumb = instance.elements.sliderThumb;
        const sliderTrack = instance.elements.sliderTrack;
        const message = instance.elements.message;

        if (!sliderImg || !sliderThumb || !sliderTrack) {
            console.error('[CaptchaX Slider] Required elements not found');
            return null;
        }

        const sliderSize = instance.config.sliderSize || 50;
        const trackWidth = sliderTrack.offsetWidth;
        const maxX = trackWidth - sliderSize;

        let isDragging = false;
        let startX = 0;
        let currentX = 0;
        let track = [];
        let startTime = 0;
        let lastTime = 0;

        const state = {
            isDragging: false,
            currentX: 0,
            startTime: 0,
            track: [],
            verified: false,
            lastMoveTime: 0
        };

        function initPosition() {
            if (data && typeof data.target_x === 'number') {
                currentX = data.target_x;
            } else {
                currentX = 0;
            }
            sliderImg.style.left = currentX + 'px';
            sliderThumb.style.left = currentX + 'px';
        }

        function getClientX(e) {
            if (e.touches && e.touches.length > 0) {
                return e.touches[0].clientX;
            }
            return e.clientX;
        }

        function getClientY(e) {
            if (e.touches && e.touches.length > 0) {
                return e.touches[0].clientY;
            }
            return e.clientY;
        }

        function onStart(e) {
            if (instance.state.verified) return;
            if (isDragging) return;

            e.preventDefault();

            isDragging = true;
            startX = getClientX(e);
            startTime = Date.now();
            lastTime = startTime;
            currentX = 0;
            track = [];

            addClass(sliderThumb, 'captchax-dragging');
            addClass(sliderThumb, 'captchax-slider-thumb-active');
            document.body.style.userSelect = 'none';
            document.body.style.cursor = 'grabbing';

            state.isDragging = true;
            state.startTime = startTime;
            state.track = [];
        }

        function onMove(e) {
            if (!isDragging) return;

            e.preventDefault();

            const clientX = getClientX(e);
            const clientY = getClientY(e);

            currentX = clientX - startX;
            currentX = Math.max(0, Math.min(currentX, maxX));

            sliderImg.style.left = currentX + 'px';
            sliderThumb.style.left = currentX + 'px';

            const now = Date.now();
            const dt = now - lastTime;

            if (dt >= 10) {
                track.push({
                    x: currentX,
                    y: 0,
                    t: now - startTime,
                    dt: dt,
                    vx: dt > 0 ? (currentX - (track.length > 0 ? track[track.length - 1].x : 0)) / dt : 0
                });
                lastTime = now;
            }

            state.currentX = currentX;
            state.lastMoveTime = now;

            updateSliderVisual();
        }

        function onEnd(e) {
            if (!isDragging) return;

            isDragging = false;
            removeClass(sliderThumb, 'captchax-dragging');
            removeClass(sliderThumb, 'captchax-slider-thumb-active');
            document.body.style.userSelect = '';
            document.body.style.cursor = '';

            state.isDragging = false;

            if (track.length > 0) {
                const duration = Date.now() - startTime;
                instance.track = track.map(p => ({
                    x: p.x,
                    y: p.y,
                    t: p.t,
                    dt: p.dt,
                    vx: p.vx
                }));
                instance.track.duration = duration;
                instance.track.totalX = currentX;
                instance.track.moveCount = track.length;
            }

            verifyPosition(currentX, data);
        }

        function updateSliderVisual() {
            const progress = currentX / maxX;
            const opacity = 0.5 + progress * 0.5;
            sliderThumb.style.boxShadow = `0 2px 8px rgba(24, 144, 255, ${opacity * 0.4 + 0.2})`;
        }

        function verifyPosition(position, captchaData) {
            if (!message) return;

            showLoadingState();

            const verifyData = {
                captcha_id: instance.state.captchaId,
                target_x: Math.round(position),
                target_y: captchaData.target_y || 0,
                track: instance.track,
                client_info: getClientInfo()
            };

            requestVerification(instance, verifyData);
        }

        function showLoadingState() {
            if (message) {
                message.innerHTML = '<span class="captchax-loading-inline">验证中...</span>';
                message.className = 'captchax-message captchax-message-show';
                removeClass(message, 'captchax-message-success captchax-message-error');
            }
        }

        function getClientInfo() {
            return {
                userAgent: navigator.userAgent,
                language: navigator.language,
                screenWidth: screen.width,
                screenHeight: screen.height,
                devicePixelRatio: window.devicePixelRatio,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
            };
        }

        function requestVerification(inst, data) {
            const url = getVerificationUrl(inst);

            fetch(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}`);
                }
                return response.json();
            })
            .then(result => {
                handleVerificationResult(inst, result);
            })
            .catch(error => {
                handleVerificationError(inst, error);
            });
        }

        function getVerificationUrl(inst) {
            const baseUrl = inst.config.serverUrl || CaptchaX.config.serverUrl;
            return `${baseUrl.replace(/\/$/, '')}/api/captcha/slider/verify`;
        }

        function handleVerificationResult(inst, result) {
            if (result.success) {
                handleSuccess(inst, result);
            } else {
                handleError(inst, result.message || CaptchaX.config.failText);
            }
        }

        function handleSuccess(inst, result) {
            inst.state.verified = true;
            state.verified = true;

            if (message) {
                message.textContent = CaptchaX.config.successText;
                message.className = 'captchax-message captchax-message-show captchax-message-success';
            }

            if (sliderThumb) {
                addClass(sliderThumb, 'captchax-slider-success');
                sliderThumb.style.cursor = 'default';
            }

            if (sliderTrack) {
                addClass(sliderTrack, 'captchax-slider-track-success');
            }

            CaptchaX.callbacks.onSuccess.forEach(cb => {
                try {
                    cb({
                        captchaId: inst.state.captchaId,
                        token: result.token || inst.state.captchaId,
                        type: 'slider',
                        response: result
                    });
                } catch (e) {
                    console.error('[CaptchaX] onSuccess callback error:', e);
                }
            });

            CaptchaX.callbacks.onVerify.forEach(cb => {
                try {
                    cb({
                        success: true,
                        captchaId: inst.state.captchaId,
                        token: result.token || inst.state.captchaId,
                        type: 'slider'
                    });
                } catch (e) {
                    console.error('[CaptchaX] onVerify callback error:', e);
                }
            });

            if (inst.config.autoClose !== false) {
                setTimeout(() => {
                    if (!inst.state.destroyed) {
                        CaptchaX.destroy(inst.id);
                    }
                }, 1500);
            }
        }

        function handleError(inst, errorMessage) {
            if (message) {
                message.textContent = errorMessage || CaptchaX.config.failText;
                message.className = 'captchax-message captchax-message-show captchax-message-error';
            }

            if (sliderThumb) {
                addClass(sliderThumb, 'captchax-slider-error');
                setTimeout(() => {
                    removeClass(sliderThumb, 'captchax-slider-error');
                }, 500);
            }

            CaptchaX.callbacks.onError.forEach(cb => {
                try {
                    cb({
                        captchaId: inst.state.captchaId,
                        error: errorMessage,
                        type: 'slider'
                    });
                } catch (e) {
                    console.error('[CaptchaX] onError callback error:', e);
                }
            });

            const isExpired = errorMessage.includes('过期') || errorMessage.includes('expired');
            const isFailed = errorMessage.includes('验证失败') || errorMessage.includes('incorrect');

            if (isExpired || isFailed) {
                setTimeout(() => {
                    if (!inst.state.destroyed && !inst.state.verified) {
                        resetSlider();
                        CaptchaX.refresh(inst.id);
                    }
                }, 1500);
            }
        }

        function resetSlider() {
            currentX = 0;
            track = [];
            startTime = 0;
            lastTime = 0;

            if (sliderImg) {
                sliderImg.style.left = '0px';
            }
            if (sliderThumb) {
                sliderThumb.style.left = '0px';
                sliderThumb.style.boxShadow = '';
            }
            if (message) {
                message.className = 'captchax-message';
            }
        }

        function destroy() {
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup', onEnd);
            document.removeEventListener('touchmove', onMove);
            document.removeEventListener('touchend', onEnd);

            if (sliderThumb) {
                sliderThumb.removeEventListener('mousedown', onStart);
                sliderThumb.removeEventListener('touchstart', onStart, { passive: false });
            }

            state.isDragging = false;
            state.verified = false;
            state.track = [];
        }

        sliderThumb.addEventListener('mousedown', onStart, { passive: false });
        sliderThumb.addEventListener('touchstart', onStart, { passive: false });

        document.addEventListener('mousemove', onMove);
        document.addEventListener('mouseup', onEnd);
        document.addEventListener('touchmove', onMove, { passive: false });
        document.addEventListener('touchend', onEnd);
        document.addEventListener('touchcancel', onEnd);

        initPosition();

        return {
            state: state,
            reset: resetSlider,
            destroy: destroy,
            getTrack: function() {
                return instance.track;
            },
            getPosition: function() {
                return currentX;
            }
        };
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

    SliderModule.create = createSliderInteraction;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = SliderModule;
    } else {
        global.CaptchaXSlider = SliderModule;
    }

})(typeof window !== 'undefined' ? window : this);
