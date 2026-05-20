/**
 * 滑块验证码轨迹采集增强版 - trajectory_collector.js
 * 
 * 功能：
 * 1. 高精度轨迹采集（时间戳、坐标、速度、加速度）
 * 2. 移动端压力和触摸数据
 * 3. 设备指纹信息
 * 4. 行为特征分析
 * 5. 数据加密和压缩
 */

class SliderTrajectoryCollector {
    constructor(options = {}) {
        this.options = {
            samplingRate: options.samplingRate || 60,
            minPoints: options.minPoints || 20,
            maxPoints: options.maxPoints || 500,
            enableCompression: options.enableCompression !== false,
            enableEncryption: options.enableEncryption !== false,
            ...options
        };

        this.state = {
            isCollecting: false,
            trajectory: [],
            startTime: 0,
            lastPoint: null,
            deviceInfo: null,
            behaviorFeatures: {
                totalDistance: 0,
                totalDuration: 0,
                avgVelocity: 0,
                maxVelocity: 0,
                velocityVariance: 0,
                accelerationChanges: 0,
                directionChanges: 0,
                pauses: 0,
                pauseDuration: 0,
                microCorrections: 0,
                backtrackDistance: 0,
                pathEfficiency: 0,
                smoothness: 0,
                jerkiness: 0,
                entropy: 0
            }
        };

        this.velocities = [];
        this.accelerations = [];
        this.directions = [];
        this.timestamps = [];

        this.initDeviceInfo();
    }

    initDeviceInfo() {
        this.state.deviceInfo = {
            userAgent: navigator.userAgent,
            platform: navigator.platform,
            language: navigator.language,
            screenWidth: window.screen.width,
            screenHeight: window.screen.height,
            windowWidth: window.innerWidth,
            windowHeight: window.innerHeight,
            pixelRatio: window.devicePixelRatio || 1,
            touchSupport: 'ontouchstart' in window,
            maxTouchPoints: navigator.maxTouchPoints || 0,
            orientation: window.orientation || 0,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            timestamp: Date.now()
        };

        if (navigator.connection) {
            this.state.deviceInfo.connectionType = navigator.connection.effectiveType;
            this.state.deviceInfo.downlink = navigator.connection.downlink;
            this.state.deviceInfo.rtt = navigator.connection.rtt;
        }
    }

    start() {
        if (this.state.isCollecting) {
            return;
        }

        this.state.isCollecting = true;
        this.state.startTime = performance.now();
        this.state.trajectory = [];
        this.velocities = [];
        this.accelerations = [];
        this.directions = [];
        this.timestamps = [];
        this.state.lastPoint = null;

        this.resetBehaviorFeatures();
    }

    stop() {
        if (!this.state.isCollecting) {
            return null;
        }

        this.state.isCollecting = false;
        const endTime = performance.now();
        
        this.calculateBehaviorFeatures();
        
        return this.getTrajectoryData();
    }

    resetBehaviorFeatures() {
        Object.keys(this.state.behaviorFeatures).forEach(key => {
            if (typeof this.state.behaviorFeatures[key] === 'number') {
                this.state.behaviorFeatures[key] = 0;
            }
        });
    }

