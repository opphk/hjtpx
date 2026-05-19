(function(global) {
    'use strict';

    var VideoCaptcha = function(containerId, options) {
        this.container = typeof containerId === 'string' ? 
            document.getElementById(containerId) : containerId;
        
        this.options = Object.assign({
            apiBase: '/api/v1',
            width: 640,
            height: 360,
            difficulty: 2,
            autoPlay: true,
            showControls: true,
            onSuccess: function() {},
            onError: function() {},
            onRefresh: function() {},
            language: 'zh-CN'
        }, options || {});

        this.sessionId = null;
        this.currentVideo = null;
        this.state = {
            isLoading: false,
            hasVideo: false,
            viewCount: 0,
            replayCount: 0,
            startTime: 0,
            answerTime: 0,
            selectedAction: null
        };

        this.translations = {
            'zh-CN': {
                title: '视频动作验证',
                instruction: '请观看视频并选择正确的动作',
                loading: '加载中...',
                play: '播放视频',
                replay: '重播',
                submit: '提交答案',
                refresh: '刷新验证码',
                correct: '验证成功！',
                incorrect: '答案错误，请重试',
                expired: '验证码已过期',
                viewVideo: '请先观看视频',
                selectAction: '请选择动作'
            },
            'en-US': {
                title: 'Video Action Verification',
                instruction: 'Watch the video and select the correct action',
                loading: 'Loading...',
                play: 'Play Video',
                replay: 'Replay',
                submit: 'Submit Answer',
                refresh: 'Refresh',
                correct: 'Verification successful!',
                incorrect: 'Incorrect answer, please try again',
                expired: 'Captcha expired',
                viewVideo: 'Please watch the video first',
                selectAction: 'Please select an action'
            }
        };

        this.init();
    };

    VideoCaptcha.prototype.init = function() {
        this.render();
        this.bindEvents();
        this.generate();
    };

    VideoCaptcha.prototype.render = function() {
        var lang = this.translations[this.options.language] || this.translations['zh-CN'];
        var html = '<div class="video-captcha-container">' +
            '<div class="video-captcha-header">' +
                '<h5 class="video-captcha-title">' + lang.title + '</h5>' +
            '</div>' +
            '<div class="video-captcha-body">' +
                '<div class="video-captcha-loading" id="videoCaptchaLoading" style="display:none;">' +
                    '<div class="spinner-border text-primary" role="status"></div>' +
                    '<span class="ms-2">' + lang.loading + '</span>' +
                '</div>' +
                '<div class="video-captcha-player" id="videoCaptchaPlayer" style="display:none;">' +
                    '<video id="videoElement" width="' + this.options.width + '" height="' + this.options.height + '" controls="' + this.options.showControls + '">' +
                    '</video>' +
                '</div>' +
                '<div class="video-captcha-question" id="videoCaptchaQuestion" style="display:none;">' +
                    '<p class="video-captcha-instruction">' + lang.instruction + '</p>' +
                    '<p class="video-captcha-question-text" id="questionText"></p>' +
                '</div>' +
                '<div class="video-captcha-options" id="videoCaptchaOptions" style="display:none;">' +
                '</div>' +
            '</div>' +
            '<div class="video-captcha-footer">' +
                '<button type="button" class="btn btn-outline-secondary btn-sm me-2" id="videoCaptchaReplay" disabled>' +
                    '<i class="fas fa-redo me-1"></i>' + lang.replay +
                '</button>' +
                '<button type="button" class="btn btn-primary" id="videoCaptchaSubmit" disabled>' +
                    lang.submit +
                '</button>' +
                '<button type="button" class="btn btn-link btn-sm ms-auto" id="videoCaptchaRefresh">' +
                    '<i class="fas fa-sync-alt me-1"></i>' + lang.refresh +
                '</button>' +
            '</div>' +
        '</div>';

        this.container.innerHTML = html;
        this.cacheElements();
    };

    VideoCaptcha.prototype.cacheElements = function() {
        this.elements = {
            loading: document.getElementById('videoCaptchaLoading'),
            player: document.getElementById('videoCaptchaPlayer'),
            video: document.getElementById('videoElement'),
            question: document.getElementById('videoCaptchaQuestion'),
            questionText: document.getElementById('questionText'),
            options: document.getElementById('videoCaptchaOptions'),
            replay: document.getElementById('videoCaptchaReplay'),
            submit: document.getElementById('videoCaptchaSubmit'),
            refresh: document.getElementById('videoCaptchaRefresh')
        };
    };

    VideoCaptcha.prototype.bindEvents = function() {
        var self = this;

        if (this.elements.replay) {
            this.elements.replay.addEventListener('click', function() {
                self.replayVideo();
            });
        }

        if (this.elements.submit) {
            this.elements.submit.addEventListener('click', function() {
                self.submitAnswer();
            });
        }

        if (this.elements.refresh) {
            this.elements.refresh.addEventListener('click', function() {
                self.refresh();
            });
        }

        if (this.elements.video) {
            this.elements.video.addEventListener('ended', function() {
                self.onVideoEnded();
            });

            this.elements.video.addEventListener('play', function() {
                self.onVideoPlay();
            });
        }
    };

    VideoCaptcha.prototype.generate = function() {
        var self = this;
        self.showLoading();

        var xhr = new XMLHttpRequest();
        xhr.open('POST', this.options.apiBase + '/captcha/video/generate', true);
        xhr.setRequestHeader('Content-Type', 'application/json');
        
        xhr.onreadystatechange = function() {
            if (xhr.readyState === 4) {
                if (xhr.status === 200) {
                    try {
                        var response = JSON.parse(xhr.responseText);
                        if (response.code === 0 && response.data) {
                            self.onGenerateSuccess(response.data);
                        } else {
                            self.onError({ message: response.message || '生成失败' });
                        }
                    } catch (e) {
                        self.onError({ message: '解析响应失败' });
                    }
                } else {
                    self.onError({ message: '网络错误' });
                }
            }
        };

        xhr.send(JSON.stringify({
            width: this.options.width,
            height: this.options.height,
            difficulty: this.options.difficulty
        }));
    };

    VideoCaptcha.prototype.onGenerateSuccess = function(data) {
        this.sessionId = data.session_id;
        this.currentVideo = data.video_url;
        this.state.question = data.question;
        this.state.options = data.options;
        this.state.targetAction = data.target_action;
        this.state.duration = data.duration;
        this.state.hasVideo = true;

        this.showVideo();
        this.displayQuestion(data.question);
        this.displayOptions(data.options);

        this.state.startTime = Date.now();
    };

    VideoCaptcha.prototype.showLoading = function() {
        this.state.isLoading = true;
        if (this.elements.loading) this.elements.loading.style.display = 'flex';
        if (this.elements.player) this.elements.player.style.display = 'none';
        if (this.elements.question) this.elements.question.style.display = 'none';
        if (this.elements.options) this.elements.options.style.display = 'none';
    };

    VideoCaptcha.prototype.showVideo = function() {
        this.state.isLoading = false;
        if (this.elements.loading) this.elements.loading.style.display = 'none';
        if (this.elements.player) this.elements.player.style.display = 'block';
        if (this.elements.question) this.elements.question.style.display = 'block';
        if (this.elements.options) this.elements.options.style.display = 'block';
        
        if (this.elements.video && this.currentVideo) {
            this.elements.video.src = this.currentVideo;
            if (this.options.autoPlay) {
                this.elements.video.play().catch(function() {});
            }
        }
    };

    VideoCaptcha.prototype.displayQuestion = function(question) {
        if (this.elements.questionText) {
            this.elements.questionText.textContent = question;
        }
    };

    VideoCaptcha.prototype.displayOptions = function(options) {
        if (!this.elements.options || !options) return;

        var self = this;
        var html = '<div class="video-action-options">';
        
        options.forEach(function(option, index) {
            html += '<button type="button" class="video-action-option" data-action="' + option + '">' +
                        '<span class="video-action-icon"><i class="fas fa-hand-paper"></i></span>' +
                        '<span class="video-action-label">' + option + '</span>' +
                    '</button>';
        });
        
        html += '</div>';
        
        this.elements.options.innerHTML = html;

        var optionBtns = this.elements.options.querySelectorAll('.video-action-option');
        optionBtns.forEach(function(btn) {
            btn.addEventListener('click', function() {
                self.selectOption(this);
            });
        });
    };

    VideoCaptcha.prototype.selectOption = function(btn) {
        var optionBtns = this.elements.options.querySelectorAll('.video-action-option');
        optionBtns.forEach(function(b) {
            b.classList.remove('selected');
        });
        
        btn.classList.add('selected');
        this.state.selectedAction = btn.getAttribute('data-action');
        
        if (this.elements.submit) {
            this.elements.submit.disabled = !this.state.viewCount;
        }
    };

    VideoCaptcha.prototype.onVideoEnded = function() {
        this.state.viewCount++;
        this.state.answerTime = Date.now();

        if (this.elements.replay) {
            this.elements.replay.disabled = false;
        }

        if (this.elements.submit) {
            this.elements.submit.disabled = !this.state.selectedAction;
        }
    };

    VideoCaptcha.prototype.onVideoPlay = function() {
        if (this.state.viewCount === 0) {
            this.state.startTime = Date.now();
        }
    };

    VideoCaptcha.prototype.replayVideo = function() {
        if (this.elements.video) {
            this.state.replayCount++;
            this.elements.video.currentTime = 0;
            this.elements.video.play().catch(function() {});
        }
    };

    VideoCaptcha.prototype.submitAnswer = function() {
        var self = this;

        if (!this.state.viewCount) {
            this.onError({ message: this.getTranslation('viewVideo') });
            return;
        }

        if (!this.state.selectedAction) {
            this.onError({ message: this.getTranslation('selectAction') });
            return;
        }

        if (this.elements.submit) {
            this.elements.submit.disabled = true;
        }

        var behaviorData = {
            start_time: this.state.startTime,
            end_time: Date.now(),
            duration: Date.now() - this.state.startTime,
            view_count: this.state.viewCount,
            replay_count: this.state.replayCount,
            answer_time: this.state.answerTime,
            is_mobile: /Android|iPhone|iPad|iPod/i.test(navigator.userAgent),
            network_type: navigator.connection ? navigator.connection.effectiveType : 'unknown'
        };

        var xhr = new XMLHttpRequest();
        xhr.open('POST', this.options.apiBase + '/captcha/video/verify', true);
        xhr.setRequestHeader('Content-Type', 'application/json');
        
        xhr.onreadystatechange = function() {
            if (xhr.readyState === 4) {
                if (xhr.status === 200) {
                    try {
                        var response = JSON.parse(xhr.responseText);
                        if (response.code === 0 && response.data) {
                            if (response.data.success) {
                                self.onSuccess(response.data);
                            } else {
                                self.onError({
                                    message: response.data.message || self.getTranslation('incorrect'),
                                    data: response.data
                                });
                            }
                        } else {
                            self.onError({ message: response.message || '验证失败' });
                        }
                    } catch (e) {
                        self.onError({ message: '解析响应失败' });
                    }
                } else {
                    self.onError({ message: '网络错误' });
                }
                
                if (self.elements.submit) {
                    self.elements.submit.disabled = false;
                }
            }
        };

        xhr.send(JSON.stringify({
            session_id: this.sessionId,
            answer: this.state.selectedAction,
            behavior_data: behaviorData
        }));
    };

    VideoCaptcha.prototype.onSuccess = function(data) {
        if (this.elements.submit) {
            this.elements.submit.classList.remove('btn-primary');
            this.elements.submit.classList.add('btn-success');
            this.elements.submit.textContent = this.getTranslation('correct');
            this.elements.submit.disabled = true;
        }

        if (this.elements.replay) {
            this.elements.replay.disabled = true;
        }

        this.options.onSuccess({
            session_id: this.sessionId,
            score: data.score,
            message: data.message
        });
    };

    VideoCaptcha.prototype.onError = function(error) {
        this.options.onError(error);
    };

    VideoCaptcha.prototype.refresh = function() {
        this.resetState();
        this.options.onRefresh();
        this.generate();
    };

    VideoCaptcha.prototype.resetState = function() {
        this.sessionId = null;
        this.currentVideo = null;
        this.state = {
            isLoading: false,
            hasVideo: false,
            viewCount: 0,
            replayCount: 0,
            startTime: 0,
            answerTime: 0,
            selectedAction: null
        };

        if (this.elements.replay) {
            this.elements.replay.disabled = true;
        }

        if (this.elements.submit) {
            this.elements.submit.disabled = true;
            this.elements.submit.classList.remove('btn-success');
            this.elements.submit.classList.add('btn-primary');
            var lang = this.translations[this.options.language] || this.translations['zh-CN'];
            this.elements.submit.textContent = lang.submit;
        }

        if (this.elements.video) {
            this.elements.video.src = '';
        }
    };

    VideoCaptcha.prototype.getTranslation = function(key) {
        var trans = this.translations[this.options.language] || this.translations['zh-CN'];
        return trans[key] || key;
    };

    VideoCaptcha.prototype.destroy = function() {
        this.container.innerHTML = '';
        this.sessionId = null;
        this.currentVideo = null;
    };

    global.VideoCaptcha = VideoCaptcha;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = VideoCaptcha;
    }

})(typeof window !== 'undefined' ? window : this);
