class TraceCollector {
    constructor() {
        this.points = [];
        this.startTime = null;
        this.isCollecting = false;
        this.lastPoint = null;
        this.deviceInfo = this.getDeviceInfo();
    }

    getDeviceInfo() {
        return {
            userAgent: navigator.userAgent,
            screenWidth: window.screen.width,
            screenHeight: window.screen.height,
            devicePixelRatio: window.devicePixelRatio,
            touchSupport: 'ontouchstart' in window,
            platform: navigator.platform,
            language: navigator.language
        };
    }

    start() {
        this.points = [];
        this.startTime = Date.now();
        this.isCollecting = true;
        this.lastPoint = null;
    }

    addPoint(eventType, x, y) {
        if (!this.isCollecting) return;

        const point = {
            t: Date.now(),
            x: x,
            y: y,
            e: eventType
        };

        this.points.push(point);

        if (eventType === 'start') {
            this.startTime = Date.now();
        }

        this.lastPoint = point;
    }

    addMovePoint(x, y) {
        if (!this.isCollecting) return;

        if (this.lastPoint) {
            const timeDiff = Date.now() - this.lastPoint.t;
            if (timeDiff < 5) {
                return;
            }
        }

        const point = {
            t: Date.now(),
            x: x,
            y: y,
            e: 'move'
        };

        this.points.push(point);
        this.lastPoint = point;
    }

    end() {
        if (!this.isCollecting) return null;
        this.isCollecting = false;

        if (this.points.length === 0) return null;

        const lastPoint = this.points[this.points.length - 1];
        const firstPoint = this.points[0];

        return {
            points: this.points,
            total_time: lastPoint.t - this.startTime,
            start_x: firstPoint.x,
            start_y: firstPoint.y,
            end_x: lastPoint.x,
            end_y: lastPoint.y,
            device_info: this.deviceInfo
        };
    }

    clear() {
        this.points = [];
        this.startTime = null;
        this.isCollecting = false;
        this.lastPoint = null;
    }

    getPoints() {
        return this.points;
    }

    getPointCount() {
        return this.points.length;
    }

    isActive() {
        return this.isCollecting;
    }
}

class TraceEncryptor {
    constructor() {
        this.secretKey = 'captcha-trajectory-secret-key-2024';
        this.saltLength = 16;
    }

    generateSalt() {
        const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        let salt = '';
        const randomValues = new Uint8Array(this.saltLength);
        crypto.getRandomValues(randomValues);
        for (let i = 0; i < this.saltLength; i++) {
            salt += chars[randomValues[i] % chars.length];
        }
        return salt;
    }

    async encryptData(data, salt) {
        const key = await this.deriveKey(salt);
        const encoder = new TextEncoder();
        const dataBytes = encoder.encode(JSON.stringify(data));

        const iv = crypto.getRandomValues(new Uint8Array(12));

        const encryptedContent = await crypto.subtle.encrypt(
            {
                name: 'AES-GCM',
                iv: iv
            },
            key,
            dataBytes
        );

        const combined = new Uint8Array(iv.length + encryptedContent.byteLength);
        combined.set(iv, 0);
        combined.set(new Uint8Array(encryptedContent), iv.length);

        return this.arrayBufferToBase64(combined);
    }

    async deriveKey(salt) {
        const encoder = new TextEncoder();
        const keyMaterial = await crypto.subtle.importKey(
            'raw',
            encoder.encode(this.secretKey),
            { name: 'PBKDF2' },
            false,
            ['deriveBits', 'deriveKey']
        );

        const saltBytes = encoder.encode(salt);

        return crypto.subtle.deriveKey(
            {
                name: 'PBKDF2',
                salt: saltBytes,
                iterations: 100000,
                hash: 'SHA-256'
            },
            keyMaterial,
            { name: 'AES-GCM', length: 256 },
            false,
            ['encrypt', 'decrypt']
        );
    }

    generateSignature(timestamp, salt, encryptedData) {
        const data = `${timestamp}:${salt}:${encryptedData}`;
        const encoder = new TextEncoder();
        const key = encoder.encode(this.secretKey);

        return this.hmacSHA256Sync(key, encoder.encode(data));
    }

    hmacSHA256Sync(key, data) {
        const hmac = {
            inner: new Uint8Array(64),
            outer: new Uint8Array(64)
        };

        for (let i = 0; i < 64; i++) {
            hmac.inner[i] = 0x36 ^ key[i % key.length];
            hmac.outer[i] = 0x5c ^ key[i % key.length];
        }

        let innerHash = this.sha256(this.concat(hmac.inner, data));
        let outerHash = this.sha256(this.concat(hmac.outer, innerHash));

        return this.arrayBufferToBase64(outerHash);
    }

