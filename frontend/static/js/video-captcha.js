(function() {
    'use strict';

    const VideoCaptcha = {
        video: null,
        canvas: null,
        ctx: null,
        sessionID: null,
        sceneType: null,
        difficulty: 2,
        isPlaying: false,
        isVerified: false,
        startTime: null,
        moveCount: 0,
        keypressData: [],
        currentFrame: 0,
        animationId: null,
        config: {
            width: 640,
            height: 360,
            container: null,
            videoUrl: '',
            onSuccess: null,
            onError: null,
            onProgress: null,
            onFrameUpdate: null,
            autoPlay: true,
            loop: false
        },
        videoFrames: [],
        correctAnswer: null,
        options: [],

        async init(options = {}) {
            Object.assign(this.config, options);
            this.setupVideo();
            this.setupCanvas();
            this.setupEventListeners();
        },

        setupVideo() {
            if (this.config.videoUrl) {
                this.video = document.createElement('video');
                this.video.src = this.config.videoUrl;
                this.video.width = this.config.width;
                this.video.height = this.config.height;
                this.video.style.cssText = `
                    width: ${this.config.width}px;
                    height: ${this.config.height}px;
                    border-radius: 8px;
                    object-fit: cover;
                `;
                this.video.muted = true;
                this.video.playsInline = true;
                this.video.playsInline = true;
                this.video.setAttribute('playsinline', '');
            } else {
                this.canvas = document.createElement('canvas');
                this.canvas.width = this.config.width;
                this.canvas.height = this.config.height;
                this.canvas.style.cssText = `
                    width: ${this.config.width}px;
                    height: ${this.config.height}px;
                    border: 2px solid #007bff;
                    border-radius: 8px;
                    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                `;
                this.ctx = this.canvas.getContext('2d');
            }
        },

        setupCanvas() {
            if (!this.canvas) {
                this.canvas = document.createElement('canvas');
                this.canvas.width = this.config.width;
                this.canvas.height = this.config.height;
                this.canvas.style.cssText = `
                    width: ${this.config.width}px;
                    height: ${this.config.height}px;
                    border: 2px solid #007bff;
                    border-radius: 8px;
                    background: #000;
                `;
                this.ctx = this.canvas.getContext('2d');
            }
        },

        setupEventListeners() {
            if (this.canvas) {
                this.canvas.addEventListener('click', (e) => this.handleCanvasClick(e));
                this.canvas.addEventListener('mousemove', (e) => this.handleMouseMove(e));
                this.canvas.addEventListener('mousedown', (e) => this.handleMouseDown(e));
                this.canvas.addEventListener('mouseup', () => this.handleMouseUp());
            }

            document.addEventListener('keydown', (e) => this.handleKeyDown(e));
            document.addEventListener('keyup', (e) => this.handleKeyUp(e));
        },

        async fetchCaptcha() {
            try {
                const response = await fetch('/api/v1/captcha/video/generate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        width: this.config.width,
                        height: this.config.height,
                        difficulty: this.difficulty
                    })
                });

                const data = await response.json();

                if (data.code === 200 || data.success) {
                    const captchaData = data.data || data;
                    this.sessionID = captchaData.captcha_id;
                    this.sceneType = captchaData.scene_type;
                    this.correctAnswer = captchaData.correct_answer;
                    this.options = captchaData.options || [];
                    this.config.videoUrl = captchaData.video_url;

                    if (this.video && this.config.videoUrl) {
                        this.video.src = this.config.videoUrl;
                    }

                    if (this.config.onProgress) {
                        this.config.onProgress({
                            type: 'loaded',
                            sessionID: this.sessionID,
                            sceneType: this.sceneType
                        });
                    }

                    return captchaData;
                }

                throw new Error('Failed to fetch captcha');
            } catch (error) {
                console.error('Error fetching video captcha:', error);
                throw error;
            }
        },

        async play() {
            if (this.isPlaying) return;

            this.isPlaying = true;
            this.startTime = Date.now();
            this.currentFrame = 0;

            if (this.video) {
                await this.video.play();
                this.animateVideo();
            } else {
                this.generateAnimatedScene();
            }
        },

        pause() {
            this.isPlaying = false;

            if (this.video) {
                this.video.pause();
            }

            if (this.animationId) {
                cancelAnimationFrame(this.animationId);
            }
        },

        animateVideo() {
            if (!this.isPlaying || !this.video) return;

            if (this.ctx && this.video.readyState >= 2) {
                this.ctx.drawImage(this.video, 0, 0, this.config.width, this.config.height);
                this.drawOverlay();
            }

            this.currentFrame++;

            if (this.config.onFrameUpdate) {
                this.config.onFrameUpdate({
                    frame: this.currentFrame,
                    time: this.video.currentTime,
                    duration: this.video.duration
                });
            }

            if (!this.video.ended) {
                this.animationId = requestAnimationFrame(() => this.animateVideo());
            } else {
                this.isPlaying = false;
            }
        },

        generateAnimatedScene() {
            if (!this.isPlaying) return;

            this.ctx.fillStyle = '#1a1a2e';
            this.ctx.fillRect(0, 0, this.config.width, this.config.height);

            this.drawAnimatedObjects();

            if (this.config.onFrameUpdate) {
                this.config.onFrameUpdate({
                    frame: this.currentFrame,
                    time: this.currentFrame / 30,
                    duration: 5
                });
            }

            this.currentFrame++;

            if (this.currentFrame < 150 || this.config.loop) {
                this.animationId = requestAnimationFrame(() => this.generateAnimatedScene());
            } else {
                this.isPlaying = false;
            }
        },

        drawAnimatedObjects() {
            const objects = this.getSceneObjects();

            objects.forEach(obj => {
                const x = obj.x + Math.sin(this.currentFrame / 30 + obj.phase) * 50;
                const y = obj.y + Math.cos(this.currentFrame / 30 + obj.phase) * 30;

                this.ctx.save();
                this.ctx.translate(x, y);
                this.ctx.rotate(this.currentFrame / 60);

                this.ctx.fillStyle = obj.color;
                this.ctx.strokeStyle = obj.color;
                this.ctx.lineWidth = 2;

                switch(obj.shape) {
                    case 'circle':
                        this.ctx.beginPath();
                        this.ctx.arc(0, 0, obj.size, 0, Math.PI * 2);
                        this.ctx.fill();
                        break;
                    case 'square':
                        this.ctx.fillRect(-obj.size, -obj.size, obj.size * 2, obj.size * 2);
                        break;
                    case 'triangle':
                        this.ctx.beginPath();
                        this.ctx.moveTo(0, -obj.size);
                        this.ctx.lineTo(obj.size, obj.size);
                        this.ctx.lineTo(-obj.size, obj.size);
                        this.ctx.closePath();
                        this.ctx.fill();
                        break;
                }

                this.ctx.restore();
            });

            this.drawProgressBar();
        },

        getSceneObjects() {
            const objects = [];
            const count = 3 + this.difficulty;
            const colors = ['#ff6b6b', '#4ecdc4', '#ffe66d', '#95e1d3', '#f38181'];

            for (let i = 0; i < count; i++) {
                objects.push({
                    x: 100 + (this.config.width - 200) * (i / count),
                    y: 100 + (this.config.height - 200) * (0.5 + 0.5 * Math.sin(i)),
                    size: 20 + this.difficulty * 5,
                    color: colors[i % colors.length],
                    shape: ['circle', 'square', 'triangle'][i % 3],
                    phase: i * 0.5
                });
            }

            return objects;
        },

        drawProgressBar() {
            const progress = Math.min(this.currentFrame / 150, 1);
            const barWidth = 200;
            const barHeight = 10;
            const x = (this.config.width - barWidth) / 2;
            const y = this.config.height - 30;

            this.ctx.fillStyle = 'rgba(255, 255, 255, 0.3)';
            this.ctx.fillRect(x, y, barWidth, barHeight);

            this.ctx.fillStyle = '#4ecdc4';
            this.ctx.fillRect(x, y, barWidth * progress, barHeight);

            this.ctx.fillStyle = '#fff';
            this.ctx.font = '14px Arial';
            this.ctx.textAlign = 'center';
            this.ctx.fillText(
                `播放进度: ${Math.round(progress * 100)}%`,
                this.config.width / 2,
                y - 5
            );
        },

        drawOverlay() {
            const progress = this.video ? this.video.currentTime / this.video.duration : 0;

            this.ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
            this.ctx.fillRect(10, 10, 150, 60);

            this.ctx.fillStyle = '#fff';
            this.ctx.font = '14px Arial';
            this.ctx.fillText(`时间: ${this.video ? Math.round(this.video.currentTime) : 0}s`, 20, 30);
            this.ctx.fillText(`帧数: ${this.currentFrame}`, 20, 50);
        },

        handleCanvasClick(event) {
            this.moveCount++;

            const rect = this.canvas.getBoundingClientRect();
            const x = event.clientX - rect.left;
            const y = event.clientY - rect.top;

            this.recordClick(x, y);

            if (this.config.onProgress) {
                this.config.onProgress({
                    type: 'click',
                    x: x,
                    y: y,
                    moveCount: this.moveCount
                });
            }
        },

        handleMouseMove(event) {
            if (event.buttons === 1) {
                this.moveCount++;

                const rect = this.canvas.getBoundingClientRect();
                const x = event.clientX - rect.left;
                const y = event.clientY - rect.top;

                this.recordClick(x, y);
            }
        },

        handleMouseDown(event) {
            this.mouseDownTime = Date.now();
        },

        handleMouseUp() {
            const duration = Date.now() - this.mouseDownTime;
            if (duration < 100) {
                this.shortClick = true;
            }
        },

        handleKeyDown(event) {
            this.keypressData.push({
                key: event.key,
                time: Date.now()
            });
        },

        handleKeyUp(event) {
            if (this.keypressData.length > 0) {
                const lastPress = this.keypressData[this.keypressData.length - 1];
                if (lastPress.key === event.key) {
                    lastPress.duration = Date.now() - lastPress.time;
                }
            }
        },

        recordClick(x, y) {
            if (!this.clickData) {
                this.clickData = [];
            }

            this.clickData.push({
                x: x,
                y: y,
                time: Date.now()
            });
        },

        async verify(answer) {
            if (this.isVerified) {
                return { success: false, message: 'Already verified' };
            }

            const timeSpent = (Date.now() - this.startTime) / 1000;
            const result = await this.sendVerification(answer, timeSpent);

            if (result.success) {
                this.isVerified = true;
                this.showSuccess(result.score);
            } else {
                this.showError(result.message, result.hint);
            }

            return result;
        },

        async sendVerification(answer, timeSpent) {
            const behaviorData = {
                mouse_move_count: this.moveCount,
                time_spent: timeSpent,
                click_data: this.clickData || [],
                keystrokes: this.keypressData.length,
                total_frames: this.currentFrame
            };

            try {
                const response = await fetch('/api/v1/captcha/video/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        captcha_id: this.sessionID,
                        answer: answer,
                        behavior_data: behaviorData
                    })
                });

                const result = await response.json();

                if (result.code === 200 || result.success) {
                    return {
                        success: result.data?.success ?? true,
                        score: result.data?.score ?? 0.9,
                        message: result.data?.message ?? '验证成功',
                        hint: result.data?.hint
                    };
                }

                return {
                    success: false,
                    message: result.message || '验证失败',
                    hint: result.hint
                };
            } catch (error) {
                console.error('Verification request failed:', error);
                return {
                    success: false,
                    message: '验证请求失败',
                    hint: '请重试'
                };
            }
        },

        showSuccess(score) {
            if (this.ctx) {
                this.ctx.fillStyle = 'rgba(0, 255, 0, 0.3)';
                this.ctx.fillRect(0, 0, this.config.width, this.config.height);

                this.ctx.fillStyle = '#fff';
                this.ctx.font = 'bold 32px Arial';
                this.ctx.textAlign = 'center';
                this.ctx.fillText('✓ 验证成功', this.config.width / 2, this.config.height / 2);
                this.ctx.font = '18px Arial';
                this.ctx.fillText(`得分: ${Math.round(score * 100)}%`, this.config.width / 2, this.config.height / 2 + 40);
            }

            if (this.config.onSuccess) {
                this.config.onSuccess({
                    score: score,
                    message: '验证成功'
                });
            }
        },

        showError(message, hint) {
            if (this.ctx) {
                this.ctx.fillStyle = 'rgba(255, 0, 0, 0.3)';
                this.ctx.fillRect(0, 0, this.config.width, this.config.height);

                this.ctx.fillStyle = '#fff';
                this.ctx.font = 'bold 32px Arial';
                this.ctx.textAlign = 'center';
                this.ctx.fillText('✗ ' + message, this.config.width / 2, this.config.height / 2);

                if (hint) {
                    this.ctx.font = '16px Arial';
                    this.ctx.fillText(hint, this.config.width / 2, this.config.height / 2 + 40);
                }
            }

            if (this.config.onError) {
                this.config.onError({
                    message: message,
                    hint: hint,
                    canRetry: true
                });
            }
        },

        getQuestion() {
            return {
                question: this.currentQuestion || '请仔细观看视频并回答问题',
                options: this.options,
                sessionID: this.sessionID
            };
        },

        reset() {
            this.isVerified = false;
            this.isPlaying = false;
            this.startTime = null;
            this.moveCount = 0;
            this.currentFrame = 0;
            this.clickData = [];
            this.keypressData = [];
            this.shortClick = false;

            if (this.animationId) {
                cancelAnimationFrame(this.animationId);
            }

            if (this.ctx) {
                this.ctx.clearRect(0, 0, this.config.width, this.config.height);
            }

            if (this.video) {
                this.video.currentTime = 0;
            }
        },

        destroy() {
            this.pause();
            this.reset();

            if (this.video && this.video.parentElement) {
                this.video.parentElement.removeChild(this.video);
            }

            if (this.canvas && this.canvas.parentElement) {
                this.canvas.parentElement.removeChild(this.canvas);
            }

            document.removeEventListener('keydown', this.handleKeyDown);
            document.removeEventListener('keyup', this.handleKeyUp);
        },

        createOptionButtons(container) {
            const optionsContainer = document.createElement('div');
            optionsContainer.style.cssText = `
                display: flex;
                flex-wrap: wrap;
                gap: 10px;
                justify-content: center;
                margin-top: 20px;
            `;

            this.options.forEach((option, index) => {
                const button = document.createElement('button');
                button.textContent = option;
                button.style.cssText = `
                    padding: 10px 20px;
                    font-size: 16px;
                    border: 2px solid #007bff;
                    border-radius: 5px;
                    background: #fff;
                    color: #007bff;
                    cursor: pointer;
                    transition: all 0.3s;
                `;

                button.addEventListener('click', () => {
                    this.handleOptionClick(option, button);
                });

                button.addEventListener('mouseenter', () => {
                    button.style.background = '#007bff';
                    button.style.color = '#fff';
                });

                button.addEventListener('mouseleave', () => {
                    button.style.background = '#fff';
                    button.style.color = '#007bff';
                });

                optionsContainer.appendChild(button);
            });

            container.appendChild(optionsContainer);
            return optionsContainer;
        },

        handleOptionClick(option, button) {
            button.style.background = '#007bff';
            button.style.color = '#fff';

            setTimeout(() => {
                this.verify(option).then(result => {
                    if (!result.success) {
                        button.style.background = '#fff';
                        button.style.color = '#007bff';
                    }
                });
            }, 100);
        },

        async start() {
            await this.fetchCaptcha();
            this.play();

            return {
                sessionID: this.sessionID,
                question: this.getQuestion(),
                canvas: this.canvas || this.video
            };
        }
    };

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = VideoCaptcha;
    } else {
        window.VideoCaptcha = VideoCaptcha;
    }
})();
