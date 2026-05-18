(function(globalContext) {
    'use strict';

    /**
     * 验证码核心模块
     * 提供滑块验证码、点选验证码等核心功能
     */
    var CaptchaCore = (function() {
        var version = '2.0.0';

        /**
         * 验证码基类
         */
        function CaptchaBase(options) {
            this.options = Object.assign({
                container: null,
                apiBase: '/api/v1',
                onSuccess: null,
                onError: null,
                onRefresh: null,
                width: 360,
                height: 200
            }, options);

            this.container = typeof options.container === 'string' ?
                document.querySelector(options.container) :
                options.container;

            this.isVerified = false;
            this.attempts = 0;
            this.maxAttempts = 3;
        }

        CaptchaBase.prototype.init = function() {
            if (!this.container) {
                console.error('验证码容器未找到');
                return;
            }
            this.render();
            this.bindEvents();
        };

        CaptchaBase.prototype.render = function() {
            throw new Error('子类必须实现 render 方法');
        };

        CaptchaBase.prototype.bindEvents = function() {
            throw new Error('子类必须实现 bindEvents 方法');
        };

        CaptchaBase.prototype.verify = async function(data) {
            throw new Error('子类必须实现 verify 方法');
        };

        CaptchaBase.prototype.reset = function() {
            this.isVerified = false;
            this.attempts = 0;
        };

        CaptchaBase.prototype.refresh = function() {
            this.reset();
            if (this.options.onRefresh) {
                this.options.onRefresh();
            }
        };

        /**
         * 滑块验证码类
         */
        function SliderCaptcha(options) {
            CaptchaBase.call(this, options);
            this.sliderPosition = 0;
            this.targetPosition = 0;
            this.isDragging = false;
            this.startX = 0;
        }

        SliderCaptcha.prototype = Object.create(CaptchaBase.prototype);
        SliderCaptcha.prototype.constructor = SliderCaptcha;

        SliderCaptcha.prototype.render = function() {
            this.container.innerHTML = this.getTemplate();
            this.sliderContainer = this.container.querySelector('.captcha-slider-container');
            this.sliderButton = this.container.querySelector('.captcha-slider-button');
            this.sliderTrack = this.container.querySelector('.captcha-slider-track');
            this.refreshButton = this.container.querySelector('.captcha-refresh');
        };

        SliderCaptcha.prototype.getTemplate = function() {
            return '<div class="captcha-slider-container">' +
                '<div class="captcha-slider-track"></div>' +
                '<span class="captcha-slider-text">拖动滑块完成拼图</span>' +
                '<div class="captcha-slider-button"><i class="fas fa-chevron-right"></i></div>' +
                '</div>' +
                '<button class="captcha-refresh"><i class="fas fa-redo"></i></button>';
        };

        SliderCaptcha.prototype.bindEvents = function() {
            var self = this;

            if (this.sliderButton) {
                this.sliderButton.addEventListener('mousedown', function(e) {
                    self.startDrag(e);
                });

                this.sliderButton.addEventListener('touchstart', function(e) {
                    self.startDrag(e);
                }, { passive: false });
            }

            if (this.refreshButton) {
                this.refreshButton.addEventListener('click', function() {
                    self.refresh();
                });
            }

            document.addEventListener('mousemove', function(e) {
                if (self.isDragging) {
                    self.onDrag(e);
                }
            });

            document.addEventListener('mouseup', function() {
                if (self.isDragging) {
                    self.endDrag();
                }
            });

            document.addEventListener('touchmove', function(e) {
                if (self.isDragging) {
                    self.onDrag(e);
                }
            }, { passive: false });

            document.addEventListener('touchend', function() {
                if (self.isDragging) {
                    self.endDrag();
                }
            });
        };

        SliderCaptcha.prototype.startDrag = function(e) {
            if (this.isVerified) return;

            this.isDragging = true;
            this.startX = e.type === 'touchstart' ? e.touches[0].clientX : e.clientX;
            this.sliderContainer.classList.add('is-dragging');
            this.sliderButton.classList.add('dragging');

            e.preventDefault();
        };

        SliderCaptcha.prototype.onDrag = function(e) {
            if (!this.isDragging) return;

            var currentX = e.type === 'touchmove' ? e.touches[0].clientX : e.clientX;
            var deltaX = currentX - this.startX;

            var containerWidth = this.sliderContainer ? this.sliderContainer.offsetWidth : this.options.width;
            var maxX = containerWidth - this.sliderButton.offsetWidth - 4;

            this.sliderPosition = Math.max(0, Math.min(deltaX, maxX));

            this.updateSliderUI();
        };

        SliderCaptcha.prototype.endDrag = function() {
            if (!this.isDragging) return;

            this.isDragging = false;
            this.sliderContainer.classList.remove('is-dragging');
            this.sliderButton.classList.remove('dragging');

            this.checkResult();
        };

        SliderCaptcha.prototype.updateSliderUI = function() {
            if (this.sliderButton) {
                this.sliderButton.style.left = this.sliderPosition + 'px';
            }
            if (this.sliderTrack) {
                this.sliderTrack.style.width = this.sliderPosition + 'px';
            }
        };

        SliderCaptcha.prototype.checkResult = function() {
            var containerWidth = this.sliderContainer ? this.sliderContainer.offsetWidth : this.options.width;
            var threshold = containerWidth * 0.1;

            this.attempts++;

            if (Math.abs(this.sliderPosition - this.targetPosition) < threshold) {
                this.onSuccess();
            } else {
                this.onError();
            }
        };

        SliderCaptcha.prototype.onSuccess = function() {
            this.isVerified = true;
            this.sliderButton.classList.add('success');
            this.sliderButton.innerHTML = '<i class="fas fa-check"></i>';

            if (this.options.onSuccess) {
                this.options.onSuccess({
                    type: 'slider',
                    position: this.sliderPosition,
                    attempts: this.attempts
                });
            }
        };

        SliderCaptcha.prototype.onError = function() {
            this.sliderButton.classList.add('error');
            this.sliderContainer.classList.add('error-flash');

            setTimeout(function() {
                this.resetSlider();
            }.bind(this), 500);

            if (this.options.onError) {
                this.options.onError({
                    type: 'slider',
                    attempts: this.attempts,
                    message: '验证失败，请重试'
                });
            }
        };

        SliderCaptcha.prototype.resetSlider = function() {
            this.sliderPosition = 0;
            this.sliderButton.classList.remove('error', 'success');
            this.sliderContainer.classList.remove('error-flash');
            this.sliderButton.innerHTML = '<i class="fas fa-chevron-right"></i>';
            this.updateSliderUI();

            if (this.attempts >= this.maxAttempts) {
                this.refresh();
            }
        };

        SliderCaptcha.prototype.reset = function() {
            CaptchaBase.prototype.reset.call(this);
            this.sliderPosition = 0;
            this.targetPosition = 0;
            this.resetSlider();
        };

        /**
         * 点选验证码类
         */
        function ClickCaptcha(options) {
            CaptchaBase.call(this, options);
            this.selectedPoints = [];
            this.requiredClicks = 4;
            this.imageLoaded = false;
        }

        ClickCaptcha.prototype = Object.create(CaptchaBase.prototype);
        ClickCaptcha.prototype.constructor = ClickCaptcha;

        ClickCaptcha.prototype.render = function() {
            this.container.innerHTML = this.getTemplate();
            this.clickImage = this.container.querySelector('.captcha-click-image');
            this.clickGrid = this.container.querySelector('.captcha-click-grid');
            this.progress = this.container.querySelector('.captcha-click-progress');
        };

        ClickCaptcha.prototype.getTemplate = function() {
            return '<div class="captcha-click-hint">' +
                '<i class="fas fa-hand-pointer hint-icon"></i>' +
                '<span>请依次点击图片中的</span>' +
                '<span class="count-badge partial">' + this.requiredClicks + '</span>' +
                '<span>个位置</span>' +
                '</div>' +
                '<div class="captcha-click-grid">' +
                '<img class="captcha-click-image" alt="验证码图片">' +
                '</div>' +
                '<div class="captcha-click-progress">已选择: 0/' + this.requiredClicks + '</div>';
        };

        ClickCaptcha.prototype.bindEvents = function() {
            var self = this;

            if (this.clickGrid) {
                this.clickGrid.addEventListener('click', function(e) {
                    self.onImageClick(e);
                });
            }
        };

        ClickCaptcha.prototype.onImageClick = function(e) {
            if (this.isVerified || this.selectedPoints.length >= this.requiredClicks) {
                return;
            }

            var rect = this.clickGrid.getBoundingClientRect();
            var x = e.clientX - rect.left;
            var y = e.clientY - rect.top;

            this.selectedPoints.push({ x: x, y: y });

            this.addMarker(x, y, this.selectedPoints.length);
            this.updateProgress();

            if (this.selectedPoints.length === this.requiredClicks) {
                this.verify(this.selectedPoints);
            }
        };

        ClickCaptcha.prototype.addMarker = function(x, y, number) {
            var marker = document.createElement('div');
            marker.className = 'captcha-click-marker';
            marker.textContent = number;
            marker.style.left = x + 'px';
            marker.style.top = y + 'px';

            this.clickGrid.appendChild(marker);
        };

        ClickCaptcha.prototype.updateProgress = function() {
            if (this.progress) {
                this.progress.textContent = '已选择: ' + this.selectedPoints.length + '/' + this.requiredClicks;
            }

            var badge = this.container.querySelector('.count-badge');
            if (badge) {
                badge.textContent = this.requiredClicks - this.selectedPoints.length;
                if (this.selectedPoints.length === this.requiredClicks) {
                    badge.classList.remove('partial');
                    badge.classList.add('complete');
                }
            }
        };

        ClickCaptcha.prototype.verify = async function(points) {
            try {
                var response = await fetch(this.options.apiBase + '/captcha/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        type: 'click',
                        points: points,
                        timestamp: Date.now()
                    })
                });

                var result = await response.json();

                if (result.success) {
                    this.onSuccess();
                } else {
                    this.onError(result.message || '验证失败');
                }
            } catch (error) {
                this.onError('网络错误，请重试');
            }
        };

        ClickCaptcha.prototype.onSuccess = function() {
            this.isVerified = true;

            var markers = this.clickGrid.querySelectorAll('.captcha-click-marker');
            markers.forEach(function(marker) {
                marker.classList.add('success-marker');
            });

            if (this.options.onSuccess) {
                this.options.onSuccess({
                    type: 'click',
                    points: this.selectedPoints
                });
            }
        };

        ClickCaptcha.prototype.onError = function(message) {
            this.attempts++;

            var markers = this.clickGrid.querySelectorAll('.captcha-click-marker');
            markers.forEach(function(marker) {
                marker.classList.add('error-marker');
            });

            setTimeout(function() {
                this.clearMarkers();
                this.selectedPoints = [];
                this.updateProgress();
                this.resetMarkers();
            }.bind(this), 500);

            if (this.options.onError) {
                this.options.onError({
                    type: 'click',
                    message: message,
                    attempts: this.attempts
                });
            }
        };

        ClickCaptcha.prototype.clearMarkers = function() {
            var markers = this.clickGrid.querySelectorAll('.captcha-click-marker');
            markers.forEach(function(marker) {
                marker.remove();
            });
        };

        ClickCaptcha.prototype.resetMarkers = function() {
            var badge = this.container.querySelector('.count-badge');
            if (badge) {
                badge.textContent = this.requiredClicks;
                badge.classList.add('partial');
                badge.classList.remove('complete');
            }
        };

        ClickCaptcha.prototype.reset = function() {
            CaptchaBase.prototype.reset.call(this);
            this.selectedPoints = [];
            this.clearMarkers();
            this.resetMarkers();
            this.updateProgress();
        };

        /**
         * 语音验证码类
         */
        function VoiceCaptcha(options) {
            CaptchaBase.call(this, options);
            this.audioElement = null;
            this.isPlaying = false;
        }

        VoiceCaptcha.prototype = Object.create(CaptchaBase.prototype);
        VoiceCaptcha.prototype.constructor = VoiceCaptcha;

        VoiceCaptcha.prototype.render = function() {
            this.container.innerHTML = this.getTemplate();
            this.playButton = this.container.querySelector('.voice-play-button');
            this.audioElement = this.container.querySelector('audio');
        };

        VoiceCaptcha.prototype.getTemplate = function() {
            return '<div class="captcha-voice-hint">' +
                '<i class="fas fa-volume-up"></i>' +
                '<span>点击播放语音验证码</span>' +
                '</div>' +
                '<button class="captcha-btn captcha-btn-primary voice-play-button">' +
                '<i class="fas fa-play"></i> 播放语音' +
                '</button>' +
                '<audio style="display:none"></audio>';
        };

        VoiceCaptcha.prototype.bindEvents = function() {
            var self = this;

            if (this.playButton) {
                this.playButton.addEventListener('click', function() {
                    self.togglePlay();
                });
            }

            if (this.audioElement) {
                this.audioElement.addEventListener('ended', function() {
                    self.onAudioEnded();
                });
            }
        };

        VoiceCaptcha.prototype.togglePlay = function() {
            if (this.isPlaying) {
                this.stopPlay();
            } else {
                this.startPlay();
            }
        };

        VoiceCaptcha.prototype.startPlay = function() {
            this.isPlaying = true;
            this.playButton.innerHTML = '<i class="fas fa-stop"></i> 停止播放';

            if (this.audioElement && this.audioElement.src) {
                this.audioElement.play();
            } else {
                this.loadAudio();
            }
        };

        VoiceCaptcha.prototype.stopPlay = function() {
            this.isPlaying = false;
            this.playButton.innerHTML = '<i class="fas fa-play"></i> 播放语音';

            if (this.audioElement) {
                this.audioElement.pause();
                this.audioElement.currentTime = 0;
            }
        };

        VoiceCaptcha.prototype.loadAudio = function() {
            var self = this;
            var url = this.options.apiBase + '/captcha/voice?' + Date.now();

            if (this.audioElement) {
                this.audioElement.src = url;
                this.audioElement.play().catch(function(error) {
                    console.error('音频播放失败:', error);
                    self.stopPlay();
                });
            }
        };

        VoiceCaptcha.prototype.onAudioEnded = function() {
            this.isPlaying = false;
            this.playButton.innerHTML = '<i class="fas fa-play"></i> 播放语音';
        };

        VoiceCaptcha.prototype.reset = function() {
            CaptchaBase.prototype.reset.call(this);
            this.stopPlay();
        };

        return {
            version: version,
            CaptchaBase: CaptchaBase,
            SliderCaptcha: SliderCaptcha,
            ClickCaptcha: ClickCaptcha,
            VoiceCaptcha: VoiceCaptcha,

            createSliderCaptcha: function(options) {
                var captcha = new SliderCaptcha(options);
                captcha.init();
                return captcha;
            },

            createClickCaptcha: function(options) {
                var captcha = new ClickCaptcha(options);
                captcha.init();
                return captcha;
            },

            createVoiceCaptcha: function(options) {
                var captcha = new VoiceCaptcha(options);
                captcha.init();
                return captcha;
            },

            create: function(type, options) {
                switch(type) {
                    case 'slider':
                        return this.createSliderCaptcha(options);
                    case 'click':
                        return this.createClickCaptcha(options);
                    case 'voice':
                        return this.createVoiceCaptcha(options);
                    default:
                        throw new Error('不支持的验证码类型: ' + type);
                }
            }
        };
    })();

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaCore;
    } else {
        globalContext.CaptchaCore = CaptchaCore;
    }

})(typeof window !== 'undefined' ? window : this);
