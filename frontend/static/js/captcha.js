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
            timeout: 60,
            imageCount: 6,
            gridColumns: 3,
            gridRows: 2,
            onSuccess: null,
            onError: null,
            onRefresh: null,
            ...options
        };

        this.sliderState = {
            isDragging: false,
            startX: 0,
            currentX: 0,
            maxX: 0,
            puzzleY: 0
        };

        this.clickState = {
            selectedPoints: [],
            maxPoints: 3,
            hintText: '请依次点击图中的文字',
            images: [],
            currentImageIndex: 0,
            countdownTimer: null,
            countdownRemaining: 60,
            clickHistory: [],
            startTime: null
        };

        this.sessionId = null;
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.refresh();
    }

    render() {
        const gridStyle = `
            grid-template-columns: repeat(${this.options.gridColumns}, 1fr);
            gap: 10px;
        `;

        this.container.innerHTML = `
            <div class="captcha-container">
                <div class="captcha-header">
                    <h3>安全验证</h3>
                    <p>请完成以下验证以继续</p>
                </div>
                <div class="captcha-body">
                    <div class="captcha-tabs">
                        <button class="captcha-tab active" data-type="slider">滑块验证</button>
                        <button class="captcha-tab" data-type="click">点选验证</button>
                    </div>
                    
                    <div class="captcha-content active" id="slider-captcha">
                        <div class="captcha-image-wrapper" id="slider-image-wrapper">
                            <img class="captcha-image" id="slider-image" alt="验证码图片">
                            <div class="captcha-puzzle" id="slider-puzzle"></div>
                            <button class="captcha-refresh" id="slider-refresh">🔄</button>
                            <div class="captcha-loading" id="slider-loading" style="display: none;">
                                <div class="captcha-loading-spinner"></div>
                            </div>
                        </div>
                        <div class="captcha-slider-container" id="slider-container">
                            <div class="captcha-slider-track" id="slider-track"></div>
                            <div class="captcha-slider-text" id="slider-text">向右滑动完成验证</div>
                            <div class="captcha-slider-button" id="slider-button">→</div>
                        </div>
                    </div>
                    
                    <div class="captcha-content" id="click-captcha">
                        <div class="captcha-click-header">
                            <div class="captcha-click-hint" id="click-hint">请依次点击图中的文字</div>
                            <div class="captcha-countdown" id="click-countdown">
                                <span class="countdown-number">60</span>
                                <span class="countdown-text">秒</span>
                            </div>
                        </div>
                        <div class="captcha-click-grid" id="click-grid" style="${gridStyle}">
                        </div>
                        <div class="captcha-click-progress" id="click-progress">
                            <span class="progress-text">已选择: <span id="selected-count">0</span>/<span id="total-count">3</span></span>
                            <div class="progress-bar">
                                <div class="progress-fill" id="progress-fill"></div>
                            </div>
                        </div>
                        <div class="captcha-actions">
                            <button class="captcha-btn captcha-btn-secondary" id="click-undo">撤销</button>
                            <button class="captcha-btn captcha-btn-secondary" id="click-clear">清除</button>
                            <button class="captcha-btn captcha-btn-primary" id="click-submit">确认</button>
                        </div>
                    </div>
                    
                    <div class="captcha-result" id="captcha-result"></div>
                </div>
            </div>
        `;

        this.elements = {
            tabs: this.container.querySelectorAll('.captcha-tab'),
            contents: this.container.querySelectorAll('.captcha-content'),
            sliderImage: this.container.querySelector('#slider-image'),
            sliderPuzzle: this.container.querySelector('#slider-puzzle'),
            sliderContainer: this.container.querySelector('#slider-container'),
            sliderTrack: this.container.querySelector('#slider-track'),
            sliderText: this.container.querySelector('#slider-text'),
            sliderButton: this.container.querySelector('#slider-button'),
            sliderRefresh: this.container.querySelector('#slider-refresh'),
            sliderLoading: this.container.querySelector('#slider-loading'),
            sliderImageWrapper: this.container.querySelector('#slider-image-wrapper'),
            clickHint: this.container.querySelector('#click-hint'),
            clickGrid: this.container.querySelector('#click-grid'),
            clickCountdown: this.container.querySelector('#click-countdown'),
            clickProgress: this.container.querySelector('#click-progress'),
            selectedCount: this.container.querySelector('#selected-count'),
            totalCount: this.container.querySelector('#total-count'),
            progressFill: this.container.querySelector('#progress-fill'),
            clickUndo: this.container.querySelector('#click-undo'),
            clickClear: this.container.querySelector('#click-clear'),
            clickSubmit: this.container.querySelector('#click-submit'),
            result: this.container.querySelector('#captcha-result')
        };
    }

    bindEvents() {
        this.elements.tabs.forEach(tab => {
            tab.addEventListener('click', () => this.switchTab(tab.dataset.type));
        });

        this.elements.sliderRefresh.addEventListener('click', () => this.refresh());
        
        this.elements.clickUndo.addEventListener('click', () => this.undoLastClick());
        this.elements.clickClear.addEventListener('click', () => this.clearClickPoints());
        this.elements.clickSubmit.addEventListener('click', () => this.verifyClick());

        this.bindSliderEvents();
        this.bindClickEvents();
    }

    bindSliderEvents() {
        const button = this.elements.sliderButton;
        const container = this.elements.sliderContainer;

        const startDrag = (e) => {
            if (this.sliderState.isDragging) return;
            
            this.sliderState.isDragging = true;
            const clientX = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX;
            this.sliderState.startX = clientX;
            this.sliderState.currentX = 0;
            this.sliderState.maxX = container.offsetWidth - button.offsetWidth - 4;
            
            button.classList.add('dragging');
        };

        const drag = (e) => {
            if (!this.sliderState.isDragging) return;
            
            e.preventDefault();
            const clientX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX;
            let deltaX = clientX - this.sliderState.startX;
            
            deltaX = Math.max(0, Math.min(deltaX, this.sliderState.maxX));
            this.sliderState.currentX = deltaX;
            
            button.style.left = (deltaX + 2) + 'px';
            this.elements.sliderTrack.style.width = deltaX + 'px';
            
            if (this.sliderState.puzzleY) {
                this.updatePuzzlePosition(deltaX);
            }
        };

        const endDrag = (e) => {
            if (!this.sliderState.isDragging) return;
            
            this.sliderState.isDragging = false;
            button.classList.remove('dragging');
            
            if (this.sliderState.currentX > 10) {
                this.verifySlider();
            }
        };

        button.addEventListener('mousedown', startDrag);
        document.addEventListener('mousemove', drag);
        document.addEventListener('mouseup', endDrag);
        
        button.addEventListener('touchstart', startDrag, { passive: false });
        document.addEventListener('touchmove', drag, { passive: false });
        document.addEventListener('touchend', endDrag);
    }

    bindClickEvents() {
        this.elements.clickGrid.addEventListener('click', (e) => {
            const grid = this.elements.clickGrid;
            
            if (e.target.classList.contains('captcha-click-image')) {
                this.handleImageClick(e, grid);
            }
        });
    }

    handleImageClick(e, grid) {
        if (this.clickState.selectedPoints.length >= this.clickState.maxPoints) {
            this.showResult('已达到最大选择数量', 'error');
            return;
        }

        const rect = grid.getBoundingClientRect();
        const imageRect = e.target.getBoundingClientRect();
        
        const x = e.clientX - imageRect.left;
        const y = e.clientY - imageRect.top;
        
        const point = {
            x: Math.round(x),
            y: Math.round(y),
            imageIndex: parseInt(e.target.dataset.index),
            timestamp: Date.now()
        };
        
        this.clickState.selectedPoints.push(point);
        this.clickState.clickHistory.push(point);
        
        this.addClickMarker(e.target, point, this.clickState.selectedPoints.length);
        this.updateProgress();
        
        this.recordClickBehavior(point);
        
        if (this.clickState.selectedPoints.length === this.clickState.maxPoints) {
            setTimeout(() => {
                this.showResult('已选择全部目标，请确认提交', 'info');
            }, 300);
        }
    }

    recordClickBehavior(point) {
        if (!this.clickState.startTime) {
            this.clickState.startTime = Date.now();
        }
        
        const timeSinceStart = Date.now() - this.clickState.startTime;
        const lastClick = this.clickState.clickHistory.length > 1 
            ? this.clickState.clickHistory[this.clickState.clickHistory.length - 2] 
            : null;
        
        const behaviorData = {
            event: 'click',
            x: point.x,
            y: point.y,
            imageIndex: point.imageIndex,
            timestamp: Date.now(),
            timeSinceStart: timeSinceStart,
            timeSinceLastClick: lastClick ? Date.now() - lastClick.timestamp : 0,
            totalClicks: this.clickState.clickHistory.length
        };
        
        this.options.behaviorData = this.options.behaviorData || [];
        this.options.behaviorData.push(behaviorData);
    }

    addClickMarker(imageElement, point, index) {
        const marker = document.createElement('div');
        marker.className = 'captcha-click-marker';
        marker.style.left = point.x + 'px';
        marker.style.top = point.y + 'px';
        marker.textContent = index;
        marker.dataset.index = index - 1;
        
        const removeBtn = document.createElement('button');
        removeBtn.className = 'captcha-marker-remove';
        removeBtn.textContent = '×';
        removeBtn.onclick = (e) => {
            e.stopPropagation();
            const idx = parseInt(marker.dataset.index);
            this.removeClickPoint(idx);
        };
        
        marker.appendChild(removeBtn);
        marker.addEventListener('click', (e) => {
            e.stopPropagation();
        });
        
        imageElement.parentElement.appendChild(marker);
        
        marker.style.animation = 'clickMarkerPop 0.3s ease-out';
    }

    removeClickPoint(index) {
        this.clickState.selectedPoints.splice(index, 1);
        this.updateClickMarkers();
        this.updateProgress();
    }

    updateClickMarkers() {
        const markers = this.elements.clickGrid.querySelectorAll('.captcha-click-marker');
        markers.forEach(m => m.remove());
        
        this.clickState.selectedPoints.forEach((point, idx) => {
            const imageElement = this.elements.clickGrid.querySelector(
                `[data-index="${point.imageIndex}"]`
            );
            if (imageElement) {
                this.addClickMarker(imageElement, point, idx + 1);
            }
        });
    }

    clearClickPoints() {
        this.clickState.selectedPoints = [];
        const markers = this.elements.clickGrid.querySelectorAll('.captcha-click-marker');
        markers.forEach(m => m.remove());
        this.updateProgress();
    }

    undoLastClick() {
        if (this.clickState.selectedPoints.length > 0) {
            this.clickState.selectedPoints.pop();
            this.updateClickMarkers();
            this.updateProgress();
        }
    }

    updateProgress() {
        const selected = this.clickState.selectedPoints.length;
        const total = this.clickState.maxPoints;
        const percentage = (selected / total) * 100;
        
        this.elements.selectedCount.textContent = selected;
        this.elements.totalCount.textContent = total;
        this.elements.progressFill.style.width = percentage + '%';
        
        if (percentage === 100) {
            this.elements.progressFill.classList.add('complete');
        } else {
            this.elements.progressFill.classList.remove('complete');
        }
    }

    updatePuzzlePosition(x) {
        this.elements.sliderPuzzle.style.left = x + 'px';
    }

    switchTab(type) {
        this.options.type = type;
        
        if (this.clickState.countdownTimer) {
            clearInterval(this.clickState.countdownTimer);
            this.clickState.countdownTimer = null;
        }
        
        this.elements.tabs.forEach(tab => {
            tab.classList.toggle('active', tab.dataset.type === type);
        });
        
        this.elements.contents.forEach(content => {
            content.classList.toggle('active', 
                (type === 'slider' && content.id === 'slider-captcha') ||
                (type === 'click' && content.id === 'click-captcha')
            );
        });
        
        this.clearResult();
        this.resetSlider();
        this.clearClickPoints();
        this.refresh();
    }

    async refresh() {
        this.clearResult();
        this.showLoading();
        
        if (this.options.onRefresh) {
            this.options.onRefresh();
        }
        
        try {
            if (this.options.type === 'slider') {
                await this.refreshSlider();
            } else {
                await this.refreshClick();
            }
        } catch (error) {
            console.error('Refresh failed:', error);
            this.showResult('加载失败，请重试', 'error');
        } finally {
            this.hideLoading();
        }
    }

    async refreshSlider() {
        this.resetSlider();
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/slider`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });
            
            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                this.elements.sliderImage.src = data.image_url;
                
                this.sliderState.puzzleY = data.puzzle_y || 50;
                this.elements.sliderPuzzle.innerHTML = `
                    <div class="captcha-puzzle-piece" style="top: ${this.sliderState.puzzleY}px; left: 0;"></div>
                `;
            } else {
                this.loadDemoSlider();
            }
        } catch (error) {
            this.loadDemoSlider();
        }
    }

    loadDemoSlider() {
        this.sessionId = 'demo_' + Date.now();
        this.elements.sliderImage.src = 'data:image/svg+xml,' + encodeURIComponent(`
            <svg xmlns="http://www.w3.org/2000/svg" width="360" height="220">
                <defs>
                    <linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
                        <stop offset="0%" style="stop-color:#667eea"/>
                        <stop offset="100%" style="stop-color:#764ba2"/>
                    </linearGradient>
                </defs>
                <rect width="100%" height="100%" fill="url(#bg)"/>
                <text x="180" y="110" text-anchor="middle" fill="white" font-size="24" font-family="Arial">
                    滑块验证演示
                </text>
                <rect x="200" y="70" width="50" height="50" fill="rgba(255,255,255,0.3)" stroke="white" stroke-width="2"/>
            </svg>
        `);
        
        this.sliderState.puzzleY = 70;
        this.elements.sliderPuzzle.innerHTML = `
            <div class="captcha-puzzle-piece" style="top: 70px; left: 0;"></div>
        `;
    }

    async refreshClick() {
        this.clearClickPoints();
        
        if (this.clickState.countdownTimer) {
            clearInterval(this.clickState.countdownTimer);
        }
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/click`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });
            
            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                
                this.clickState.hintText = data.hint || '请依次点击图中的文字';
                this.clickState.maxPoints = data.max_points || 3;
                this.elements.clickHint.textContent = this.clickState.hintText;
                
                this.renderClickGrid(data.image_url);
                this.startCountdown();
            } else {
                this.loadDemoClick();
            }
        } catch (error) {
            this.loadDemoClick();
        }
    }

    renderClickGrid(imageUrl) {
        const grid = this.elements.clickGrid;
        grid.innerHTML = '';
        
        const imageCount = this.options.imageCount;
        
        for (let i = 0; i < imageCount; i++) {
            const wrapper = document.createElement('div');
            wrapper.className = 'captcha-image-cell';
            
            const img = document.createElement('img');
            img.className = 'captcha-click-image';
            img.src = imageUrl;
            img.alt = `验证码图片 ${i + 1}`;
            img.dataset.index = i;
            
            const number = document.createElement('div');
            number.className = 'captcha-image-number';
            number.textContent = i + 1;
            
            wrapper.appendChild(img);
            wrapper.appendChild(number);
            grid.appendChild(wrapper);
        }
    }

    startCountdown() {
        this.clickState.countdownRemaining = this.options.timeout;
        const countdownNumber = this.elements.clickCountdown.querySelector('.countdown-number');
        
        if (this.clickState.countdownTimer) {
            clearInterval(this.clickState.countdownTimer);
        }
        
        this.clickState.countdownTimer = setInterval(() => {
            this.clickState.countdownRemaining--;
            countdownNumber.textContent = this.clickState.countdownRemaining;
            
            if (this.clickState.countdownRemaining <= 10) {
                this.elements.clickCountdown.classList.add('warning');
            }
            
            if (this.clickState.countdownRemaining <= 0) {
                clearInterval(this.clickState.countdownTimer);
                this.showResult('验证超时，请重试', 'error');
                setTimeout(() => this.refreshClick(), 1500);
            }
        }, 1000);
    }

    loadDemoClick() {
        this.sessionId = 'demo_' + Date.now();
        this.clickState.hintText = '请依次点击: 1, 2, 3';
        this.clickState.maxPoints = 3;
        this.elements.clickHint.textContent = this.clickState.hintText;
        
        const demoImage = 'data:image/svg+xml,' + encodeURIComponent(`
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
        
        this.renderClickGrid(demoImage);
        this.startCountdown();
    }

    async verifySlider() {
        this.showLoading();
        
        const payload = {
            session_id: this.sessionId,
            x: Math.round(this.sliderState.currentX),
            y: this.sliderState.puzzleY,
            type: 'slider'
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
            } else {
                success = this.simulateSliderVerify();
            }
            
            if (success) {
                this.elements.sliderButton.classList.add('success');
                this.elements.sliderText.textContent = '验证成功!';
                this.showResult('验证成功!', 'success');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ type: 'slider', session_id: this.sessionId });
                }
            } else {
                this.elements.sliderButton.classList.add('error');
                this.showResult('验证失败，请重试', 'error');
                if (this.options.onError) {
                    this.options.onError({ type: 'slider', error: '验证失败' });
                }
                setTimeout(() => this.refresh(), 1500);
            }
        } catch (error) {
            const success = this.simulateSliderVerify();
            if (success) {
                this.elements.sliderButton.classList.add('success');
                this.elements.sliderText.textContent = '验证成功!';
                this.showResult('验证成功!', 'success');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ type: 'slider', session_id: this.sessionId });
                }
            } else {
                this.showResult('验证失败，请重试', 'error');
                setTimeout(() => this.refresh(), 1500);
            }
        } finally {
            this.hideLoading();
        }
    }

    simulateSliderVerify() {
        const targetX = 200;
        const tolerance = 10;
        return Math.abs(this.sliderState.currentX - targetX) <= tolerance;
    }

    async verifyClick() {
        if (this.clickState.selectedPoints.length !== this.clickState.maxPoints) {
            this.showResult(`请选择全部 ${this.clickState.maxPoints} 个目标`, 'error');
            return;
        }
        
        this.showLoading();
        
        const clickData = this.clickState.selectedPoints.map((point, index) => ({
            x: point.x,
            y: point.y,
            imageIndex: point.imageIndex,
            clickOrder: index + 1
        }));
        
        const payload = {
            session_id: this.sessionId,
            points: clickData,
            type: 'click',
            behavior_data: this.options.behaviorData || [],
            verification_time: this.clickState.startTime ? Date.now() - this.clickState.startTime : 0
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
            } else {
                success = this.simulateClickVerify();
            }
            
            if (success) {
                if (this.clickState.countdownTimer) {
                    clearInterval(this.clickState.countdownTimer);
                }
                this.showResult('验证成功!', 'success');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ 
                        type: 'click', 
                        session_id: this.sessionId,
                        click_count: this.clickState.selectedPoints.length,
                        verification_time: payload.verification_time
                    });
                }
            } else {
                this.showResult('验证失败，请重试', 'error');
                if (this.options.onError) {
                    this.options.onError({ type: 'click', error: '验证失败' });
                }
                setTimeout(() => this.refreshClick(), 1500);
            }
        } catch (error) {
            const success = this.simulateClickVerify();
            if (success) {
                this.showResult('验证成功!', 'success');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ 
                        type: 'click', 
                        session_id: this.sessionId 
                    });
                }
            } else {
                this.showResult('验证失败，请重试', 'error');
                setTimeout(() => this.refreshClick(), 1500);
            }
        } finally {
            this.hideLoading();
        }
    }

    simulateClickVerify() {
        return this.clickState.selectedPoints.length === this.clickState.maxPoints;
    }

    resetSlider() {
        this.sliderState.isDragging = false;
        this.sliderState.currentX = 0;
        this.elements.sliderButton.style.left = '2px';
        this.elements.sliderButton.classList.remove('success', 'error', 'dragging');
        this.elements.sliderTrack.style.width = '0px';
        this.elements.sliderText.textContent = '向右滑动完成验证';
        this.elements.sliderPuzzle.style.left = '0px';
    }

    showResult(message, type) {
        this.elements.result.textContent = message;
        this.elements.result.className = 'captcha-result show ' + type;
    }

    clearResult() {
        this.elements.result.classList.remove('show', 'success', 'error', 'info');
    }

    showLoading() {
        if (this.options.type === 'slider') {
            this.elements.sliderLoading.style.display = 'flex';
        }
    }

    hideLoading() {
        this.elements.sliderLoading.style.display = 'none';
    }

    reset() {
        if (this.clickState.countdownTimer) {
            clearInterval(this.clickState.countdownTimer);
        }
        this.clearResult();
        this.resetSlider();
        this.clearClickPoints();
        this.switchTab('slider');
        this.refresh();
    }

    destroy() {
        if (this.clickState.countdownTimer) {
            clearInterval(this.clickState.countdownTimer);
        }
        this.container.innerHTML = '';
    }
}

document.addEventListener('DOMContentLoaded', function() {
    window.Captcha = Captcha;
});
