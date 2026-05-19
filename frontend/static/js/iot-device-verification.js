(function(window) {
    'use strict';

    const IoTDeviceVerification = {
        API_BASE: '/api/iot',

        async registerDevice(deviceData) {
            const payload = {
                device_type: deviceData.deviceType,
                device_name: deviceData.deviceName,
                manufacturer: deviceData.manufacturer,
                model: deviceData.model,
                firmware: deviceData.firmware,
                mac_address: deviceData.macAddress,
                ip_address: deviceData.ipAddress,
                location: deviceData.location,
                owner_id: deviceData.ownerId,
                capabilities: deviceData.capabilities || [],
                metadata: deviceData.metadata || {}
            };

            const response = await fetch(`${this.API_BASE}/devices/register`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Failed to register device');
            }

            return response.json();
        },

        async getDevice(deviceId) {
            const response = await fetch(`${this.API_BASE}/devices/${deviceId}`);

            if (!response.ok) {
                throw new Error('Failed to get device');
            }

            return response.json();
        },

        async verifyDevice(request) {
            const payload = {
                device_id: request.deviceId,
                fingerprint: request.fingerprint,
                challenge: request.challenge || '',
                signature: request.signature || '',
                timestamp: request.timestamp || new Date().toISOString(),
                connection_info: request.connectionInfo || {}
            };

            const response = await fetch(`${this.API_BASE}/devices/verify`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Failed to verify device');
            }

            return response.json();
        },

        async getDeviceFingerprint(deviceId) {
            const response = await fetch(`${this.API_BASE}/devices/${deviceId}/fingerprint`);

            if (!response.ok) {
                throw new Error('Failed to get device fingerprint');
            }

            return response.json();
        },

        async authenticateSmartHome(request) {
            const payload = {
                device_id: request.deviceId,
                device_type: request.deviceType,
                location: request.location,
                sensors: request.sensors || {},
                auth_method: request.authMethod || 'token'
            };

            const response = await fetch(`${this.API_BASE}/smart-home/auth`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Smart home authentication failed');
            }

            return response.json();
        },

        async authenticateVehicle(request) {
            const payload = {
                vehicle_id: request.vehicleId,
                vin: request.vin,
                obd_device_id: request.obdDeviceId || '',
                odometer: request.odometer,
                location: request.location,
                auth_challenge: request.authChallenge || '',
                signature: request.signature || ''
            };

            const response = await fetch(`${this.API_BASE}/vehicle/auth`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Vehicle authentication failed');
            }

            return response.json();
        },

        async authenticateIIoT(request) {
            const payload = {
                device_id: request.deviceId,
                device_type: request.deviceType,
                plant_id: request.plantId,
                production_line: request.productionLine,
                certificates: request.certificates || [],
                measurements: request.measurements || {},
                timestamp: request.timestamp || new Date().toISOString()
            };

            const response = await fetch(`${this.API_BASE}/iiot/auth`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('IIoT authentication failed');
            }

            return response.json();
        },

        async getDeviceHistory(deviceId, limit = 50, offset = 0) {
            const params = new URLSearchParams({
                device_id: deviceId,
                limit: limit,
                offset: offset
            });

            const response = await fetch(`${this.API_BASE}/devices/${deviceId}/history?${params}`);

            if (!response.ok) {
                throw new Error('Failed to get device history');
            }

            return response.json();
        },

        collectDeviceFingerprint() {
            const fp = {
                userAgent: navigator.userAgent,
                platform: navigator.platform,
                language: navigator.language,
                hardwareConcurrency: navigator.hardwareConcurrency,
                deviceMemory: navigator.deviceMemory || 'unknown',
                screenResolution: `${screen.width}x${screen.height}`,
                colorDepth: screen.colorDepth,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                touchSupport: 'ontouchstart' in window,
                canvas: this.getCanvasFingerprint(),
                webgl: this.getWebGLFingerprint(),
                fonts: this.getFontFingerprint(),
                plugins: Array.from(navigator.plugins).map(p => p.name).join(',')
            };

            fp.hash = this.computeFingerprintHash(fp);

            return fp;
        },

        getCanvasFingerprint() {
            try {
                const canvas = document.createElement('canvas');
                const ctx = canvas.getContext('2d');
                canvas.width = 200;
                canvas.height = 50;

                ctx.textBaseline = 'top';
                ctx.font = "14px 'Arial'";
                ctx.fillStyle = '#f60';
                ctx.fillRect(125, 1, 62, 20);
                ctx.fillStyle = '#069';
                ctx.fillText('IoT Fingerprint', 2, 15);
                ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
                ctx.fillText('IoT Fingerprint', 4, 17);

                return canvas.toDataURL().substring(0, 100);
            } catch (e) {
                return 'canvas_not_supported';
            }
        },

        getWebGLFingerprint() {
            try {
                const canvas = document.createElement('canvas');
                const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');

                if (!gl) return 'webgl_not_supported';

                const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                if (debugInfo) {
                    return gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) + '~' +
                           gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL);
                }
                return 'webgl_no_debug_info';
            } catch (e) {
                return 'webgl_error';
            }
        },

        getFontFingerprint() {
            const baseFonts = ['monospace', 'sans-serif', 'serif'];
            const testFonts = ['Arial', 'Verdana', 'Times New Roman', 'Courier New', 'Georgia'];

            const testString = 'mmmmmmmmmmlli';
            const testSize = '72px';

            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');

            const getWidth = (fontFamily) => {
                ctx.font = `${testSize} ${fontFamily}`;
                return ctx.measureText(testString).width;
            };

            const baseWidths = {};
            baseFonts.forEach(f => {
                baseWidths[f] = getWidth(f);
            });

            const detected = [];
            testFonts.forEach(font => {
                const detected_font = `${font}, ${baseFonts[0]}`;
                if (getWidth(detected_font) !== baseWidths[baseFonts[0]]) {
                    detected.push(font);
                }
            });

            return detected.join(',');
        },

        computeFingerprintHash(fp) {
            const data = JSON.stringify(fp);
            let hash = 0;
            for (let i = 0; i < data.length; i++) {
                const char = data.charCodeAt(i);
                hash = ((hash << 5) - hash) + char;
                hash = hash & hash;
            }
            return Math.abs(hash).toString(16);
        },

        collectSmartHomeSensors() {
            const sensors = {};

            if ('AmbientLightSensor' in window) {
                try {
                    const lightSensor = new AmbientLightSensor();
                    lightSensor.addEventListener('reading', () => {
                        sensors.ambient_light = lightSensor.illuminance;
                    });
                    lightSensor.addEventListener('error', () => {
                        sensors.ambient_light = null;
                    });
                    lightSensor.start();
                    setTimeout(() => lightSensor.stop(), 1000);
                } catch (e) {
                    sensors.ambient_light = null;
                }
            }

            if ('Accelerometer' in window) {
                try {
                    const accel = new Accelerometer();
                    accel.addEventListener('reading', () => {
                        sensors.acceleration_x = accel.x;
                        sensors.acceleration_y = accel.y;
                        sensors.acceleration_z = accel.z;
                    });
                    accel.start();
                    setTimeout(() => accel.stop(), 1000);
                } catch (e) {
                    sensors.accelerometer = null;
                }
            }

            sensors.battery_level = navigator.getBattery ?
                navigator.getBattery().then(b => sensors.battery = b.level) : null;
            sensors.connection_type = navigator.connection ?
                navigator.connection.effectiveType : 'unknown';

            return sensors;
        },

        initSmartHomeUI(container) {
            if (!container) return;

            container.innerHTML = `
                <div class="iot-smart-home-panel">
                    <div class="panel-header">
                        <h3>Smart Home Device Authentication</h3>
                    </div>
                    <form id="smarthome-auth-form">
                        <div class="form-group">
                            <label for="sh-device-id">Device ID</label>
                            <input type="text" id="sh-device-id" required placeholder="Enter device ID">
                        </div>
                        <div class="form-group">
                            <label for="sh-device-type">Device Type</label>
                            <select id="sh-device-type" required>
                                <option value="">Select Type</option>
                                <option value="thermostat">Thermostat</option>
                                <option value="camera">Camera</option>
                                <option value="lock">Smart Lock</option>
                                <option value="light">Smart Light</option>
                                <option value="sensor">Sensor</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="sh-location">Location</label>
                            <input type="text" id="sh-location" required placeholder="e.g., Living Room">
                        </div>
                        <div class="sensor-readings">
                            <h4>Sensor Readings</h4>
                            <div class="sensor-grid">
                                <div class="sensor-item">
                                    <label>Temperature (°C)</label>
                                    <input type="number" id="sh-temp" step="0.1" value="22.5">
                                </div>
                                <div class="sensor-item">
                                    <label>Humidity (%)</label>
                                    <input type="number" id="sh-humidity" min="0" max="100" value="45">
                                </div>
                                <div class="sensor-item">
                                    <label>Motion Detected</label>
                                    <select id="sh-motion">
                                        <option value="0">No</option>
                                        <option value="1">Yes</option>
                                    </select>
                                </div>
                            </div>
                        </div>
                        <button type="submit" class="btn-auth">Authenticate</button>
                    </form>
                    <div id="sh-result" class="result-container"></div>
                </div>
            `;

            document.getElementById('smarthome-auth-form').addEventListener('submit', async (e) => {
                e.preventDefault();
                await this.handleSmartHomeAuth();
            });
        },

        async handleSmartHomeAuth() {
            const deviceId = document.getElementById('sh-device-id').value;
            const deviceType = document.getElementById('sh-device-type').value;
            const location = document.getElementById('sh-location').value;
            const temp = parseFloat(document.getElementById('sh-temp').value);
            const humidity = parseFloat(document.getElementById('sh-humidity').value);
            const motion = parseInt(document.getElementById('sh-motion').value);

            const resultContainer = document.getElementById('sh-result');
            resultContainer.innerHTML = '<div class="loading">Authenticating...</div>';

            try {
                const response = await this.authenticateSmartHome({
                    deviceId,
                    deviceType,
                    location,
                    sensors: {
                        temperature: temp,
                        humidity: humidity,
                        motion: motion
                    }
                });

                this.displaySmartHomeResult(response, resultContainer);
            } catch (err) {
                resultContainer.innerHTML = `<div class="error">${err.message}</div>`;
            }
        },

        displaySmartHomeResult(response, container) {
            if (response.authenticated) {
                container.innerHTML = `
                    <div class="success-result">
                        <div class="success-icon">✓</div>
                        <h4>Authentication Successful</h4>
                        <p>${response.message}</p>
                        <div class="details">
                            <div><strong>Access Token:</strong> <code>${response.access_token}</code></div>
                            <div><strong>Expires In:</strong> ${response.expires_in} seconds</div>
                            <div><strong>Zone ID:</strong> ${response.zone_id}</div>
                            <div><strong>Permissions:</strong></div>
                            <ul>${response.permissions.map(p => `<li>${p}</li>`).join('')}</ul>
                        </div>
                    </div>
                `;
            } else {
                container.innerHTML = `
                    <div class="error-result">
                        <div class="error-icon">✗</div>
                        <h4>Authentication Failed</h4>
                        <p>${response.message}</p>
                    </div>
                `;
            }
        },

        initVehicleUI(container) {
            if (!container) return;

            container.innerHTML = `
                <div class="iot-vehicle-panel">
                    <div class="panel-header">
                        <h3>Vehicle Authentication</h3>
                    </div>
                    <form id="vehicle-auth-form">
                        <div class="form-group">
                            <label for="v-vehicle-id">Vehicle ID</label>
                            <input type="text" id="v-vehicle-id" required placeholder="Enter vehicle ID">
                        </div>
                        <div class="form-group">
                            <label for="v-vin">VIN (17 characters)</label>
                            <input type="text" id="v-vin" required minlength="17" maxlength="17" placeholder="1HGBH41JXMN109186">
                        </div>
                        <div class="form-group">
                            <label for="v-odometer">Odometer (km)</label>
                            <input type="number" id="v-odometer" required value="0">
                        </div>
                        <div class="form-group">
                            <label for="v-location">Location</label>
                            <input type="text" id="v-location" placeholder="City, State">
                        </div>
                        <button type="submit" class="btn-auth">Authenticate Vehicle</button>
                    </form>
                    <div id="v-result" class="result-container"></div>
                </div>
            `;

            document.getElementById('vehicle-auth-form').addEventListener('submit', async (e) => {
                e.preventDefault();
                await this.handleVehicleAuth();
            });
        },

        async handleVehicleAuth() {
            const vehicleId = document.getElementById('v-vehicle-id').value;
            const vin = document.getElementById('v-vin').value.toUpperCase();
            const odometer = parseInt(document.getElementById('v-odometer').value);
            const location = document.getElementById('v-location').value;

            const resultContainer = document.getElementById('v-result');
            resultContainer.innerHTML = '<div class="loading">Authenticating...</div>';

            try {
                const response = await this.authenticateVehicle({
                    vehicleId,
                    vin,
                    odometer,
                    location
                });

                this.displayVehicleResult(response, resultContainer);
            } catch (err) {
                resultContainer.innerHTML = `<div class="error">${err.message}</div>`;
            }
        },

        displayVehicleResult(response, container) {
            if (response.authorized) {
                container.innerHTML = `
                    <div class="success-result">
                        <div class="success-icon">✓</div>
                        <h4>Vehicle Authorized</h4>
                        <p>${response.message}</p>
                        <div class="details">
                            <div><strong>Security Level:</strong> ${response.security_level}</div>
                            <div><strong>Expires:</strong> ${new Date(response.expires_at).toLocaleString()}</div>
                            <div><strong>Available Features:</strong></div>
                            <ul>${response.features.map(f => `<li>${f}</li>`).join('')}</ul>
                        </div>
                    </div>
                `;
            } else {
                container.innerHTML = `
                    <div class="error-result">
                        <div class="error-icon">✗</div>
                        <h4>Authorization Failed</h4>
                        <p>${response.message}</p>
                    </div>
                `;
            }
        },

        initIIoTUI(container) {
            if (!container) return;

            container.innerHTML = `
                <div class="iot-iiot-panel">
                    <div class="panel-header">
                        <h3>Industrial IoT Authentication</h3>
                    </div>
                    <form id="iiot-auth-form">
                        <div class="form-group">
                            <label for="iiot-device-id">Device ID</label>
                            <input type="text" id="iiot-device-id" required placeholder="Enter IIoT device ID">
                        </div>
                        <div class="form-group">
                            <label for="iiot-device-type">Device Type</label>
                            <select id="iiot-device-type" required>
                                <option value="">Select Type</option>
                                <option value="plc">PLC Controller</option>
                                <option value="sensor">Industrial Sensor</option>
                                <option value="actuator">Actuator</option>
                                <option value="gateway">Gateway</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="iiot-plant">Plant ID</label>
                            <input type="text" id="iiot-plant" required placeholder="e.g., plant-001">
                        </div>
                        <div class="form-group">
                            <label for="iiot-line">Production Line</label>
                            <input type="text" id="iiot-line" required placeholder="e.g., assembly-line-1">
                        </div>
                        <div class="form-group">
                            <label>Certificates (one per line)</label>
                            <textarea id="iiot-certificates" rows="3" placeholder="iso9001&#10;iec62443"></textarea>
                        </div>
                        <div class="measurements">
                            <h4>Measurements</h4>
                            <div class="measurement-grid">
                                <div class="measurement-item">
                                    <label>Temperature (°C)</label>
                                    <input type="number" id="iiot-temp" step="0.1" value="45.0">
                                </div>
                                <div class="measurement-item">
                                    <label>Pressure (bar)</label>
                                    <input type="number" id="iiot-pressure" step="0.1" value="2.5">
                                </div>
                            </div>
                        </div>
                        <button type="submit" class="btn-auth">Authenticate IIoT Device</button>
                    </form>
                    <div id="iiot-result" class="result-container"></div>
                </div>
            `;

            document.getElementById('iiot-auth-form').addEventListener('submit', async (e) => {
                e.preventDefault();
                await this.handleIIoTAuth();
            });
        },

        async handleIIoTAuth() {
            const deviceId = document.getElementById('iiot-device-id').value;
            const deviceType = document.getElementById('iiot-device-type').value;
            const plantId = document.getElementById('iiot-plant').value;
            const productionLine = document.getElementById('iiot-line').value;
            const certsText = document.getElementById('iiot-certificates').value;
            const temp = parseFloat(document.getElementById('iiot-temp').value);
            const pressure = parseFloat(document.getElementById('iiot-pressure').value);

            const certificates = certsText.split('\n').filter(c => c.trim());

            const resultContainer = document.getElementById('iiot-result');
            resultContainer.innerHTML = '<div class="loading">Authenticating...</div>';

            try {
                const response = await this.authenticateIIoT({
                    deviceId,
                    deviceType,
                    plantId,
                    productionLine,
                    certificates,
                    measurements: {
                        temperature: temp,
                        pressure: pressure
                    }
                });

                this.displayIIoTResult(response, resultContainer);
            } catch (err) {
                resultContainer.innerHTML = `<div class="error">${err.message}</div>`;
            }
        },

        displayIIoTResult(response, container) {
            if (response.authorized) {
                container.innerHTML = `
                    <div class="success-result">
                        <div class="success-icon">✓</div>
                        <h4>IIoT Device Authorized</h4>
                        <p>${response.message}</p>
                        <div class="details">
                            <div><strong>Session Token:</strong> <code>${response.session_token}</code></div>
                            <div><strong>Expires:</strong> ${new Date(response.expires_at).toLocaleString()}</div>
                            <div><strong>Compliance:</strong> <span class="badge ${response.compliance}">${response.compliance}</span></div>
                            <div><strong>Access Level:</strong> ${response.access_level}</div>
                        </div>
                    </div>
                `;
            } else {
                container.innerHTML = `
                    <div class="error-result">
                        <div class="error-icon">✗</div>
                        <h4>Authorization Failed</h4>
                        <p>${response.message}</p>
                    </div>
                `;
            }
        }
    };

    window.IoTDeviceVerification = IoTDeviceVerification;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = IoTDeviceVerification;
    }

})(typeof window !== 'undefined' ? window : global);
