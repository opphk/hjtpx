/**
 * CaptchaX Click Module
 * @version 1.0.0
 * @license MIT
 */
(function(global) {
    'use strict';

    const ClickModule = {
        name: 'click',
        version: '1.0.0'
    };

    function createClickInteraction(instance, data) {
        const clickImage = instance.elements.clickImage;
        const clickIndicators = instance.elements.clickIndicators;
        const clickInstruction = instance.elements.clickInstruction;
        const message = instance.elements.message;

        if (!clickImage) {
            console.error('[CaptchaX Click] Required element clickImage not found');
            return null;
        }

        let clicks = [];
        let clickMarkers = [];
        let startTime = 0;
        let isVerifying = false;
        const targetChars = data.target_chars || [];
        const totalClicks = targetChars.length;

        const state = {
            clicks: [],
            currentIndex: 0,
            isVerifying: false,
            verified: false,
            startTime: 0
        };

        function init() {
            startTime = Date.now();
            state.startTime = startTime;
            updateProgress();
            updateInstruction();
        }

        function getImageCoordinates(e) {
            const rect = clickImage.getBoundingClientRect();
            const scaleX = clickImage.naturalWidth / rect.width;
            const scaleY = clickImage.naturalHeight / rect.height;

            const x = Math.round((e.clientX - rect.left) * scaleX);
            const y = Math.round((e.clientY - rect.top) * scaleY);

            return { x, y, scaleX, scaleY };
        }

        function onImageClick(e) {
            if (instance.state.verified) return;
            if (state.isVerifying) return;
            if (clicks.length >= totalClicks) return;

            const coords = getImageCoordinates(e);
            const time = Date.now() - startTime;

            clicks.push({
                x: coords.x,
                y: coords.y,
                t: time,
                screenX: e.clientX,
                screenY: e.clientY
            });

            instance.track = clicks;
            state.clicks = clicks;
            state.currentIndex = clicks.length;

            addClickMarker(coords, clicks.length);

            if (clicks.length === totalClicks) {
                verifyClicks();
            } else {
                updateProgress();
                updateInstruction();
            }
        }

        function addClickMarker(coords, index) {
            if (!clickIndicators) return;

            const marker = document.createElement('span');
            marker.className = 'captchax-click-indicator';
            marker.textContent = index;

            const rect = clickImage.getBoundingClientRect();
            const displayX = (coords.x / coords.scaleX);
            const displayY = (coords.y / coords.scaleY);

            marker.style.left = displayX + 'px';
            marker.style.top = displayY + 'px';

            clickIndicators.appendChild(marker);
            clickMarkers.push(marker);

            requestAnimationFrame(() => {
                marker.classList.add('captchax-click-indicator-animate');
            });
        }

        function removeClickMarkers() {
            clickMarkers.forEach(marker => {
                if (marker.parentNode) {
                    marker.parentNode.removeChild(marker);
                }
            });
            clickMarkers = [];
        }

        function updateProgress() {
            const progressFill = instance.container.querySelector('.captchax-progress-fill');
            const clickedCount = instance.container.querySelector('.captchax-clicked-count');

            if (progressFill) {
                const progress = totalClicks > 0 ? (clicks.length / totalClicks) * 100 : 0;
                progressFill.style.width = progress + '%';
            }

            if (clickedCount) {
                clickedCount.textContent = clicks.length;
            }
        }

        function updateInstruction() {
            if (!clickInstruction) return;

            const charsContainer = clickInstruction.querySelector('.captchax-target-chars');
            const instructionText = clickInstruction.querySelector('.captchax-instruction-text');

            if (charsContainer) {
                const remaining = targetChars.slice(clicks.length).join(' ');
                charsContainer.textContent = remaining;

                if (clicks.length > 0) {
                    const clicked = targetChars.slice(0, clicks.length);
                    const remaining = targetChars.slice(clicks.length);
                    charsContainer.innerHTML = `<span class="captchax-chars-clicked">${clicked.join(' ')}</span> ${remaining.join(' ')}`;
                }
            }

            if (instructionText) {
                if (clicks.length === 0) {
                    instructionText.textContent = '请依次点击：';
                } else if (clicks.length < totalClicks) {
                    instructionText.textContent = `请继续点击 (${clicks.length}/${totalClicks})：`;
                }
            }
        }

        function verifyClicks() {
            if (state.isVerifying) return;
            state.isVerifying = true;
            isVerifying = true;

            showLoadingState();

            const verifyData = {
                captcha_id: instance.state.captchaId,
                clicks: clicks,
                click_count: clicks.length,
                target_chars: targetChars,
                duration: Date.now() - startTime,
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

            if (clickImage) {
                clickImage.style.pointerEvents = 'none';
            }
        }

        function hideLoadingState() {
            if (clickImage) {
                clickImage.style.pointerEvents = '';
            }
        }

        function getClientInfo() {
            return {
                userAgent: navigator.userAgent,
                language: navigator.language,
                screenWidth: screen.width,
                screenHeight: screen.height,
                devicePixelRatio: window.devicePixelRatio,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                clickCount: clicks.length,
                duration: Date.now() - startTime
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
            return `${baseUrl.replace(/\/$/, '')}/api/captcha/click/verify`;
        }

        function handleVerificationResult(inst, result) {
            state.isVerifying = false;
            isVerifying = false;

            if (result.success) {
                handleSuccess(inst, result);
            } else {
                handleError(inst, result.message || CaptchaX.config.failText, result);
            }
        }

        function handleSuccess(inst, result) {
            inst.state.verified = true;
            state.verified = true;

            hideLoadingState();

            if (message) {
                message.textContent = CaptchaX.config.successText;
                message.className = 'captchax-message captchax-message-show captchax-message-success';
            }

            highlightClickMarkersSuccess();

            CaptchaX.callbacks.onSuccess.forEach(cb => {
                try {
                    cb({
                        captchaId: inst.state.captchaId,
                        token: result.token || inst.state.captchaId,
                        score: result.score,
                        type: 'click',
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
                        score: result.score,
                        type: 'click'
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

        function handleError(inst, errorMessage, result) {
            hideLoadingState();

            if (message) {
                message.textContent = errorMessage || CaptchaX.config.failText;
                message.className = 'captchax-message captchax-message-show captchax-message-error';
            }

            highlightClickMarkersError(result);

            CaptchaX.callbacks.onError.forEach(cb => {
                try {
                    cb({
                        captchaId: inst.state.captchaId,
                        error: errorMessage,
                        score: result ? result.score : 0,
                        type: 'click'
                    });
                } catch (e) {
                    console.error('[CaptchaX] onError callback error:', e);
                }
            });

            const isExpired = errorMessage.includes('过期') || errorMessage.includes('expired');
            const isFailed = errorMessage.includes('失败') || errorMessage.includes('incorrect') || errorMessage.includes('failed');

            if (isExpired || isFailed) {
                setTimeout(() => {
                    if (!inst.state.destroyed && !inst.state.verified) {
                        resetClicks();
                        CaptchaX.refresh(inst.id);
                    }
                }, 2000);
            }
        }

        function highlightClickMarkersSuccess() {
            clickMarkers.forEach(marker => {
                removeClass(marker, 'captchax-click-indicator-error');
                addClass(marker, 'captchax-click-indicator-success');
            });
        }

        function highlightClickMarkersError(result) {
            clickMarkers.forEach((marker, index) => {
                if (result && result.wrong_indices && result.wrong_indices.includes(index)) {
                    addClass(marker, 'captchax-click-indicator-error');
                    removeClass(marker, 'captchax-click-indicator-animate');
                }
            });
        }

        function resetClicks() {
            clicks = [];
            state.clicks = [];
            state.currentIndex = 0;
            state.isVerifying = false;
            isVerifying = false;

            removeClickMarkers();

            if (message) {
                message.className = 'captchax-message';
            }

            updateProgress();
            updateInstruction();
        }

        function onRefreshClick() {
            if (instance.state.verified) return;

            resetClicks();

            CaptchaX.refresh(instance.id);
        }

        function destroy() {
            clickImage.removeEventListener('click', onImageClick);

            const refreshBtn = instance.container.querySelector('.captchax-refresh-btn');
            if (refreshBtn) {
                refreshBtn.removeEventListener('click', onRefreshClick);
            }

            removeClickMarkers();

            state.clicks = [];
            state.verified = false;
            clicks = [];
        }

        clickImage.addEventListener('click', onImageClick);

        const refreshBtn = instance.container.querySelector('.captchax-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', onRefreshClick);
        }

        init();

        return {
            state: state,
            reset: resetClicks,
            destroy: destroy,
            getClicks: function() {
                return clicks;
            },
            getClickCount: function() {
                return clicks.length;
            },
            getRemainingChars: function() {
                return targetChars.slice(clicks.length);
            },
            isVerifying: function() {
                return isVerifying;
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

    ClickModule.create = createClickInteraction;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = ClickModule;
    } else {
        global.CaptchaXClick = ClickModule;
    }

})(typeof window !== 'undefined' ? window : this);
