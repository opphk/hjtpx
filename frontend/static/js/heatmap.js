class HeatmapVisualization {
    constructor(containerId, options = {}) {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            console.error('Heatmap container not found');
            return;
        }

        this.options = {
            cellSize: options.cellSize || 20,
            cellGap: options.cellGap || 2,
            maxValue: options.maxValue || 100,
            minValue: options.minValue || 0,
            colorScale: options.colorScale || 'default',
            showLegend: options.showLegend !== false,
            showTooltip: options.showTooltip !== false,
            interactive: options.interactive !== false,
            gradient: options.gradient || [
                { stop: 0.0, color: '#1a1a2e' },
                { stop: 0.25, color: '#16213e' },
                { stop: 0.5, color: '#0f3460' },
                { stop: 0.75, color: '#e94560' },
                { stop: 1.0, color: '#ff6b6b' }
            ],
            ...options
        };

        this.data = [];
        this.canvas = null;
        this.ctx = null;
        this.tooltip = null;
        this.hoveredCell = null;
        this.animationFrame = null;
        this.lastUpdate = 0;
        this.frameInterval = options.frameInterval || 16;

        this.init();
    }

    init() {
        this.setupCanvas();
        this.setupTooltip();
        this.setupLegend();
        if (this.options.interactive) {
            this.setupEventListeners();
        }
    }

    setupCanvas() {
        this.canvas = document.createElement('canvas');
        this.canvas.style.width = '100%';
        this.canvas.style.height = '100%';
        this.canvas.style.display = 'block';
        this.container.appendChild(this.canvas);

        this.ctx = this.canvas.getContext('2d');
        this.resizeCanvas();
    }

    resizeCanvas() {
        const rect = this.container.getBoundingClientRect();
        const dpr = window.devicePixelRatio || 1;

        this.canvas.width = rect.width * dpr;
        this.canvas.height = rect.height * dpr;
        this.canvas.style.width = rect.width + 'px';
        this.canvas.style.height = rect.height + 'px';

        this.ctx.scale(dpr, dpr);
        this.drawWidth = rect.width;
        this.drawHeight = rect.height;

        this.cols = Math.floor(this.drawWidth / (this.options.cellSize + this.options.cellGap));
        this.rows = Math.floor(this.drawHeight / (this.options.cellSize + this.options.cellGap));

        this.render();
    }

    setupTooltip() {
        if (!this.options.showTooltip) return;

        this.tooltip = document.createElement('div');
        this.tooltip.className = 'heatmap-tooltip';
        this.tooltip.style.cssText = `
            position: absolute;
            background: rgba(0, 0, 0, 0.85);
            color: white;
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 12px;
            pointer-events: none;
            z-index: 1000;
            display: none;
            white-space: nowrap;
        `;
        this.container.style.position = 'relative';
        this.container.appendChild(this.tooltip);
    }

    setupLegend() {
        if (!this.options.showLegend) return;

        const legend = document.createElement('div');
        legend.className = 'heatmap-legend';
        legend.style.cssText = `
            position: absolute;
            bottom: 10px;
            right: 10px;
            display: flex;
            align-items: center;
            gap: 5px;
            background: rgba(255,255,255,0.9);
            padding: 5px 10px;
            border-radius: 4px;
            font-size: 11px;
        `;

        const gradientCanvas = document.createElement('canvas');
        gradientCanvas.width = 100;
        gradientCanvas.height = 12;
        gradientCanvas.style.borderRadius = '2px';

        const gradientCtx = gradientCanvas.getContext('2d');
        const gradient = gradientCtx.createLinearGradient(0, 0, 100, 0);
        this.options.gradient.forEach(g => {
            gradient.addColorStop(g.stop, g.color);
        });
        gradientCtx.fillStyle = gradient;
        gradientCtx.fillRect(0, 0, 100, 12);

        legend.innerHTML = `<span>${this.options.minValue}</span>`;
        legend.appendChild(gradientCanvas);
        legend.innerHTML += `<span>${this.options.maxValue}</span>`;

        this.container.appendChild(legend);
    }

    setupEventListeners() {
        this.canvas.addEventListener('mousemove', (e) => this.handleMouseMove(e));
        this.canvas.addEventListener('mouseleave', () => this.handleMouseLeave());

        let resizeTimeout;
        window.addEventListener('resize', () => {
            clearTimeout(resizeTimeout);
            resizeTimeout = setTimeout(() => this.resizeCanvas(), 100);
        });
    }

    handleMouseMove(e) {
        const rect = this.canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;

        const col = Math.floor(x / (this.options.cellSize + this.options.cellGap));
        const row = Math.floor(y / (this.options.cellSize + this.options.cellGap));

        if (col >= 0 && col < this.cols && row >= 0 && row < this.rows) {
            const index = row * this.cols + col;
            const cellData = this.data[index];

            if (cellData && this.tooltip) {
                const value = cellData.value || 0;
                const time = cellData.time || '';
                const count = cellData.count || value;

                this.tooltip.innerHTML = `
                    <strong>Position:</strong> (${col}, ${row})<br>
                    <strong>Value:</strong> ${value.toFixed(2)}<br>
                    <strong>Count:</strong> ${count}<br>
                    <strong>Time:</strong> ${time}
                `;
                this.tooltip.style.display = 'block';
                this.tooltip.style.left = (e.clientX - rect.left + 10) + 'px';
                this.tooltip.style.top = (e.clientY - rect.top - 10) + 'px';

                this.hoveredCell = { col, row };
                this.render();
            }
        }
    }

    handleMouseLeave() {
        if (this.tooltip) {
            this.tooltip.style.display = 'none';
        }
        this.hoveredCell = null;
        this.render();
    }

    setData(data) {
        this.data = data;
        this.render();
    }

    addDataPoint(x, y, value, time = '') {
        const index = y * this.cols + x;
        if (index >= 0 && index < this.data.length) {
            this.data[index] = {
                x, y,
                value: value,
                count: (this.data[index]?.count || 0) + 1,
                time: time || new Date().toLocaleTimeString()
            };
        }
    }

    clearData() {
        this.data = [];
        this.render();
    }

    getColorForValue(value) {
        const min = this.options.minValue;
        const max = this.options.maxValue;
        const normalized = Math.max(0, Math.min(1, (value - min) / (max - min)));

        const gradient = this.options.gradient;
        for (let i = 0; i < gradient.length - 1; i++) {
            if (normalized >= gradient[i].stop && normalized <= gradient[i + 1].stop) {
                const localT = (normalized - gradient[i].stop) / (gradient[i + 1].stop - gradient[i].stop);
                return this.interpolateColor(gradient[i].color, gradient[i + 1].color, localT);
            }
        }

        return gradient[gradient.length - 1].color;
    }

    interpolateColor(color1, color2, t) {
        const r1 = parseInt(color1.slice(1, 3), 16);
        const g1 = parseInt(color1.slice(3, 5), 16);
        const b1 = parseInt(color1.slice(5, 7), 16);

        const r2 = parseInt(color2.slice(1, 3), 16);
        const g2 = parseInt(color2.slice(3, 5), 16);
        const b2 = parseInt(color2.slice(5, 7), 16);

        const r = Math.round(r1 + (r2 - r1) * t);
        const g = Math.round(g1 + (g2 - g1) * t);
        const b = Math.round(b1 + (b2 - b1) * t);

        return `rgb(${r}, ${g}, ${b})`;
    }

    render() {
        const now = performance.now();
        if (now - this.lastUpdate < this.frameInterval) {
            if (!this.animationFrame) {
                this.animationFrame = requestAnimationFrame(() => {
                    this.animationFrame = null;
                    this.render();
                });
            }
            return;
        }
        this.lastUpdate = now;

        this.ctx.clearRect(0, 0, this.drawWidth, this.drawHeight);

        const cellSize = this.options.cellSize;
        const gap = this.options.cellGap;

        for (let row = 0; row < this.rows; row++) {
            for (let col = 0; col < this.cols; col++) {
                const index = row * this.cols + col;
                const cellData = this.data[index] || { value: 0 };
                const value = cellData.value || 0;

                const x = col * (cellSize + gap);
                const y = row * (cellSize + gap);

                const color = this.getColorForValue(value);

                this.ctx.fillStyle = color;
                this.ctx.beginPath();
                this.ctx.roundRect(x, y, cellSize, cellSize, 3);
                this.ctx.fill();

                if (this.hoveredCell && this.hoveredCell.col === col && this.hoveredCell.row === row) {
                    this.ctx.strokeStyle = '#fff';
                    this.ctx.lineWidth = 2;
                    this.ctx.stroke();
                }
            }
        }
    }

    destroy() {
        if (this.animationFrame) {
            cancelAnimationFrame(this.animationFrame);
        }
        if (this.canvas) {
            this.canvas.remove();
        }
        if (this.tooltip) {
            this.tooltip.remove();
        }
    }
}

