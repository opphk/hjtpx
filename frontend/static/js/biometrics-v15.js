/**
 * v15.0 多模态生物特征采集模块
 * 功能：鼠标压力感应、触摸力度分析、眼动追踪模拟
 * 包含：MousePressureCollector、TouchForceCollector、EyeTrackingCollector
 */

class MousePressureCollector {
    constructor(options = {}) {
        this.options = {
            minSamples: 50,
            pressureLevels: 256,
            samplingRate: 60,
            ...options
        };

        this.pressureData = [];
        this.isCollecting = false;
        this.lastMouseDown = null;
        this.lastMouseUp = null;
        this.clickPatterns = [];
        this.hoverPatterns = [];
        this.dragPatterns = [];
        this.moveHistory = [];
        this.maxMoveHistory = 1000;
    }

    start() {
        this.pressureData = [];
        this.clickPatterns = [];
        this.hoverPatterns = [];
        this.dragPatterns = [];
        this.moveHistory = [];
        this.isCollecting = true;

        window.addEventListener('mousedown', this.handleMouseDown.bind(this));
        window.addEventListener('mouseup', this.handleMouseUp.bind(this));
        window.addEventListener('mousemove', this.handleMouseMove.bind(this));
        window.addEventListener('click', this.handleClick.bind(this));
        window.addEventListener('contextmenu', this.handleContextMenu.bind(this));

        if (this.options.enablePressureSimulation) {
            this.startPressureSimulation();
        }
    }

    stop() {
        this.isCollecting = false;
        this.stopPressureSimulation();

        window.removeEventListener('mousedown', this.handleMouseDown);
        window.removeEventListener('mouseup', this.handleMouseUp);
        window.removeEventListener('mousemove', this.handleMouseMove);
        window.removeEventListener('click', this.handleClick);
        window.removeEventListener('contextmenu', this.handleContextMenu);
    }

    handleMouseDown(event) {
        if (!this.isCollecting) return;

        this.lastMouseDown = {
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now(),
            pressure: this.simulatePressure('down', event)
        };

        this.pressureData.push({
            type: 'mousedown',
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now(),
            pressure: this.lastMouseDown.pressure,
            force: this.calculateForce(this.lastMouseDown.pressure)
        });
    }

    handleMouseUp(event) {
        if (!this.isCollecting) return;

        this.lastMouseUp = {
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now(),
            pressure: this.simulatePressure('up', event)
        };

        this.pressureData.push({
            type: 'mouseup',
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now(),
            pressure: this.lastMouseUp.pressure,
            force: this.calculateForce(this.lastMouseUp.pressure)
        });

        if (this.lastMouseDown) {
            const clickDuration = this.lastMouseUp.timestamp - this.lastMouseDown.timestamp;
            const dx = this.lastMouseUp.x - this.lastMouseDown.x;
            const dy = this.lastMouseUp.y - this.lastMouseDown.y;
            const distance = Math.sqrt(dx * dx + dy * dy);

            this.clickPatterns.push({
                startX: this.lastMouseDown.x,
                startY: this.lastMouseDown.y,
                endX: this.lastMouseUp.x,
                endY: this.lastMouseUp.y,
                duration: clickDuration,
                distance: distance,
                startPressure: this.lastMouseDown.pressure,
                endPressure: this.lastMouseUp.pressure,
                button: event.button
            });

            if (distance > 5) {
                this.dragPatterns.push({
                    startX: this.lastMouseDown.x,
                    startY: this.lastMouseDown.y,
                    endX: this.lastMouseUp.x,
                    endY: this.lastMouseUp.y,
                    duration: clickDuration,
                    distance: distance,
                    avgPressure: (this.lastMouseDown.pressure + this.lastMouseUp.pressure) / 2
                });
            }
        }
    }

    handleMouseMove(event) {
        if (!this.isCollecting) return;

        const pressure = this.simulatePressure('move', event);
        const point = {
            x: event.clientX,
            y: event.clientY,
            timestamp: Date.now(),
            pressure: pressure,
            force: this.calculateForce(pressure),
            movementX: event.movementX || 0,
            movementY: event.movementY || 0
        };

        this.pressureData.push({
            type: 'mousemove',
            ...point
        });

        if (this.moveHistory.length >= this.maxMoveHistory) {
            this.moveHistory.shift();
        }
        this.moveHistory.push(point);

        if (this.lastMouseDown && event.buttons > 0) {
            const timeSinceDown = Date.now() - this.lastMouseDown.timestamp;
            if (timeSinceDown > 100) {
                this.hoverPatterns.push({
                    x: event.clientX,
                    y: event.clientY,
                    duration: timeSinceDown,
                    pressure: pressure
                });
            }
        }
    }

    handleClick(event) {
        if (!this.isCollecting) return;

        this.pressureData.push({
            type: 'click',
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now(),
            pressure: this.simulatePressure('click', event),
            force: this.calculateForce(this.simulatePressure('click', event))
        });
    }

    handleContextMenu(event) {
        if (!this.isCollecting) return;

        this.pressureData.push({
            type: 'contextmenu',
            x: event.clientX,
            y: event.clientY,
            timestamp: Date.now(),
            pressure: this.simulatePressure('context', event)
        });
    }

    simulatePressure(action, event) {
        const basePressure = {
            'down': 0.7 + Math.random() * 0.25,
            'up': 0.3 + Math.random() * 0.2,
            'move': 0.1 + Math.random() * 0.15,
            'click': 0.5 + Math.random() * 0.3,
            'context': 0.6 + Math.random() * 0.25
        };

        let pressure = basePressure[action] || 0.5;

        const movement = Math.abs(event.movementX || 0) + Math.abs(event.movementY || 0);
        if (movement > 10) {
            pressure *= 0.8;
        } else if (movement < 2) {
            pressure *= 1.1;
        }

        const rect = event.target?.getBoundingClientRect?.();
        if (rect) {
            const x = event.clientX - rect.left;
            const y = event.clientY - rect.top;
            const centerX = rect.width / 2;
            const centerY = rect.height / 2;
            const distFromCenter = Math.sqrt(Math.pow(x - centerX, 2) + Math.pow(y - centerY, 2));
            const maxDist = Math.sqrt(Math.pow(centerX, 2) + Math.pow(centerY, 2));
            const normalizedDist = distFromCenter / maxDist;
            pressure *= (1 - normalizedDist * 0.1);
        }

        return Math.min(1, Math.max(0, pressure));
    }

    calculateForce(pressure) {
        return pressure * 9.81;
    }

    startPressureSimulation() {
        this.simulationInterval = setInterval(() => {
            if (this.isCollecting && this.moveHistory.length > 0) {
                const lastPoint = this.moveHistory[this.moveHistory.length - 1];
                if (lastPoint) {
                    this.pressureData.push({
                        type: 'pressure_sample',
                        x: lastPoint.x,
                        y: lastPoint.y,
                        timestamp: Date.now(),
                        pressure: lastPoint.pressure,
                        force: lastPoint.force
                    });
                }
            }
        }, 1000 / this.options.samplingRate);
    }

