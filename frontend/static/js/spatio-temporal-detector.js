class SpatioTemporalDetector {
    constructor(options = {}) {
        this.options = {
            apiBase: '/api/v1/captcha/spatio-temporal',
            trackingInterval: 1000,
            minPoints: 10,
            predictionWindow: 3600,
            riskThreshold: 0.7,
            ...options
        };

        this.points = [];
        this.flow = null;
        this.riskScore = null;
        this.isTracking = false;
        this.trackingInterval = null;
        this.startTime = null;
    }

    async init() {
        this.setupGeolocation();
        this.setupEventListeners();
        return this;
    }

    setupGeolocation() {
        if ('geolocation' in navigator) {
            this.watchId = navigator.geolocation.watchPosition(
                (position) => this.handlePositionUpdate(position),
                (error) => this.handleGeolocationError(error),
                {
                    enableHighAccuracy: true,
                    timeout: 5000,
                    maximumAge: 0
                }
            );
        }
    }

    handlePositionUpdate(position) {
        const point = {
            timestamp: Date.now(),
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
            altitude: position.coords.altitude || 0,
            accuracy: this.getAccuracyLevel(position.coords.accuracy),
            confidence: this.calculateConfidence(position.coords),
            velocity: position.coords.speed || 0,
            heading: position.coords.heading || 0
        };

        this.points.push(point);
        this.trimPoints();
        this.updateFlow();
    }

    handleGeolocationError(error) {
        console.error('Geolocation error:', error.message);
        switch (error.code) {
            case error.PERMISSION_DENIED:
                this.emit('error', { type: 'permission_denied', message: '位置权限被拒绝' });
                break;
            case error.POSITION_UNAVAILABLE:
                this.emit('error', { type: 'unavailable', message: '位置信息不可用' });
                break;
            case error.TIMEOUT:
                this.emit('error', { type: 'timeout', message: '获取位置超时' });
                break;
        }
    }

    getAccuracyLevel(accuracy) {
        if (accuracy <= 10) return 'gps';
        if (accuracy <= 100) return 'city';
        if (accuracy <= 1000) return 'region';
        return 'ip';
    }

    calculateConfidence(coords) {
        let confidence = 0.5;
        if (coords.accuracy <= 10) confidence += 0.3;
        else if (coords.accuracy <= 50) confidence += 0.2;
        else if (coords.accuracy <= 100) confidence += 0.1;

        if (coords.altitude !== null) confidence += 0.1;
        if (coords.heading !== null) confidence += 0.05;
        if (coords.speed !== null) confidence += 0.05;

        return Math.min(1.0, confidence);
    }

    trimPoints() {
        const maxPoints = 1000;
        if (this.points.length > maxPoints) {
            this.points = this.points.slice(-maxPoints);
        }
    }

    updateFlow() {
        if (this.points.length < 2) return;

        const trajectory = this.buildTrajectory();
        const anomalies = this.detectAnomalies(trajectory);
        const riskScore = this.calculateRiskScore(trajectory, anomalies);

        this.flow = {
            flow_id: `flow_${Date.now()}`,
            user_id: this.options.userId || 'anonymous',
            points: this.points,
            start_time: this.startTime || Date.now(),
            end_time: Date.now(),
            total_distance: this.calculateTotalDistance(),
            avg_velocity: this.calculateAvgVelocity(),
            max_velocity: this.calculateMaxVelocity(),
            min_velocity: this.calculateMinVelocity(),
            trajectory: trajectory,
            anomalies: anomalies,
            risk_score: riskScore
        };

        this.emit('flowUpdate', this.flow);
    }

    buildTrajectory() {
        const trajectory = [];

        for (let i = 0; i < this.points.length; i++) {
            const p = this.points[i];
            const tp = {
                timestamp: p.timestamp,
                x: p.latitude,
                y: p.longitude,
                z: p.altitude,
                velocity: p.velocity,
                direction: p.heading
            };

            if (i > 0) {
                const prev = this.points[i - 1];
                const dt = (p.timestamp - prev.timestamp) / 1000;
                if (dt > 0) {
                    const dist = this.haversineDistance(
                        prev.latitude, prev.longitude,
                        p.latitude, p.longitude
                    );
                    tp.velocity = (dist / dt) * 3.6;
                }

                if (i > 1) {
                    const prevTraj = trajectory[i - 1];
                    const dt = (p.timestamp - prev.timestamp) / 1000;
                    if (dt > 0 && prevTraj.velocity > 0) {
                        tp.acceleration = (tp.velocity - prevTraj.velocity) / dt;
                    }

                    if (i > 2) {
                        const prevAcc = prevTraj.acceleration;
                        if (prevAcc !== undefined && dt > 0) {
                            tp.jerk = (tp.acceleration - prevAcc) / dt;
                        }
                    }
                }
            }

            trajectory.push(tp);
        }

        return trajectory;
    }

    detectAnomalies(trajectory) {
        const anomalies = [];

        for (let i = 2; i < trajectory.length; i++) {
            const tp = trajectory[i];

            if (tp.acceleration !== undefined && Math.abs(tp.acceleration) > 50) {
                anomalies.push({
                    anomaly_id: `accel_${Date.now()}_${i}`,
                    anomaly_type: 'high_acceleration',
                    timestamp: tp.timestamp,
                    location: [tp.x, tp.y],
                    severity: Math.min(1.0, Math.abs(tp.acceleration) / 100),
                    description: '检测到异常加速度',
                    confidence: 0.85,
                    risk_contribution: 0.3
                });
            }

            if (tp.jerk !== undefined && Math.abs(tp.jerk) > 20) {
                anomalies.push({
                    anomaly_id: `jerk_${Date.now()}_${i}`,
                    anomaly_type: 'high_jerk',
                    timestamp: tp.timestamp,
                    location: [tp.x, tp.y],
                    severity: Math.min(1.0, Math.abs(tp.jerk) / 50),
                    description: '检测到运动不平滑',
                    confidence: 0.75,
                    risk_contribution: 0.2
                });
            }
        }

        for (let i = 1; i < trajectory.length - 1; i++) {
            const angle = this.calculateTurnAngle(
                trajectory[i - 1],
                trajectory[i],
                trajectory[i + 1]
            );

            if (angle > 150 || angle < -150) {
                anomalies.push({
                    anomaly_id: `sharp_turn_${Date.now()}_${i}`,
                    anomaly_type: 'sharp_turn',
                    timestamp: trajectory[i].timestamp,
                    location: [trajectory[i].x, trajectory[i].y],
                    severity: 0.6,
                    description: '检测到急转弯',
                    confidence: 0.7,
                    risk_contribution: 0.15
                });
            }
        }

        return anomalies;
    }

    calculateTurnAngle(p1, p2, p3) {
        const v1 = [p2.x - p1.x, p2.y - p1.y];
        const v2 = [p3.x - p2.x, p3.y - p2.y];

        const dot = v1[0] * v2[0] + v1[1] * v2[1];
        const mag1 = Math.sqrt(v1[0] * v1[0] + v1[1] * v1[1]);
        const mag2 = Math.sqrt(v2[0] * v2[0] + v2[1] * v2[1]);

        if (mag1 === 0 || mag2 === 0) return 0;

        let cosAngle = dot / (mag1 * mag2);
        cosAngle = Math.max(-1, Math.min(1, cosAngle));

        return Math.acos(cosAngle) * 180 / Math.PI;
    }

    calculateRiskScore(trajectory, anomalies) {
        let score = 0.5;

        if (trajectory.length > 0) {
            const velocities = trajectory.map(t => t.velocity).filter(v => v > 0);
            if (velocities.length > 0) {
                const maxVel = Math.max(...velocities);
                const avgVel = velocities.reduce((a, b) => a + b, 0) / velocities.length;

                if (maxVel > 200) score += 0.2;
                if (avgVel > 100) score += 0.1;
            }
        }

        let anomalyContribution = 0;
        for (const a of anomalies) {
            anomalyContribution += a.severity * a.risk_contribution;
        }
        score += anomalyContribution;

        return Math.min(1.0, Math.max(0, score));
    }

    calculateTotalDistance() {
        let total = 0;
        for (let i = 1; i < this.points.length; i++) {
            total += this.haversineDistance(
                this.points[i - 1].latitude, this.points[i - 1].longitude,
                this.points[i].latitude, this.points[i].longitude
            );
        }
        return total;
    }

    calculateAvgVelocity() {
        const velocities = this.points
            .map(p => p.velocity)
            .filter(v => v > 0);
        if (velocities.length === 0) return 0;
        return velocities.reduce((a, b) => a + b, 0) / velocities.length;
    }

    calculateMaxVelocity() {
        return Math.max(...this.points.map(p => p.velocity || 0));
    }

    calculateMinVelocity() {
        const velocities = this.points
            .map(p => p.velocity)
            .filter(v => v > 0);
        if (velocities.length === 0) return 0;
        return Math.min(...velocities);
    }

    haversineDistance(lat1, lon1, lat2, lon2) {
        const R = 6371;
        const dLat = this.toRad(lat2 - lat1);
        const dLon = this.toRad(lon2 - lon1);
        const a =
            Math.sin(dLat / 2) * Math.sin(dLat / 2) +
            Math.cos(this.toRad(lat1)) * Math.cos(this.toRad(lat2)) *
            Math.sin(dLon / 2) * Math.sin(dLon / 2);
        const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
        return R * c;
    }

    toRad(deg) {
        return deg * Math.PI / 180;
    }

    startTracking() {
        if (this.isTracking) return;

        this.isTracking = true;
        this.startTime = Date.now();
        this.points = [];

        this.trackingInterval = setInterval(() => {
            this.updateFlow();
        }, this.options.trackingInterval);

        this.emit('trackingStarted', { startTime: this.startTime });
    }

    stopTracking() {
        if (!this.isTracking) return;

        this.isTracking = false;
        if (this.trackingInterval) {
            clearInterval(this.trackingInterval);
            this.trackingInterval = null;
        }

        this.updateFlow();
        this.emit('trackingStopped', { flow: this.flow });
    }

    async predictTrajectory(historicalData, predictionSteps = 5) {
        try {
            const response = await fetch(`${this.options.apiBase}/predict`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    user_id: this.options.userId || 'anonymous',
                    historical_data: historicalData || this.points,
                    current_location: this.points.length > 0 ? 
                        [this.points[this.points.length - 1].latitude, this.points[this.points.length - 1].longitude] : null,
                    current_time: Date.now(),
                    prediction_steps: predictionSteps,
                    prediction_method: 'kalman_filter'
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                return result.data;
            }
            return null;
        } catch (error) {
            console.error('Prediction error:', error);
            return this.localPrediction(predictionSteps);
        }
    }

    localPrediction(steps) {
        if (this.points.length < 2) return null;

        const predictions = [];
        const lastPoint = this.points[this.points.length - 1];
        const secondLast = this.points[this.points.length - 2];

        const dt = (lastPoint.timestamp - secondLast.timestamp) / 1000;
        let latSlope = 0, lngSlope = 0;
        if (dt > 0) {
            latSlope = (lastPoint.latitude - secondLast.latitude) / dt;
            lngSlope = (lastPoint.longitude - secondLast.longitude) / dt;
        }

        for (let i = 1; i <= steps; i++) {
            const window = i * 600;
            predictions.push({
                prediction_id: `pred_${Date.now()}_${i}`,
                user_id: this.options.userId || 'anonymous',
                predicted_location: [
                    lastPoint.latitude + latSlope * window,
                    lastPoint.longitude + lngSlope * window
                ],
                predicted_time: Date.now() + window,
                prediction_window: window,
                confidence: Math.max(0.3, 0.9 - i * 0.1),
                method: 'linear_extrapolation',
                features: {},
                trajectory: this.buildTrajectory().slice(-10),
                anomaly_indicators: []
            });
        }

        return {
            predictions: predictions,
            confidence: 0.7,
            method: 'linear_extrapolation',
            model_version: 'v1.0'
        };
    }

    async assessRisk(behaviorData) {
        try {
            const response = await fetch(`${this.options.apiBase}/risk-assess`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    user_id: this.options.userId || 'anonymous',
                    behavior_data: behaviorData || this.flow,
                    context_data: {
                        client_ip: await this.getClientIP(),
                        user_agent: navigator.userAgent
                    },
                    threshold: this.options.riskThreshold
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                this.riskScore = result.data.risk_score;
                return result.data;
            }
            return this.localRiskAssessment();
        } catch (error) {
            console.error('Risk assessment error:', error);
            return this.localRiskAssessment();
        }
    }

    localRiskAssessment() {
        const locationScore = 0.7 + Math.random() * 0.3;
        const timeScore = 0.6 + Math.random() * 0.4;
        const behaviorScore = 0.5 + Math.random() * 0.5;
        const velocityScore = 0.6 + Math.random() * 0.4;

        const overallScore = locationScore * 0.25 + timeScore * 0.2 + 
                            behaviorScore * 0.25 + velocityScore * 0.15;

        let riskLevel = 'low';
        const riskFactors = [];

        if (overallScore > 0.8) {
            riskLevel = 'low';
        } else if (overallScore > 0.6) {
            riskLevel = 'medium';
            riskFactors.push('slight_deviation_detected');
        } else if (overallScore > 0.4) {
            riskLevel = 'high';
            riskFactors.push('significant_deviation', 'possible_automation');
        } else {
            riskLevel = 'critical';
            riskFactors.push('high_risk_behavior');
        }

        if (this.flow && this.flow.anomalies && this.flow.anomalies.length > 0) {
            riskFactors.push(`${this.flow.anomalies.length}_anomalies_detected`);
        }

        return {
            assessment_id: `ar_${Date.now()}`,
            risk_score: {
                score_id: `rs_${Date.now()}`,
                user_id: this.options.userId || 'anonymous',
                overall_score: overallScore,
                location_score: locationScore,
                time_score: timeScore,
                behavior_score: behaviorScore,
                velocity_score: velocityScore,
                risk_level: riskLevel,
                risk_factors: riskFactors,
                recommendations: this.generateRecommendations(riskLevel),
                calculated_at: Date.now(),
                valid_until: Date.now() + 300000
            },
            anomalies: this.flow ? this.flow.anomalies : [],
            recommendations: this.generateRecommendations(riskLevel),
            factors: {
                velocity_factor: this.flow ? this.flow.avg_velocity / 100 : 0,
                distance_factor: this.flow ? this.flow.total_distance / 1000 : 0,
                anomaly_factor: this.flow ? this.flow.anomalies.length / 10 : 0
            }
        };
    }

    generateRecommendations(riskLevel) {
        const recommendations = {
            critical: ['立即进行人工审核', '要求额外身份验证', '限制账户操作'],
            high: ['加强监控', '要求多因素认证', '审查最近活动'],
            medium: ['建议启用增强安全', '记录审计日志'],
            low: ['保持当前安全策略']
        };
        return recommendations[riskLevel] || recommendations.low;
    }

    async getClientIP() {
        try {
            const response = await fetch('https://api.ipify.org?format=json');
            const data = await response.json();
            return data.ip;
        } catch {
            return 'unknown';
        }
    }

    setupEventListeners() {
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.emit('pageHidden');
            } else {
                this.emit('pageVisible');
            }
        });
    }

    emit(event, data) {
        if (typeof this.options.onEvent === 'function') {
            this.options.onEvent(event, data);
        }
    }

    getPoints() {
        return [...this.points];
    }

    getFlow() {
        return this.flow;
    }

    getRiskScore() {
        return this.riskScore;
    }

    getStatus() {
        return {
            isTracking: this.isTracking,
            pointCount: this.points.length,
            hasFlow: this.flow !== null,
            hasRiskScore: this.riskScore !== null,
            startTime: this.startTime
        };
    }

    reset() {
        this.stopTracking();
        this.points = [];
        this.flow = null;
        this.riskScore = null;
        this.startTime = null;
    }

    destroy() {
        this.reset();
        if (this.watchId !== undefined) {
            navigator.geolocation.clearWatch(this.watchId);
        }
    }
}

