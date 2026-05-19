class VRCaptcha {
    constructor() {
        this.sessionId = null;
        this.vrConfig = null;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.objects = [];
        this.targets = [];
        this.selectedObject = null;
        this.isDragging = false;
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.timerInterval = null;
        this.seconds = 0;
        this.animationFrameId = null;
        this.isAnimating = false;
        this.xrSession = null;
        this.xrViewerSpace = null;
        this.xrReferenceSpace = null;
        this.xrSupported = false;
        this.currentMode = 'interactive';
        this.gestureData = null;
        this.interactionData = {
            objectPositions: {},
            objectRotations: {},
            completionOrder: [],
            timeSpent: 0,
            movementCount: 0
        };
        this.startTime = Date.now();
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.checkWebXRSupport();
        this.refresh();
    }

    bindEvents() {
        document.getElementById('refresh-btn').addEventListener('click', () => this.refresh());
        document.getElementById('submit-btn').addEventListener('click', () => this.submit());
        document.getElementById('difficulty-select').addEventListener('change', (e) => {
            this.updateDifficultyDisplay(e.target.value);
        });
        document.getElementById('xr-button').addEventListener('click', () => this.enterVR());
        
        document.querySelectorAll('.mode-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.mode-btn').forEach(b => b.classList.remove('active'));
                e.target.closest('.mode-btn').classList.add('active');
                this.currentMode = e.target.closest('.mode-btn').dataset.mode;
                this.updateInstruction();
                this.refresh();
            });
        });
    }

    updateDifficultyDisplay(difficulty) {
        const displayNames = {
            'easy': '简单',
            'medium': '中等',
            'hard': '困难',
            'expert': '专家'
        };
        document.getElementById('difficulty-display').textContent = `难度: ${displayNames[difficulty]}`;
    }

    updateInstruction() {
        const instructions = {
            'interactive': '点击物体并拖动，将它们放置到对应的目标位置',
            'gesture': '根据提示做出相应的手势动作',
            'puzzle': '按顺序完成空间拼图，将每个块放到正确位置'
        };
        document.getElementById('instruction-text').textContent = instructions[this.currentMode];
    }

    async checkWebXRSupport() {
        if (navigator.xr) {
            try {
                const supported = await navigator.xr.isSessionSupported('immersive-vr');
                this.xrSupported = supported;
                document.getElementById('xr-status').textContent = `WebXR: ${supported ? '支持' : '不支持'}`;
                document.getElementById('xr-button').disabled = !supported;
                if (!supported) {
                    document.getElementById('xr-button').innerHTML = '<i class="fas fa-desktop me-2"></i>仅桌面模式';
                }
            } catch (error) {
                console.error('WebXR support check failed:', error);
                this.xrSupported = false;
                document.getElementById('xr-status').textContent = 'WebXR: 检测失败';
            }
        } else {
            this.xrSupported = false;
            document.getElementById('xr-status').textContent = 'WebXR: 不支持';
            document.getElementById('xr-button').disabled = true;
            document.getElementById('xr-button').innerHTML = '<i class="fas fa-desktop me-2"></i>仅桌面模式';
        }
    }

    async enterVR() {
        if (!this.xrSupported) {
            alert('您的浏览器或设备不支持WebXR');
            return;
        }

        try {
            this.xrSession = await navigator.xr.requestSession('immersive-vr', {
                requiredFeatures: ['local-floor'],
                optionalFeatures: ['hand-tracking', 'hit-test']
            });

            this.xrSession.addEventListener('end', () => this.onXREnd());

            this.renderer.xr.setSession(this.xrSession);
            
            this.xrReferenceSpace = await this.xrSession.requestReferenceSpace('local-floor');

            document.getElementById('xr-status-display').innerHTML = '<i class="fas fa-vr-cardboard"></i> VR 模式';

        } catch (error) {
            console.error('Failed to enter VR:', error);
            alert('无法进入VR模式: ' + error.message);
        }
    }

    onXREnd() {
        this.xrSession = null;
        this.xrReferenceSpace = null;
        document.getElementById('xr-status-display').innerHTML = '<i class="fas fa-desktop"></i> 桌面模式';
    }

    async refresh() {
        this.showLoading(true);
        this.hideResult();
        this.resetTimer();
        this.cleanup();
        this.startTime = Date.now();
        this.interactionData = {
            objectPositions: {},
            objectRotations: {},
            completionOrder: [],
            timeSpent: 0,
            movementCount: 0
        };

        const difficulty = document.getElementById('difficulty-select').value;
        const modeMap = {
            'interactive': { mode: 'interactive', type: '3d_placement' },
            'gesture': { mode: 'gesture', type: 'vr_gesture' },
            'puzzle': { mode: 'puzzle', type: 'spatial_puzzle' }
        };
        const { mode, type } = modeMap[this.currentMode];

        try {
            const response = await fetch('/api/v1/captcha/vr/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ mode, type, difficulty })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.sessionId = result.data.sessionID;
                this.vrConfig = result.data.vrConfig;
                this.initThreeJS();
                this.renderScene();
                this.startTimer();
            } else {
                this.showResult(false, result.message || '生成验证码失败');
            }
        } catch (error) {
            console.error('Error:', error);
            this.showResult(false, '网络错误，请重试');
        } finally {
            this.showLoading(false);
        }
    }

    initThreeJS() {
        const container = document.getElementById('canvas-container');
        const width = container.clientWidth;
        const height = container.clientHeight;

        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x0d1b2a);

        this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
        this.camera.position.set(0, 1.6, 5);

        this.renderer = new THREE.WebGLRenderer({ 
            antialias: true,
            alpha: true,
            powerPreference: 'high-performance'
        });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        this.renderer.xr.enabled = true;
        
        const xrOverlay = container.querySelector('.xr-overlay');
        this.renderer.domElement.style.position = 'absolute';
        this.renderer.domElement.style.top = '0';
        this.renderer.domElement.style.left = '0';
        container.insertBefore(this.renderer.domElement, xrOverlay);

        const ambientLight = new THREE.AmbientLight(0xffffff, 0.4);
        this.scene.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
        directionalLight.position.set(5, 10, 5);
        directionalLight.castShadow = true;
        directionalLight.shadow.mapSize.width = 2048;
        directionalLight.shadow.mapSize.height = 2048;
        directionalLight.shadow.camera.near = 0.5;
        directionalLight.shadow.camera.far = 50;
        this.scene.add(directionalLight);

        const pointLight = new THREE.PointLight(0x6366f1, 0.5);
        pointLight.position.set(-5, 5, -5);
        this.scene.add(pointLight);

        const floorGeometry = new THREE.PlaneGeometry(20, 20);
        const floorMaterial = new THREE.MeshStandardMaterial({ 
            color: 0x1e3a5f,
            roughness: 0.8,
            metalness: 0.2
        });
        const floor = new THREE.Mesh(floorGeometry, floorMaterial);
        floor.rotation.x = -Math.PI / 2;
        floor.position.y = -0.01;
        floor.receiveShadow = true;
        this.scene.add(floor);

        this.renderer.domElement.addEventListener('mousedown', (e) => this.onMouseDown(e));
        this.renderer.domElement.addEventListener('mousemove', (e) => this.onMouseMove(e));
        this.renderer.domElement.addEventListener('mouseup', () => this.onMouseUp());
        this.renderer.domElement.addEventListener('mouseleave', () => this.onMouseUp());

        window.addEventListener('resize', () => this.onWindowResize());

        this.animate();
    }

    renderScene() {
        this.objects = [];
        this.targets = [];

        if (this.vrConfig.objects) {
            this.vrConfig.objects.forEach((objData) => {
                const geometry = this.createGeometry(objData.type);
                const material = this.createMaterial(objData);
                const mesh = new THREE.Mesh(geometry, material);
                
                mesh.position.set(...objData.position);
                mesh.rotation.set(...objData.rotation.map(r => THREE.MathUtils.degToRad(r)));
                mesh.scale.set(...objData.scale);
                mesh.castShadow = true;
                mesh.receiveShadow = true;
                mesh.userData = { 
                    id: objData.id,
                    type: objData.type,
                    targetPosition: objData.targetPosition,
                    targetRotation: objData.targetRotation,
                    interactable: objData.interactable !== false,
                    grabbable: objData.grabbable !== false
                };

                this.scene.add(mesh);
                this.objects.push(mesh);
            });
        }

        if (this.vrConfig.targets) {
            this.vrConfig.targets.forEach((targetData) => {
                const geometry = new THREE.BoxGeometry(...targetData.size);
                const material = new THREE.MeshStandardMaterial({
                    color: targetData.color,
                    transparent: true,
                    opacity: 0.5,
                    wireframe: true
                });
                const mesh = new THREE.Mesh(geometry, material);
                mesh.position.set(...targetData.position);
                mesh.userData = {
                    id: targetData.id,
                    objectId: targetData.objectId,
                    sequence: targetData.sequence
                };
                
                if (targetData.visible !== false) {
                    this.scene.add(mesh);
                }
                this.targets.push(mesh);
            });
        }

        if (this.currentMode === 'gesture' && this.vrConfig.targetGesture) {
            this.showGestureIndicator(this.vrConfig.targetGesture);
        }
    }

    createGeometry(type) {
        const size = 0.5;
        switch (type) {
            case 'cube':
                return new THREE.BoxGeometry(size, size, size);
            case 'sphere':
                return new THREE.SphereGeometry(size / 2, 32, 32);
            case 'cylinder':
                return new THREE.CylinderGeometry(size / 2, size / 2, size, 32);
            case 'pyramid':
                return new THREE.ConeGeometry(size / 2, size, 4);
            case 'torus':
                return new THREE.TorusGeometry(size / 3, size / 6, 16, 100);
            default:
                return new THREE.BoxGeometry(size, size, size);
        }
    }

    createMaterial(objData) {
        const color = new THREE.Color(objData.color);
        return new THREE.MeshStandardMaterial({
            color: color,
            roughness: 0.5,
            metalness: 0.3
        });
    }

    showGestureIndicator(gesture) {
        const gestureNames = {
            'pinch': '捏合手势',
            'point': '指向前方',
            'wave': '挥手',
            'fist': '握拳',
            'open_palm': '张开手掌',
            'thumbs_up': '竖大拇指',
            'peace': '剪刀手',
            'ok_sign': 'OK手势'
        };
        const indicator = document.getElementById('gesture-indicator');
        indicator.textContent = `请做出: ${gestureNames[gesture] || gesture}`;
        indicator.style.display = 'block';
    }

    onMouseDown(event) {
        if (this.currentMode === 'gesture') return;

        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.objects);

        if (intersects.length > 0) {
            const obj = intersects[0].object;
            if (obj.userData.interactable) {
                this.selectedObject = obj;
                this.isDragging = true;
                this.interactionData.movementCount++;
                
                obj.material.emissive = new THREE.Color(0x6366f1);
                obj.material.emissiveIntensity = 0.3;
            }
        }
    }

    onMouseMove(event) {
        if (!this.isDragging || !this.selectedObject) return;

        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        const vector = new THREE.Vector3(this.mouse.x, this.mouse.y, 0.5);
        vector.unproject(this.camera);
        const dir = vector.sub(this.camera.position).normalize();
        const distance = (1 - this.camera.position.z) / dir.z;
        const pos = this.camera.position.clone().add(dir.multiplyScalar(distance));
        
        this.selectedObject.position.x = pos.x;
        this.selectedObject.position.y = pos.y;
        
        this.interactionData.objectPositions[this.selectedObject.userData.id] = [
            this.selectedObject.position.x,
            this.selectedObject.position.y,
            this.selectedObject.position.z
        ];
        this.interactionData.objectRotations[this.selectedObject.userData.id] = [
            this.selectedObject.rotation.x,
            this.selectedObject.rotation.y,
            this.selectedObject.rotation.z
        ];
    }

    onMouseUp() {
        if (this.selectedObject) {
            this.selectedObject.material.emissiveIntensity = 0;
            this.checkTargetAlignment(this.selectedObject);
        }
        this.selectedObject = null;
        this.isDragging = false;
    }

    checkTargetAlignment(obj) {
        const target = this.targets.find(t => t.userData.objectId === obj.userData.id);
        if (target) {
            const distance = obj.position.distanceTo(target.position);
            if (distance < 0.5) {
                obj.position.copy(target.position);
                if (!this.interactionData.completionOrder.includes(obj.userData.id)) {
                    this.interactionData.completionOrder.push(obj.userData.id);
                }
            }
        }
    }

    onWindowResize() {
        const container = document.getElementById('canvas-container');
        const width = container.clientWidth;
        const height = container.clientHeight;

        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    }

    animate() {
        this.animationFrameId = requestAnimationFrame(() => this.animate());

        if (this.xrSession) {
            this.updateXR();
        }

        this.objects.forEach(obj => {
            if (obj !== this.selectedObject) {
                obj.rotation.y += 0.005;
            }
        });

        this.renderer.render(this.scene, this.camera);
    }

    updateXR() {
    }

    startTimer() {
        this.seconds = 0;
        this.updateTimerDisplay();
        this.timerInterval = setInterval(() => {
            this.seconds++;
            this.updateTimerDisplay();
        }, 1000);
    }

    resetTimer() {
        if (this.timerInterval) {
            clearInterval(this.timerInterval);
            this.timerInterval = null;
        }
        this.seconds = 0;
        this.updateTimerDisplay();
    }

    updateTimerDisplay() {
        const minutes = Math.floor(this.seconds / 60);
        const seconds = this.seconds % 60;
        document.getElementById('timer').textContent = 
            `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }

    async submit() {
        if (!this.sessionId) {
            this.showResult(false, '请先生成验证码');
            return;
        }

        this.showLoading(true);
        this.interactionData.timeSpent = (Date.now() - this.startTime) / 1000;

        try {
            const response = await fetch('/api/v1/captcha/vr/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    sessionID: this.sessionId,
                    interaction: this.interactionData,
                    gestureData: this.gestureData
                })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.showResult(true, result.data.message || '验证成功！');
            } else {
                this.showResult(false, result.message || '验证失败');
            }
        } catch (error) {
            console.error('Error:', error);
            this.showResult(false, '网络错误，请重试');
        } finally {
            this.showLoading(false);
        }
    }

    showResult(success, message) {
        const banner = document.getElementById('result-banner');
        banner.className = 'result-banner show ' + (success ? 'success' : 'error');
        banner.innerHTML = `
            <i class="fas fa-${success ? 'check-circle' : 'times-circle'}"></i>
            ${message}
        `;
    }

    hideResult() {
        const banner = document.getElementById('result-banner');
        banner.className = 'result-banner';
    }

    showLoading(show) {
        const overlay = document.getElementById('loading-overlay');
        if (show) {
            overlay.classList.add('show');
        } else {
            overlay.classList.remove('show');
        }
    }

    cleanup() {
        if (this.animationFrameId) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }

        if (this.renderer && this.renderer.domElement && this.renderer.domElement.parentNode) {
            this.renderer.domElement.parentNode.removeChild(this.renderer.domElement);
        }

        if (this.xrSession && this.xrSession.state !== 'ended') {
            this.xrSession.end();
        }

        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.objects = [];
        this.targets = [];
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new VRCaptcha();
});