    stopPressureSimulation() {
        if (this.simulationInterval) {
            clearInterval(this.simulationInterval);
            this.simulationInterval = null;
        }
    }

    analyzePressurePattern() {
        if (this.pressureData.length < 10) {
            return null;
        }

        const downEvents = this.pressureData.filter(e => e.type === 'mousedown');
        const upEvents = this.pressureData.filter(e => e.type === 'mouseup');

        const pressures = this.pressureData.map(e => e.pressure).filter(p => p !== undefined);
        const forces = this.pressureData.map(e => e.force).filter(f => f !== undefined);

        return {
            average_pressure: this.calculateAverage(pressures),
            pressure_std: this.calculateStdDev(pressures),
            average_force: this.calculateAverage(forces),
            force_std: this.calculateStdDev(forces),
            max_pressure: Math.max(...pressures),
            min_pressure: Math.min(...pressures),
            down_pressure_avg: this.calculateAverage(downEvents.map(e => e.pressure)),
            up_pressure_avg: this.calculateAverage(upEvents.map(e => e.pressure)),
            pressure_range: Math.max(...pressures) - Math.min(...pressures),
            pressure_skewness: this.calculateSkewness(pressures),
            pressure_kurtosis: this.calculateKurtosis(pressures)
        };
    }

    analyzeClickPattern() {
        if (this.clickPatterns.length === 0) {
            return null;
        }

        const durations = this.clickPatterns.map(p => p.duration);
        const distances = this.clickPatterns.map(p => p.distance);
        const startPressures = this.clickPatterns.map(p => p.startPressure);
        const endPressures = this.clickPatterns.map(p => p.endPressure);

        return {
            click_count: this.clickPatterns.length,
            avg_click_duration: this.calculateAverage(durations),
            click_duration_std: this.calculateStdDev(durations),
            avg_distance: this.calculateAverage(distances),
            avg_start_pressure: this.calculateAverage(startPressures),
            avg_end_pressure: this.calculateAverage(endPressures),
            pressure_change_avg: this.calculateAverage(
                this.clickPatterns.map(p => p.endPressure - p.startPressure)
            )
        };
    }

    analyzeDragPattern() {
        if (this.dragPatterns.length === 0) {
            return null;
        }

        const durations = this.dragPatterns.map(p => p.duration);
        const distances = this.dragPatterns.map(p => p.distance);
        const avgPressures = this.dragPatterns.map(p => p.avgPressure);

        return {
            drag_count: this.dragPatterns.length,
            avg_drag_duration: this.calculateAverage(durations),
            avg_drag_distance: this.calculateAverage(distances),
            avg_drag_speed: this.calculateAverage(
                this.dragPatterns.map(p => p.distance / (p.duration || 1))
            ),
            avg_drag_pressure: this.calculateAverage(avgPressures),
            drag_pressure_std: this.calculateStdDev(avgPressures)
        };
    }

    analyzeMovementPattern() {
        if (this.moveHistory.length < 10) {
            return null;
        }

        const speeds = [];
        const accelerations = [];
        const jerkPatterns = [];
        const directions = [];

        for (let i = 1; i < this.moveHistory.length; i++) {
            const prev = this.moveHistory[i - 1];
            const curr = this.moveHistory[i];

            const dx = curr.x - prev.x;
            const dy = curr.y - prev.y;
            const dt = curr.timestamp - prev.timestamp;

            if (dt > 0) {
                const speed = Math.sqrt(dx * dx + dy * dy) / dt;
                speeds.push(speed);

                if (dx !== 0 || dy !== 0) {
                    directions.push(Math.atan2(dy, dx));
                }
            }
        }

        for (let i = 1; i < speeds.length; i++) {
            const dt = this.moveHistory[i + 1]?.timestamp - this.moveHistory[i - 1]?.timestamp;
            if (dt > 0) {
                accelerations.push((speeds[i] - speeds[i - 1]) / dt);
            }
        }

        for (let i = 1; i < accelerations.length; i++) {
            const dt = this.moveHistory[i + 1]?.timestamp - this.moveHistory[i - 1]?.timestamp;
            if (dt > 0) {
                jerkPatterns.push((accelerations[i] - accelerations[i - 1]) / dt);
            }
        }

        return {
            avg_speed: this.calculateAverage(speeds),
            speed_std: this.calculateStdDev(speeds),
            max_speed: Math.max(...speeds),
            min_speed: Math.min(...speeds),
            avg_acceleration: this.calculateAverage(accelerations),
            acceleration_std: this.calculateStdDev(accelerations),
            avg_jerk: this.calculateAverage(jerkPatterns),
            jerk_std: this.calculateStdDev(jerkPatterns),
            movement_entropy: this.calculateEntropy(speeds),
            direction_preference: this.calculateDirectionPreference(directions)
        };
    }

    calculateDirectionPreference(directions) {
        if (directions.length === 0) return { horizontal: 0.5, vertical: 0.5 };

        let horizontalCount = 0;
        let verticalCount = 0;

        for (const dir of directions) {
            const absDir = Math.abs(dir);
            if (absDir < Math.PI / 4 || absDir > 3 * Math.PI / 4) {
                horizontalCount++;
            } else {
                verticalCount++;
            }
        }

        const total = horizontalCount + verticalCount;
        return {
            horizontal: total > 0 ? horizontalCount / total : 0.5,
            vertical: total > 0 ? verticalCount / total : 0.5
        };
    }

    calculateSkewness(values) {
        if (values.length < 3) return 0;
        const avg = this.calculateAverage(values);
        const std = this.calculateStdDev(values);
        if (std === 0) return 0;

        const n = values.length;
        let sum = 0;
        for (const v of values) {
            sum += Math.pow((v - avg) / std, 3);
        }
        return sum / n;
    }

    calculateKurtosis(values) {
        if (values.length < 4) return 0;
        const avg = this.calculateAverage(values);
        const std = this.calculateStdDev(values);
        if (std === 0) return 0;

        const n = values.length;
        let sum = 0;
        for (const v of values) {
            sum += Math.pow((v - avg) / std, 4);
        }
        return (sum / n) - 3;
    }

    calculateEntropy(values) {
        if (values.length === 0) return 0;

        const bins = 10;
        const min = Math.min(...values);
        const max = Math.max(...values);
        const binSize = (max - min) / bins || 1;

        const histogram = new Array(bins).fill(0);
        for (const v of values) {
            const binIndex = Math.min(bins - 1, Math.floor((v - min) / binSize));
            histogram[binIndex]++;
        }

        let entropy = 0;
        const total = values.length;
        for (const count of histogram) {
            if (count > 0) {
                const p = count / total;
                entropy -= p * Math.log2(p);
            }
        }

        return entropy;
    }

    calculateAverage(arr) {
        if (!arr || arr.length === 0) return 0;
        return arr.reduce((sum, val) => sum + val, 0) / arr.length;
    }

    calculateStdDev(arr) {
        if (!arr || arr.length < 2) return 0;
        const avg = this.calculateAverage(arr);
        const sumOfSquares = arr.reduce((sum, val) => sum + Math.pow(val - avg, 2), 0);
        return Math.sqrt(sumOfSquares / arr.length);
    }

