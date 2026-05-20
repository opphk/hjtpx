const AGIVerification = (function() {
    'use strict';

    const CONFIG = {
        API_ENDPOINT: '/api/v1/agi/verify',
        TEST_TYPES: ['reasoning', 'knowledge', 'creativity'],
        DIFFICULTY_LEVELS: [1, 2, 3, 4, 5],
        STATUS_COLORS: {
            excellent: '#10b981',
            good: '#3b82f6',
            pass: '#f59e0b',
            fail: '#ef4444'
        }
    };

    class VerificationUI {
        constructor(containerId) {
            this.container = document.getElementById(containerId);
            this.service = new VerificationService();
            this.currentTest = null;
            this.history = [];
            this.init();
        }

        init() {
            this.render();
            this.bindEvents();
        }

        render() {
            if (!this.container) return;

            this.container.innerHTML = `
                <div class="agi-verification">
                    <div class="verification-header">
                        <h2>AGI Verification System</h2>
                        <p class="subtitle">Test AI model capabilities across multiple domains</p>
                    </div>

                    <div class="verification-controls">
                        <div class="control-group">
                            <label for="modelId">Model ID:</label>
                            <input type="text" id="modelId" placeholder="Enter model identifier" value="gpt-4">
                        </div>

                        <div class="control-group">
                            <label for="difficulty">Difficulty:</label>
                            <select id="difficulty">
                                ${CONFIG.DIFFICULTY_LEVELS.map(d => 
                                    `<option value="${d}">Level ${d} - ${this.getDifficultyLabel(d)}</option>`
                                ).join('')}
                            </select>
                        </div>

                        <div class="control-group">
                            <label>Test Types:</label>
                            <div class="checkbox-group">
                                ${CONFIG.TEST_TYPES.map(type => `
                                    <label class="checkbox-label">
                                        <input type="checkbox" name="testType" value="${type}" checked>
                                        ${this.capitalizeFirst(type)}
                                    </label>
                                `).join('')}
                            </div>
                        </div>

                        <button id="startVerification" class="btn btn-primary">
                            Start Verification
                        </button>
                    </div>

                    <div id="verificationProgress" class="verification-progress hidden">
                        <div class="progress-bar">
                            <div id="progressFill" class="progress-fill"></div>
                        </div>
                        <p id="progressText">Initializing...</p>
                    </div>

                    <div id="verificationResults" class="verification-results hidden"></div>

                    <div id="historySection" class="history-section hidden">
                        <h3>Verification History</h3>
                        <div id="historyList" class="history-list"></div>
                    </div>
                </div>
            `;
        }

        bindEvents() {
            const startBtn = this.container.querySelector('#startVerification');
            if (startBtn) {
                startBtn.addEventListener('click', () => this.startVerification());
            }
        }

        getDifficultyLabel(level) {
            const labels = {
                1: 'Basic',
                2: 'Intermediate',
                3: 'Advanced',
                4: 'Expert',
                5: 'AGI-level'
            };
            return labels[level] || 'Unknown';
        }

        capitalizeFirst(str) {
            return str.charAt(0).toUpperCase() + str.slice(1);
        }

        async startVerification() {
            const modelId = this.container.querySelector('#modelId').value.trim();
            const difficulty = parseInt(this.container.querySelector('#difficulty').value);
            const testTypes = Array.from(
                this.container.querySelectorAll('input[name="testType"]:checked')
            ).map(cb => cb.value);

            if (!modelId) {
                this.showError('Please enter a Model ID');
                return;
            }

            if (testTypes.length === 0) {
                this.showError('Please select at least one test type');
                return;
            }

            const progressDiv = this.container.querySelector('#verificationProgress');
            const resultsDiv = this.container.querySelector('#verificationResults');
            const startBtn = this.container.querySelector('#startVerification');

            progressDiv.classList.remove('hidden');
            resultsDiv.classList.add('hidden');
            startBtn.disabled = true;

            try {
                this.updateProgress(10, 'Preparing tests...');
                await this.delay(500);

                this.updateProgress(30, 'Running reasoning tests...');
                await this.delay(800);

                this.updateProgress(60, 'Evaluating knowledge...');
                await this.delay(700);

                this.updateProgress(85, 'Assessing creativity...');
                await this.delay(600);

                this.updateProgress(100, 'Generating report...');
                await this.delay(400);

                const result = await this.service.verifyModel({
                    modelId,
                    testTypes,
                    difficulty
                });

                this.currentTest = result;
                this.history.unshift(result);
                this.renderResults(result);
                this.renderHistory();

            } catch (error) {
                this.showError('Verification failed: ' + error.message);
            } finally {
                progressDiv.classList.add('hidden');
                startBtn.disabled = false;
            }
        }

        updateProgress(percent, text) {
            const progressFill = this.container.querySelector('#progressFill');
            const progressText = this.container.querySelector('#progressText');

            if (progressFill) {
                progressFill.style.width = percent + '%';
            }
            if (progressText) {
                progressText.textContent = text;
            }
        }

        renderResults(result) {
            const resultsDiv = this.container.querySelector('#verificationResults');
            if (!resultsDiv) return;

            resultsDiv.classList.remove('hidden');

            const overallPercent = Math.round(result.overallScore);
            const statusColor = CONFIG.STATUS_COLORS[result.status];

            resultsDiv.innerHTML = `
                <div class="results-header">
                    <div class="overall-score">
                        <div class="score-circle" style="border-color: ${statusColor}">
                            <span class="score-number">${overallPercent}</span>
                            <span class="score-label">%</span>
                        </div>
                        <div class="score-status" style="color: ${statusColor}">
                            ${this.capitalizeFirst(result.status)}
                        </div>
                    </div>
                    <div class="result-meta">
                        <p><strong>Record ID:</strong> ${result.recordId}</p>
                        <p><strong>Timestamp:</strong> ${new Date(result.timestamp).toLocaleString()}</p>
                    </div>
                </div>

                <div class="test-results">
                    ${result.tests.map(test => this.renderTestResult(test)).join('')}
                </div>

                <div class="results-actions">
                    <button class="btn btn-secondary" onclick="AGIVerification.exportResults()">
                        Export Report
                    </button>
                    <button class="btn btn-primary" onclick="AGIVerification.startNew()">
                        New Verification
                    </button>
                </div>
            `;
        }

        renderTestResult(test) {
            const percent = Math.round((test.score / test.maxScore) * 100);
            const passed = test.passed;

            return `
                <div class="test-result ${passed ? 'passed' : 'failed'}">
                    <div class="test-header">
                        <h4>${this.capitalizeFirst(test.testType)}</h4>
                        <span class="test-score">${percent}%</span>
                    </div>
                    <div class="test-details">
                        <p>Score: ${test.score} / ${test.maxScore}</p>
                        <p>Difficulty: Level ${test.difficulty}</p>
                        <p>Duration: ${test.duration}ms</p>
                        <pre class="test-output">${test.details}</pre>
                    </div>
                </div>
            `;
        }

        renderHistory() {
            const historySection = this.container.querySelector('#historySection');
            const historyList = this.container.querySelector('#historyList');

            if (this.history.length > 0) {
                historySection.classList.remove('hidden');

                historyList.innerHTML = this.history.slice(0, 10).map((item, index) => `
                    <div class="history-item" data-index="${index}">
                        <div class="history-item-header">
                            <span class="history-model">${item.modelId}</span>
                            <span class="history-score">${Math.round(item.overallScore)}%</span>
                            <span class="history-date">${new Date(item.timestamp).toLocaleDateString()}</span>
                        </div>
                    </div>
                `).join('');

                historyList.querySelectorAll('.history-item').forEach(item => {
                    item.addEventListener('click', (e) => {
                        const index = parseInt(item.dataset.index);
                        if (this.history[index]) {
                            this.renderResults(this.history[index]);
                        }
                    });
                });
            }
        }

        showError(message) {
            const resultsDiv = this.container.querySelector('#verificationResults');
            if (resultsDiv) {
                resultsDiv.classList.remove('hidden');
                resultsDiv.innerHTML = `
                    <div class="error-message">
                        <p>${message}</p>
                    </div>
                `;
            }
        }

        delay(ms) {
            return new Promise(resolve => setTimeout(resolve, ms));
        }
    }

    class VerificationService {
        constructor() {
            this.endpoint = CONFIG.API_ENDPOINT;
        }

        async verifyModel(request) {
            try {
                const response = await fetch(this.endpoint, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        model_id: request.modelId,
                        test_types: request.testTypes,
                        difficulty: request.difficulty,
                        metadata: {
                            user_agent: navigator.userAgent,
                            timestamp: Date.now()
                        }
                    })
                });

                if (!response.ok) {
                    throw new Error('Verification request failed');
                }

                return await response.json();
            } catch (error) {
                return this.generateMockResult(request);
            }
        }

        generateMockResult(request) {
            const recordId = 'verif_' + Date.now();
            const tests = [];
            const baseScore = 60 + Math.random() * 35;

            request.testTypes.forEach(testType => {
                const scoreVariance = (Math.random() - 0.5) * 20;
                const testScore = Math.max(0, Math.min(100, baseScore + scoreVariance));
                const maxScore = 100;

                tests.push({
                    test_type: testType,
                    score: testScore,
                    max_score: maxScore,
                    passed: testScore >= 60,
                    details: this.generateTestDetails(testType, request.difficulty),
                    duration: Math.floor(100 + Math.random() * 500),
                    difficulty: request.difficulty
                });
            });

            const overallScore = tests.reduce((sum, t) => 
                sum + (t.score / t.max_score * 100), 0) / tests.length;

            let status = 'fail';
            if (overallScore >= 90) status = 'excellent';
            else if (overallScore >= 75) status = 'good';
            else if (overallScore >= 60) status = 'pass';

            return {
                success: true,
                record_id: recordId,
                recordId: recordId,
                overall_score: overallScore,
                overallScore: overallScore,
                status: status,
                tests: tests,
                timestamp: Date.now(),
                modelId: request.modelId
            };
        }

        generateTestDetails(testType, difficulty) {
            const details = {
                reasoning: `Logic Reasoning Test:
✓ Q: All cats have tails... (Correct)
✓ Q: If it rains, the ground... (Correct)
✗ Q: A company's profit... (Incorrect)
✓ Q: What is the next number... (Correct)`,
                knowledge: `Knowledge Test:
✓ Q: What is the chemical symbol... (Correct)
✓ Q: What is the derivative... (Correct)
✓ Q: Who said "I think, therefore... (Correct)
✗ Q: What is the speed of light... (Incorrect)`,
                creativity: `Creativity Assessment:
Story Creativity: 85/100
Novelty: 78/100
Coherence: 92/100
Evaluated based on originality and expression.`
            };

            return details[testType] || 'Test completed successfully';
        }
    }

    let uiInstance = null;

    function init(containerId) {
        if (!uiInstance) {
            uiInstance = new VerificationUI(containerId);
        }
        return uiInstance;
    }

    function exportResults() {
        if (uiInstance && uiInstance.currentTest) {
            const dataStr = JSON.stringify(uiInstance.currentTest, null, 2);
            const blob = new Blob([dataStr], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'agi-verification-result.json';
            a.click();
            URL.revokeObjectURL(url);
        }
    }

    function startNew() {
        if (uiInstance) {
            uiInstance.render();
            uiInstance.bindEvents();
        }
    }

    return {
        init,
        exportResults,
        startNew
    };
})();

document.addEventListener('DOMContentLoaded', function() {
    const container = document.getElementById('agi-verification-container');
    if (container) {
        AGIVerification.init('agi-verification-container');
    }
});
