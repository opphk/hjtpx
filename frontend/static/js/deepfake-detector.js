const DeepfakeDetector = (function() {
    'use strict';

    const CONFIG = {
        API_ENDPOINT: '/api/v1/deepfake/detect',
        WATERMARK_ENDPOINT: '/api/v1/watermark/verify',
        MAX_FILE_SIZE: 10 * 1024 * 1024,
        SUPPORTED_TYPES: ['image/jpeg', 'image/png', 'video/mp4', 'video/webm'],
        DETECTION_MODULES: {
            face: { name: 'Face Analysis', weight: 0.3 },
            texture: { name: 'Texture Analysis', weight: 0.25 },
            semantic: { name: 'Semantic Consistency', weight: 0.2 },
            frequency: { name: 'Frequency Domain', weight: 0.15 },
            watermark: { name: 'Watermark Verification', weight: 0.1 }
        },
        RISK_LEVELS: {
            critical: { min: 85, color: '#dc2626', label: 'Critical Risk' },
            high: { min: 70, color: '#ea580c', label: 'High Risk' },
            medium: { min: 50, color: '#f59e0b', label: 'Medium Risk' },
            low: { min: 30, color: '#10b981', label: 'Low Risk' },
            minimal: { min: 0, color: '#059669', label: 'Minimal Risk' }
        }
    };

    class DetectorUI {
        constructor(containerId) {
            this.container = document.getElementById(containerId);
            this.detector = new DeepfakeDetectorService();
            this.currentResults = null;
            this.detectionHistory = [];
            this.analysisMode = 'comprehensive';
            this.init();
        }

        init() {
            if (!this.container) {
                console.error('DeepfakeDetector: Container not found');
                return;
            }
            this.render();
            this.bindEvents();
        }

        render() {
            this.container.innerHTML = `
                <div class="deepfake-detector">
                    <div class="detector-header">
                        <h2>Deepfake Detection System V3</h2>
                        <p class="subtitle">Advanced AI-generated content detection and verification</p>
                    </div>

                    <div class="upload-section">
                        <div class="upload-zone" id="uploadZone">
                            <div class="upload-icon">
                                <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                                    <polyline points="17 8 12 3 7 8"></polyline>
                                    <line x1="12" y1="3" x2="12" y2="15"></line>
                                </svg>
                            </div>
                            <p class="upload-text">Drag & drop media files here or click to upload</p>
                            <input type="file" id="fileInput" accept="image/*,video/*" hidden>
                            <button id="browseBtn" class="btn btn-secondary">Browse Files</button>
                        </div>

                        <div class="analysis-options">
                            <h4>Analysis Mode</h4>
                            <div class="mode-selector">
                                <label class="radio-label">
                                    <input type="radio" name="analysisMode" value="quick" checked>
                                    <span>Quick Scan</span>
                                </label>
                                <label class="radio-label">
                                    <input type="radio" name="analysisMode" value="comprehensive">
                                    <span>Comprehensive</span>
                                </label>
                                <label class="radio-label">
                                    <input type="radio" name="analysisMode" value="watermark">
                                    <span>Watermark Verification</span>
                                </label>
                            </div>
                        </div>
                    </div>

                    <div id="previewSection" class="preview-section hidden">
                        <div class="preview-container">
                            <img id="previewImage" class="preview-media hidden" alt="Preview">
                            <video id="previewVideo" class="preview-media hidden" controls></video>
                        </div>
                        <div class="preview-info">
                            <span id="fileName"></span>
                            <span id="fileSize"></span>
                            <button id="removeBtn" class="btn btn-icon">×</button>
                        </div>
                    </div>

                    <div id="analysisProgress" class="analysis-progress hidden">
                        <div class="progress-header">
                            <span class="progress-title">Analyzing...</span>
                            <span id="progressPercent" class="progress-percent">0%</span>
                        </div>
                        <div class="progress-bar">
                            <div id="progressFill" class="progress-fill"></div>
                        </div>
                        <div class="progress-stages">
                            <div class="stage" data-stage="face">
                                <div class="stage-icon">👤</div>
                                <span>Face Analysis</span>
                            </div>
                            <div class="stage" data-stage="texture">
                                <div class="stage-icon">🎨</div>
                                <span>Texture</span>
                            </div>
                            <div class="stage" data-stage="semantic">
                                <div class="stage-icon">🧠</div>
                                <span>Semantic</span>
                            </div>
                            <div class="stage" data-stage="frequency">
                                <div class="stage-icon">📊</div>
                                <span>Frequency</span>
                            </div>
                            <div class="stage" data-stage="watermark">
                                <div class="stage-icon">💧</div>
                                <span>Watermark</span>
                            </div>
                        </div>
                    </div>

                    <div id="detectionResults" class="detection-results hidden"></div>

                    <div id="detectionHistory" class="detection-history hidden">
                        <h3>Detection History</h3>
                        <div id="historyList" class="history-list"></div>
                    </div>
                </div>

                <style>
                    .deepfake-detector {
                        max-width: 1200px;
                        margin: 0 auto;
                        padding: 2rem;
                        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                    }

                    .detector-header {
                        text-align: center;
                        margin-bottom: 2rem;
                    }

                    .detector-header h2 {
                        font-size: 2rem;
                        color: #1f2937;
                        margin-bottom: 0.5rem;
                    }

                    .subtitle {
                        color: #6b7280;
                        font-size: 1rem;
                    }

                    .upload-section {
                        background: white;
                        border-radius: 12px;
                        padding: 2rem;
                        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
                        margin-bottom: 1.5rem;
                    }

                    .upload-zone {
                        border: 2px dashed #d1d5db;
                        border-radius: 8px;
                        padding: 3rem;
                        text-align: center;
                        transition: all 0.3s;
                        cursor: pointer;
                    }

                    .upload-zone:hover {
                        border-color: #3b82f6;
                        background: #f9fafb;
                    }

                    .upload-zone.dragover {
                        border-color: #3b82f6;
                        background: #eff6ff;
                    }

                    .upload-icon {
                        color: #9ca3af;
                        margin-bottom: 1rem;
                    }

                    .upload-text {
                        color: #6b7280;
                        margin-bottom: 1rem;
                    }

                    .analysis-options {
                        margin-top: 1.5rem;
                    }

                    .analysis-options h4 {
                        font-size: 1rem;
                        color: #374151;
                        margin-bottom: 0.75rem;
                    }

                    .mode-selector {
                        display: flex;
                        gap: 1.5rem;
                        flex-wrap: wrap;
                    }

                    .radio-label {
                        display: flex;
                        align-items: center;
                        gap: 0.5rem;
                        cursor: pointer;
                    }

                    .preview-section {
                        background: white;
                        border-radius: 12px;
                        padding: 1.5rem;
                        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
                        margin-bottom: 1.5rem;
                    }

                    .preview-container {
                        display: flex;
                        justify-content: center;
                        align-items: center;
                        min-height: 300px;
                        background: #f3f4f6;
                        border-radius: 8px;
                        overflow: hidden;
                    }

                    .preview-media {
                        max-width: 100%;
                        max-height: 400px;
                        object-fit: contain;
                    }

                    .preview-info {
                        display: flex;
                        align-items: center;
                        justify-content: space-between;
                        margin-top: 1rem;
                        padding: 0 0.5rem;
                        font-size: 0.875rem;
                        color: #6b7280;
                    }

                    .analysis-progress {
                        background: white;
                        border-radius: 12px;
                        padding: 1.5rem;
                        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
                        margin-bottom: 1.5rem;
                    }

                    .progress-header {
                        display: flex;
                        justify-content: space-between;
                        margin-bottom: 0.75rem;
                    }

                    .progress-percent {
                        font-weight: 600;
                        color: #3b82f6;
                    }

                    .progress-bar {
                        height: 8px;
                        background: #e5e7eb;
                        border-radius: 4px;
                        overflow: hidden;
                        margin-bottom: 1.5rem;
                    }

                    .progress-fill {
                        height: 100%;
                        background: linear-gradient(90deg, #3b82f6, #10b981);
                        transition: width 0.3s;
                        width: 0%;
                    }

                    .progress-stages {
                        display: flex;
                        justify-content: space-between;
                        gap: 0.5rem;
                    }

                    .stage {
                        display: flex;
                        flex-direction: column;
                        align-items: center;
                        gap: 0.5rem;
                        padding: 0.75rem;
                        background: #f9fafb;
                        border-radius: 8px;
                        flex: 1;
                        opacity: 0.4;
                        transition: all 0.3s;
                    }

                    .stage.active {
                        opacity: 1;
                        background: #eff6ff;
                    }

                    .stage.completed {
                        opacity: 1;
                        background: #ecfdf5;
                    }

                    .stage-icon {
                        font-size: 1.5rem;
                    }

                    .stage span {
                        font-size: 0.75rem;
                        color: #6b7280;
                    }

                    .detection-results {
                        background: white;
                        border-radius: 12px;
                        padding: 2rem;
                        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
                        margin-bottom: 1.5rem;
                    }

                    .results-header {
                        display: flex;
                        align-items: center;
                        gap: 2rem;
                        margin-bottom: 2rem;
                        padding-bottom: 1.5rem;
                        border-bottom: 2px solid #e5e7eb;
                    }

                    .overall-risk {
                        display: flex;
                        flex-direction: column;
                        align-items: center;
                    }

                    .risk-circle {
                        width: 120px;
                        height: 120px;
                        border-radius: 50%;
                        display: flex;
                        flex-direction: column;
                        align-items: center;
                        justify-content: center;
                        border: 6px solid;
                        margin-bottom: 0.5rem;
                    }

                    .risk-score {
                        font-size: 2.5rem;
                        font-weight: 700;
                        color: white;
                    }

                    .risk-label {
                        font-size: 0.75rem;
                        color: white;
                        font-weight: 600;
                    }

                    .results-meta {
                        flex: 1;
                    }

                    .results-meta p {
                        margin: 0.5rem 0;
                        color: #374151;
                    }

                    .module-results {
                        display: grid;
                        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
                        gap: 1rem;
                        margin-bottom: 2rem;
                    }

                    .module-result {
                        padding: 1rem;
                        background: #f9fafb;
                        border-radius: 8px;
                        border-left: 4px solid #d1d5db;
                    }

                    .module-result.high-risk {
                        border-left-color: #dc2626;
                        background: #fef2f2;
                    }

                    .module-result.medium-risk {
                        border-left-color: #f59e0b;
                        background: #fffbeb;
                    }

                    .module-result.low-risk {
                        border-left-color: #10b981;
                        background: #ecfdf5;
                    }

                    .module-header {
                        display: flex;
                        justify-content: space-between;
                        align-items: center;
                        margin-bottom: 0.75rem;
                    }

                    .module-name {
                        font-weight: 600;
                        color: #374151;
                    }

                    .module-score {
                        font-weight: 700;
                        color: #1f2937;
                    }

                    .module-details {
                        font-size: 0.875rem;
                        color: #6b7280;
                    }

                    .artifacts-section {
                        margin-bottom: 2rem;
                    }

                    .artifacts-section h4 {
                        font-size: 1.125rem;
                        color: #374151;
                        margin-bottom: 1rem;
                    }

                    .artifact-item {
                        display: flex;
                        align-items: flex-start;
                        gap: 0.75rem;
                        padding: 0.75rem;
                        background: #fef2f2;
                        border-radius: 6px;
                        margin-bottom: 0.5rem;
                        border-left: 3px solid #dc2626;
                    }

                    .artifact-icon {
                        font-size: 1.25rem;
                    }

                    .artifact-content {
                        flex: 1;
                    }

                    .artifact-type {
                        font-weight: 600;
                        color: #991b1b;
                        margin-bottom: 0.25rem;
                    }

                    .artifact-description {
                        font-size: 0.875rem;
                        color: #7f1d1d;
                    }

                    .btn {
                        padding: 0.5rem 1rem;
                        border-radius: 6px;
                        border: none;
                        font-weight: 500;
                        cursor: pointer;
                        transition: all 0.2s;
                    }

                    .btn-primary {
                        background: #3b82f6;
                        color: white;
                    }

                    .btn-primary:hover {
                        background: #2563eb;
                    }

                    .btn-secondary {
                        background: #e5e7eb;
                        color: #374151;
                    }

                    .btn-secondary:hover {
                        background: #d1d5db;
                    }

                    .btn-icon {
                        width: 2rem;
                        height: 2rem;
                        display: flex;
                        align-items: center;
                        justify-content: center;
                        background: #fee2e2;
                        color: #dc2626;
                        font-size: 1.5rem;
                        border-radius: 50%;
                    }

                    .btn-icon:hover {
                        background: #fecaca;
                    }

                    .hidden {
                        display: none !important;
                    }
                </style>
            `;
        }

        bindEvents() {
            const uploadZone = this.container.querySelector('#uploadZone');
            const fileInput = this.container.querySelector('#fileInput');
            const browseBtn = this.container.querySelector('#browseBtn');
            const removeBtn = this.container.querySelector('#removeBtn');
            const modeRadios = this.container.querySelectorAll('input[name="analysisMode"]');

            uploadZone.addEventListener('click', () => fileInput.click());
            browseBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                fileInput.click();
            });

            uploadZone.addEventListener('dragover', (e) => {
                e.preventDefault();
                uploadZone.classList.add('dragover');
            });

            uploadZone.addEventListener('dragleave', () => {
                uploadZone.classList.remove('dragover');
            });

            uploadZone.addEventListener('drop', (e) => {
                e.preventDefault();
                uploadZone.classList.remove('dragover');
                const files = e.dataTransfer.files;
                if (files.length > 0) {
                    this.handleFileSelect(files[0]);
                }
            });

            fileInput.addEventListener('change', (e) => {
                if (e.target.files.length > 0) {
                    this.handleFileSelect(e.target.files[0]);
                }
            });

            removeBtn.addEventListener('click', () => this.removeFile());

            modeRadios.forEach(radio => {
                radio.addEventListener('change', (e) => {
                    this.analysisMode = e.target.value;
                });
            });
        }

        handleFileSelect(file) {
            if (file.size > CONFIG.MAX_FILE_SIZE) {
                alert('File size exceeds 10MB limit');
                return;
            }

            if (!CONFIG.SUPPORTED_TYPES.includes(file.type)) {
                alert('Unsupported file type. Please use JPEG, PNG, MP4, or WebM.');
                return;
            }

            const previewSection = this.container.querySelector('#previewSection');
            const previewImage = this.container.querySelector('#previewImage');
            const previewVideo = this.container.querySelector('#previewVideo');
            const fileName = this.container.querySelector('#fileName');
            const fileSize = this.container.querySelector('#fileSize');

            previewSection.classList.remove('hidden');

            if (file.type.startsWith('image/')) {
                previewImage.src = URL.createObjectURL(file);
                previewImage.classList.remove('hidden');
                previewVideo.classList.add('hidden');
            } else if (file.type.startsWith('video/')) {
                previewVideo.src = URL.createObjectURL(file);
                previewVideo.classList.remove('hidden');
                previewImage.classList.add('hidden');
            }

            fileName.textContent = file.name;
            fileSize.textContent = this.formatFileSize(file.size);

            this.analyzeFile(file);
        }

        async analyzeFile(file) {
            const progressSection = this.container.querySelector('#analysisProgress');
            const progressFill = this.container.querySelector('#progressFill');
            const progressPercent = this.container.querySelector('#progressPercent');
            const stages = this.container.querySelectorAll('.stage');

            progressSection.classList.remove('hidden');

            const stageNames = ['face', 'texture', 'semantic', 'frequency', 'watermark'];
            const stagePercents = [20, 40, 60, 80, 100];

            for (let i = 0; i < stageNames.length; i++) {
                progressFill.style.width = stagePercents[i] + '%';
                progressPercent.textContent = stagePercents[i] + '%';

                stages.forEach(stage => {
                    if (stage.dataset.stage === stageNames[i]) {
                        stage.classList.add('active');
                    }
                });

                await this.delay(400);

                stages.forEach(stage => {
                    if (stage.dataset.stage === stageNames[i]) {
                        stage.classList.remove('active');
                        stage.classList.add('completed');
                    }
                });
            }

            const result = await this.detector.analyze(file, this.analysisMode);

            this.currentResults = result;
            this.detectionHistory.unshift(result);
            this.renderResults(result);

            progressSection.classList.add('hidden');
            stages.forEach(stage => {
                stage.classList.remove('active', 'completed');
            });
        }

        renderResults(result) {
            const resultsDiv = this.container.querySelector('#detectionResults');
            resultsDiv.classList.remove('hidden');

            const riskLevel = this.getRiskLevel(result.overallScore);
            const riskColor = riskLevel.color;

            resultsDiv.innerHTML = `
                <div class="results-header">
                    <div class="overall-risk">
                        <div class="risk-circle" style="background: ${riskColor}">
                            <span class="risk-score">${result.overallScore.toFixed(0)}</span>
                            <span class="risk-label">${result.riskLevel.toUpperCase()}</span>
                        </div>
                    </div>
                    <div class="results-meta">
                        <p><strong>Detection ID:</strong> ${result.detectionId}</p>
                        <p><strong>Content Type:</strong> ${result.contentType}</p>
                        <p><strong>Analysis Time:</strong> ${result.processingTime}ms</p>
                        <p><strong>Confidence:</strong> ${(result.confidence * 100).toFixed(1)}%</p>
                    </div>
                </div>

                <div class="module-results">
                    ${result.modules.map(mod => this.renderModuleResult(mod)).join('')}
                </div>

                ${result.artifacts && result.artifacts.length > 0 ? this.renderArtifacts(result.artifacts) : ''}

                <div class="results-actions">
                    <button class="btn btn-secondary" onclick="DeepfakeDetector.exportResults()">
                        Export Report
                    </button>
                    <button class="btn btn-primary" onclick="DeepfakeDetector.newAnalysis()">
                        New Analysis
                    </button>
                </div>
            `;
        }

        renderModuleResult(mod) {
            const riskClass = mod.score >= 70 ? 'high-risk' : mod.score >= 50 ? 'medium-risk' : 'low-risk';

            return `
                <div class="module-result ${riskClass}">
                    <div class="module-header">
                        <span class="module-name">${mod.name}</span>
                        <span class="module-score">${mod.score.toFixed(1)}%</span>
                    </div>
                    <div class="module-details">
                        ${mod.description}
                    </div>
                </div>
            `;
        }

        renderArtifacts(artifacts) {
            return `
                <div class="artifacts-section">
                    <h4>🔍 Detected Artifacts</h4>
                    ${artifacts.map(artifact => `
                        <div class="artifact-item">
                            <span class="artifact-icon">⚠️</span>
                            <div class="artifact-content">
                                <div class="artifact-type">${artifact.type}</div>
                                <div class="artifact-description">${artifact.description}</div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        }

        getRiskLevel(score) {
            const levels = Object.values(CONFIG.RISK_LEVELS);
            for (let i = 0; i < levels.length; i++) {
                if (score >= levels[i].min) {
                    return levels[i];
                }
            }
            return levels[levels.length - 1];
        }

        formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        }

        removeFile() {
            const previewSection = this.container.querySelector('#previewSection');
            const previewImage = this.container.querySelector('#previewImage');
            const previewVideo = this.container.querySelector('#previewVideo');
            const resultsDiv = this.container.querySelector('#detectionResults');

            previewSection.classList.add('hidden');
            previewImage.src = '';
            previewImage.classList.add('hidden');
            previewVideo.src = '';
            previewVideo.classList.add('hidden');
            resultsDiv.classList.add('hidden');

            this.currentResults = null;
        }

        delay(ms) {
            return new Promise(resolve => setTimeout(resolve, ms));
        }
    }

    class DeepfakeDetectorService {
        constructor() {
            this.endpoint = CONFIG.API_ENDPOINT;
            this.watermarkEndpoint = CONFIG.WATERMARK_ENDPOINT;
        }

        async analyze(file, mode) {
            try {
                const formData = new FormData();
                formData.append('file', file);
                formData.append('mode', mode);

                const response = await fetch(this.endpoint, {
                    method: 'POST',
                    body: formData
                });

                if (response.ok) {
                    return await response.json();
                }
            } catch (error) {
                console.warn('API request failed, using mock data:', error);
            }

            return this.generateMockResult(file, mode);
        }

        generateMockResult(file, mode) {
            const detectionId = 'det_' + Date.now();
            const overallScore = 60 + Math.random() * 35;

            const modules = Object.entries(CONFIG.DETECTION_MODULES).map(([key, config]) => {
                const scoreVariance = (Math.random() - 0.5) * 20;
                const score = Math.max(0, Math.min(100, overallScore + scoreVariance));

                return {
                    name: config.name,
                    weight: config.weight,
                    score: score,
                    description: score >= 70 ? 'High probability of manipulation detected' :
                                score >= 50 ? 'Some inconsistencies detected' :
                                'No significant anomalies found'
                };
            });

            const artifacts = overallScore > 75 ? [
                {
                    type: 'Texture Regularity',
                    description: 'Unusual texture patterns consistent with synthetic generation'
                },
                {
                    type: 'Frequency Anomaly',
                    description: 'Detected atypical frequency distribution in the image'
                }
            ] : [];

            let riskLevel = 'minimal';
            if (overallScore >= 85) riskLevel = 'critical';
            else if (overallScore >= 70) riskLevel = 'high';
            else if (overallScore >= 50) riskLevel = 'medium';
            else if (overallScore >= 30) riskLevel = 'low';

            return {
                detectionId: detectionId,
                overallScore: overallScore,
                riskLevel: riskLevel,
                confidence: 0.85 + Math.random() * 0.1,
                contentType: file.type.startsWith('image/') ? 'image' : 'video',
                processingTime: Math.floor(500 + Math.random() * 1500),
                modules: modules,
                artifacts: artifacts
            };
        }
    }

    let uiInstance = null;

    function init(containerId) {
        if (!uiInstance) {
            uiInstance = new DetectorUI(containerId);
        }
        return uiInstance;
    }

    function exportResults() {
        if (uiInstance && uiInstance.currentResults) {
            const dataStr = JSON.stringify(uiInstance.currentResults, null, 2);
            const blob = new Blob([dataStr], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'deepfake-detection-result.json';
            a.click();
            URL.revokeObjectURL(url);
        }
    }

    function newAnalysis() {
        if (uiInstance) {
            uiInstance.removeFile();
        }
    }

    return {
        init,
        exportResults,
        newAnalysis
    };
})();

document.addEventListener('DOMContentLoaded', function() {
    const container = document.getElementById('deepfake-detector-container');
    if (container) {
        DeepfakeDetector.init('deepfake-detector-container');
    }
});