    getPressureSample() {
        return {
            pressure_data: this.pressureData,
            pressure_analysis: this.analyzePressurePattern(),
            click_analysis: this.analyzeClickPattern(),
            drag_analysis: this.analyzeDragPattern(),
            movement_analysis: this.analyzeMovementPattern(),
            timestamp: Date.now()
        };
    }

    clear() {
        this.pressureData = [];
        this.clickPatterns = [];
        this.hoverPatterns = [];
        this.dragPatterns = [];
        this.moveHistory = [];
        this.lastMouseDown = null;
        this.lastMouseUp = null;
    }
}


class TouchForceCollector {
    constructor(options = {}) {
        this.options = {
            minTouchPoints: 10,
            trackMultiTouch: true,
            maxTouchPoints: 10,
            ...options
        };

        this.touchEvents = [];
        this.gestureEvents = [];
        this.swipeEvents = [];
        this.pinchEvents = [];
        this.isCollecting = false;
        this.activeTouches = new Map();
        this.gestureStartDistance = 0;
        this.gestureStartAngle = 0;
    }

    start() {
        this.touchEvents = [];
        this.gestureEvents = [];
        this.swipeEvents = [];
        this.pinchEvents = [];
        this.isCollecting = true;
        this.activeTouches.clear();

        window.addEventListener('touchstart', this.handleTouchStart.bind(this), { passive: false });
        window.addEventListener('touchmove', this.handleTouchMove.bind(this), { passive: false });
        window.addEventListener('touchend', this.handleTouchEnd.bind(this), { passive: false });
        window.addEventListener('touchcancel', this.handleTouchCancel.bind(this));
        window.addEventListener('gesturestart', this.handleGestureStart.bind(this));
        window.addEventListener('gesturechange', this.handleGestureChange.bind(this));
        window.addEventListener('gestureend', this.handleGestureEnd.bind(this));
    }

    stop() {
        this.isCollecting = false;

        window.removeEventListener('touchstart', this.handleTouchStart);
        window.removeEventListener('touchmove', this.handleTouchMove);
        window.removeEventListener('touchend', this.handleTouchEnd);
        window.removeEventListener('touchcancel', this.handleTouchCancel);
        window.removeEventListener('gesturestart', this.handleGestureStart);
        window.removeEventListener('gesturechange', this.handleGestureChange);
        window.removeEventListener('gestureend', this.handleGestureEnd);
    }

    handleTouchStart(event) {
        if (!this.isCollecting) return;
        event.preventDefault();

        const timestamp = Date.now();

        for (const touch of event.changedTouches) {
            const force = this.simulateTouchForce(touch);
            const touchData = {
                id: touch.identifier,
                x: touch.clientX,
                y: touch.clientY,
                radiusX: touch.radiusX || 10,
                radiusY: touch.radiusY || 10,
                force: force,
                pressure: force / 1.0,
                rotationAngle: touch.rotationAngle || 0,
                timestamp: timestamp
            };

            this.activeTouches.set(touch.identifier, touchData);

            this.touchEvents.push({
                type: 'touchstart',
                ...touchData
            });
        }

        if (this.activeTouches.size >= 2 && this.options.trackMultiTouch) {
            this.detectSwipeStart();
        }
    }

    handleTouchMove(event) {
        if (!this.isCollecting) return;
        event.preventDefault();

        const timestamp = Date.now();

        for (const touch of event.changedTouches) {
            const prevTouch = this.activeTouches.get(touch.identifier);
            if (!prevTouch) continue;

            const force = this.simulateTouchForce(touch);
            const velocity = this.calculateVelocity(prevTouch, {
                clientX: touch.clientX,
                clientY: touch.clientY,
                timestamp: timestamp
            });

            const touchData = {
                id: touch.identifier,
                x: touch.clientX,
                y: touch.clientY,
                prevX: prevTouch.x,
                prevY: prevTouch.y,
                dx: touch.clientX - prevTouch.x,
                dy: touch.clientY - prevTouch.y,
                radiusX: touch.radiusX || 10,
                radiusY: touch.radiusY || 10,
                force: force,
                pressure: force / 1.0,
                velocity: velocity,
                speed: velocity.speed,
                direction: velocity.direction,
                timestamp: timestamp
            };

            this.activeTouches.set(touch.identifier, touchData);

            this.touchEvents.push({
                type: 'touchmove',
                ...touchData
            });
        }

        if (this.activeTouches.size >= 2 && this.options.trackMultiTouch) {
            this.detectSwipe();
        }
    }

    handleTouchEnd(event) {
        if (!this.isCollecting) return;
        event.preventDefault();

        const timestamp = Date.now();

        for (const touch of event.changedTouches) {
            const prevTouch = this.activeTouches.get(touch.identifier);
            if (!prevTouch) continue;

            const force = this.simulateTouchForce(touch);
            const touchData = {
                id: touch.identifier,
                x: touch.clientX,
                y: touch.clientY,
                force: force,
                pressure: force / 1.0,
                timestamp: timestamp,
                duration: timestamp - prevTouch.timestamp
            };

            this.activeTouches.delete(touch.identifier);

            this.touchEvents.push({
                type: 'touchend',
                ...touchData
            });

            if (prevTouch) {
                this.recordSwipeEvent(prevTouch, touchData);
            }
        }
    }

    handleTouchCancel(event) {
        if (!this.isCollecting) return;

        for (const touch of event.changedTouches) {
            this.activeTouches.delete(touch.identifier);

            this.touchEvents.push({
                type: 'touchcancel',
                id: touch.identifier,
                timestamp: Date.now()
            });
        }
    }

    handleGestureStart(event) {
        if (!this.isCollecting) return;

        this.gestureStartDistance = event.scale;
        this.gestureStartAngle = event.rotation;

        this.gestureEvents.push({
            type: 'gesturestart',
            scale: event.scale,
            rotation: event.rotation,
            timestamp: Date.now()
        });
    }

    handleGestureChange(event) {
        if (!this.isCollecting) return;

        this.gestureEvents.push({
            type: 'gesturechange',
            scale: event.scale,
            rotation: event.rotation,
            scaleDelta: event.scale - (this.gestureEvents[this.gestureEvents.length - 1]?.scale || event.scale),
            rotationDelta: event.rotation - (this.gestureEvents[this.gestureEvents.length - 1]?.rotation || event.rotation),
            timestamp: Date.now()
        });
    }

    handleGestureEnd(event) {
        if (!this.isCollecting) return;

        this.gestureEvents.push({
            type: 'gestureend',
            scale: event.scale,
            rotation: event.rotation,
            totalScale: event.scale / this.gestureStartDistance,
            totalRotation: event.rotation - this.gestureStartAngle,
            timestamp: Date.now()
        });

        if (Math.abs(event.scale / this.gestureStartDistance - 1) > 0.1) {
            this.pinchEvents.push({
                scaleFactor: event.scale / this.gestureStartDistance,
                rotation: event.rotation - this.gestureStartAngle,
                timestamp: Date.now()
            });
        }

        this.gestureStartDistance = 0;
        this.gestureStartAngle = 0;
    }