class SpatioTemporalCaptcha {
    constructor(options = {}) {
        this.options = {
            apiBase: '/api/v1/captcha/spatio-temporal',
            difficulty: 'medium',
            enablePredictions: true,
            predictionWindow: 3600,
            onSuccess: null,
            onError: null,
            ...options
        };

        this.sessionData = null;
        this.detector = null;
        this.selectedOption = null;
    }

    async init() {
        this.detector = new SpatioTemporalDetector({
            userId: this.options.userId,
            predictionWindow: this.options.predictionWindow
        });
        await this.detector.init();
        await this.generateCaptcha();
        return this;
    }

    async generateCaptcha() {
        try {
            const currentLocation = this.detector.points.length > 0 ? 
                this.detector.points[this.detector.points.length - 1] : null;

            const response = await fetch(`${this.options.apiBase}/create`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    user_id: this.options.userId || 'anonymous',
                    pattern_type: 'daily',
                    difficulty: this.options.difficulty,
                    client_ip: await this.detector.getClientIP(),
                    user_agent: navigator.userAgent,
                    current_location: currentLocation,
                    include_predictions: this.options.enablePredictions,
                    prediction_window: this.options.predictionWindow
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                this.sessionData = result.data;
                this.renderCaptcha();
            } else {
                this.showError('生成验证码失败');
            }
        } catch (error) {
            console.error('Generate error:', error);
            this.showError('网络错误，请重试');
        }
    }

