class BiometricsCollector {
    constructor(options = {}) {
        this.options = {
            minKeyEvents: 50,
            minMouseEvents: 100,
            ...options
        };
        
        this.keyboardEvents = [];
        this.mouseEvents = [];
        this.isCollecting = false;
        this.keyDownMap = new Map();
        this.userId = null;
    }

    /**
     * 开始生物特征采集
     */
    start(userId = null) {
        this.userId = userId;
        this.keyboardEvents = [];
        this.mouseEvents = [];
        this.keyDownMap.clear();
        this.isCollecting = true;

        window.addEventListener('keydown', this.handleKeyDown.bind(this));
        window.addEventListener('keyup', this.handleKeyUp.bind(this));
        window.addEventListener('mousemove', this.handleMouseMove.bind(this));
        window.addEventListener('mousedown', this.handleMouseDown.bind(this));
        window.addEventListener('mouseup', this.handleMouseUp.bind(this));
        window.addEventListener('click', this.handleClick.bind(this));
    }

    /**
     * 停止生物特征采集
     */
    stop() {
        this.isCollecting = false;
        window.removeEventListener('keydown', this.handleKeyDown);
        window.removeEventListener('keyup', this.handleKeyUp);
        window.removeEventListener('mousemove', this.handleMouseMove);
        window.removeEventListener('mousedown', this.handleMouseDown);
        window.removeEventListener('mouseup', this.handleMouseUp);
        window.removeEventListener('click', this.handleClick);
    }

    /**
     * 键盘按下事件处理
     * @param {KeyboardEvent} event 
     */
    handleKeyDown(event) {
        if (!this.isCollecting) return;

        const timestamp = Date.now();
        const key = event.key || 'unknown';
        const keyCode = event.keyCode || event.which;

        const keyEvent = {
            key: key,
            key_code: keyCode,
            type: 'keydown',
            timestamp: timestamp
        };

        this.keyboardEvents.push(keyEvent);
        this.keyDownMap.set(`${key}-${keyCode}`, timestamp);
    }

    /**
     * 键盘释放事件处理
     * @param {KeyboardEvent} event 
     */
    handleKeyUp(event) {
        if (!this.isCollecting) return;

        const timestamp = Date.now();
        const key = event.key || 'unknown';
        const keyCode = event.keyCode || event.which;

        this.keyboardEvents.push({
            key: key,
            key_code: keyCode,
            type: 'keyup',
            timestamp: timestamp
        });

        const downKey = `${key}-${keyCode}`;
        if (this.keyDownMap.has(downKey)) {
            this.keyDownMap.delete(downKey);
        }
    }

    /**
     * 鼠标移动事件处理
     * @param {MouseEvent} event 
     */
    handleMouseMove(event) {
        if (!this.isCollecting) return;

        this.mouseEvents.push({
            type: 'mousemove',
            x: event.clientX,
            y: event.clientY,
            timestamp: Date.now()
        });
    }

    /**
     * 鼠标按下事件处理
     * @param {MouseEvent} event 
     */
    handleMouseDown(event) {
        if (!this.isCollecting) return;

        this.mouseEvents.push({
            type: 'mousedown',
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now()
        });
    }

    /**
     * 鼠标释放事件处理
     * @param {MouseEvent} event 
     */
    handleMouseUp(event) {
        if (!this.isCollecting) return;

        this.mouseEvents.push({
            type: 'mouseup',
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now()
        });
    }

    /**
     * 鼠标点击事件处理
     * @param {MouseEvent} event 
     */
    handleClick(event) {
        if (!this.isCollecting) return;

        this.mouseEvents.push({
            type: 'click',
            x: event.clientX,
            y: event.clientY,
            button: event.button,
            timestamp: Date.now()
        });
    }

    /**
     * 获取采集到的键盘样本
     * @returns {Object}
     */
    getKeyboardSample() {
        return {
            key_events: this.keyboardEvents.map(event => ({
                key: event.key,
                type: event.type,
                timestamp: event.timestamp,
                key_code: event.key_code
            })),
            timestamp: Date.now()
        };
    }

    /**
     * 获取采集到的鼠标样本
     * @returns {Object}
     */
    getMouseSample() {
        return {
            mouse_events: this.mouseEvents.map(event => ({
                type: event.type,
                x: event.x,
                y: event.y,
                timestamp: event.timestamp,
                button: event.button
            })),
            timestamp: Date.now()
        };
    }