    simulateTouchForce(touch) {
        let force = 0.5 + Math.random() * 0.4;

        if (touch.force !== undefined && touch.force > 0) {
            force = touch.force;
        }

        if (touch.radiusX && touch.radiusY) {
            const area = Math.PI * touch.radiusX * touch.radiusY;
            const normalizedArea = Math.min(1, area / 500);
            force = Math.max(force, normalizedArea);
        }

        return Math.min(1, force);
    }

    calculateVelocity(prev, curr) {
        const dx = curr.clientX - prev.x;
        const dy = curr.clientY - prev.y;
        const dt = curr.timestamp - prev.timestamp;

        if (dt === 0) {
            return { speed: 0, direction: 0 };
        }

        const distance = Math.sqrt(dx * dx + dy * dy);
        const speed = distance / dt;
        const direction = Math.atan2(dy, dx);

        return { speed, direction, dx, dy, dt };
    }

    detectSwipeStart() {
        const touches = Array.from(this.activeTouches.values());
        if (touches.length < 2) return;

        const dx = touches[1].x - touches[0].x;
        const dy = touches[1].y - touches[0].y;
        this.swipeStartData = {
            x: (touches[0].x + touches[1].x) / 2,
            y: (touches[0].y + touches[1].y) / 2,
            distance: Math.sqrt(dx * dx + dy * dy),
            angle: Math.atan2(dy, dx),
            timestamp: Date.now()
        };
    }

    detectSwipe() {
        if (!this.swipeStartData) return;

        const touches = Array.from(this.activeTouches.values());
        if (touches.length < 2) return;

        const dx = touches[1].x - touches[0].x;
        const dy = touches[1].y - touches[0].y;
        const currentDistance = Math.sqrt(dx * dx + dy * dy);
        const currentAngle = Math.atan2(dy, dx);

        this.swipeData = {
            x: (touches[0].x + touches[1].x) / 2,
            y: (touches[0].y + touches[1].y) / 2,
            distance: currentDistance,
            angle: currentAngle,
            distanceDelta: currentDistance - this.swipeStartData.distance,
            angleDelta: currentAngle - this.swipeStartData.angle,
            timestamp: Date.now()
        };
    }

    recordSwipeEvent(start, end) {
        const dx = end.x - start.x;
        const dy = end.y - start.y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        const duration = end.timestamp - start.timestamp;
        const speed = distance / duration;

        let direction = 'unknown';
        const angle = Math.atan2(dy, dx);
        if (Math.abs(angle) < Math.PI / 4) {
            direction = 'right';
        } else if (Math.abs(angle) > 3 * Math.PI / 4) {
            direction = 'left';
        } else if (angle > 0) {
            direction = 'down';
        } else {
            direction = 'up';
        }

        if (distance > 30) {
            this.swipeEvents.push({
                startX: start.x,
                startY: start.y,
                endX: end.x,
                endY: end.y,
                distance: distance,
                duration: duration,
                speed: speed,
                direction: direction,
                angle: angle,
                avgForce: (start.force + end.force) / 2,
                timestamp: end.timestamp
            });
        }
    }

    analyzeTouchForcePattern() {
        if (this.touchEvents.length === 0) {
            return null;
        }

        const forces = this.touchEvents
            .filter(e => e.force !== undefined)
            .map(e => e.force);

        const pressures = this.touchEvents
            .filter(e => e.pressure !== undefined)
            .map(e => e.pressure);

        const speeds = this.touchEvents
            .filter(e => e.speed !== undefined)
            .map(e => e.speed);

        return {
            touch_count: this.touchEvents.length,
            avg_force: this.calculateAverage(forces),
            force_std: this.calculateStdDev(forces),
            max_force: Math.max(...forces),
            min_force: Math.min(...forces),
            avg_pressure: this.calculateAverage(pressures),
            pressure_std: this.calculateStdDev(pressures),
            force_range: Math.max(...forces) - Math.min(...forces),
            force_skewness: this.calculateSkewness(forces),
            avg_speed: this.calculateAverage(speeds),
            speed_std: this.calculateStdDev(speeds)
        };
    }

    analyzeSwipePattern() {
        if (this.swipeEvents.length === 0) {
            return null;
        }

        const directions = {};
        const speeds = this.swipeEvents.map(e => e.speed);
        const forces = this.swipeEvents.map(e => e.avgForce);
        const angles = this.swipeEvents.map(e => e.angle);
        const distances = this.swipeEvents.map(e => e.distance);
        const durations = this.swipeEvents.map(e => e.duration);

        for (const event of this.swipeEvents) {
            directions[event.direction] = (directions[event.direction] || 0) + 1;
        }

        return {
            swipe_count: this.swipeEvents.length,
            directions: directions,
            direction_entropy: this.calculateEntropy(Object.values(directions)),
            avg_speed: this.calculateAverage(speeds),
            speed_std: this.calculateStdDev(speeds),
            avg_force: this.calculateAverage(forces),
            force_std: this.calculateStdDev(forces),
            avg_angle: this.calculateAverage(angles),
            angle_std: this.calculateStdDev(angles),
            avg_distance: this.calculateAverage(distances),
            avg_duration: this.calculateAverage(durations),
            velocity_profile: this.calculateVelocityProfile(speeds, durations)
        };
    }

    analyzeMultiTouchPattern() {
        if (this.gestureEvents.length === 0 && this.pinchEvents.length === 0) {
            return null;
        }

        const pinchScales = this.pinchEvents.map(e => e.scaleFactor);
        const pinchRotations = this.pinchEvents.map(e => e.rotation);

        return {
            gesture_count: this.gestureEvents.length,
            pinch_count: this.pinchEvents.length,
            avg_pinch_scale: this.calculateAverage(pinchScales),
            avg_pinch_rotation: this.calculateAverage(pinchRotations),
            max_pinch_scale: Math.max(...pinchScales),
            min_pinch_scale: Math.min(...pinchScales)
        };
    }

    calculateVelocityProfile(speeds, durations) {
        if (speeds.length === 0) return null;

        const profiles = [];
        for (let i = 0; i < speeds.length; i++) {
            profiles.push({
                speed: speeds[i],
                acceleration: i > 0 ? (speeds[i] - speeds[i - 1]) / (durations[i] || 1) : 0
            });
        }

        return profiles;
    }

    calculateSkewness(values) {
        if (values.length < 3) return 0;
        const avg = this.calculateAverage(values);
        const std = this.calculateStdDev(values);
        if (std === 0) return 0;

        let sum = 0;
        for (const v of values) {
            sum += Math.pow((v - avg) / std, 3);
        }
        return sum / values.length;
    }

    calculateEntropy(values) {
        if (values.length === 0) return 0;

        const total = values.reduce((sum, v) => sum + v, 0);
        if (total === 0) return 0;

        let entropy = 0;
        for (const v of values) {
            if (v > 0) {
                const p = v / total;
                entropy -= p * Math.log2(p);
            }
        }

        return entropy;
    }