class TrajectoryHeatmap {
    constructor(containerId, options = {}) {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            console.error('Trajectory heatmap container not found');
            return;
        }

        this.options = {
            width: options.width || 800,
            height: options.height || 600,
            radius: options.radius || 15,
            blur: options.blur || 20,
            maxOpacity: options.maxOpacity || 0.8,
            minOpacity: options.minOpacity || 0.1,
            gradient: options.gradient || {
                0.2: 'blue',
                0.4: 'cyan',
                0.6: 'lime',
                0.8: 'yellow',
                1.0: 'red'
            },
            ...options
        };

        this.points = [];
        this.heatmap = null;
        this.canvas = null;
        this.ctx = null;

        this.init();
    }

    init() {
        this.canvas = document.createElement('canvas');
        this.canvas.width = this.options.width;
        this.canvas.height = this.options.height;
        this.canvas.style.width = '100%';
        this.canvas.style.height = 'auto';
        this.container.appendChild(this.canvas);

        this.ctx = this.canvas.getContext('2d');
    }

    addPoint(x, y, value = 1) {
        this.points.push({ x, y, value });
        this.render();
    }

    addTrajectory(points, value = 1) {
        for (const point of points) {
            this.points.push({
                x: point.x,
                y: point.y,
                value: value
            });
        }
        this.render();
    }

    clearPoints() {
        this.points = [];
        this.ctx.clearRect(0, 0, this.options.width, this.options.height);
    }

    render() {
        this.ctx.clearRect(0, 0, this.options.width, this.options.height);

        const max = Math.max(...this.points.map(p => p.value), 1);
        const gradient = this.createGradient();

        for (const point of this.points) {
            const intensity = point.value / max;
            const radius = this.options.radius * intensity;

            const radialGradient = this.ctx.createRadialGradient(
                point.x, point.y, 0,
                point.x, point.y, radius
            );

            const color = this.getColorForIntensity(intensity);
            radialGradient.addColorStop(0, this.hexToRgba(color, this.options.maxOpacity * intensity));
            radialGradient.addColorStop(1, this.hexToRgba(color, 0));

            this.ctx.fillStyle = radialGradient;
            this.ctx.beginPath();
            this.ctx.arc(point.x, point.y, radius, 0, Math.PI * 2);
            this.ctx.fill();
        }
    }

    createGradient() {
        const canvas = document.createElement('canvas');
        canvas.width = 1;
        canvas.height = 256;
        const ctx = canvas.getContext('2d');

        const gradient = ctx.createLinearGradient(0, 0, 0, 256);
        for (const [stop, color] of Object.entries(this.options.gradient)) {
            gradient.addColorStop(parseFloat(stop), color);
        }

        ctx.fillStyle = gradient;
        ctx.fillRect(0, 0, 1, 256);

        return ctx.getImageData(0, 0, 1, 256).data;
    }

    getColorForIntensity(intensity) {
        const gradient = this.options.gradient;
        const stops = Object.keys(gradient).map(parseFloat).sort((a, b) => a - b);

        for (let i = 0; i < stops.length - 1; i++) {
            if (intensity >= stops[i] && intensity <= stops[i + 1]) {
                return gradient[stops[i]];
            }
        }

        return gradient[stops[stops.length - 1]];
    }

    hexToRgba(hex, alpha) {
        const r = parseInt(hex.slice(1, 3), 16);
        const g = parseInt(hex.slice(3, 5), 16);
        const b = parseInt(hex.slice(5, 7), 16);
        return `rgba(${r}, ${g}, ${b}, ${alpha})`;
    }

    getImageData() {
        return this.ctx.getImageData(0, 0, this.options.width, this.options.height);
    }

    exportImage(format = 'png') {
        return this.canvas.toDataURL(`image/${format}`);
    }

    destroy() {
        if (this.canvas) {
            this.canvas.remove();
        }
        this.points = [];
    }
}