    /**
     * 分析键盘打字特征
     */
    analyzeKeyboardPattern() {
        if (this.keyboardEvents.length < 10) {
            return null;
        }

        const holdTimes = [];
        const flightTimes = [];
        const keyDownTimes = new Map();

        for (let i = 0; i < this.keyboardEvents.length; i++) {
            const event = this.keyboardEvents[i];
            const keyId = `${event.key}-${event.key_code}`;

            if (event.type === 'keydown') {
                keyDownTimes.set(keyId, event.timestamp);
                
                if (i > 0 && this.keyboardEvents[i-1].type === 'keydown') {
                    flightTimes.push(event.timestamp - this.keyboardEvents[i-1].timestamp);
                }
            } else if (event.type === 'keyup') {
                if (keyDownTimes.has(keyId)) {
                    holdTimes.push(event.timestamp - keyDownTimes.get(keyId));
                    keyDownTimes.delete(keyId);
                }
            }
        }

        const stats = {
            hold_times: holdTimes,
            flight_times: flightTimes,
            avg_hold_time: this.calculateAverage(holdTimes),
            avg_flight_time: this.calculateAverage(flightTimes),
            hold_time_std: this.calculateStdDev(holdTimes),
            flight_time_std: this.calculateStdDev(flightTimes)
        };

        return stats;
    }

    /**
     * 分析鼠标移动模式
     */
    analyzeMousePattern() {
        if (this.mouseEvents.length < 10) {
            return null;
        }

        const speeds = [];
        const trajectories = [];
        const clickEvents = [];

        for (let i = 1; i < this.mouseEvents.length; i++) {
            const prev = this.mouseEvents[i - 1];
            const curr = this.mouseEvents[i];

            if (prev.type === 'mousemove' && curr.type === 'mousemove') {
                const dx = curr.x - prev.x;
                const dy = curr.y - prev.y;
                const distance = Math.sqrt(dx * dx + dy * dy);
                const dt = curr.timestamp - prev.timestamp;

                if (dt > 0) {
                    speeds.push(distance / dt);
                }
            }

            if (curr.type === 'click') {
                clickEvents.push(curr);
            }
        }

        return {
            speeds: speeds,
            avg_speed: this.calculateAverage(speeds),
            speed_std: this.calculateStdDev(speeds),
            click_count: clickEvents.length
        };
    }

    /**
     * 计算平均值
     */
    calculateAverage(arr) {
        if (!arr || arr.length === 0) return 0;
        return arr.reduce((sum, val) => sum + val, 0) / arr.length;
    }

    /**
     * 计算标准差
     */
    calculateStdDev(arr) {
        if (!arr || arr.length < 2) return 0;
        const avg = this.calculateAverage(arr);
        const sumOfSquares = arr.reduce((sum, val) => sum + Math.pow(val - avg, 2), 0);
        return Math.sqrt(sumOfSquares / arr.length);
    }

    /**
     * 获取完整的生物识别数据
     */
    getBiometricData() {
        return {
            user_id: this.userId,
            keyboard_sample: this.getKeyboardSample(),
            mouse_sample: this.getMouseSample(),
            keyboard_analysis: this.analyzeKeyboardPattern(),
            mouse_analysis: this.analyzeMousePattern()
        };
    }

    /**
     * 清空采集的数据
     */
    clear() {
        this.keyboardEvents = [];
        this.mouseEvents = [];
        this.keyDownMap.clear();
    }
}

class BiometricsService {
    constructor(apiBaseUrl = '/api/v1/biometrics') {
        this.apiBaseUrl = apiBaseUrl;
        this.collector = new BiometricsCollector();
    }

    /**
     * 开始采集
     */
    startCollecting(userId = null) {
        this.collector.start(userId);
    }

    /**
     * 停止采集
     */
    stopCollecting() {
        this.collector.stop();
    }

    /**
     * 注册生物特征档案
     */
    async registerProfile(userId) {
        const keyboardSample = this.collector.getKeyboardSample();
        const mouseSample = this.collector.getMouseSample();

        const response = await fetch(`${this.apiBaseUrl}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                keyboard_sample: keyboardSample,
                mouse_sample: mouseSample
            })
        });

        return await response.json();
    }

    /**
     * 验证生物特征
     */
    async verify(userId) {
        const keyboardSample = this.collector.getKeyboardSample();
        const mouseSample = this.collector.getMouseSample();

        const response = await fetch(`${this.apiBaseUrl}/verify`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                user_id: userId,
                keyboard_sample: keyboardSample,
                mouse_sample: mouseSample
            })
        });

        return await response.json();
    }

    /**
     * 获取生物特征档案
     */
    async getProfile(userId) {
        const response = await fetch(`${this.apiBaseUrl}/profile?user_id=${encodeURIComponent(userId)}`);
        return await response.json();
    }

    /**
     * 清空采集的数据
     */
    clear() {
        this.collector.clear();
    }
}

window.BiometricsCollector = BiometricsCollector;
window.BiometricsService = BiometricsService;
window.biometricsService = new BiometricsService();
