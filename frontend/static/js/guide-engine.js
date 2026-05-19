class GuideEngine {
    constructor(config = {}) {
        this.config = {
            guideId: null,
            userId: null,
            apiEndpoint: '/api/v1/guide',
            autoStart: false,
            showProgress: true,
            allowSkip: true,
            allowSkipAll: true,
            saveProgress: true,
            storageKey: 'hjtpx_guide_progress',
            onStepChange: null,
            onComplete: null,
            onSkip: null,
            onError: null,
            ...config
        };

        this.session = null;
        this.currentStep = null;
        this.currentStepIndex = 0;
        this.steps = [];
        this.overlay = null;
        this.tooltip = null;
        this.progressBar = null;
        this.initialized = false;
    }

    async init(guideId, userId) {
        if (this.initialized) {
            this.destroy();
        }

        this.config.guideId = guideId;
        this.config.userId = userId;

        try {
            const response = await fetch(`${this.config.apiEndpoint}/session`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    guide_id: guideId,
                    user_id: userId
                })
            });

            if (!response.ok) {
                throw new Error('Failed to start guide session');
            }

            const data = await response.json();
            this.session = data.session;
            this.steps = data.steps || [];

            this.loadProgress();
            this.createOverlay();
            this.createTooltip();

            if (this.config.showProgress) {
                this.createProgressBar();
            }

            this.initialized = true;

            if (this.config.autoStart) {
                await this.start();
            }

            return true;
        } catch (error) {
            console.error('Guide initialization failed:', error);
            if (this.config.onError) {
                this.config.onError(error);
            }
            return false;
        }
    }

    async start() {
        if (!this.session || this.steps.length === 0) {
            console.warn('No steps available');
            return;
        }

        this.showStep(this.currentStepIndex);
        this.updateProgress();
    }

    async startGuide(guideId) {
        this.config.guideId = guideId;

        try {
            const response = await fetch(`${this.config.apiEndpoint}/personalized`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    guide_id: guideId,
                    context: this.collectContext()
                })
            });

            const data = await response.json();
            const personalizedGuide = data.guide_id || guideId;

            return await this.init(personalizedGuide, this.config.userId);
        } catch (error) {
            console.error('Failed to get personalized guide:', error);
            return await this.init(guideId, this.config.userId);
        }
    }

    collectContext() {
        return {
            device: this.detectDevice(),
            browser: this.detectBrowser(),
            os: this.detectOS(),
            screen_size: `${window.innerWidth}x${window.innerHeight}`,
            language: navigator.language || 'en',
            experience: 'intermediate',
            success_rate: this.getSuccessRate(),
            total_attempts: this.getTotalAttempts(),
            failed_attempts: this.getFailedAttempts(),
            time_spent: this.getTimeSpent(),
            verification_type: this.detectVerificationType()
        };
    }

    detectDevice() {
        const ua = navigator.userAgent.toLowerCase();
        if (/(tablet|ipad|playbook|silk)/.test(ua)) {
            return 'tablet';
        }
        if (/mobile/.test(ua)) {
            return 'mobile';
        }
        return 'desktop';
    }

    detectBrowser() {
        const ua = navigator.userAgent;
        if (ua.indexOf('Firefox') > -1) return 'Firefox';
        if (ua.indexOf('Chrome') > -1) return 'Chrome';
        if (ua.indexOf('Safari') > -1) return 'Safari';
        if (ua.indexOf('MSIE') > -1 || ua.indexOf('Trident/') > -1) return 'IE';
        return 'Unknown';
    }

    detectOS() {
        const ua = navigator.userAgent;
        if (ua.indexOf('Win') > -1) return 'Windows';
        if (ua.indexOf('Mac') > -1) return 'MacOS';
        if (ua.indexOf('Linux') > -1) return 'Linux';
        if (ua.indexOf('Android') > -1) return 'Android';
        if (ua.indexOf('iOS') > -1) return 'iOS';
        return 'Unknown';
    }

    detectVerificationType() {
        const captchaContainer = document.querySelector('.captcha-container');
        if (!captchaContainer) return 'unknown';

        if (captchaContainer.classList.contains('slider')) return 'slider';
        if (captchaContainer.classList.contains('click')) return 'click';
        if (captchaContainer.classList.contains('voice')) return 'voice';
        return 'unknown';
    }

    getSuccessRate() {
        const stored = localStorage.getItem('hjtpx_verification_stats');
        if (!stored) return 0;

        try {
            const stats = JSON.parse(stored);
            if (stats.total === 0) return 0;
            return stats.success / stats.total;
        } catch {
            return 0;
        }
    }

    getTotalAttempts() {
        const stored = localStorage.getItem('hjtpx_verification_stats');
        if (!stored) return 0;

        try {
            const stats = JSON.parse(stored);
            return stats.total || 0;
        } catch {
            return 0;
        }
    }

    getFailedAttempts() {
        const stored = localStorage.getItem('hjtpx_verification_stats');
        if (!stored) return 0;

        try {
            const stats = JSON.parse(stored);
            return (stats.total || 0) - (stats.success || 0);
        } catch {
            return 0;
        }
    }

    getTimeSpent() {
        const stored = localStorage.getItem('hjtpx_guide_time');
        if (!stored) return 0;

        try {
            const startTime = parseInt(stored, 10);
            return Date.now() - startTime;
        } catch {
            return 0;
        }
    }

    showStep(index) {
        if (index < 0 || index >= this.steps.length) {
            this.completeGuide();
            return;
        }

        this.currentStepIndex = index;
        this.currentStep = this.steps[index];

        const step = this.currentStep;

        if (step.type === 'welcome' || step.type === 'success' || step.type === 'warning') {
            this.showCenteredTooltip(step);
        } else if (step.target) {
            this.showTargetedTooltip(step);
        } else {
            this.showCenteredTooltip(step);
        }

        this.highlightTarget(step);
        this.setupStepActions(step);

        if (this.config.onStepChange) {
            this.config.onStepChange(step, index);
        }
    }

    showCenteredTooltip(step) {
        const tooltip = this.tooltip;

        tooltip.innerHTML = `
            <div class="guide-tooltip-header">
                <h3>${step.title || ''}</h3>
                ${this.config.allowSkip ? '<button class="guide-skip-btn" aria-label="Skip">&times;</button>' : ''}
            </div>
            <div class="guide-tooltip-content">
                <p>${step.description || ''}</p>
            </div>
            <div class="guide-tooltip-footer">
                <span class="guide-step-indicator">${this.currentStepIndex + 1} / ${this.steps.length}</span>
                ${this.currentStepIndex < this.steps.length - 1
                    ? '<button class="guide-next-btn">下一步</button>'
                    : '<button class="guide-finish-btn">完成</button>'}
            </div>
        `;

        tooltip.className = `guide-tooltip guide-tooltip-center guide-tooltip-${step.type || 'default'}`;
        tooltip.style.top = '50%';
        tooltip.style.left = '50%';
        tooltip.style.transform = 'translate(-50%, -50%)';
        tooltip.style.display = 'block';

        this.bindTooltipEvents(tooltip);
    }

    showTargetedTooltip(step) {
        const target = document.querySelector(step.target);
        if (!target) {
            console.warn('Target element not found:', step.target);
            return;
        }

        const tooltip = this.tooltip;
        const rect = target.getBoundingClientRect();
        const position = step.position || 'bottom';

        tooltip.innerHTML = `
            <div class="guide-tooltip-header">
                <h3>${step.title || ''}</h3>
                ${this.config.allowSkip ? '<button class="guide-skip-btn" aria-label="Skip">&times;</button>' : ''}
            </div>
            <div class="guide-tooltip-content">
                <p>${step.description || ''}</p>
            </div>
            <div class="guide-tooltip-footer">
                <span class="guide-step-indicator">${this.currentStepIndex + 1} / ${this.steps.length}</span>
                ${this.currentStepIndex < this.steps.length - 1
                    ? '<button class="guide-next-btn">下一步</button>'
                    : '<button class="guide-finish-btn">完成</button>'}
                ${this.config.allowSkipAll && this.currentStepIndex > 0
                    ? '<button class="guide-skip-all-btn">跳过全部</button>'
                    : ''}
            </div>
        `;

        tooltip.className = `guide-tooltip guide-tooltip-${position} guide-tooltip-${step.type || 'default'}`;

        let top, left;
        const tooltipWidth = 300;
        const tooltipHeight = 200;

        switch (position) {
            case 'top':
                top = rect.top - tooltipHeight - 10;
                left = rect.left + rect.width / 2 - tooltipWidth / 2;
                break;
            case 'bottom':
                top = rect.bottom + 10;
                left = rect.left + rect.width / 2 - tooltipWidth / 2;
                break;
            case 'left':
                top = rect.top + rect.height / 2 - tooltipHeight / 2;
                left = rect.left - tooltipWidth - 10;
                break;
            case 'right':
                top = rect.top + rect.height / 2 - tooltipHeight / 2;
                left = rect.right + 10;
                break;
            default:
                top = rect.bottom + 10;
                left = rect.left;
        }

        tooltip.style.top = `${top}px`;
        tooltip.style.left = `${left}px`;
        tooltip.style.display = 'block';

        this.bindTooltipEvents(tooltip);
    }

    bindTooltipEvents(tooltip) {
        const nextBtn = tooltip.querySelector('.guide-next-btn');
        if (nextBtn) {
            nextBtn.addEventListener('click', () => this.nextStep());
        }

        const finishBtn = tooltip.querySelector('.guide-finish-btn');
        if (finishBtn) {
            finishBtn.addEventListener('click', () => this.completeGuide());
        }

        const skipBtn = tooltip.querySelector('.guide-skip-btn');
        if (skipBtn) {
            skipBtn.addEventListener('click', () => this.skipStep());
        }

        const skipAllBtn = tooltip.querySelector('.guide-skip-all-btn');
        if (skipAllBtn) {
            skipAllBtn.addEventListener('click', () => this.skipAllSteps());
        }
    }

    highlightTarget(step) {
        if (!step.target) return;

        const target = document.querySelector(step.target);
        if (target) {
            target.classList.add('guide-highlight');
        }
    }

    setupStepActions(step) {
        if (!step.actions) return;

        step.actions.forEach(action => {
            const elements = document.querySelectorAll(action.selector);
            elements.forEach(el => {
                switch (action.type) {
                    case 'pulse':
                        el.classList.add('guide-pulse');
                        break;
                    case 'highlight':
                        el.classList.add('guide-highlight-action');
                        break;
                    case 'enable':
                        el.removeAttribute('disabled');
                        el.classList.remove('disabled');
                        break;
                }
            });
        });
    }

    async nextStep() {
        await this.completeCurrentStep();

        this.clearHighlights();
        this.currentStepIndex++;

        if (this.currentStepIndex >= this.steps.length) {
            this.completeGuide();
        } else {
            this.showStep(this.currentStepIndex);
            this.updateProgress();
        }

        this.saveProgress();
    }

    async skipStep() {
        const stepIndex = this.currentStepIndex;
        const reason = 'user_skip';

        try {
            await fetch(`${this.config.apiEndpoint}/session/${this.session.id}/skip`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    step_index: stepIndex,
                    reason: reason
                })
            });
        } catch (error) {
            console.error('Failed to report skip:', error);
        }

        this.clearHighlights();

        if (this.config.onSkip) {
            this.config.onSkip(this.steps[stepIndex], stepIndex);
        }

        this.currentStepIndex++;

        if (this.currentStepIndex >= this.steps.length) {
            this.completeGuide();
        } else {
            this.showStep(this.currentStepIndex);
            this.updateProgress();
        }

        this.saveProgress();
    }

    async skipAllSteps() {
        try {
            await fetch(`${this.config.apiEndpoint}/session/${this.session.id}/complete`, {
                method: 'POST'
            });
        } catch (error) {
            console.error('Failed to complete guide:', error);
        }

        this.clearHighlights();
        this.hide();
        this.saveProgress();

        if (this.config.onComplete) {
            this.config.onComplete();
        }
    }

    async completeCurrentStep() {
        try {
            await fetch(`${this.config.apiEndpoint}/session/${this.session.id}/step`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    step_index: this.currentStepIndex
                })
            });
        } catch (error) {
            console.error('Failed to report step completion:', error);
        }
    }

    async completeGuide() {
        try {
            await fetch(`${this.config.apiEndpoint}/session/${this.session.id}/complete`, {
                method: 'POST'
            });
        } catch (error) {
            console.error('Failed to complete guide:', error);
        }

        this.clearHighlights();
        this.hide();
        this.saveProgress();

        if (this.config.onComplete) {
            this.config.onComplete();
        }
    }

    clearHighlights() {
        document.querySelectorAll('.guide-highlight').forEach(el => {
            el.classList.remove('guide-highlight');
        });

        document.querySelectorAll('.guide-pulse').forEach(el => {
            el.classList.remove('guide-pulse');
        });

        document.querySelectorAll('.guide-highlight-action').forEach(el => {
            el.classList.remove('guide-highlight-action');
        });
    }

    createOverlay() {
        if (this.overlay) {
            this.overlay.remove();
        }

        this.overlay = document.createElement('div');
        this.overlay.className = 'guide-overlay';
        this.overlay.innerHTML = '<div class="guide-overlay-hole"></div>';
        document.body.appendChild(this.overlay);
    }

    createTooltip() {
        if (this.tooltip) {
            this.tooltip.remove();
        }

        this.tooltip = document.createElement('div');
        this.tooltip.className = 'guide-tooltip';
        this.tooltip.setAttribute('role', 'dialog');
        this.tooltip.setAttribute('aria-live', 'polite');
        document.body.appendChild(this.tooltip);
    }

    createProgressBar() {
        if (this.progressBar) {
            this.progressBar.remove();
        }

        this.progressBar = document.createElement('div');
        this.progressBar.className = 'guide-progress-bar';
        this.progressBar.innerHTML = `
            <div class="guide-progress-track">
                <div class="guide-progress-fill" style="width: 0%"></div>
            </div>
            <div class="guide-progress-text">0 / ${this.steps.length}</div>
        `;
        document.body.appendChild(this.progressBar);
    }

    updateProgress() {
        if (!this.config.showProgress || !this.progressBar) return;

        const fill = this.progressBar.querySelector('.guide-progress-fill');
        const text = this.progressBar.querySelector('.guide-progress-text');

        const percentage = ((this.currentStepIndex) / this.steps.length) * 100;
        fill.style.width = `${percentage}%`;
        text.textContent = `${this.currentStepIndex + 1} / ${this.steps.length}`;
    }

    saveProgress() {
        if (!this.config.saveProgress) return;

        const progress = {
            sessionId: this.session?.id,
            guideId: this.config.guideId,
            currentStep: this.currentStepIndex,
            timestamp: Date.now()
        };

        localStorage.setItem(this.config.storageKey, JSON.stringify(progress));
    }

    loadProgress() {
        if (!this.config.saveProgress) return;

        const stored = localStorage.getItem(this.config.storageKey);
        if (!stored) return;

        try {
            const progress = JSON.parse(stored);

            if (progress.sessionId === this.session?.id) {
                const timeDiff = Date.now() - progress.timestamp;
                if (timeDiff < 24 * 60 * 60 * 1000) {
                    this.currentStepIndex = progress.currentStep || 0;
                }
            }
        } catch (error) {
            console.error('Failed to load progress:', error);
        }
    }

    hide() {
        if (this.overlay) {
            this.overlay.style.display = 'none';
        }
        if (this.tooltip) {
            this.tooltip.style.display = 'none';
        }
    }

    show() {
        if (this.overlay) {
            this.overlay.style.display = 'block';
        }
        if (this.tooltip && this.currentStep) {
            this.tooltip.style.display = 'block';
        }
    }

    destroy() {
        this.clearHighlights();

        if (this.overlay) {
            this.overlay.remove();
            this.overlay = null;
        }

        if (this.tooltip) {
            this.tooltip.remove();
            this.tooltip = null;
        }

        if (this.progressBar) {
            this.progressBar.remove();
            this.progressBar = null;
        }

        this.session = null;
        this.currentStep = null;
        this.steps = [];
        this.currentStepIndex = 0;
        this.initialized = false;

        if (this.config.saveProgress) {
            localStorage.removeItem(this.config.storageKey);
        }
    }
}

window.GuideEngine = GuideEngine;