    calculateAverage(arr) {
        if (!arr || arr.length === 0) return 0;
        return arr.reduce((sum, val) => sum + val, 0) / arr.length;
    }

    calculateStdDev(arr) {
        if (!arr || arr.length < 2) return 0;
        const avg = this.calculateAverage(arr);
        const sumOfSquares = arr.reduce((sum, val) => sum + Math.pow(val - avg, 2), 0);
        return Math.sqrt(sumOfSquares / arr.length);
    }

    getTouchSample() {
        return {
            touch_events: this.touchEvents,
            gesture_events: this.gestureEvents,
            swipe_events: this.swipeEvents,
            pinch_events: this.pinchEvents,
            force_analysis: this.analyzeTouchForcePattern(),
            swipe_analysis: this.analyzeSwipePattern(),
            multitouch_analysis: this.analyzeMultiTouchPattern(),
            timestamp: Date.now()
        };
    }

    clear() {
        this.touchEvents = [];
        this.gestureEvents = [];
        this.swipeEvents = [];
        this.pinchEvents = [];
        this.activeTouches.clear();
        this.swipeStartData = null;
        this.swipeData = null;
    }
}


class EyeTrackingCollector {
    constructor(options = {}) {
        this.options = {
            minSamples: 30,
            samplingRate: 30,
            trackBlink: true,
            trackFocus: true,
            simulateGaze: true,
            ...options
        };

        this.gazeData = [];
        this.blinkData = [];
        this.focusData = [];
        this.isCollecting = false;
        this.lastGazePoint = null;
        this.blinkStartTime = null;
        this.pageLoadTime = Date.now();
        this.eyeClosed = false;
        this.focusLostCount = 0;
        this.fixationPoints = [];
        this.saccadeEvents = [];
        this.dwellTimeData = [];
        this.currentFixation = null;
        this.currentDwell = null;
    }

    start() {
        this.gazeData = [];
        this.blinkData = [];
        this.focusData = [];
        this.fixationPoints = [];
        this.saccadeEvents = [];
        this.dwellTimeData = [];
        this.isCollecting = true;
        this.lastGazePoint = null;
        this.eyeClosed = false;
        this.focusLostCount = 0;

        window.addEventListener('mousemove', this.handleMouseMove.bind(this));
        window.addEventListener('click', this.handleClick.bind(this));
        window.addEventListener('scroll', this.handleScroll.bind(this));
        window.addEventListener('keydown', this.handleKeyDown.bind(this));
        window.addEventListener('focus', this.handleFocus.bind(this));
        window.addEventListener('blur', this.handleBlur.bind(this));
        window.addEventListener('mouseleave', this.handleMouseLeave.bind(this));
        window.addEventListener('mouseenter', this.handleMouseEnter.bind(this));

        this.startSampling();
        this.startBlinkDetection();
    }

    stop() {
        this.isCollecting = false;

        window.removeEventListener('mousemove', this.handleMouseMove);
        window.removeEventListener('click', this.handleClick);
        window.removeEventListener('scroll', this.handleScroll);
        window.removeEventListener('keydown', this.handleKeyDown);
        window.removeEventListener('focus', this.handleFocus);
        window.removeEventListener('blur', this.handleBlur);
        window.removeEventListener('mouseleave', this.handleMouseLeave);
        window.removeEventListener('mouseenter', this.handleMouseEnter);

        this.stopSampling();
        this.stopBlinkDetection();

        if (this.currentFixation) {
            this.endFixation();
        }
        if (this.currentDwell) {
            this.endDwell();
        }
    }

    startSampling() {
        this.samplingInterval = setInterval(() => {
            if (this.isCollecting && this.options.simulateGaze) {
                this.recordSimulatedGaze();
            }
        }, 1000 / this.options.samplingRate);
    }

    stopSampling() {
        if (this.samplingInterval) {
            clearInterval(this.samplingInterval);
            this.samplingInterval = null;
        }
    }

    startBlinkDetection() {
        if (!this.options.trackBlink) return;

        this.blinkInterval = setInterval(() => {
            if (this.isCollecting) {
                this.checkBlink();
            }
        }, 50);
    }

    stopBlinkDetection() {
        if (this.blinkInterval) {
            clearInterval(this.blinkInterval);
            this.blinkInterval = null;
        }
    }

    recordSimulatedGaze() {
        const gazePoint = this.simulateGazePoint();

        if (this.lastGazePoint) {
            const dx = gazePoint.x - this.lastGazePoint.x;
            const dy = gazePoint.y - this.lastGazePoint.y;
            const dt = gazePoint.timestamp - this.lastGazePoint.timestamp;

            if (dt > 0) {
                const speed = Math.sqrt(dx * dx + dy * dy) / dt;

                if (speed > 0.5) {
                    this.recordSaccade(gazePoint, speed);
                } else {
                    this.recordFixation(gazePoint);
                }
            }
        }

        this.gazeData.push(gazePoint);
        this.lastGazePoint = gazePoint;
    }

    simulateGazePoint() {
        const mousePos = this.getCurrentMousePosition();

        const noise = {
            x: (Math.random() - 0.5) * 20,
            y: (Math.random() - 0.5) * 20
        };

        const viewportCenter = {
            x: window.innerWidth / 2,
            y: window.innerHeight / 3
        };

        const attentionBias = 0.3;
        const gazeX = mousePos.x * (1 - attentionBias) + viewportCenter.x * attentionBias + noise.x;
        const gazeY = mousePos.y * (1 - attentionBias) + viewportCenter.y * attentionBias + noise.y;

        return {
            x: Math.max(0, Math.min(window.innerWidth, gazeX)),
            y: Math.max(0, Math.min(window.innerHeight, gazeY)),
            timestamp: Date.now(),
            pupilSize: this.simulatePupilSize(),
            confidence: 0.7 + Math.random() * 0.25
        };
    }

    simulatePupilSize() {
        const baseSize = 3 + Math.random() * 2;

        const lightLevel = this.estimateLightLevel();
        const adjustedSize = baseSize * (1.2 - lightLevel * 0.4);

        return Math.max(2, Math.min(6, adjustedSize));
    }

    estimateLightLevel() {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        canvas.width = 1;
        canvas.height = 1;

        try {
            ctx.fillStyle = 'white';
            ctx.fillRect(0, 0, 1, 1);

            const bgColor = window.getComputedStyle(document.body).backgroundColor;
            const rgb = this.parseRGB(bgColor);

            const brightness = (rgb.r * 299 + rgb.g * 587 + rgb.b * 114) / 1000 / 255;
            return brightness;
        } catch (e) {
            return 0.5;
        }
    }

    parseRGB(color) {
        const match = color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
        if (match) {
            return {
                r: parseInt(match[1]),
                g: parseInt(match[2]),
                b: parseInt(match[3])
            };
        }
        return { r: 128, g: 128, b: 128 };
    }