    renderCaptcha() {
        const container = document.getElementById('spatio-temporal-container');
        if (!container) return;

        container.innerHTML = `
            <div class="st-captcha-wrapper">
                <div class="st-header">
                    <h3>时空行为验证</h3>
                    <p class="instruction">${this.sessionData.instructions}</p>
                </div>

                <div class="st-map-container" id="st-map"></div>

                <div class="st-options" id="st-options">
                    ${this.sessionData.options.map((opt, idx) => `
                        <button class="st-option" data-option="${opt.option_id}" data-index="${idx}">
                            <span class="option-marker">${String.fromCharCode(65 + idx)}</span>
                            <span class="option-info">
                                <span class="option-coords">${opt.point.latitude.toFixed(4)}, ${opt.point.longitude.toFixed(4)}</span>
                                <span class="option-accuracy">精度: ${opt.point.accuracy}</span>
                            </span>
                        </button>
                    `).join('')}
                </div>

                ${this.sessionData.risk_score ? `
                    <div class="st-risk-display">
                        <div class="risk-indicator ${this.sessionData.risk_score.risk_level}">
                            <span class="risk-label">风险等级</span>
                            <span class="risk-value">${this.sessionData.risk_score.risk_level}</span>
                        </div>
                        <div class="risk-score-bar">
                            <div class="risk-score-fill" style="width: ${this.sessionData.risk_score.overall_score * 100}%"></div>
                        </div>
                    </div>
                ` : ''}

                <div class="st-actions">
                    <button id="st-refresh-btn" class="btn btn-secondary">
                        <i class="fas fa-sync-alt"></i> 重新生成
                    </button>
                    <button id="st-verify-btn" class="btn btn-primary" disabled>
                        <i class="fas fa-check"></i> 验证
                    </button>
                </div>

                <div class="st-result" id="st-result"></div>

                ${this.sessionData.prediction ? `
                    <div class="st-prediction-panel">
                        <h4>轨迹预测</h4>
                        <div class="prediction-list">
                            ${this.sessionData.prediction.anomaly_indicators.map(ind => 
                                `<span class="prediction-alert">${ind}</span>`
                            ).join('')}
                        </div>
                    </div>
                ` : ''}
            </div>
        `;

        this.setupEventListeners();
        this.initMap();
    }

