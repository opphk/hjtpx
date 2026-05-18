/**
 * WebSocket Verification Client
 * 实时双向验证客户端
 */
class WebSocketVerificationClient {
    constructor(options = {}) {
        this.url = options.url || this.getDefaultWebSocketUrl();
        this.reconnectInterval = options.reconnectInterval || 3000;
        this.maxReconnectAttempts = options.maxReconnectAttempts || 5;
        this.pingInterval = options.pingInterval || 25000;
        
        this.clientId = this.generateClientId();
        this.sessionId = null;
        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.pingTimer = null;
        this.challengeId = null;
        this.currentChallenge = null;
        
        // 事件回调
        this.callbacks = {
            onOpen: options.onOpen || (() => {}),
            onClose: options.onClose || (() => {}),
            onError: options.onError || (() => {}),
            onChallenge: options.onChallenge || (() => {}),
            onResult: options.onResult || (() => {}),
            onMessage: options.onMessage || (() => {})
        };
    }

    /**
     * 获取默认 WebSocket URL
     */
    getDefaultWebSocketUrl() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        return `${protocol}//${host}/api/v1/websocket/verify`;
    }

    /**
     * 生成客户端 ID
     */
    generateClientId() {
        return 'client_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now();
    }

    /**
     * 连接 WebSocket
     */
    connect() {
        try {
            console.log('[WebSocket] Connecting to', this.url);
            this.ws = new WebSocket(this.url);
            
            this.ws.onopen = (event) => this.handleOpen(event);
            this.ws.onmessage = (event) => this.handleMessage(event);
            this.ws.onclose = (event) => this.handleClose(event);
            this.ws.onerror = (error) => this.handleError(error);
            
        } catch (error) {
            console.error('[WebSocket] Connection error:', error);
            this.callbacks.onError(error);
            this.attemptReconnect();
        }
    }

    /**
     * 处理连接打开事件
     */
    handleOpen(event) {
        console.log('[WebSocket] Connection established');
        this.isConnected = true;
        this.reconnectAttempts = 0;
        
        // 开始心跳
        this.startPing();
        
        // 发送握手消息
        this.sendHello();
        
        // 调用回调
        this.callbacks.onOpen(event);
    }

    /**
     * 处理消息事件
     */
    handleMessage(event) {
        try {
            const message = JSON.parse(event.data);
            console.log('[WebSocket] Received:', message.type);
            
            switch (message.type) {
                case 'hello_ack':
                    this.handleHelloAck(message);
                    break;
                case 'challenge':
                    this.handleChallenge(message);
                    break;
                case 'result':
                    this.handleResult(message);
                    break;
                case 'pong':
                    this.handlePong(message);
                    break;
                case 'error':
                    this.handleErrorResponse(message);
                    break;
                default:
                    this.callbacks.onMessage(message);
            }
        } catch (error) {
            console.error('[WebSocket] Message parsing error:', error);
        }
    }

    /**
     * 处理连接关闭事件
     */
    handleClose(event) {
        console.log('[WebSocket] Connection closed:', event.code, event.reason);
        this.isConnected = false;
        this.stopPing();
        this.callbacks.onClose(event);
        this.attemptReconnect();
    }

    /**
     * 处理错误事件
     */
    handleError(error) {
        console.error('[WebSocket] Error:', error);
        this.callbacks.onError(error);
    }

    /**
     * 发送握手消息
     */
    sendHello() {
        const message = {
            type: 'hello',
            payload: {
                client_id: this.clientId,
                user_agent: navigator.userAgent
            },
            timestamp: Date.now()
        };
        this.send(message);
    }

    /**
     * 处理握手确认
     */
    handleHelloAck(message) {
        console.log('[WebSocket] Handshake complete, session:', message.payload.session_id);
        this.sessionId = message.payload.session_id;
    }

    /**
     * 处理验证挑战
     */
    handleChallenge(message) {
        this.challengeId = message.payload.challenge_id;
        this.currentChallenge = message.payload;
        console.log('[WebSocket] Challenge received:', message.payload.type);
        this.callbacks.onChallenge(message.payload);
    }

    /**
     * 处理验证结果
     */
    handleResult(message) {
        console.log('[WebSocket] Result received:', message.payload.success);
        this.callbacks.onResult(message.payload);
        
        if (message.payload.success && message.payload.token) {
            // 验证成功，可以保存 token
            console.log('[WebSocket] Verification token:', message.payload.token);
        }
    }

    /**
     * 处理错误响应
     */
    handleErrorResponse(message) {
        console.error('[WebSocket] Server error:', message.payload);
        this.callbacks.onError(new Error(message.payload.message));
    }

    /**
     * 发送验证答案
     */
    sendAnswer(answerData) {
        if (!this.challengeId) {
            console.warn('[WebSocket] No active challenge');
            return;
        }

        const message = {
            type: 'answer',
            payload: {
                challenge_id: this.challengeId,
                data: answerData
            },
            timestamp: Date.now()
        };
        this.send(message);
    }

    /**
     * 发送 ping
     */
    sendPing() {
        const message = {
            type: 'ping',
            timestamp: Date.now()
        };
        this.send(message);
    }

    /**
     * 处理 pong
     */
    handlePong(message) {
        // 可以在这里计算延迟等
    }

    /**
     * 开始心跳
     */
    startPing() {
        this.stopPing();
        this.pingTimer = setInterval(() => {
            if (this.isConnected) {
                this.sendPing();
            }
        }, this.pingInterval);
    }

    /**
     * 停止心跳
     */
    stopPing() {
        if (this.pingTimer) {
            clearInterval(this.pingTimer);
            this.pingTimer = null;
        }
    }

    /**
     * 发送消息
     */
    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.warn('[WebSocket] Cannot send, connection not open');
        }
    }

    /**
     * 尝试重连
     */
    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`[WebSocket] Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
            
            setTimeout(() => {
                this.connect();
            }, this.reconnectInterval);
        } else {
            console.error('[WebSocket] Max reconnect attempts reached');
        }
    }

    /**
     * 断开连接
     */
    disconnect() {
        this.stopPing();
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.isConnected = false;
    }

    /**
     * 获取连接状态
     */
    getStatus() {
        return {
            isConnected: this.isConnected,
            clientId: this.clientId,
            sessionId: this.sessionId,
            readyState: this.ws ? this.ws.readyState : -1
        };
    }
}

// 全局实例（可选）
let wsClient = null;

/**
 * 获取 WebSocket 客户端单例
 */
function getWebSocketClient(options = {}) {
    if (!wsClient) {
        wsClient = new WebSocketVerificationClient(options);
    }
    return wsClient;
}

/**
 * 创建简单的验证 UI
 */
function createVerificationUI(containerId, options = {}) {
    const container = document.getElementById(containerId);
    if (!container) {
        console.error('Container not found:', containerId);
        return;
    }

    container.innerHTML = `
        <div class="ws-verification-container">
            <div class="ws-status-bar">
                <span class="ws-status-icon" id="ws-status-icon">🔴</span>
                <span class="ws-status-text" id="ws-status-text">未连接</span>
            </div>
            
            <div class="ws-challenge-area" id="ws-challenge-area">
                <p class="ws-placeholder">点击下方按钮开始验证</p>
            </div>
            
            <div class="ws-actions">
                <button class="ws-btn ws-btn-primary" id="ws-start-btn">
                    <span class="ws-btn-icon">▶</span>
                    开始验证
                </button>
                <button class="ws-btn ws-btn-secondary" id="ws-reset-btn" style="display: none;">
                    <span class="ws-btn-icon">🔄</span>
                    重新验证
                </button>
            </div>
            
            <div class="ws-result-area" id="ws-result-area" style="display: none;">
                <div class="ws-result-icon" id="ws-result-icon"></div>
                <div class="ws-result-message" id="ws-result-message"></div>
                <div class="ws-result-token" id="ws-result-token" style="display: none;"></div>
            </div>
        </div>
    `;

    const client = getWebSocketClient({
        onOpen: () => updateStatus('🟢', '已连接'),
        onClose: () => updateStatus('🔴', '已断开'),
        onError: (error) => {
            updateStatus('🟡', '连接出错');
            showError(error.message || '连接错误');
        },
        onChallenge: (challenge) => showChallenge(challenge),
        onResult: (result) => showResult(result)
    });

    // 更新状态
    function updateStatus(icon, text) {
        document.getElementById('ws-status-icon').textContent = icon;
        document.getElementById('ws-status-text').textContent = text;
    }

    // 显示挑战
    function showChallenge(challenge) {
        const area = document.getElementById('ws-challenge-area');
        let content = '';

        switch (challenge.type) {
            case 'slider':
                content = createSliderChallenge(challenge);
                break;
            case 'click':
                content = createClickChallenge(challenge);
                break;
            case 'rotation':
                content = createRotationChallenge(challenge);
                break;
            case 'gesture':
                content = createGestureChallenge(challenge);
                break;
            default:
                content = `<p>请完成验证: ${challenge.type}</p>`;
        }

        area.innerHTML = content;
    }

    // 滑块验证 UI
    function createSliderChallenge(challenge) {
        return `
            <div class="ws-slider-challenge">
                <h3>🔲 滑块验证</h3>
                <p>拖动滑块完成拼图</p>
                <div class="ws-slider-track">
                    <div class="ws-slider-progress" id="ws-slider-progress"></div>
                    <div class="ws-slider-handle" id="ws-slider-handle">></div>
                </div>
                <div class="ws-slider-value" id="ws-slider-value">0%</div>
                <button class="ws-btn ws-btn-small" id="ws-submit-slider">提交答案</button>
            </div>
        `;
    }

    // 点击验证 UI
    function createClickChallenge(challenge) {
        const points = challenge.data.points || [];
        return `
            <div class="ws-click-challenge">
                <h3>👆 点击验证</h3>
                <p>${challenge.data.hint || '依次点击指定位置'}</p>
                <div class="ws-click-area" id="ws-click-area">
                    ${points.map((p, i) => `<div class="ws-click-point" data-x="${p.x}" data-y="${p.y}">${i + 1}</div>`).join('')}
                </div>
                <button class="ws-btn ws-btn-small" id="ws-submit-click">提交答案</button>
            </div>
        `;
    }

    // 旋转验证 UI
    function createRotationChallenge(challenge) {
        return `
            <div class="ws-rotation-challenge">
                <h3>🔄 旋转验证</h3>
                <p>将图片旋转到正确角度</p>
                <div class="ws-rotation-track">
                    <div class="ws-rotation-progress" id="ws-rotation-progress"></div>
                    <div class="ws-rotation-handle" id="ws-rotation-handle">↻</div>
                </div>
                <div class="ws-rotation-value" id="ws-rotation-value">0°</div>
                <button class="ws-btn ws-btn-small" id="ws-submit-rotation">提交答案</button>
            </div>
        `;
    }

    // 手势验证 UI
    function createGestureChallenge(challenge) {
        return `
            <div class="ws-gesture-challenge">
                <h3>✍️ 手势验证</h3>
                <p>${challenge.data.hint || '绘制指定手势'}</p>
                <canvas class="ws-gesture-canvas" id="ws-gesture-canvas" width="300" height="200"></canvas>
                <button class="ws-btn ws-btn-small" id="ws-clear-gesture">清除</button>
                <button class="ws-btn ws-btn-small" id="ws-submit-gesture">提交答案</button>
            </div>
        `;
    }

    // 显示结果
    function showResult(result) {
        const area = document.getElementById('ws-result-area');
        const icon = document.getElementById('ws-result-icon');
        const message = document.getElementById('ws-result-message');
        const token = document.getElementById('ws-result-token');
        const resetBtn = document.getElementById('ws-reset-btn');

        area.style.display = 'block';
        resetBtn.style.display = 'inline-block';

        if (result.success) {
            icon.textContent = '✅';
            message.textContent = result.message || '验证成功！';
            if (result.token) {
                token.style.display = 'block';
                token.textContent = `Token: ${result.token}`;
            }
        } else {
            icon.textContent = '❌';
            message.textContent = result.message || '验证失败，请重试';
        }
    }

    // 显示错误
    function showError(message) {
        const area = document.getElementById('ws-result-area');
        area.style.display = 'block';
        document.getElementById('ws-result-icon').textContent = '⚠️';
        document.getElementById('ws-result-message').textContent = message;
    }

    // 事件绑定
    document.getElementById('ws-start-btn').addEventListener('click', () => {
        client.connect();
        document.getElementById('ws-start-btn').style.display = 'none';
    });

    document.getElementById('ws-reset-btn').addEventListener('click', () => {
        document.getElementById('ws-result-area').style.display = 'none';
        document.getElementById('ws-challenge-area').innerHTML = '<p class="ws-placeholder">正在重新连接...</p>';
        document.getElementById('ws-reset-btn').style.display = 'none';
        client.connect();
    });

    // 为滑块添加事件监听
    container.addEventListener('click', (e) => {
        if (e.target.id === 'ws-submit-slider') {
            client.sendAnswer({ type: 'slider', position: 50 });
        } else if (e.target.id === 'ws-submit-click') {
            client.sendAnswer({ type: 'click', points: [] });
        } else if (e.target.id === 'ws-submit-rotation') {
            client.sendAnswer({ type: 'rotation', angle: 0 });
        } else if (e.target.id === 'ws-submit-gesture') {
            client.sendAnswer({ type: 'gesture', gesture: 'checkmark' });
        }
    });

    return client;
}