    recordPoint(x, y, extra = {}) {
        if (!this.state.isCollecting) {
            return;
        }

        const now = performance.now();
        const timestamp = now - this.state.startTime;

        const point = {
            x: Math.round(x),
            y: Math.round(y),
            t: Math.round(timestamp),
            ...extra
        };

        if (this.state.lastPoint) {
            const dx = point.x - this.state.lastPoint.x;
            const dy = point.y - this.state.lastPoint.y;
            const dt = point.t - this.state.lastPoint.t;

            if (dt > 0) {
                const distance = Math.sqrt(dx * dx + dy * dy);
                const velocity = distance / dt * 1000;
                const angle = Math.atan2(dy, dx);

                point.dx = dx;
                point.dy = dy;
                point.dt = dt;
                point.distance = distance;
                point.velocity = velocity;
                point.angle = angle;

                this.state.behaviorFeatures.totalDistance += distance;
                this.velocities.push(velocity);
                this.directions.push(angle);
                this.timestamps.push(timestamp);

                if (this.velocities.length > 1) {
                    const prevVelocity = this.velocities[this.velocities.length - 2];
                    const acceleration = (velocity - prevVelocity) / dt * 1000;
                    point.acceleration = acceleration;
                    this.accelerations.push(acceleration);

                    if (Math.abs(acceleration) > 5000) {
                        this.state.behaviorFeatures.accelerationChanges++;
                    }
                }

                if (this.directions.length > 1) {
                    const prevAngle = this.directions[this.directions.length - 2];
                    let angleDiff = Math.abs(angle - prevAngle);
                    if (angleDiff > Math.PI) {
                        angleDiff = 2 * Math.PI - angleDiff;
                    }
                    
                    if (angleDiff > 0.5) {
                        this.state.behaviorFeatures.directionChanges++;
                    }

                    if (angleDiff > 0.1 && angleDiff < 0.5) {
                        this.state.behaviorFeatures.microCorrections++;
                    }
                }

                if (distance < 3 && dt > 100) {
                    this.state.behaviorFeatures.pauses++;
                    this.state.behaviorFeatures.pauseDuration += dt;
                }

                if (dx < 0 && point.x > this.state.lastPoint.maxX - 5) {
                    const backtrackDist = this.state.lastPoint.maxX - point.x;
                    if (backtrackDist > 5) {
                        this.state.behaviorFeatures.backtrackDistance += backtrackDist;
                    }
                }

                if (!this.state.lastPoint.maxX || point.x > this.state.lastPoint.maxX) {
                    point.maxX = point.x;
                } else {
                    point.maxX = this.state.lastPoint.maxX;
                }
            }
        } else {
            point.maxX = point.x;
            point.startX = point.x;
        }

        this.state.trajectory.push(point);
        this.state.lastPoint = point;

        if (this.state.trajectory.length >= this.options.maxPoints) {
            return this.stop();
        }
    }

    calculateBehaviorFeatures() {
        const endTime = this.timestamps.length > 0 
            ? this.timestamps[this.timestamps.length - 1] 
            : 0;
        
        this.state.behaviorFeatures.totalDuration = endTime;
        this.state.behaviorFeatures.avgVelocity = this.calculateAverage(this.velocities);
        this.state.behaviorFeatures.maxVelocity = this.calculateMax(this.velocities);
        this.state.behaviorFeatures.velocityVariance = this.calculateVariance(
            this.velocities, 
            this.state.behaviorFeatures.avgVelocity
        );

        if (this.state.trajectory.length > 0) {
            const startX = this.state.trajectory[0].x;
            const endX = this.state.trajectory[this.state.trajectory.length - 1].x;
            const directDistance = Math.abs(endX - startX);
            
            if (this.state.behaviorFeatures.totalDistance > 0) {
                this.state.behaviorFeatures.pathEfficiency = 
                    directDistance / this.state.behaviorFeatures.totalDistance;
            }
        }

        this.state.behaviorFeatures.smoothness = this.calculateSmoothness();
        this.state.behaviorFeatures.jerkiness = this.calculateJerkiness();
        this.state.behaviorFeatures.entropy = this.calculateEntropy();
    }

    calculateAverage(values) {
        if (values.length === 0) return 0;
        const sum = values.reduce((a, b) => a + b, 0);
        return sum / values.length;
    }

    calculateMax(values) {
        if (values.length === 0) return 0;
        return Math.max(...values);
    }

    calculateVariance(values, mean) {
        if (values.length < 2) return 0;
        const squaredDiffs = values.map(v => Math.pow(v - mean, 2));
        return this.calculateAverage(squaredDiffs);
    }

