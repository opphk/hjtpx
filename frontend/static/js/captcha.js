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
            onSuccess: null,
            onError: null,
            onRefresh: null,
            onLoadStart: null,
            onLoadEnd: null,
            ...options
        };

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

        this.bindSliderEvents();
        this.bindClickEvents();
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
                           (type === 'click' && content.id === 'click-captcha');
            content.classList.toggle('active', isActive);
            content.hidden = !isActive;
        });

        this.clearResult();
        this.resetSlider();
        this.clearClickPoints();
        this.refresh();
        
        const tabName = type === 'slider' ? this.i18n.t('sliderVerify') : this.i18n.t('clickVerify');
        this.announceToScreenReader(`${this.i18n.t('switchedTo')} ${tabName}`);
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
            } else {
                await this.refreshClick();
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

        const payload = {
            session_id: this.sessionId,
            x: Math.round(this.sliderState.currentX),
            y: this.sliderState.puzzleY,
            type: 'slider',
            behavior_data: this.trajectoryData,
            speed_data: speedData,
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
            this.elements.clickLoadingOverlay;
        
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
            this.elements.clickProgressFill;
        const loadingMessage = type === 'slider' ? 
            this.elements.sliderLoadingMessage : 
            this.elements.clickLoadingMessage;

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
            this.elements.clickLoadingMessage;
        if (loadingMessage) {
            loadingMessage.textContent = message;
        }
    }

    hideLoading(type) {
        this.isLoading = false;
        this.loadingState.isLoading = false;

        const overlay = type === 'slider' ? 
            this.elements.sliderLoadingOverlay : 
            this.elements.clickLoadingOverlay;
        
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
                textSpan.textContent = tab.dataset.type === 'slider' ? 
                    this.i18n.t('sliderVerify') : 
                    this.i18n.t('clickVerify');
            }
        });

        this.resetSlider();
        this.clearClickPoints();
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
                demoClickHint: '请依次点击: 1, 2, 3'
            },
            'en-US': {
                captchaLabel: 'Security captcha component',
                securityVerify: 'Security Verification',
                completeVerify: 'Please complete the verification to continue',
                verifyType: 'Verification type',
                sliderVerify: 'Slider Verification',
                clickVerify: 'Click Verification',
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
                demoClickHint: 'Click: 1, 2, 3 in order'
            },
            'ja-JP': {
                captchaLabel: 'セキュリティキャプチャコンポーネント',
                securityVerify: 'セキュリティ確認',
                completeVerify: '続行するには確認を完了してください',
                verifyType: '確認方法',
                sliderVerify: 'スライダー確認',
                clickVerify: 'クリック確認',
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
                demoClickHint: '順番にクリック: 1, 2, 3'
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

document.addEventListener('DOMContentLoaded', function() {
    window.Captcha = Captcha;
    window.CaptchaI18n = CaptchaI18n;
    window.CaptchaLanguageManager = CaptchaLanguageManager;
});
