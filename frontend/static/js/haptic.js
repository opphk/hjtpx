class HapticCaptcha {
    constructor(options = {}) {
        this.options = {
            apiBase: options.apiBase || '/api/v1',
            onSuccess: options.onSuccess || (() => {}),
            onError: options.onError || (() => {}),
            onInit: options.onInit || (() => {}),
            onPatternChange: options.onPatternChange || (() => {}),
            difficulty: options.difficulty || 'medium',
            gridSize: options.gridSize || 3,
            patternType: options.patternType || 'sequence',
            enableVibration: options.enableVibration !== false,
            enableVisualHint: options.enableVisualHint !== false
        };
        
        this.session = null;
        this.userSequence = [];
        this.userTimestamps = [];
        this.userPressures = [];
        this.startTime = null;
        this.isVibrationSupported = 'vibrate' in navigator;
    }

    async init() {
        await this.generateCaptcha();
        this.options.onInit(this.session);
        return this;
    }

    async generateCaptcha() {
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/haptic/create`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    difficulty: this.options.difficulty,
                    grid_size: this.options.gridSize,
                    pattern_type: this.options.patternType
                })
            });

            const data = await response.json();
            
            if (data.code === 0 && data.data) {
                this.session = data.data;
                this.options.onPatternChange(this.session.pattern);
                return this.session;
            } else {
                throw new Error(data.message || 'Failed to generate captcha');
            }
        } catch (error) {
            this.options.onError(error);
            throw error;
        }
    }

    startSequence() {
        this.userSequence = [];
        this.userTimestamps = [];
        this.userPressures = [];
        this.startTime = Date.now();
    }

    async recordTap(position, event) {
        if (!this.startTime) {
            this.startSequence();
        }

        const timestamp = Date.now() - this.startTime;
        let pressure = 0.5;

        if (event.touches && event.touches[0]) {
            pressure = event.touches[0].force || 0.5;
        } else if (event.originalEvent && event.originalEvent.touches && event.originalEvent.touches[0]) {
            pressure = event.originalEvent.touches[0].force || 0.5;
        }

        this.userSequence.push(position);
        this.userTimestamps.push(timestamp);
        this.userPressures.push(pressure);

        if (this.options.enableVibration && this.isVibrationSupported && this.session) {
            const pattern = this.session.pattern;
            const vibrationDuration = Math.floor(100 + pattern.intensity * 100);
            navigator.vibrate(vibrationDuration);
        }

        return {
            position,
            timestamp,
            pressure
        };
    }

    endSequence() {
        this.startTime = null;
    }

    async verify() {
        if (!this.session || this.userSequence.length === 0) {
            throw new Error('No user input recorded');
        }

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/haptic/verify`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    session_id: this.session.session_id,
                    user_input: {
                        sequence: this.userSequence,
                        timestamps: this.userTimestamps,
                        pressures: this.userPressures
                    }
                })
            });

            const data = await response.json();
            
            if (data.code === 0 && data.data) {
                const result = data.data;
                if (result.success) {
                    this.options.onSuccess(result);
                } else {
                    this.options.onError(new Error(result.message));
                }
                return result;
            } else {
                throw new Error(data.message || 'Verification failed');
            }
        } catch (error) {
            this.options.onError(error);
            throw error;
        }
    }

    async analyzePattern() {
        if (this.userSequence.length === 0) {
            return null;
        }

        try {
            const response = await fetch(`${this.options.apiBase}/captcha/haptic/analyze`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    session_id: this.session ? this.session.session_id : '',
                    user_input: {
                        sequence: this.userSequence,
                        timestamps: this.userTimestamps,
                        pressures: this.userPressures
                    }
                })
            });

            const data = await response.json();
            return data.data || null;
        } catch (error) {
            console.error('Pattern analysis failed:', error);
            return null;
        }
    }

    getGridPositions(gridSize) {
        const positions = [];
        for (let i = 0; i < gridSize * gridSize; i++) {
            positions.push({
                index: i,
                row: Math.floor(i / gridSize),
                col: i % gridSize
            });
        }
        return positions;
    }

    createGridElement(container, gridSize, onTap) {
        const grid = document.createElement('div');
        grid.className = 'haptic-grid';
        grid.style.display = 'grid';
        grid.style.gridTemplateColumns = `repeat(${gridSize}, 1fr)`;
        grid.style.gap = '8px';
        grid.style.padding = '16px';

        const positions = this.getGridPositions(gridSize);
        
        positions.forEach(pos => {
            const cell = document.createElement('div');
            cell.className = 'haptic-cell';
            cell.dataset.position = pos.index;
            cell.style.width = '60px';
            cell.style.height = '60px';
            cell.style.borderRadius = '12px';
            cell.style.border = '2px solid #c9a96e';
            cell.style.backgroundColor = '#f8f9fa';
            cell.style.cursor = 'pointer';
            cell.style.transition = 'all 0.2s ease';
            cell.style.display = 'flex';
            cell.style.alignItems = 'center';
            cell.style.justifyContent = 'center';
            cell.style.fontSize = '20px';
            cell.style.fontWeight = 'bold';
            cell.style.color = '#1a1a2e';

            cell.addEventListener('click', (e) => this.handleCellTap(pos.index, e));
            cell.addEventListener('touchstart', (e) => {
                e.preventDefault();
                this.handleCellTap(pos.index, e);
            }, { passive: false });

            grid.appendChild(cell);
        });

        if (container) {
            container.innerHTML = '';
            container.appendChild(grid);
        }

        return grid;
    }

    handleCellTap(position, event) {
        const cell = event.currentTarget;
        
        this.recordTap(position, event);
        
        cell.style.backgroundColor = '#c9a96e';
        cell.style.transform = 'scale(0.95)';
        
        setTimeout(() => {
            cell.style.backgroundColor = '#f8f9fa';
            cell.style.transform = 'scale(1)';
        }, 200);
    }

    highlightSequence(hints, gridElement) {
        if (!gridElement || !hints || hints.length === 0) {
            return;
        }

        const cells = gridElement.querySelectorAll('.haptic-cell');
        
        hints.forEach((pos, index) => {
            if (cells[pos]) {
                setTimeout(() => {
                    cells[pos].style.borderColor = '#28a745';
                    cells[pos].style.borderWidth = '3px';
                    cells[pos].textContent = (index + 1).toString();
                }, index * 500);
            }
        });
    }

    async playHapticPattern() {
        if (!this.session || !this.session.pattern) {
            return;
        }

        const pattern = this.session.pattern;
        
        if (!this.isVibrationSupported) {
            console.warn('Vibration API not supported');
            return;
        }

        for (let i = 0; i < pattern.Taps.length; i++) {
            const tap = pattern.Taps[i];
            
            await new Promise(resolve => setTimeout(resolve, i > 0 ? 200 : 0));
            
            navigator.vibrate(tap.Duration);
            
            await new Promise(resolve => setTimeout(resolve, tap.Duration));
            navigator.vibrate(0);
        }
    }

    async getOptions() {
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/haptic/options`);
            const data = await response.json();
            return data.data || {};
        } catch (error) {
            console.error('Failed to get options:', error);
            return {};
        }
    }

    reset() {
        this.session = null;
        this.userSequence = [];
        this.userTimestamps = [];
        this.userPressures = [];
        this.startTime = null;
    }

    getCurrentInput() {
        return {
            sequence: this.userSequence,
            timestamps: this.userTimestamps,
            pressures: this.userPressures,
            length: this.userSequence.length
        };
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = HapticCaptcha;
}
