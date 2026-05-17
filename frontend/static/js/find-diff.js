class FindDiffCaptcha {
    constructor() {
        this.sessionId = null;
        this.image = null;
        this.foundDifferences = [];
        this.timerInterval = null;
        this.seconds = 0;
        this.init();
    }

    init() {
        this.bindEvents();
        this.refresh();
    }

    bindEvents() {
        document.getElementById('refresh-btn').addEventListener('click', () => this.refresh());
        document.getElementById('submit-btn').addEventListener('click', () => this.submit());
    }

    async refresh() {
        this.showLoading(true);
        this.hideResult();
        this.resetTimer();
        this.foundDifferences = [];

        try {
            const response = await fetch('/api/v1/captcha/find-diff/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    width: 400,
                    height: 400,
                    diff_count: 5
                })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.sessionId = result.data.session_id;
                this.image = result.data.image;
                this.renderImages();
                this.startTimer();
            } else {
                this.showResult(false, result.message || '生成验证码失败');
            }
        } catch (error) {
            console.error('Error:', error);
            this.showResult(false, '网络错误，请重试');
        } finally {
            this.showLoading(false);
        }
    }

    renderImages() {
        document.getElementById('found-count').textContent = '0';
        document.getElementById('total-count').textContent = this.image.diff_count;

        const container = document.getElementById('images-container');
        container.innerHTML = '';

        this.renderImage(container, this.image.image1_data, '图片 1', 0);
        this.renderImage(container, this.image.image2_data, '图片 2', 1);
    }

    renderImage(container, imageData, label, index) {
        const wrapper = document.createElement('div');
        wrapper.className = 'image-wrapper';

        const canvas = document.createElement('canvas');
        canvas.width = this.image.width;
        canvas.height = this.image.height;
        canvas.className = 'find-diff-canvas';
        canvas.dataset.index = index;

        const ctx = canvas.getContext('2d');
        const img = new Image();
        img.onload = () => {
            ctx.drawImage(img, 0, 0);
        };
        img.src = imageData;

        canvas.addEventListener('click', (e) => this.handleCanvasClick(e, canvas, index));

        const labelDiv = document.createElement('div');
        labelDiv.className = 'image-label';
        labelDiv.textContent = label;

        wrapper.appendChild(canvas);
        wrapper.appendChild(labelDiv);
        container.appendChild(wrapper);
    }

    handleCanvasClick(e, canvas, index) {
        const rect = canvas.getBoundingClientRect();
        const x = Math.floor((e.clientX - rect.left) * (canvas.width / rect.width));
        const y = Math.floor((e.clientY - rect.top) * (canvas.height / rect.height));

        const tolerance = 40;
        let alreadyFound = false;
        for (const diff of this.foundDifferences) {
            const distance = this.calculateDistance(x, y, diff.x, diff.y);
            if (distance < tolerance) {
                alreadyFound = true;
                break;
            }
        }

        if (!alreadyFound) {
            this.foundDifferences.push({ x, y });
            this.drawMarker(canvas, x, y);
            this.drawMarkerOnOtherCanvas(index, x, y);
            this.updateFoundCount();

            if (this.foundDifferences.length === this.image.diff_count) {
                setTimeout(() => this.submit(), 500);
            }
        }
    }

    drawMarker(canvas, x, y) {
        const ctx = canvas.getContext('2d');
        const radius = 25;

        ctx.strokeStyle = '#28a745';
        ctx.lineWidth = 3;
        ctx.beginPath();
        ctx.arc(x, y, radius, 0, 2 * Math.PI);
        ctx.stroke();

        ctx.beginPath();
        ctx.moveTo(x, y - radius);
        ctx.lineTo(x, y + radius);
        ctx.moveTo(x - radius, y);
        ctx.lineTo(x + radius, y);
        ctx.stroke();
    }

    drawMarkerOnOtherCanvas(sourceIndex, x, y) {
        const canvases = document.querySelectorAll('.find-diff-canvas');
        canvases.forEach((canvas, i) => {
            if (i !== sourceIndex) {
                this.drawMarker(canvas, x, y);
            }
        });
    }

    updateFoundCount() {
        document.getElementById('found-count').textContent = this.foundDifferences.length;
    }

    calculateDistance(x1, y1, x2, y2) {
        const dx = x1 - x2;
        const dy = y1 - y2;
        return Math.sqrt(dx * dx + dy * dy);
    }

    async submit() {
        if (!this.sessionId) {
            this.showResult(false, '请先生成验证码');
            return;
        }

        this.showLoading(true);

        try {
            const response = await fetch('/api/v1/captcha/find-diff/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    session_id: this.sessionId,
                    differences: this.foundDifferences
                })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.showResult(result.data.success, result.data.message || (result.data.success ? '验证成功！' : '验证失败'));
                if (result.data.success) {
                    this.stopTimer();
                }
            } else {
                this.showResult(false, result.message || '验证失败');
            }
        } catch (error) {
            console.error('Error:', error);
            this.showResult(false, '网络错误，请重试');
        } finally {
            this.showLoading(false);
        }
    }

    startTimer() {
        this.seconds = 0;
        this.updateTimerDisplay();
        this.timerInterval = setInterval(() => {
            this.seconds++;
            this.updateTimerDisplay();
        }, 1000);
    }

    stopTimer() {
        if (this.timerInterval) {
            clearInterval(this.timerInterval);
            this.timerInterval = null;
        }
    }

    resetTimer() {
        this.stopTimer();
        this.seconds = 0;
        this.updateTimerDisplay();
    }

    updateTimerDisplay() {
        const minutes = Math.floor(this.seconds / 60).toString().padStart(2, '0');
        const seconds = (this.seconds % 60).toString().padStart(2, '0');
        document.getElementById('timer').textContent = `${minutes}:${seconds}`;
    }

    showResult(success, message) {
        const banner = document.getElementById('result-banner');
        banner.className = `result-banner show ${success ? 'success' : 'error'}`;
        banner.innerHTML = `
            <i class="fas fa-${success ? 'check-circle' : 'times-circle'} me-2"></i>
            <strong>${message}</strong>
        `;
    }

    hideResult() {
        const banner = document.getElementById('result-banner');
        banner.classList.remove('show');
    }

    showLoading(show) {
        const overlay = document.getElementById('loading-overlay');
        if (show) {
            overlay.classList.add('show');
        } else {
            overlay.classList.remove('show');
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new FindDiffCaptcha();
});
