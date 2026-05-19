const AdaptiveAIEngine = (function() {
    'use strict';
    
    const CONFIG = {
        API_ENDPOINT: '/api/v1/ai/adaptive',
        RISK_THRESHOLDS: {
            CRITICAL: 85,
            HIGH: 70,
            MEDIUM: 50,
            LOW: 30
        },
        RETRY_ATTEMPTS: 3,
        RETRY_DELAY: 1000
    };

    class RiskAssessor {
        constructor() {
            this.featureWeights = {
                mouseVelocity: 0.15,
                clickTiming: 0.12,
                scrollPattern: 0.10,
                deviceFingerprint: 0.15,
                sessionBehavior: 0.12,
                networkPattern: 0.08,
                timeDistribution: 0.05,
                errorPattern: 0.05
            };
        }

        calculateMouseVelocity(traceData) {
            if (!traceData || !traceData.points || traceData.points.length < 2) {
                return 0.5;
            }

            let totalVelocity = 0;
            let count = 0;

            for (let i = 1; i < traceData.points.length; i++) {
                const dx = traceData.points[i].x - traceData.points[i - 1].x;
                const dy = traceData.points[i].y - traceData.points[i - 1].y;
                const dist = Math.sqrt(dx * dx + dy * dy);
                const dt = traceData.points[i].timestamp - traceData.points[i - 1].timestamp;

                if (dt > 0) {
                    const velocity = dist / dt;
                    totalVelocity += velocity;
                    count++;
                }
            }

            if (count === 0) return 0.5;

            const avgVelocity = totalVelocity / count;

            if (avgVelocity > 5) return 0.9;
            if (avgVelocity > 2) return 0.6;
            if (avgVelocity > 0.5) return 0.3;
            return 0.1;
        }

        calculateClickTiming(traceData) {
            if (!traceData || !traceData.points || traceData.points.length < 2) {
                return 0.5;
            }

            const intervals = [];
            for (let i = 1; i < traceData.points.length; i++) {
                if (traceData.points[i].event === 'click') {
                    intervals.push(traceData.points[i].timestamp - traceData.points[i - 1].timestamp);
                }
            }

            if (intervals.length < 2) return 0.5;

            const variance = this.calculateVariance(intervals);

            if (variance < 10) return 0.9;
            if (variance < 100) return 0.5;
            return 0.2;
        }

        calculateVariance(values) {
            if (values.length < 2) return 0;

            const mean = values.reduce((a, b) => a + b, 0) / values.length;
            const squaredDiffs = values.map(v => Math.pow(v - mean, 2));
            return squaredDiffs.reduce((a, b) => a + b, 0) / (values.length - 1);
        }

        evaluateDeviceFingerprint(deviceInfo) {
            if (!deviceInfo) return 0.5;

            let score = 0.5;

            if (deviceInfo.isProxy || deviceInfo.isVPN || deviceInfo.isTor) {
                score += 0.3;
            }

            if (deviceInfo.fingerprint && deviceInfo.fingerprint.length > 32) {
                score += 0.1;
            }

            if (deviceInfo.plugins && deviceInfo.plugins.length > 0) {
                score += 0.1;
            }

            return Math.min(score, 1.0);
        }

        evaluateSessionBehavior(sessionData) {
            if (!sessionData) return 0.5;

            let score = 0.5;

            if (sessionData.failureCount > 3) {
                score += 0.3;
            } else if (sessionData.failureCount > 1) {
                score += 0.1;
            }

            if (sessionData.verificationCount > 10) {
                score -= 0.2;
            }

            return Math.min(Math.max(score, 0), 1.0);
        }

        evaluateNetworkPattern(networkData) {
            if (!networkData || !networkData.reputation) return 0.5;

            switch (networkData.reputation) {
                case 'clean': return 0.2;
                case 'suspicious': return 0.6;
                case 'malicious': return 0.9;
                default: return 0.5;
            }
        }

        evaluateTimeDistribution() {
            const hour = new Date().getHours();

            if (hour >= 2 && hour <= 5) {
                return 0.7;
            }

            return 0.3;
        }

        evaluateErrorPattern(errorData) {
            if (!errorData || !errorData.failureCount) return 0.3;

            if (errorData.failureCount > 5) return 0.9;
            if (errorData.failureCount > 3) return 0.7;
            if (errorData.failureCount > 1) return 0.5;
            return 0.4;
        }

        computeWeightedScore(features) {
            let totalScore = 0;
            let totalWeight = 0;

            for (const [feature, value] of Object.entries(features)) {
                const weight = this.featureWeights[feature] || 0.1;
                totalScore += value * weight;
                totalWeight += weight;
            }

            if (totalWeight === 0) return 50.0;

            return (totalScore / totalWeight) * 100;
        }

        determineRiskLevel(score) {
            if (score >= CONFIG.RISK_THRESHOLDS.CRITICAL) return 'critical';
            if (score >= CONFIG.RISK_THRESHOLDS.HIGH) return 'high';
            if (score >= CONFIG.RISK_THRESHOLDS.MEDIUM) return 'medium';
            if (score >= CONFIG.RISK_THRESHOLDS.LOW) return 'low';
            return 'minimal';
        }

        identifyContributingFactors(features) {
            const factors = [];
            const threshold = 0.7;

            for (const [feature, value] of Object.entries(features)) {
                if (value >= threshold) {
                    const factorMap = {
                        mouseVelocity: '异常鼠标移动速度',
                        clickTiming: '规律的点击时序',
                        scrollPattern: '异常的滚动行为',
                        deviceFingerprint: '可疑的设备指纹',
                        sessionBehavior: '异常的会话行为',
                        networkPattern: '可疑的网络模式',
                        timeDistribution: '非正常访问时间',
                        errorPattern: '高失败率'
                    };

                    if (factorMap[feature]) {
                        factors.push(factorMap[feature]);
                    }
                }
            }

            return factors;
        }

        generateRecommendations(score, level, factors) {
            const recommendations = [];

            if (level === 'critical' || level === 'high') {
                recommendations.push('建议启用增强验证');
                recommendations.push('考虑添加多因素认证');
            } else if (level === 'medium') {
                recommendations.push('建议增加验证难度');
            }

            if (factors.length > 3) {
                recommendations.push('检测到多种异常行为，建议人工审核');
            }

            if (recommendations.length === 0) {
                recommendations.push('当前行为正常');
            }

            return recommendations;
        }

        calculateConfidence(features) {
            const dataPoints = Object.keys(features).length;
            const completeness = dataPoints / 8;

            const values = Object.values(features);
            const variance = this.calculateVariance(values);
            const consistency = 1 - Math.min(variance, 1);

            const confidence = (completeness * 0.6 + consistency * 0.4) * 100;

            return Math.min(Math.max(confidence, 0), 100);
        }

        assessRisk(traceData, deviceInfo, sessionData, networkData, errorData) {
            const features = {
                mouseVelocity: this.calculateMouseVelocity(traceData),
                clickTiming: this.calculateClickTiming(traceData),
                deviceFingerprint: this.evaluateDeviceFingerprint(deviceInfo),
                sessionBehavior: this.evaluateSessionBehavior(sessionData),
                networkPattern: this.evaluateNetworkPattern(networkData),
                timeDistribution: this.evaluateTimeDistribution(),
                errorPattern: this.evaluateErrorPattern(errorData),
                scrollPattern: 0.5
            };

            const score = this.computeWeightedScore(features);
            const level = this.determineRiskLevel(score);
            const factors = this.identifyContributingFactors(features);
            const recommendations = this.generateRecommendations(score, level, factors);
            const confidence = this.calculateConfidence(features);

            return {
                riskScore: score,
                riskLevel: level,
                confidence: confidence,
                contributingFactors: factors,
                recommendations: recommendations
            };
        }
    }

    class StrategyEngine {
        constructor() {
            this.strategies = {
                minimal: {
                    name: 'Minimal Verification',
                    difficulty: 1,
                    timeout: 30000,
                    maxAttempts: 3
                },
                standard: {
                    name: 'Standard Verification',
                    difficulty: 2,
                    timeout: 45000,
                    maxAttempts: 3
                },
                enhanced: {
                    name: 'Enhanced Verification',
                    difficulty: 3,
                    timeout: 60000,
                    maxAttempts: 2
                },
                critical: {
                    name: 'Critical Verification',
                    difficulty: 4,
                    timeout: 90000,
                    maxAttempts: 2
                }
            };
        }

        selectStrategyBasedOnRisk(riskResult) {
            switch (riskResult.riskLevel) {
                case 'critical': return 'critical';
                case 'high': return 'enhanced';
                case 'medium': return 'standard';
                default: return 'minimal';
            }
        }

        adjustStrategyForTrust(baseStrategy, trustScore) {
            if (trustScore > 0.9) return 'minimal';
            if (trustScore > 0.8) return 'standard';
            return baseStrategy;
        }

        adjustStrategyForContext(strategy, riskResult) {
            const adjusted = { ...strategy };

            if (riskResult.riskScore > 80) {
                adjusted.timeout = strategy.timeout * 0.8;
                adjusted.maxAttempts = Math.max(1, strategy.maxAttempts - 1);
            } else if (riskResult.riskScore < 30) {
                adjusted.timeout = strategy.timeout * 1.2;
            }

            return adjusted;
        }

        generateSpecialInstructions(strategy, riskResult) {
            const instructions = [];

            if (strategy.difficulty >= 3) {
                instructions.push('请注意验证时间限制');
                instructions.push('仔细观察图像细节');
            }

            if (riskResult.contributingFactors && riskResult.contributingFactors.length > 2) {
                instructions.push('系统检测到异常行为，请按正常方式完成验证');
            }

            return instructions;
        }

        determineStrategy(riskResult, userProfile) {
            let strategyKey = this.selectStrategyBasedOnRisk(riskResult);

            if (userProfile && userProfile.trustScore > 0.8) {
                strategyKey = this.adjustStrategyForTrust(strategyKey, userProfile.trustScore);
            }

            let strategy = this.strategies[strategyKey] || this.strategies.standard;
            strategy = this.adjustStrategyForContext(strategy, riskResult);

            return {
                strategy: strategy,
                recommendedDifficulty: strategy.difficulty,
                adjustedTimeout: strategy.timeout,
                specialInstructions: this.generateSpecialInstructions(strategy, riskResult),
                reasoning: `基于风险评分 ${riskResult.riskScore.toFixed(2)} (级别: ${riskResult.riskLevel}) 和置信度 ${riskResult.confidence.toFixed(2)}%`
            };
        }
    }

    class PersonalizationEngine {
        constructor() {
            this.profiles = {};
        }

        createDefaultProfile(userId) {
            return {
                userId: userId,
                interactionStyle: 'normal',
                preferredCaptcha: 'slider',
                difficultyHistory: [],
                successRate: 0.7,
                avgCompletionTime: 5000,
                failurePatterns: [],
                lastInteraction: Date.now(),
                trustScore: 0.5
            };
        }

        calculateOptimalDifficulty(profile, riskResult) {
            let baseDifficulty = 2;

            if (profile.difficultyHistory && profile.difficultyHistory.length > 0) {
                const avgDifficulty = profile.difficultyHistory.reduce((a, b) => a + b, 0) / profile.difficultyHistory.length;
                baseDifficulty += Math.floor(avgDifficulty * 0.5);
            }

            let riskAdjust = 0;
            if (riskResult.riskScore > 70) {
                riskAdjust = 1;
            } else if (riskResult.riskScore < 30) {
                riskAdjust = -1;
            }

            let optimal = baseDifficulty + riskAdjust;
            return Math.max(1, Math.min(5, optimal));
        }

        generateCustomInstructions(profile) {
            const instructions = [];

            if (profile.successRate < 0.5) {
                instructions.push('请仔细阅读验证提示');
                instructions.push('按照提示要求完成操作');
            }

            if (profile.failurePatterns && profile.failurePatterns.length > 0) {
                const lastFailure = profile.failurePatterns[profile.failurePatterns.length - 1];
                if (lastFailure === 'timeout') {
                    instructions.push('注意控制操作时间');
                } else if (lastFailure === 'wrong_answer') {
                    instructions.push('请仔细确认选择是否正确');
                }
            }

            return instructions;
        }

        getPersonalization(userId, riskResult) {
            let profile = this.profiles[userId];
            if (!profile) {
                profile = this.createDefaultProfile(userId);
                this.profiles[userId] = profile;
            }

            return {
                preferredCaptcha: profile.preferredCaptcha,
                optimalDifficulty: this.calculateOptimalDifficulty(profile, riskResult),
                customInstructions: this.generateCustomInstructions(profile),
                trustAdjustment: 0,
                confidence: profile.trustScore * 100
            };
        }

        updateProfile(userId, interaction) {
            let profile = this.profiles[userId];
            if (!profile) {
                profile = this.createDefaultProfile(userId);
                this.profiles[userId] = profile;
            }

            profile.lastInteraction = Date.now();

            if (interaction.success) {
                profile.successRate = profile.successRate * 0.9 + 0.1;
            } else {
                profile.successRate = profile.successRate * 0.9;
                if (interaction.errorType) {
                    profile.failurePatterns.push(interaction.errorType);
                    if (profile.failurePatterns.length > 10) {
                        profile.failurePatterns = profile.failurePatterns.slice(-10);
                    }
                }
            }

            if (interaction.difficulty) {
                profile.difficultyHistory.push(interaction.difficulty);
                if (profile.difficultyHistory.length > 20) {
                    profile.difficultyHistory = profile.difficultyHistory.slice(-20);
                }
            }
        }
    }

    class AdaptiveEngine {
        constructor() {
            this.riskAssessor = new RiskAssessor();
            this.strategyEngine = new StrategyEngine();
            this.personalizationEngine = new PersonalizationEngine();
            this.initialized = false;
            this.sessionId = this.generateSessionId();
        }

        generateSessionId() {
            return 'adaptive_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
        }

        async initialize() {
            if (this.initialized) return;

            this.initialized = true;
            console.log('[AdaptiveAI] Engine initialized');
        }

        async performAssessment(traceData, deviceInfo, sessionData, networkData, errorData, userId) {
            if (!this.initialized) {
                await this.initialize();
            }

            const riskResult = this.riskAssessor.assessRisk(traceData, deviceInfo, sessionData, networkData, errorData);
            const strategyResult = this.strategyEngine.determineStrategy(riskResult, null);
            const personalizationResult = this.personalizationEngine.getPersonalization(userId || this.sessionId, riskResult);

            const recommendedAction = this.determineAction(riskResult, strategyResult);

            return {
                sessionId: this.sessionId,
                timestamp: Date.now(),
                riskAssessment: riskResult,
                strategy: strategyResult,
                personalization: personalizationResult,
                recommendedAction: recommendedAction
            };
        }

        determineAction(riskResult, strategyResult) {
            if (riskResult.riskLevel === 'critical') {
                return 'block';
            }

            if (riskResult.riskLevel === 'high') {
                return 'challenge';
            }

            if (strategyResult.strategy.difficulty >= 3) {
                return 'verify';
            }

            return 'allow';
        }

        recordInteraction(userId, interaction) {
            this.personalizationEngine.updateProfile(userId || this.sessionId, interaction);
        }

        getRecommendedCaptcha(assessment) {
            if (!assessment || !assessment.personalization) {
                return 'slider';
            }

            return assessment.personalization.preferredCaptcha;
        }

        getRecommendedDifficulty(assessment) {
            if (!assessment || !assessment.personalization) {
                return 2;
            }

            return assessment.personalization.optimalDifficulty;
        }

        shouldChallengeUser(assessment) {
            if (!assessment || !assessment.riskAssessment) {
                return false;
            }

            return assessment.riskAssessment.riskLevel === 'critical' ||
                   assessment.riskAssessment.riskLevel === 'high' ||
                   assessment.recommendedAction === 'challenge' ||
                   assessment.recommendedAction === 'verify';
        }

        getUserInstructions(assessment) {
            if (!assessment) {
                return [];
            }

            const instructions = [];

            if (assessment.strategy && assessment.strategy.specialInstructions) {
                instructions.push(...assessment.strategy.specialInstructions);
            }

            if (assessment.personalization && assessment.personalization.customInstructions) {
                instructions.push(...assessment.personalization.customInstructions);
            }

            return instructions;
        }
    }

    return {
        createEngine: function() {
            return new AdaptiveEngine();
        },

        RiskAssessor: RiskAssessor,
        StrategyEngine: StrategyEngine,
        PersonalizationEngine: PersonalizationEngine
    };
})();

if (typeof module !== 'undefined' && module.exports) {
    module.exports = AdaptiveAIEngine;
}