    recordFixation(gazePoint) {
        if (!this.currentFixation) {
            this.currentFixation = {
                startX: gazePoint.x,
                startY: gazePoint.y,
                points: [gazePoint],
                startTime: gazePoint.timestamp
            };
        } else {
            this.currentFixation.points.push(gazePoint);
            this.currentFixation.endX = gazePoint.x;
            this.currentFixation.endY = gazePoint.y;
        }
    }

    endFixation() {
        if (!this.currentFixation || this.currentFixation.points.length < 3) {
            this.currentFixation = null;
            return;
        }

        const duration = Date.now() - this.currentFixation.startTime;
        const points = this.currentFixation.points;

        const centroidX = points.reduce((sum, p) => sum + p.x, 0) / points.length;
        const centroidY = points.reduce((sum, p) => sum + p.y, 0) / points.length;

        let variance = 0;
        for (const p of points) {
            variance += Math.pow(p.x - centroidX, 2) + Math.pow(p.y - centroidY, 2);
        }
        variance /= points.length;

        if (duration > 100) {
            this.fixationPoints.push({
                centroidX: centroidX,
                centroidY: centroidY,
                duration: duration,
                pointCount: points.length,
                dispersion: Math.sqrt(variance),
                avgPupilSize: points.reduce((sum, p) => sum + p.pupilSize, 0) / points.length,
                timestamp: this.currentFixation.startTime
            });
        }

        this.currentFixation = null;
    }

    recordSaccade(currentPoint, speed) {
        if (this.currentFixation && this.currentFixation.points.length >= 3) {
            this.endFixation();
        }

        const targetArea = this.findTargetArea(currentPoint.x, currentPoint.y);

        this.saccadeEvents.push({
            startX: this.lastGazePoint.x,
            startY: this.lastGazePoint.y,
            endX: currentPoint.x,
            endY: currentPoint.y,
            speed: speed,
            duration: currentPoint.timestamp - this.lastGazePoint.timestamp,
            targetArea: targetArea,
            timestamp: currentPoint.timestamp
        });
    }

    findTargetArea(x, y) {
        const elements = document.elementsFromPoint(x, y);

        for (const el of elements) {
            if (el.id) return { type: 'id', value: el.id };
            if (el.className) return { type: 'class', value: el.className };
            if (el.tagName) return { type: 'tag', value: el.tagName };
        }

        return { type: 'none', value: 'background' };
    }

    checkBlink() {
        const now = Date.now();

        if (this.eyeClosed) {
            const closedDuration = now - this.blinkStartTime;

            if (closedDuration > 50 && closedDuration < 1000) {
                this.blinkData.push({
                    duration: closedDuration,
                    timestamp: now
                });
            }

            this.eyeClosed = false;
        } else {
            const shouldBlink = Math.random() < 0.02;

            if (shouldBlink) {
                this.eyeClosed = true;
                this.blinkStartTime = now;
            }
        }
    }

    handleMouseMove(event) {
        if (!this.isCollecting) return;

        if (this.currentDwell) {
            const dx = event.clientX - this.currentDwell.x;
            const dy = event.clientY - this.currentDwell.y;
            if (Math.sqrt(dx * dx + dy * dy) > 10) {
                this.endDwell();
            }
        }
    }

    handleClick(event) {
        if (!this.isCollecting) return;

        this.startDwell(event.clientX, event.clientY);
    }

    handleScroll() {
        if (!this.isCollecting) return;

        if (this.currentDwell) {
            this.endDwell();
        }
        if (this.currentFixation) {
            this.endFixation();
        }
    }

    handleKeyDown(event) {
        if (!this.isCollecting) return;

        if (event.key.length === 1) {
            if (this.currentDwell) {
                this.endDwell();
            }
        }
    }

    handleFocus() {
        if (!this.isCollecting) return;

        this.focusData.push({
            type: 'focus',
            timestamp: Date.now(),
            pageTime: Date.now() - this.pageLoadTime
        });
    }

    handleBlur() {
        if (!this.isCollecting) return;

        this.focusLostCount++;
        this.focusData.push({
            type: 'blur',
            timestamp: Date.now(),
            pageTime: Date.now() - this.pageLoadTime
        });

        if (this.currentDwell) {
            this.endDwell();
        }
    }

    handleMouseLeave() {
        if (!this.isCollecting) return;

        this.gazeData.push({
            type: 'leave',
            x: -1,
            y: -1,
            timestamp: Date.now()
        });
    }

    handleMouseEnter() {
        if (!this.isCollecting) return;

        this.gazeData.push({
            type: 'enter',
            x: -1,
            y: -1,
            timestamp: Date.now()
        });
    }

    startDwell(x, y) {
        this.currentDwell = {
            x: x,
            y: y,
            startTime: Date.now(),
            target: this.findTargetArea(x, y)
        };
    }

    endDwell() {
        if (!this.currentDwell) return;

        const duration = Date.now() - this.currentDwell.startTime;

        if (duration > 100) {
            this.dwellTimeData.push({
                x: this.currentDwell.x,
                y: this.currentDwell.y,
                duration: duration,
                target: this.currentDwell.target,
                timestamp: this.currentDwell.startTime
            });
        }

        this.currentDwell = null;
    }

    getCurrentMousePosition() {
        return {
            x: window.mouseX || window.innerWidth / 2,
            y: window.mouseY || window.innerHeight / 2
        };
    }

    analyzeGazePattern() {
        if (this.gazeData.length < 10) {
            return null;
        }

        const xCoords = this.gazeData.map(g => g.x);
        const yCoords = this.gazeData.map(g => g.y);
        const pupilSizes = this.gazeData.map(g => g.pupilSize).filter(p => p !== undefined);

        return {
            gaze_count: this.gazeData.length,
            avg_x: this.calculateAverage(xCoords),
            avg_y: this.calculateAverage(yCoords),
            x_std: this.calculateStdDev(xCoords),
            y_std: this.calculateStdDev(yCoords),
            coverage_area: this.calculateCoverageArea(xCoords, yCoords),
            avg_pupil_size: this.calculateAverage(pupilSizes),
            pupil_std: this.calculateStdDev(pupilSizes),
            scan_pattern: this.analyzeScanPattern()
        };
    }

    analyzeScanPattern() {
        const quadrants = { topLeft: 0, topRight: 0, bottomLeft: 0, bottomRight: 0 };
        const centerX = window.innerWidth / 2;
        const centerY = window.innerHeight / 2;

        for (const gaze of this.gazeData) {
            if (gaze.x < centerX && gaze.y < centerY) {
                quadrants.topLeft++;
            } else if (gaze.x >= centerX && gaze.y < centerY) {
                quadrants.topRight++;
            } else if (gaze.x < centerX && gaze.y >= centerY) {
                quadrants.bottomLeft++;
            } else {
                quadrants.bottomRight++;
            }
        }

        const total = this.gazeData.length;
        return {
            topLeft: total > 0 ? quadrants.topLeft / total : 0,
            topRight: total > 0 ? quadrants.topRight / total : 0,
            bottomLeft: total > 0 ? quadrants.bottomLeft / total : 0,
            bottomRight: total > 0 ? quadrants.bottomRight / total : 0,
            preference: this.determinePreference(quadrants)
        };
    }