class ClickHeatmap extends HeatmapVisualization {
    constructor(containerId, options = {}) {
        super(containerId, {
            ...options,
            colorScale: 'clicks'
        });

        this.clickData = new Map();
        this.timeWindow = options.timeWindow || 60000;
        this.initClickTracking();
    }

    initClickTracking() {
        this.container.addEventListener('click', (e) => {
            const rect = this.canvas.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;

            const col = Math.floor(x / (this.options.cellSize + this.options.cellGap));
            const row = Math.floor(y / (this.options.cellSize + this.options.cellGap));

            const key = `${col},${row}`;
            const existing = this.clickData.get(key) || { count: 0, lastClick: 0 };
            existing.count++;
            existing.lastClick = Date.now();

            this.clickData.set(key, existing);
            this.updateFromClickData();

            this.cleanOldData();
        });
    }

    updateFromClickData() {
        this.data = [];
        for (let row = 0; row < this.rows; row++) {
            for (let col = 0; col < this.cols; col++) {
                const key = `${col},${row}`;
                const clickInfo = this.clickData.get(key);
                this.data.push({
                    x: col,
                    y: row,
                    value: clickInfo ? clickInfo.count : 0,
                    count: clickInfo ? clickInfo.count : 0,
                    time: clickInfo ? new Date(clickInfo.lastClick).toLocaleTimeString() : ''
                });
            }
        }
        this.render();
    }