    initMap() {
        const mapContainer = document.getElementById('st-map');
        if (!mapContainer) return;

        const pattern = this.sessionData.target_pattern;
        if (!pattern || !pattern.centroid) return;

        mapContainer.innerHTML = `
            <div class="map-placeholder">
                <div class="centroid-marker" style="left: 50%; top: 50%;"></div>
                ${this.sessionData.options.map((opt, idx) => {
                    const offsetX = (opt.point.longitude - pattern.centroid[1]) * 100;
                    const offsetY = -(opt.point.latitude - pattern.centroid[0]) * 100;
                    return `<div class="option-marker ${idx}" style="left: calc(50% + ${offsetX}px); top: calc(50% + ${offsetY}px);">${String.fromCharCode(65 + idx)}</div>`;
                }).join('')}
            </div>
        `;
    }

    setupEventListeners() {
        const options = document.querySelectorAll('.st-option');
        options.forEach(opt => {
            opt.addEventListener('click', () => {
                options.forEach(o => o.classList.remove('selected'));
                opt.classList.add('selected');
                this.selectedOption = opt.dataset.option;
                document.getElementById('st-verify-btn').disabled = false;
            });
        });

        document.getElementById('st-refresh-btn')?.addEventListener('click', () => {
            this.generateCaptcha();
        });

        document.getElementById('st-verify-btn')?.addEventListener('click', () => {
            this.verify();
        });
    }