    determinePreference(quadrants) {
        const max = Math.max(...Object.values(quadrants));
        for (const [key, value] of Object.entries(quadrants)) {
            if (value === max) {
                return key;
            }
        }
        return 'center';
    }

    calculateCoverageArea(xCoords, yCoords) {
        if (xCoords.length === 0) return 0;

        const minX = Math.min(...xCoords);
        const maxX = Math.max(...xCoords);
        const minY = Math.min(...yCoords);
        const maxY = Math.max(...yCoords);

        const width = maxX - minX;
        const height = maxY - minY;
        const area = width * height;

        const viewportArea = window.innerWidth * window.innerHeight;

        return area / viewportArea;
    }

    analyzeBlinkPattern() {
        if (this.blinkData.length === 0) {
            return null;
        }

        const durations = this.blinkData.map(b => b.duration);
        const intervals = [];

        for (let i = 1; i < this.blinkData.length; i++) {
            intervals.push(this.blinkData[i].timestamp - this.blinkData[i - 1].timestamp);
        }

        const now = Date.now();
        const totalDuration = (this.blinkData[this.blinkData.length - 1]?.timestamp || now) - this.blinkData[0]?.timestamp;

        return {
            blink_count: this.blinkData.length,
            blink_rate: totalDuration > 0 ? this.blinkData.length / (totalDuration / 60000) : 0,
            avg_blink_duration: this.calculateAverage(durations),
            blink_duration_std: this.calculateStdDev(durations),
            avg_interval: this.calculateAverage(intervals),
            interval_std: this.calculateStdDev(intervals),
            min_duration: Math.min(...durations),
            max_duration: Math.max(...durations)
        };
    }

    analyzeFixationPattern() {
        if (this.fixationPoints.length === 0) {
            return null;
        }

        const durations = this.fixationPoints.map(f => f.duration);
        const dispersions = this.fixationPoints.map(f => f.dispersion);
        const pupilSizes = this.fixationPoints.map(f => f.avgPupilSize);

        return {
            fixation_count: this.fixationPoints.length,
            avg_duration: this.calculateAverage(durations),
            duration_std: this.calculateStdDev(durations),
            avg_dispersion: this.calculateAverage(dispersions),
            dispersion_std: this.calculateStdDev(dispersions),
            avg_pupil_size: this.calculateAverage(pupilSizes),
            pupil_std: this.calculateStdDev(pupilSizes),
            long_fixations: durations.filter(d => d > 300).length,
            short_fixations: durations.filter(d => d <= 150).length
        };
    }

    analyzeSaccadePattern() {
        if (this.saccadeEvents.length === 0) {
            return null;
        }

        const speeds = this.saccadeEvents.map(s => s.speed);
        const durations = this.saccadeEvents.map(s => s.duration);
        const distances = this.saccadeEvents.map(s => {
            const dx = s.endX - s.startX;
            const dy = s.endY - s.startY;
            return Math.sqrt(dx * dx + dy * dy);
        });

        return {
            saccade_count: this.saccadeEvents.length,
            avg_speed: this.calculateAverage(speeds),
            speed_std: this.calculateStdDev(speeds),
            avg_duration: this.calculateAverage(durations),
            duration_std: this.calculateStdDev(durations),
            avg_distance: this.calculateAverage(distances),
            distance_std: this.calculateStdDev(distances),
            max_speed: Math.max(...speeds)
        };
    }

    analyzeDwellPattern() {
        if (this.dwellTimeData.length === 0) {
            return null;
        }

        const durations = this.dwellTimeData.map(d => d.duration);
        const targetTypes = {};
        const targets = {};

        for (const dwell of this.dwellTimeData) {
            const type = dwell.target.type;
            const value = dwell.target.value;

            targetTypes[type] = (targetTypes[type] || 0) + 1;
            targets[value] = (targets[value] || 0) + 1;
        }

        return {
            dwell_count: this.dwellTimeData.length,
            avg_duration: this.calculateAverage(durations),
            duration_std: this.calculateStdDev(durations),
            target_types: targetTypes,
            top_targets: this.getTopItems(targets, 5),
            longest_dwell: Math.max(...durations)
        };
    }

    analyzeFocusPattern() {
        if (this.focusData.length === 0) {
            return null;
        }

        const blurEvents = this.focusData.filter(f => f.type === 'blur');
        const focusEvents = this.focusData.filter(f => f.type === 'focus');

        let totalBlurDuration = 0;
        if (blurEvents.length > 0) {
            for (let i = 1; i < blurEvents.length; i++) {
                if (focusEvents.some(f => f.timestamp > blurEvents[i - 1].timestamp && f.timestamp <= blurEvents[i].timestamp)) {
                    const correspondingFocus = focusEvents.find(f => f.timestamp > blurEvents[i - 1].timestamp && f.timestamp <= blurEvents[i].timestamp);
                    if (correspondingFocus) {
                        totalBlurDuration += correspondingFocus.timestamp - blurEvents[i].timestamp;
                    }
                }
            }
        }

        return {
            focus_count: focusEvents.length,
            blur_count: blurEvents.length,
            blur_rate: blurEvents.length,
            total_blur_duration: totalBlurDuration,
            focus_lost_count: this.focusLostCount,
            attention_ratio: 1 - (totalBlurDuration / (Date.now() - this.pageLoadTime))
        };
    }

    getTopItems(obj, n) {
        return Object.entries(obj)
            .sort((a, b) => b[1] - a[1])
            .slice(0, n)
            .reduce((acc, [key, value]) => {
                acc[key] = value;
                return acc;
            }, {});
    }

    calculateAverage(arr) {
        if (!arr || arr.length === 0) return 0;
        return arr.reduce((sum, val) => sum + val, 0) / arr.length;
    }

    calculateStdDev(arr) {
        if (!arr || arr.length < 2) return 0;
        const avg = this.calculateAverage(arr);
        const sumOfSquares = arr.reduce((sum, val) => sum + Math.pow(val - avg, 2), 0);
        return Math.sqrt(sumOfSquares / arr.length);
    }

    getEyeTrackingSample() {
        return {
            gaze_data: this.gazeData,
            blink_data: this.blinkData,
            fixation_data: this.fixationPoints,
            saccade_data: this.saccadeEvents,
            dwell_data: this.dwellTimeData,
            focus_data: this.focusData,
            gaze_analysis: this.analyzeGazePattern(),
            blink_analysis: this.analyzeBlinkPattern(),
            fixation_analysis: this.analyzeFixationPattern(),
            saccade_analysis: this.analyzeSaccadePattern(),
            dwell_analysis: this.analyzeDwellPattern(),
            focus_analysis: this.analyzeFocusPattern(),
            timestamp: Date.now()
        };
    }

    clear() {
        this.gazeData = [];
        this.blinkData = [];
        this.focusData = [];
        this.fixationPoints = [];
        this.saccadeEvents = [];
        this.dwellTimeData = [];
        this.lastGazePoint = null;
        this.eyeClosed = false;
        this.focusLostCount = 0;
        this.currentFixation = null;
        this.currentDwell = null;
    }
}


