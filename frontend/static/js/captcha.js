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
            hintText: '请依次点击图中的文字'
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
                        <div class="captcha-click-hint" id="click-hint">请依次点击图中的文字</div>
                        <div class="captcha-click-grid" id="click-grid">
                            <img class="captcha-click-image" id="click-image" alt="点选验证码图片">
                            <button class="captcha-refresh" id="click-refresh">🔄</button>
                            <div class="captcha-loading" id="click-loading" style="display: none;">
                                <div class="captcha-loading-spinner"></div>
                            </div>
                        </div>
                        <div class="captcha-actions">
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
            clickImage: this.container.querySelector('#click-image'),
            clickRefresh: this.container.querySelector('#click-refresh'),
            clickLoading: this.container.querySelector('#click-loading'),
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
        this.elements.clickRefresh.addEventListener('click', () => this.refresh());

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
        const grid = this.elements.clickGrid;

        grid.addEventListener('click', (e) => {
            if (e.target === this.elements.clickRefresh || 
                e.target === this.elements.clickImage) {
                if (e.target === this.elements.clickRefresh) return;
            }

            if (this.clickState.selectedPoints.length >= this.clickState.maxPoints) {
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
        });

        this.elements.clickClear.addEventListener('click', () => {
            this.clearClickPoints();
        });

        this.elements.clickSubmit.addEventListener('click', () => {
            if (this.clickState.selectedPoints.length > 0) {
                this.verifyClick();
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
        
        marker.addEventListener('click', (e) => {
            e.stopPropagation();
            const idx = parseInt(marker.dataset.index);
            this.removeClickPoint(idx);
        });
        
        this.elements.clickGrid.appendChild(marker);
    }

    removeClickPoint(index) {
        this.clickState.selectedPoints.splice(index, 1);
        this.updateClickMarkers();
    }

    updateClickMarkers() {
        const markers = this.elements.clickGrid.querySelectorAll('.captcha-click-marker');
        markers.forEach(m => m.remove());
        
        this.clickState.selectedPoints.forEach((point, idx) => {
            this.addClickMarker(point, idx + 1);
        });
    }

    clearClickPoints() {
        this.clickState.selectedPoints = [];
        const markers = this.elements.clickGrid.querySelectorAll('.captcha-click-marker');
        markers.forEach(m => m.remove());
    }

    updatePuzzlePosition(x) {
        this.elements.sliderPuzzle.style.left = x + 'px';
    }

    switchTab(type) {
        this.options.type = type;
        
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
        
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/click`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' }
            });
            
            if (response.ok) {
                const data = await response.json();
                this.sessionId = data.session_id;
                this.elements.clickImage.src = data.image_url;
                this.clickState.hintText = data.hint || '请依次点击图中的文字';
                this.clickState.maxPoints = data.max_points || 3;
                this.elements.clickHint.textContent = this.clickState.hintText;
            } else {
                this.loadDemoClick();
            }
        } catch (error) {
            this.loadDemoClick();
        }
    }

    loadDemoClick() {
        this.sessionId = 'demo_' + Date.now();
        this.clickState.hintText = '请依次点击: 1, 2, 3';
        this.clickState.maxPoints = 3;
        this.elements.clickHint.textContent = this.clickState.hintText;
        
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
        this.showLoading();
        
        const payload = {
            session_id: this.sessionId,
            points: this.clickState.selectedPoints,
            type: 'click'
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
                this.showResult('验证成功!', 'success');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ type: 'click', session_id: this.sessionId });
                }
            } else {
                this.showResult('验证失败，请重试', 'error');
                if (this.options.onError) {
                    this.options.onError({ type: 'click', error: '验证失败' });
                }
                setTimeout(() => this.refresh(), 1500);
            }
        } catch (error) {
            const success = this.simulateClickVerify();
            if (success) {
                this.showResult('验证成功!', 'success');
                if (this.options.onSuccess) {
                    this.options.onSuccess({ type: 'click', session_id: this.sessionId });
                }
            } else {
                this.showResult('验证失败，请重试', 'error');
                setTimeout(() => this.refresh(), 1500);
            }
        } finally {
            this.hideLoading();
        }
    }

    simulateClickVerify() {
        return this.clickState.selectedPoints.length === 3;
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
        this.elements.result.classList.remove('show', 'success', 'error');
    }

    showLoading() {
        if (this.options.type === 'slider') {
            this.elements.sliderLoading.style.display = 'flex';
        } else {
            this.elements.clickLoading.style.display = 'flex';
        }
    }

    hideLoading() {
        this.elements.sliderLoading.style.display = 'none';
        this.elements.clickLoading.style.display = 'none';
    }

    reset() {
        this.clearResult();
        this.resetSlider();
        this.clearClickPoints();
        this.switchTab('slider');
        this.refresh();
    }
}

document.addEventListener('DOMContentLoaded', function() {
    window.Captcha = Captcha;
});