    sha256(data) {
        const hash = new Uint8Array(32);
        let h0 = 0x6a09e667, h1 = 0xbb67ae85, h2 = 0x3c6ef372, h3 = 0xa54ff53a;
        let h4 = 0x510e527f, h5 = 0x9b05688c, h6 = 0x1f83d9ab, h7 = 0x5be0cd19;

        const k = new Uint32Array(64);
        for (let i = 0; i < 64; i++) {
            k[i] = Math.floor(Math.abs(Math.sin(i + 1) * 0x100000000));
        }

        const bytes = data;
        const ml = bytes.length * 8 + 1;
        const mlBytes = Math.ceil(ml / 8);
        const paddedLength = (mlBytes % 64 === 0 ? mlBytes + 64 : Math.ceil(mlBytes / 64) * 64);
        const padded = new Uint8Array(paddedLength);
        padded.set(bytes);
        padded[bytes.length] = 0x80;
        const view = new DataView(padded.buffer);
        view.setUint32(paddedLength - 4, bytes.length * 8, false);

        for (let chunk = 0; chunk < paddedLength; chunk += 64) {
            const w = new Uint32Array(64);
            for (let i = 0; i < 16; i++) {
                w[i] = view.getUint32(chunk + i * 4, false);
            }
            for (let i = 16; i < 64; i++) {
                const s0 = this.rotr32(w[i-15], 7) ^ this.rotr32(w[i-15], 18) ^ (w[i-15] >>> 3);
                const s1 = this.rotr32(w[i-2], 17) ^ this.rotr32(w[i-2], 19) ^ (w[i-2] >>> 10);
                w[i] = (w[i-16] + s0 + w[i-7] + s1) >>> 0;
            }

            let a = h0, b = h1, c = h2, d = h3, e = h4, f = h5, g = h6, hh = h7;

            for (let i = 0; i < 64; i++) {
                const S1 = this.rotr32(e, 6) ^ this.rotr32(e, 11) ^ this.rotr32(e, 25);
                const ch = (e & f) ^ ((~e) & g);
                const temp1 = (hh + S1 + ch + k[i] + w[i]) >>> 0;
                const S0 = this.rotr32(a, 2) ^ this.rotr32(a, 13) ^ this.rotr32(a, 22);
                const maj = (a & b) ^ (a & c) ^ (b & c);
                const temp2 = (S0 + maj) >>> 0;

                hh = g;
                g = f;
                f = e;
                e = (d + temp1) >>> 0;
                d = c;
                c = b;
                b = a;
                a = (temp1 + temp2) >>> 0;
            }

            h0 = (h0 + a) >>> 0;
            h1 = (h1 + b) >>> 0;
            h2 = (h2 + c) >>> 0;
            h3 = (h3 + d) >>> 0;
            h4 = (h4 + e) >>> 0;
            h5 = (h5 + f) >>> 0;
            h6 = (h6 + g) >>> 0;
            h7 = (h7 + hh) >>> 0;
        }

        const result = new Uint8Array(32);
        const view2 = new DataView(result.buffer);
        view2.setUint32(0, h0, false);
        view2.setUint32(4, h1, false);
        view2.setUint32(8, h2, false);
        view2.setUint32(12, h3, false);
        view2.setUint32(16, h4, false);
        view2.setUint32(20, h5, false);
        view2.setUint32(24, h6, false);
        view2.setUint32(28, h7, false);

        return result;
    }

    rotr32(x, n) {
        return (x >>> n) | (x << (32 - n));
    }

    concat(a, b) {
        const result = new Uint8Array(a.length + b.length);
        result.set(a);
        result.set(b, a.length);
        return result;
    }

    arrayBufferToBase64(buffer) {
        const bytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
    }

    base64ToArrayBuffer(base64) {
        const binaryString = atob(base64);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes;
    }
}

class TraceTransport {
    constructor() {
        this.encryptor = new TraceEncryptor();
        this.currentSessionId = null;
    }

    setSessionId(sessionId) {
        this.currentSessionId = sessionId;
    }

    async encryptData(data) {
        const salt = this.encryptor.generateSalt();
        const timestamp = Date.now();
        const encrypted = await this.encryptor.encryptData(data, salt);
        const signature = this.encryptor.generateSignature(timestamp, salt, encrypted);

        return {
            salt: salt,
            timestamp: timestamp,
            encrypted: encrypted,
            signature: signature
        };
    }

    async sendTraceData(traceData, positionX, positionY) {
        if (!this.currentSessionId) {
            console.error('Session ID not set');
            return { success: false, error: 'Session ID not set' };
        }

        try {
            const encrypted = await this.encryptData(traceData);

            const response = await fetch('/api/v1/captcha/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Trace-Encrypted': encrypted.encrypted,
                    'X-Trace-Salt': encrypted.salt,
                    'X-Trace-Timestamp': encrypted.timestamp.toString(),
                    'X-Trace-Signature': encrypted.signature
                },
                body: JSON.stringify({
                    session_id: this.currentSessionId,
                    position_x: positionX,
                    position_y: positionY,
                    trace_data: traceData,
                    trace_encrypted: encrypted.encrypted
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            console.error('Failed to send trace data:', error);
            return { success: false, error: error.message };
        }
    }
}

window.traceCollector = new TraceCollector();
window.traceEncryptor = new TraceEncryptor();
window.traceTransport = new TraceTransport();