class MultimodalBiometricsCollector {
    constructor(options = {}) {
        this.options = {
            enableMousePressure: true,
            enableTouchForce: true,
            enableEyeTracking: true,
            ...options
        };

        this.mousePressureCollector = new MousePressureCollector(options.mousePressure);
        this.touchForceCollector = new TouchForceCollector(options.touchForce);
        this.eyeTrackingCollector = new EyeTrackingCollector(options.eyeTracking);

        this.isCollecting = false;
        this.userId = null;
        this.sessionId = this.generateSessionId();
        this.startTime = null;
    }

    generateSessionId() {
        return 'bio_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }

    start(userId = null) {
        this.userId = userId;
        this.isCollecting = true;
        this.startTime = Date.now();
        this.sessionId = this.generateSessionId();

        if (this.options.enableMousePressure) {
            this.mousePressureCollector.start();
        }

        if (this.options.enableTouchForce) {
            this.touchForceCollector.start();
        }

        if (this.options.enableEyeTracking) {
            this.eyeTrackingCollector.start();
        }
    }

    stop() {
        this.isCollecting = false;

        if (this.options.enableMousePressure) {
            this.mousePressureCollector.stop();
        }

        if (this.options.enableTouchForce) {
            this.touchForceCollector.stop();
        }

        if (this.options.enableEyeTracking) {
            this.eyeTrackingCollector.stop();
        }
    }

    getBiometricData() {
        const mouseData = this.options.enableMousePressure ?
            this.mousePressureCollector.getPressureSample() : null;
        const touchData = this.options.enableTouchForce ?
            this.touchForceCollector.getTouchSample() : null;
        const eyeData = this.options.enableEyeTracking ?
            this.eyeTrackingCollector.getEyeTrackingSample() : null;

        return {
            session_id: this.sessionId,
            user_id: this.userId,
            collection_duration: this.startTime ? Date.now() - this.startTime : 0,
            timestamp: Date.now(),
            mouse_pressure: mouseData,
            touch_force: touchData,
            eye_tracking: eyeData,
            device_info: this.getDeviceInfo(),
            features_summary: this.generateFeaturesSummary(mouseData, touchData, eyeData)
        };
    }

    generateFeaturesSummary(mouseData, touchData, eyeData) {
        const summary = {
            feature_count: 0,
            modalities: []
        };

        if (mouseData && mouseData.pressure_analysis) {
            summary.modalities.push('mouse_pressure');
            summary.feature_count += Object.keys(mouseData.pressure_analysis).length +
                Object.keys(mouseData.click_analysis || {}).length +
                Object.keys(mouseData.movement_analysis || {}).length;
        }

        if (touchData && touchData.force_analysis) {
            summary.modalities.push('touch_force');
            summary.feature_count += Object.keys(touchData.force_analysis).length +
                Object.keys(touchData.swipe_analysis || {}).length;
        }

        if (eyeData && eyeData.gaze_analysis) {
            summary.modalities.push('eye_tracking');
            summary.feature_count += Object.keys(eyeData.gaze_analysis || {}).length +
                Object.keys(eyeData.blink_analysis || {}).length +
                Object.keys(eyeData.fixation_analysis || {}).length;
        }

        return summary;
    }

    getDeviceInfo() {
        return {
            userAgent: navigator.userAgent,
            screenWidth: window.screen.width,
            screenHeight: window.screen.height,
            windowWidth: window.innerWidth,
            windowHeight: window.innerHeight,
            devicePixelRatio: window.devicePixelRatio,
            touchSupport: 'ontouchstart' in window,
            platform: navigator.platform,
            language: navigator.language,
            hardwareConcurrency: navigator.hardwareConcurrency || 0,
            maxTouchPoints: navigator.maxTouchPoints || 0
        };
    }

    clear() {
        this.mousePressureCollector.clear();
        this.touchForceCollector.clear();
        this.eyeTrackingCollector.clear();
        this.sessionId = this.generateSessionId();
    }

    isReadyForVerification() {
        const mouseReady = !this.options.enableMousePressure ||
            this.mousePressureCollector.pressureData.length >= 50;
        const touchReady = !this.options.enableTouchForce ||
            this.touchForceCollector.touchEvents.length >= 10;
        const eyeReady = !this.options.enableEyeTracking ||
            this.eyeTrackingCollector.gazeData.length >= 30;

        return {
            ready: mouseReady && touchReady && eyeReady,
            mouse_ready: mouseReady,
            touch_ready: touchReady,
            eye_ready: eyeReady,
            sample_counts: {
                mouse: this.mousePressureCollector.pressureData.length,
                touch: this.touchForceCollector.touchEvents.length,
                eye: this.eyeTrackingCollector.gazeData.length
            }
        };
    }
}


class MultimodalBiometricsService {
    constructor(apiBaseUrl = '/api/v1/biometrics') {
        this.apiBaseUrl = apiBaseUrl;
        this.collector = new MultimodalBiometricsCollector();
    }

    startCollecting(userId = null) {
        this.collector.start(userId);
    }

    stopCollecting() {
        this.collector.stop();
    }

    async registerProfile(userId) {
        const biometricData = this.collector.getBiometricData();

        const response = await fetch(`${this.apiBaseUrl}/register-v15`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                biometric_data: biometricData
            })
        });

        return await response.json();
    }

    async verify(userId) {
        const biometricData = this.collector.getBiometricData();

        const response = await fetch(`${this.apiBaseUrl}/verify-v15`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                biometric_data: biometricData
            })
        });

        return await response.json();
    }

    async registerMouseProfile(userId) {
        const mouseData = this.collector.mousePressureCollector.getPressureSample();

        const response = await fetch(`${this.apiBaseUrl}/register/mouse`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                mouse_data: mouseData
            })
        });

        return await response.json();
    }

    async registerTouchProfile(userId) {
        const touchData = this.collector.touchForceCollector.getTouchSample();

        const response = await fetch(`${this.apiBaseUrl}/register/touch`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                touch_data: touchData
            })
        });

        return await response.json();
    }

    async registerEyeProfile(userId) {
        const eyeData = this.collector.eyeTrackingCollector.getEyeTrackingSample();

        const response = await fetch(`${this.apiBaseUrl}/register/eye`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                eye_tracking_data: eyeData
            })
        });

        return await response.json();
    }

    async fusionVerify(userId) {
        const biometricData = this.collector.getBiometricData();

        const response = await fetch(`${this.apiBaseUrl}/fusion/verify`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                biometric_data: biometricData
            })
        });

        return await response.json();
    }

    getCollector() {
        return this.collector;
    }

    clear() {
        this.collector.clear();
    }

    isReady() {
        return this.collector.isReadyForVerification();
    }
}


window.MousePressureCollector = MousePressureCollector;
window.TouchForceCollector = TouchForceCollector;
window.EyeTrackingCollector = EyeTrackingCollector;
window.MultimodalBiometricsCollector = MultimodalBiometricsCollector;
window.MultimodalBiometricsService = MultimodalBiometricsService;
window.multimodalBiometricsService = new MultimodalBiometricsService();