    async verify() {
        if (!this.selectedOption) {
            this.showError('请选择一个选项');
            return;
        }

        try {
            const response = await fetch(`${this.options.apiBase}/verify`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    session_id: this.sessionData.session_id,
                    selected_option: this.selectedOption,
                    user_location: this.detector.points.length > 0 ? 
                        this.detector.points[this.detector.points.length - 1] : null,
                    response_time: Date.now() - (this.detector.startTime || Date.now()),
                    behavior_data: this.detector.getFlow()
                })
            });

            const result = await response.json();
            if (result.code === 0 && result.data) {
                if (result.data.success) {
                    this.showSuccess(result.data.message);
                    if (this.options.onSuccess) {
                        this.options.onSuccess(result.data);
                    }
                } else {
                    this.showError(result.data.message);
                    if (this.options.onError) {
                        this.options.onError(result.data);
                    }
                }
            } else {
                this.showError('验证失败，请重试');
            }
        } catch (error) {
            console.error('Verify error:', error);
            this.showError('网络错误，请重试');
        }
    }

    showSuccess(message) {
        const resultDiv = document.getElementById('st-result');
        if (resultDiv) {
            resultDiv.className = 'st-result success show';
            resultDiv.innerHTML = `<i class="fas fa-check-circle"></i> ${message}`;
        }
    }

    showError(message) {
        const resultDiv = document.getElementById('st-result');
        if (resultDiv) {
            resultDiv.className = 'st-result error show';
            resultDiv.innerHTML = `<i class="fas fa-exclamation-circle"></i> ${message}`;
        }
    }

    destroy() {
        if (this.detector) {
            this.detector.destroy();
        }
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { SpatioTemporalDetector, SpatioTemporalCaptcha };
}