    cleanOldData() {
        const now = Date.now();
        for (const [key, value] of this.clickData.entries()) {
            if (now - value.lastClick > this.timeWindow) {
                this.clickData.delete(key);
            }
        }
    }

    clearData() {
        super.clearData();
        this.clickData.clear();
    }

    getClickStats() {
        const clicks = Array.from(this.clickData.values());
        return {
            totalClicks: clicks.reduce((sum, c) => sum + c.count, 0),
            uniquePositions: this.clickData.size,
            avgClicksPerPosition: clicks.length > 0
                ? clicks.reduce((sum, c) => sum + c.count, 0) / clicks.length
                : 0,
            maxClicks: clicks.length > 0
                ? Math.max(...clicks.map(c => c.count))
                : 0
        };
    }
}

class TrajectoryRecorder {
    constructor(options = {}) {
        this.points = [];
        this.maxPoints = options.maxPoints || 10000;
        this.sampleRate = options.sampleRate || 50;
        this.lastSampleTime = 0;
        this.isRecording = false;
        this.startTime = 0;
        this.callbacks = [];
    }

    start() {
        this.isRecording = true;
        this.startTime = Date.now();
        this.points = [];
        this.lastSampleTime = 0;
    }

    stop() {
        this.isRecording = false;
        return this.getTrajectory();
    }

    addPoint(x, y, metadata = {}) {
        if (!this.isRecording) return;

        const now = Date.now();
        if (now - this.lastSampleTime < this.sampleRate) return;
        this.lastSampleTime = now;

        if (this.points.length >= this.maxPoints) {
            this.points.shift();
        }

        this.points.push({
            x,
            y,
            timestamp: now - this.startTime,
            metadata
        });

        this.notifyCallbacks();
    }

    addPointsFromEvent(e, containerRect) {
        if (e.touches) {
            for (const touch of e.touches) {
                this.addPoint(
                    touch.clientX - containerRect.left,
                    touch.clientY - containerRect.top,
                    { type: 'touch' }
                );
            }
        } else {
            this.addPoint(
                e.clientX - containerRect.left,
                e.clientY - containerRect.top,
                { type: 'mouse' }
            );
        }
    }

    getTrajectory() {
        return {
            points: [...this.points],
            duration: this.points.length > 0
                ? this.points[this.points.length - 1].timestamp
                : 0,
            pointCount: this.points.length
        };
    }

    getDuration() {
        if (this.points.length < 2) return 0;
        return this.points[this.points.length - 1].timestamp - this.points[0].timestamp;
    }

    clear() {
        this.points = [];
    }

    onUpdate(callback) {
        this.callbacks.push(callback);
    }

    notifyCallbacks() {
        for (const callback of this.callbacks) {
            callback(this.getTrajectory());
        }
    }

    exportJSON() {
        return JSON.stringify(this.getTrajectory(), null, 2);
    }

    importJSON(json) {
        try {
            const data = JSON.parse(json);
            if (data.points && Array.isArray(data.points)) {
                this.points = data.points;
                this.isRecording = false;
                return true;
            }
        } catch (e) {
            console.error('Failed to import trajectory:', e);
        }
        return false;
    }
}

window.HeatmapVisualization = HeatmapVisualization;
window.TrajectoryHeatmap = TrajectoryHeatmap;
window.ClickHeatmap = ClickHeatmap;
window.TrajectoryRecorder = TrajectoryRecorder;