    calculateSmoothness() {
        if (this.directions.length < 3) return 1;
        
        let totalAngleChange = 0;
        for (let i = 1; i < this.directions.length; i++) {
            let diff = Math.abs(this.directions[i] - this.directions[i-1]);
            if (diff > Math.PI) {
                diff = 2 * Math.PI - diff;
            }
            totalAngleChange += diff;
        }
        
        const avgAngleChange = totalAngleChange / (this.directions.length - 1);
        return 1 - Math.min(avgAngleChange / Math.PI, 1);
    }

    calculateJerkiness() {
        if (this.accelerations.length < 2) return 0;
        
        let totalJerk = 0;
        for (let i = 1; i < this.accelerations.length; i++) {
            totalJerk += Math.abs(this.accelerations[i] - this.accelerations[i-1]);
        }
        
        return totalJerk / this.accelerations.length;
    }

    calculateEntropy() {
        if (this.velocities.length < 10) return 0;
        
        const bins = 20;
        const minV = Math.min(...this.velocities);
        const maxV = Math.max(...this.velocities);
        const binWidth = (maxV - minV) / bins || 1;
        
        const histogram = new Array(bins).fill(0);
        this.velocities.forEach(v => {
            const binIndex = Math.min(Math.floor((v - minV) / binWidth), bins - 1);
            histogram[binIndex]++;
        });
        
        const total = this.velocities.length;
        let entropy = 0;
        histogram.forEach(count => {
            if (count > 0) {
                const p = count / total;
                entropy -= p * Math.log2(p);
            }
        });
        
        return entropy;
    }

    getTrajectoryData() {
        const trajectoryData = {
            version: '2.0',
            timestamp: Date.now(),
            deviceInfo: this.state.deviceInfo,
            behaviorFeatures: { ...this.state.behaviorFeatures },
            trajectory: this.state.trajectory,
            summary: {
                pointCount: this.state.trajectory.length,
                duration: this.state.behaviorFeatures.totalDuration,
                distance: this.state.behaviorFeatures.totalDistance,
                isValid: this.isValidTrajectory()
            }
        };

        if (this.options.enableCompression) {
            trajectoryData.compressed = this.compress(trajectoryData.trajectory);
        }

        if (this.options.enableEncryption) {
            trajectoryData.encrypted = this.encrypt(JSON.stringify(trajectoryData));
            delete trajectoryData.trajectory;
            delete trajectoryData.behaviorFeatures;
            delete trajectoryData.deviceInfo;
        }

        return trajectoryData;
    }

    compress(trajectory) {
        if (!trajectory || trajectory.length === 0) {
            return [];
        }

        const compressed = [];
        let prevX = trajectory[0].x;
        let prevY = trajectory[0].y;
        let prevT = trajectory[0].t;

        compressed.push({
            x: trajectory[0].x,
            y: trajectory[0].y,
            t: trajectory[0].t
        });

        for (let i = 1; i < trajectory.length; i++) {
            const point = trajectory[i];
            const dx = point.x - prevX;
            const dy = point.y - prevY;
            const dt = point.t - prevT;

            const compressedPoint = {
                dx: dx,
                dy: dy,
                dt: dt
            };

            if (point.velocity !== undefined) {
                compressedPoint.v = Math.round(point.velocity);
            }

            if (point.acceleration !== undefined) {
                compressedPoint.a = Math.round(point.acceleration / 100);
            }

            if (point.angle !== undefined) {
                compressedPoint.an = Math.round(point.angle * 100);
            }

            if (point.pressure !== undefined) {
                compressedPoint.p = Math.round(point.pressure * 255);
            }

            if (point.maxX !== undefined) {
                compressedPoint.mx = point.maxX - prevX;
            }

            compressed.push(compressedPoint);

            prevX = point.x;
            prevY = point.y;
            prevT = point.t;
        }

        return compressed;
    }

