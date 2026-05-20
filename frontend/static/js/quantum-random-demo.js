class QuantumRandomDemo {
    constructor(options = {}) {
        this.options = {
            apiBase: '/api/v1/captcha/quantum',
            canvasWidth: 300,
            canvasHeight: 200,
            noiseIntensity: 0.5,
            complexity: 5,
            ...options
        };

        this.sessionData = null;
        this.isGenerating = false;
        this.noiseData = null;
        this.listeners = {};
    }

    async init() {
        this.setupCanvas();
        this.setupEventListeners();
        await this.generateChallenge();
        return this;
    }

    setupCanvas() {
        const canvas = document.getElementById('quantum-canvas');
        if (!canvas) return;

        canvas.width = this.options.canvasWidth;
        canvas.height = this.options.canvasHeight;

        this.ctx = canvas.getContext('2d');
    }

    setupEventListeners() {
        document.getElementById('generate-btn')?.addEventListener('click', () => {
            this.generateChallenge();
        });

        document.getElementById('verify-btn')?.addEventListener('click', () => {
            this.verifyResponse();
        });

        document.getElementById('quality-check-btn')?.addEventListener('click', () => {
            this.runQualityCheck();
        });
    }

    async generateChallenge() {
        if (this.isGenerating) return;
        this.isGenerating = true;

        this.updateStatus('正在生成量子随机验证码...');

        try {
            const response = await fetch(`${this.options.apiBase}/create`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    type: 'quantum_pattern',
                    seed_length: 32,
                    noise_intensity: this.options.noiseIntensity,
                    pattern_complexity: this.options.complexity
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                this.sessionData = result.data;
                this.renderChallenge();
                this.updateStatus('验证码已生成');
                this.emit('challengeGenerated', result.data);
            }
        } catch (error) {
            console.error('Generate error:', error);
            this.updateStatus('生成失败: ' + error.message);
            this.generateLocalChallenge();
        } finally {
            this.isGenerating = false;
        }
    }

    generateLocalChallenge() {
        const seedLength = 32;
        const seedData = new Uint8Array(seedLength);
        crypto.getRandomValues(seedData);

        const challengeType = this.getRandomChallengeType();
        const positions = this.generatePositions(seedData, this.options.complexity);
        const sequence = this.generateSequence(seedData, this.options.complexity + 2);

        this.sessionData = {
            session_id: 'local_' + Date.now(),
            type: 'quantum_pattern',
            seed_data: this.arrayBufferToBase64(seedData),
            noise_pattern: this.generateNoisePattern(256),
            challenge_data: {
                type: challengeType,
                positions: positions,
                sequence: sequence
            },
            verification_data: {
                correct_positions: positions,
                correct_sequence: sequence
            },
            entropy_estimate: this.estimateEntropy(seedData),
            timestamp: Date.now(),
            expires_at: Date.now() + 300000,
            metadata: {
                source_type: 'local_quantum_simulated',
                complexity: this.options.complexity
            }
        };

        this.renderChallenge();
        this.updateStatus('本地验证码已生成');
    }

    getRandomChallengeType() {
        const types = ['simple_pattern', 'sequence', 'complex_pattern'];
        return types[Math.floor(Math.random() * types.length)];
    }

    generatePositions(seed, count) {
        const positions = [];
        const used = new Set();

        for (let i = 0; i < count; i++) {
            let pos;
            do {
                pos = seed[i % seed.length] % 16;
            } while (used.has(pos));
            used.add(pos);
            positions.push(pos);
        }

        return positions;
    }

    generateSequence(seed, length) {
        const sequence = [];
        for (let i = 0; i < length; i++) {
            sequence.push(seed[i % seed.length] % 10);
        }
        return sequence;
    }

    generateNoisePattern(length) {
        const pattern = new Uint8Array(length);
        crypto.getRandomValues(pattern);
        return pattern;
    }

    estimateEntropy(seed) {
        const entropy = seed.length * 7.5;
        return Math.min(entropy, seed.length * 8);
    }

    arrayBufferToBase64(buffer) {
        let binary = '';
        const bytes = new Uint8Array(buffer);
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    renderChallenge() {
        this.renderVisualPattern();
        this.renderChallengeUI();
        this.renderMetadata();
    }

    renderVisualPattern() {
        if (!this.ctx || !this.sessionData) return;

        const canvas = this.ctx.canvas;
        const width = canvas.width;
        const height = canvas.height;

        this.ctx.fillStyle = '#1a1a2e';
        this.ctx.fillRect(0, 0, width, height);

        const gridSize = 20;
        const cols = Math.floor(width / gridSize);
        const rows = Math.floor(height / gridSize);

        this.ctx.strokeStyle = '#333355';
        this.ctx.lineWidth = 1;
        for (let x = 0; x <= cols; x++) {
            this.ctx.beginPath();
            this.ctx.moveTo(x * gridSize, 0);
            this.ctx.lineTo(x * gridSize, height);
            this.ctx.stroke();
        }
        for (let y = 0; y <= rows; y++) {
            this.ctx.beginPath();
            this.ctx.moveTo(0, y * gridSize);
            this.ctx.lineTo(width, y * gridSize);
            this.ctx.stroke();
        }

        const noisePattern = this.sessionData.noise_pattern;
        const imageData = this.ctx.getImageData(0, 0, width, height);
        const data = imageData.data;

        for (let i = 0; i < data.length; i += 4) {
            if (noisePattern && i/4 < noisePattern.length) {
                const noise = noisePattern[i/4] * 0.1;
                data[i] = Math.min(255, data[i] + noise);
                data[i+1] = Math.min(255, data[i+1] + noise);
                data[i+2] = Math.min(255, data[i+2] + noise);
            }
        }

        this.ctx.putImageData(imageData, 0, 0);

        const challengeData = this.sessionData.challenge_data;
        if (challengeData && challengeData.positions) {
            this.ctx.fillStyle = '#c9a96e';
            for (const pos of challengeData.positions) {
                const col = pos % 4;
                const row = Math.floor(pos / 4);
                const x = col * gridSize * 4 + gridSize / 2;
                const y = row * gridSize * 2 + gridSize / 2;

                this.ctx.beginPath();
                this.ctx.arc(x, y, 8, 0, Math.PI * 2);
                this.ctx.fill();
            }
        }

        if (this.options.showAnswers) {
            this.renderAnswerPattern();
        }
    }

    renderAnswerPattern() {
        const challengeData = this.sessionData?.challenge_data;
        if (!challengeData) return;

        const infoDiv = document.getElementById('challenge-info');
        if (!infoDiv) return;

        let info = '<div class="answer-info">';

        if (challengeData.positions) {
            info += `<p>位置: [${challengeData.positions.join(', ')}]</p>`;
        }

        if (challengeData.sequence) {
            info += `<p>序列: [${challengeData.sequence.join(', ')}]</p>`;
        }

        info += '</div>';
        infoDiv.innerHTML += info;
    }

    renderChallengeUI() {
        const container = document.getElementById('challenge-ui');
        if (!container) return;

        const challengeType = this.sessionData?.challenge_data?.type || 'simple_pattern';

        let html = '<div class="challenge-interface">';

        if (challengeType === 'simple_pattern' || challengeType === 'complex_pattern') {
            html += `
                <div class="pattern-selector">
                    <p>点击选择正确的位置（按顺序）:</p>
                    <div class="pattern-grid">
                        ${Array.from({length: 16}, (_, i) => `
                            <button class="pattern-cell" data-index="${i}">
                                <span class="cell-number">${i}</span>
                                <span class="cell-marker"></span>
                            </button>
                        `).join('')}
                    </div>
                </div>
            `;
        }

        if (challengeType === 'sequence' || challengeType === 'complex_pattern') {
            html += `
                <div class="sequence-input">
                    <p>输入正确的数字序列:</p>
                    <input type="text" id="sequence-input" 
                           placeholder="例如: 123456" 
                           maxlength="10" 
                           pattern="[0-9]*"
                           inputmode="numeric">
                </div>
            `;
        }

        html += '</div>';
        container.innerHTML = html;

        this.setupChallengeInteraction();
    }

    setupChallengeInteraction() {
        const cells = document.querySelectorAll('.pattern-cell');
        const selectedPositions = [];

        cells.forEach(cell => {
            cell.addEventListener('click', () => {
                const index = parseInt(cell.dataset.index);

                if (cell.classList.contains('selected')) {
                    cell.classList.remove('selected');
                    const idx = selectedPositions.indexOf(index);
                    if (idx > -1) {
                        selectedPositions.splice(idx, 1);
                    }
                } else {
                    cell.classList.add('selected');
                    selectedPositions.push(index);
                }

                const verifyBtn = document.getElementById('verify-btn');
                if (verifyBtn) {
                    verifyBtn.disabled = selectedPositions.length === 0;
                }
            });
        });
    }

    renderMetadata() {
        const container = document.getElementById('metadata-info');
        if (!container || !this.sessionData) return;

        const meta = this.sessionData.metadata || {};
        const entropy = this.sessionData.entropy_estimate || 0;

        container.innerHTML = `
            <div class="metadata-panel">
                <div class="meta-item">
                    <span class="meta-label">会话ID:</span>
                    <span class="meta-value">${this.sessionData.session_id}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">熵估计:</span>
                    <span class="meta-value">${entropy.toFixed(2)} bits</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">复杂度:</span>
                    <span class="meta-value">${meta.complexity || this.options.complexity}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">来源:</span>
                    <span class="meta-value">${meta.source_type || 'unknown'}</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">有效期:</span>
                    <span class="meta-value">${Math.round((this.sessionData.expires_at - Date.now())/1000)}秒</span>
                </div>
            </div>
        `;
    }

    async verifyResponse() {
        const challengeType = this.sessionData?.challenge_data?.type;
        let response;

        if (challengeType === 'simple_pattern' || challengeType === 'complex_pattern') {
            const selectedCells = document.querySelectorAll('.pattern-cell.selected');
            const positions = Array.from(selectedCells).map(cell => parseInt(cell.dataset.index));
            response = positions;
        }

        if (challengeType === 'sequence' || challengeType === 'complex_pattern') {
            const seqInput = document.getElementById('sequence-input');
            const sequence = seqInput?.value.split('').map(Number) || [];
            if (response && typeof response === 'object' && !Array.isArray(response)) {
                response.sequence = sequence;
            } else {
                response = sequence;
            }
        }

        const startTime = Date.now();

        try {
            const response_ = await fetch(`${this.options.apiBase}/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    session_id: this.sessionData.session_id,
                    response: response,
                    response_time: Date.now() - startTime
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                this.showVerificationResult(result.data);
            }
        } catch (error) {
            console.error('Verify error:', error);
            this.verifyLocal(response);
        }
    }

    verifyLocal(response) {
        const verification = this.sessionData.verification_data;
        const challenge = this.sessionData.challenge_data;

        let isCorrect = false;

        if (challenge.type === 'simple_pattern' || challenge.type === 'complex_pattern') {
            if (Array.isArray(response)) {
                isCorrect = this.compareArrays(response, verification.correct_positions);
            }
        }

        if (challenge.type === 'sequence' || challenge.type === 'complex_pattern') {
            const seqResponse = Array.isArray(response) ? response : response?.sequence;
            if (seqResponse) {
                isCorrect = this.compareArrays(seqResponse, verification.correct_sequence);
            }
        }

        this.showVerificationResult({
            success: isCorrect,
            score: isCorrect ? 1.0 : 0.0,
            message: isCorrect ? '验证成功' : '验证失败',
            entropy_used: this.sessionData.entropy_estimate,
            unpredictability: this.sessionData.entropy_estimate / (this.sessionData.seed_data?.length * 8 || 1)
        });
    }

    compareArrays(a, b) {
        if (!Array.isArray(a) || !Array.isArray(b)) return false;
        if (a.length !== b.length) return false;
        for (let i = 0; i < a.length; i++) {
            if (a[i] !== b[i]) return false;
        }
        return true;
    }

    showVerificationResult(result) {
        const container = document.getElementById('verification-result');
        if (!container) return;

        container.innerHTML = `
            <div class="result-panel ${result.success ? 'success' : 'error'}">
                <div class="result-icon">
                    <i class="fas ${result.success ? 'fa-check-circle' : 'fa-times-circle'}"></i>
                </div>
                <div class="result-message">${result.message}</div>
                <div class="result-score">得分: ${(result.score * 100).toFixed(1)}%</div>
                ${result.entropy_used ? `<div class="entropy-info">熵使用量: ${result.entropy_used.toFixed(2)} bits</div>` : ''}
                ${result.unpredictability ? `<div class="unpredictability-info">不可预测性: ${(result.unpredictability * 100).toFixed(1)}%</div>` : ''}
            </div>
        `;

        this.emit('verificationComplete', result);
    }

    async runQualityCheck() {
        this.updateStatus('正在进行随机性质量检查...');

        const seedData = this.sessionData?.seed_data;
        if (!seedData) {
            this.showQualityResult(null, '无可用数据');
            return;
        }

        try {
            const result = this.localQualityCheck(seedData);
            this.showQualityResult(result, null);
        } catch (error) {
            console.error('Quality check error:', error);
            this.showQualityResult(null, error.message);
        }
    }

    localQualityCheck(base64Data) {
        const binaryString = atob(base64Data);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }

        const result = {
            frequency: this.runFrequencyTest(bytes),
            runs: this.runRunsTest(bytes),
            entropy: this.runEntropyTest(bytes),
            serial: this.runSerialTest(bytes)
        };

        let passed = 0;
        let total = 0;
        for (const test in result) {
            total++;
            if (result[test].passed) passed++;
        }

        return {
            overall_score: (passed / total) * 100,
            grade: this.calculateGrade(passed / total),
            tests_passed: passed,
            tests_failed: total - passed,
            total_tests: total,
            results: result
        };
    }

    runFrequencyTest(bytes) {
        let ones = 0;
        for (const b of bytes) {
            for (let i = 0; i < 8; i++) {
                if ((b >> i) & 1) ones++;
            }
        }
        const proportion = ones / (bytes.length * 8);
        const passed = proportion > 0.45 && proportion < 0.55;
        return { statistic: proportion, passed, name: 'Monobit Frequency' };
    }

    runRunsTest(bytes) {
        const bits = [];
        for (const b of bytes) {
            for (let i = 0; i < 8; i++) {
                bits.push((b >> i) & 1);
            }
        }

        let runs = 1;
        for (let i = 1; i < bits.length; i++) {
            if (bits[i] !== bits[i-1]) runs++;
        }

        const expectedRuns = 2 * bits.length * 0.5 * 0.5;
        const passed = Math.abs(runs - expectedRuns) < expectedRuns * 0.3;
        return { statistic: runs, passed, name: 'Runs Test' };
    }

    runEntropyTest(bytes) {
        const freq = {};
        for (const b of bytes) {
            freq[b] = (freq[b] || 0) + 1;
        }

        let entropy = 0;
        for (const f of Object.values(freq)) {
            const p = f / bytes.length;
            entropy -= p * Math.log2(p);
        }

        const passed = entropy > 7.5;
        return { statistic: entropy, passed, name: 'Entropy Test' };
    }

    runSerialTest(bytes) {
        const bits = [];
        for (const b of bytes) {
            for (let i = 0; i < 8; i++) {
                bits.push((b >> i) & 1);
            }
        }

        const freq = {};
        for (let i = 0; i < bits.length - 1; i++) {
            const pair = bits[i] * 2 + bits[i+1];
            freq[pair] = (freq[pair] || 0) + 1;
        }

        const chiSquare = Object.values(freq).reduce((sum, f) => {
            const expected = (bits.length - 1) / 4;
            return sum + Math.pow(f - expected, 2) / expected;
        }, 0);

        const passed = chiSquare < 7.815;
        return { statistic: chiSquare, passed, name: 'Serial Test' };
    }

    calculateGrade(score) {
        if (score >= 1.0) return 'A+ (优秀)';
        if (score >= 0.75) return 'A (良好)';
        if (score >= 0.5) return 'B (合格)';
        if (score >= 0.25) return 'C (较差)';
        return 'F (失败)';
    }

    showQualityResult(result, error) {
        const container = document.getElementById('quality-result');
        if (!container) return;

        if (error) {
            container.innerHTML = `<div class="quality-error">检查失败: ${error}</div>`;
            return;
        }

        let html = `
            <div class="quality-panel">
                <div class="quality-header">
                    <h3>随机性质量报告</h3>
                    <div class="quality-grade ${result.grade.split(' ')[0]}">${result.grade}</div>
                </div>
                <div class="quality-score">总体评分: ${result.overall_score.toFixed(1)}%</div>
                <div class="quality-summary">
                    通过: ${result.tests_passed}/${result.total_tests} | 失败: ${result.tests_failed}/${result.total_tests}
                </div>
                <div class="test-results">
        `;

        for (const [name, test] of Object.entries(result.results)) {
            html += `
                <div class="test-item ${test.passed ? 'passed' : 'failed'}">
                    <span class="test-name">${test.name}</span>
                    <span class="test-statistic">${test.statistic.toFixed(4)}</span>
                    <span class="test-status">${test.passed ? '通过' : '失败'}</span>
                </div>
            `;
        }

        html += '</div></div>';
        container.innerHTML = html;

        this.emit('qualityCheckComplete', result);
    }

    updateStatus(message) {
        const statusEl = document.getElementById('status-message');
        if (statusEl) {
            statusEl.textContent = message;
        }
    }

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(callback => callback(data));
        }
    }

    destroy() {
        this.listeners = {};
        this.sessionData = null;
    }
}

class QuantumNoiseVisualizer {
    constructor(canvas) {
        this.canvas = canvas;
        this.ctx = canvas?.getContext('2d');
        this.animationId = null;
    }

    visualizeNoise(intensity = 0.5) {
        if (!this.ctx) return;

        const width = this.canvas.width;
        const height = this.canvas.height;
        const imageData = this.ctx.createImageData(width, height);
        const data = imageData.data;

        for (let y = 0; y < height; y++) {
            for (let x = 0; x < width; x++) {
                const i = (y * width + x) * 4;

                const time = Date.now() / 1000;
                const noise = this.quantumNoise(x, y, time, intensity);

                data[i] = noise;
                data[i + 1] = noise;
                data[i + 2] = noise + 20;
                data[i + 3] = 255;
            }
        }

        this.ctx.putImageData(imageData, 0, 0);
    }

    quantumNoise(x, y, t, intensity) {
        const freq1 = 0.1;
        const freq2 = 0.15;
        const freq3 = 0.08;

        const noise1 = Math.sin(2 * Math.PI * freq1 * x + t * 2);
        const noise2 = Math.sin(2 * Math.PI * freq2 * y + t * 3);
        const noise3 = Math.sin(2 * Math.PI * freq3 * (x + y) + t);

        const combined = (noise1 + noise2 + noise3) / 3;
        const scaled = (combined + 1) / 2;

        return Math.floor(scaled * intensity * 255);
    }

    startAnimation(intensity = 0.5) {
        const animate = () => {
            this.visualizeNoise(intensity);
            this.animationId = requestAnimationFrame(animate);
        };
        animate();
    }

    stopAnimation() {
        if (this.animationId) {
            cancelAnimationFrame(this.animationId);
            this.animationId = null;
        }
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { QuantumRandomDemo, QuantumNoiseVisualizer };
}