    decompress(compressed) {
        if (!compressed || compressed.length === 0) {
            return [];
        }

        const trajectory = [];
        let prevX = 0;
        let prevY = 0;
        let prevT = 0;
        let prevMaxX = 0;

        compressed.forEach((point, index) => {
            const x = point.x !== undefined ? point.x : prevX + (point.dx || 0);
            const y = point.y !== undefined ? point.y : prevY + (point.dy || 0);
            const t = point.t !== undefined ? point.t : prevT + (point.dt || 0);

            const decompressedPoint = {
                x: x,
                y: y,
                t: t
            };

            if (point.v !== undefined) {
                decompressedPoint.velocity = point.v;
            }

            if (point.a !== undefined) {
                decompressedPoint.acceleration = point.a * 100;
            }

            if (point.an !== undefined) {
                decompressedPoint.angle = point.an / 100;
            }

            if (point.p !== undefined) {
                decompressedPoint.pressure = point.p / 255;
            }

            if (point.mx !== undefined) {
                decompressedPoint.maxX = prevMaxX + point.mx;
                prevMaxX = decompressedPoint.maxX;
            }

            trajectory.push(decompressedPoint);

            prevX = x;
            prevY = y;
            prevT = t;
        });

        return trajectory;
    }

    encrypt(data) {
        const key = this.generateKey();
        const encoded = btoa(unescape(encodeURIComponent(data)));
        
        let encrypted = '';
        for (let i = 0; i < encoded.length; i++) {
            const charCode = encoded.charCodeAt(i) ^ key.charCodeAt(i % key.length);
            encrypted += String.fromCharCode(charCode);
        }
        
        return btoa(encrypted);
    }

    decrypt(encrypted) {
        const key = this.generateKey();
        const decoded = atob(encrypted);
        
        let decrypted = '';
        for (let i = 0; i < decoded.length; i++) {
            const charCode = decoded.charCodeAt(i) ^ key.charCodeAt(i % key.length);
            decrypted += String.fromCharCode(charCode);
        }
        
        return decodeURIComponent(escape(atob(decrypted)));
    }

    generateKey() {
        const salt = this.state.deviceInfo?.timestamp || Date.now();
        return `${this.options.encryptionKey || 'default'}_${salt}_v2`;
    }

    isValidTrajectory() {
        const features = this.state.behaviorFeatures;
        
        if (this.state.trajectory.length < this.options.minPoints) {
            return false;
        }

        if (features.totalDuration < 300) {
            return false;
        }

        if (features.totalDistance < 50) {
            return false;
        }

        if (features.maxVelocity > 5000) {
            return false;
        }

        return true;
    }

    getQualityScore() {
        let score = 0;
        
        if (this.state.trajectory.length >= this.options.minPoints) {
            score += 25;
        }
        
        if (this.state.behaviorFeatures.totalDuration >= 500 && 
            this.state.behaviorFeatures.totalDuration <= 10000) {
            score += 25;
        }
        
        if (this.state.behaviorFeatures.pathEfficiency > 0.5 && 
            this.state.behaviorFeatures.pathEfficiency < 0.99) {
            score += 25;
        }
        
        if (this.state.behaviorFeatures.avgVelocity > 50 && 
            this.state.behaviorFeatures.avgVelocity < 1500) {
            score += 25;
        }
        
        return score;
    }

    reset() {
        this.state.trajectory = [];
        this.state.lastPoint = null;
        this.velocities = [];
        this.accelerations = [];
        this.directions = [];
        this.timestamps = [];
        this.resetBehaviorFeatures();
    }

    getState() {
        return {
            isCollecting: this.state.isCollecting,
            pointCount: this.state.trajectory.length,
            isValid: this.isValidTrajectory(),
            qualityScore: this.getQualityScore(),
            behaviorFeatures: { ...this.state.behaviorFeatures }
        };
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = SliderTrajectoryCollector;
}
