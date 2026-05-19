class ARCaptcha {
    constructor() {
        this.sessionId = null;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.objects = [];
        this.selectedObject = null;
        this.targetPosition = null;
        this.isDragging = false;
        this.previousMousePosition = { x: 0, y: 0 };
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.timerInterval = null;
        this.seconds = 0;
        this.animationFrameId = null;
        this.currentGesture = null;
        this.placedObjectID = -1;
        this.requiredGesture = '';
        this.targetColor = '';
        this.targetShape = '';
        this.gestureHistory = [];
        this.lastTouchStart = null;
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.checkWebXRSupport();
        this.refresh();
    }

    async checkWebXRSupport() {
        try {
            const response = await fetch('/api/v1/captcha/ar/webxr-support');
            const result = await response.json();
            if (result.code === 0) {
                this.webXRSupport = result.data;
                this.updateWebXRSupportDisplay(result.data);
            }
        } catch (error) {
            console.error('Failed to check WebXR support:', error);
        }
    }

    updateWebXRSupportDisplay(data) {
        const supportEl = document.getElementById('webxr-support');
        if (supportEl) {
            supportEl.innerHTML = `
                <div class="support-item">
                    <i class="fas fa-check-circle ${data.supportsAR ? 'text-success' : 'text-danger'}"></i>
                    <span>AR支持: ${data.supportsAR ? '是' : '否'}</span>
                </div>
                <div class="support-item">
                    <i class="fas fa-check-circle ${data.mobileSupport ? 'text-success' : 'text-danger'}"></i>
                    <span>移动设备: ${data.mobileSupport ? '支持' : '不支持'}</span>
                </div>
            `;
        }
    }

    bindEvents() {
        document.getElementById('refresh-btn')?.addEventListener('click', () => this.refresh());
        document.getElementById('submit-btn')?.addEventListener('click', () => this.submit());
        document.getElementById('difficulty-select')?.addEventListener('change', (e) => {
            this.updateDifficultyDisplay(e.target.value);
        });
        
        this.setupGestureEvents();
    }

    setupGestureEvents() {
        const container = document.getElementById('canvas-container');
        if (!container) return;

        let touchStartX = 0;
        let touchStartY = 0;
        let touchStartTime = 0;
        let lastDistance = 0;

        container.addEventListener('touchstart', (e) => {
            this.lastTouchStart = Date.now();
            if (e.touches.length === 1) {
                touchStartX = e.touches[0].clientX;
                touchStartY = e.touches[0].clientY;
                touchStartTime = Date.now();
            } else if (e.touches.length === 2) {
                lastDistance = this.getDistance(e.touches[0], e.touches[1]);
            }
        }, { passive: true });

        container.addEventListener('touchmove', (e) => {
            if (e.touches.length === 1) {
                const deltaX = e.touches[0].clientX - touchStartX;
                const deltaY = e.touches[0].clientY - touchStartY;
                
                if (Math.abs(deltaX) > 50 || Math.abs(deltaY) > 50) {
                    const swipeDirection = this.getSwipeDirection(deltaX, deltaY);
                    this.currentGesture = swipeDirection;
                    this.recordGesture('swipe');
                }
            } else if (e.touches.length === 2) {
                const currentDistance = this.getDistance(e.touches[0], e.touches[1]);
                if (Math.abs(currentDistance - lastDistance) > 10) {
                    this.currentGesture = currentDistance > lastDistance ? 'pinch-out' : 'pinch-in';
                    this.recordGesture('pinch');
                    lastDistance = currentDistance;
                }
            }
        }, { passive: true });

        container.addEventListener('touchend', (e) => {
            const touchDuration = Date.now() - this.lastTouchStart;
            if (touchDuration < 200 && e.changedTouches.length === 1) {
                this.currentGesture = 'tap';
                this.recordGesture('tap');
                this.onTap(e.changedTouches[0]);
            }
        }, { passive: true });

        container.addEventListener('wheel', (e) => {
            if (Math.abs(e.deltaY) > 5) {
                this.currentGesture = e.deltaY > 0 ? 'scroll-down' : 'scroll-up';
                this.recordGesture('scroll');
            }
        }, { passive: true });
    }

    getDistance(touch1, touch2) {
        const dx = touch2.clientX - touch1.clientX;
        const dy = touch2.clientY - touch1.clientY;
        return Math.sqrt(dx * dx + dy * dy);
    }

    getSwipeDirection(dx, dy) {
        if (Math.abs(dx) > Math.abs(dy)) {
            return dx > 0 ? 'swipe-right' : 'swipe-left';
        }
        return dy > 0 ? 'swipe-down' : 'swipe-up';
    }

    recordGesture(gestureType) {
        this.gestureHistory.push({
            type: gestureType,
            timestamp: Date.now(),
            position: this.selectedObject ? {
                x: this.selectedObject.position.x,
                y: this.selectedObject.position.y,
                z: this.selectedObject.position.z
            } : null
        });
        
        if (this.gestureHistory.length > 10) {
            this.gestureHistory.shift();
        }
    }

    updateDifficultyDisplay(difficulty) {
        const displayNames = {
            'easy': '简单',
            'medium': '中等',
            'hard': '困难',
            'expert': '专家'
        };
        document.getElementById('difficulty-display')?.textContent = `难度: ${displayNames[difficulty]}`;
    }

    async refresh() {
        this.showLoading(true);
        this.hideResult();
        this.resetTimer();
        this.cleanup();
        this.currentGesture = null;
        this.placedObjectID = -1;
        this.gestureHistory = [];

        const difficulty = document.getElementById('difficulty-select')?.value || 'medium';

        try {
            const response = await fetch('/api/v1/captcha/ar/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ difficulty })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.sessionId = result.data.sessionID;
                this.sceneData = result.data.scene;
                this.targetPosition = result.data.scene.targetPosition;
                this.requiredGesture = result.data.scene.requiredGesture;
                this.targetColor = result.data.scene.targetColor;
                this.targetShape = result.data.scene.targetShape;
                
                this.updateTargetDisplay();
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

    updateTargetDisplay() {
        const targetInfo = document.getElementById('target-info');
        if (targetInfo) {
            const gestureNames = {
                'tap': '点击',
                'swipe': '滑动',
                'pinch': '捏合',
                'rotate': '旋转'
            };
            
            targetInfo.innerHTML = `
                <div class="target-item">
                    <i class="fas fa-target"></i>
                    <span>目标颜色: <span style="color: ${this.targetColor}; font-weight: bold;">${this.targetColor}</span></span>
                </div>
                <div class="target-item">
                    <i class="fas fa-hand-pointer"></i>
                    <span>所需手势: <strong>${gestureNames[this.requiredGesture] || this.requiredGesture}</strong></span>
                </div>
            `;
        }
    }

    initThreeJS() {
        const container = document.getElementById('canvas-container');
        if (!container) return;

        const width = container.clientWidth;
        const height = container.clientHeight;

        this.scene = new THREE.Scene();
        
        const bgColor = this.sceneData?.environment?.background === 'solid' ? '#e0e0e0' : '#f5f5f5';
        this.scene.background = new THREE.Color(bgColor);

        if (this.sceneData?.environment?.floorPlane) {
            this.addFloorPlane();
        }

        this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
        this.camera.position.z = 8;

        this.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        container.appendChild(this.renderer.domElement);

        this.setupLighting();

        this.renderer.domElement.addEventListener('mousedown', (e) => this.onMouseDown(e));
        this.renderer.domElement.addEventListener('mousemove', (e) => this.onMouseMove(e));
        this.renderer.domElement.addEventListener('mouseup', () => this.onMouseUp());
        this.renderer.domElement.addEventListener('mouseleave', () => this.onMouseUp());

        window.addEventListener('resize', () => this.onWindowResize());

        this.animate();
    }

    addFloorPlane() {
        const geometry = new THREE.PlaneGeometry(10, 10);
        const material = new THREE.MeshStandardMaterial({ 
            color: 0xcccccc,
            roughness: 0.8,
            metalness: 0.2
        });
        const plane = new THREE.Mesh(geometry, material);
        plane.rotation.x = -Math.PI / 2;
        plane.position.y = -2;
        plane.receiveShadow = true;
        this.scene.add(plane);

        const gridHelper = new THREE.GridHelper(10, 10, 0x888888, 0xcccccc);
        gridHelper.position.y = -1.99;
        this.scene.add(gridHelper);
    }

    setupLighting() {
        const lighting = this.sceneData?.environment?.lighting || 'day';
        let intensity = 1.0;
        
        switch (lighting) {
            case 'night':
                intensity = 0.5;
                this.scene.background = new THREE.Color(0x1a1a2e);
                break;
            case 'sunset':
                const ambientLight = new THREE.AmbientLight(0xffaa66, intensity * 0.5);
                this.scene.add(ambientLight);
                const directionalLight = new THREE.DirectionalLight(0xffcc88, intensity);
                directionalLight.position.set(5, 5, 5);
                this.scene.add(directionalLight);
                return;
            case 'studio':
                intensity = 1.2;
                break;
        }

        const ambientLight = new THREE.AmbientLight(0xffffff, intensity * 0.5);
        this.scene.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(0xffffff, intensity);
        directionalLight.position.set(5, 5, 5);
        directionalLight.castShadow = true;
        this.scene.add(directionalLight);

        const pointLight = new THREE.PointLight(0xffffff, intensity * 0.3);
        pointLight.position.set(-5, -5, 5);
        this.scene.add(pointLight);
    }

    renderScene() {
        if (!this.sceneData?.objects) return;

        this.sceneData.objects.forEach((objData, index) => {
            const geometry = this.createGeometry(objData.type);
            const material = this.createMaterial(objData);

            const mesh = new THREE.Mesh(geometry, material);
            mesh.position.set(objData.position.x, objData.position.y, objData.position.z);
            mesh.rotation.x = THREE.MathUtils.degToRad(objData.rotation.x);
            mesh.rotation.y = THREE.MathUtils.degToRad(objData.rotation.y);
            mesh.rotation.z = THREE.MathUtils.degToRad(objData.rotation.z);
            mesh.scale.set(objData.scale, objData.scale, objData.scale);
            mesh.castShadow = true;
            mesh.receiveShadow = true;
            mesh.userData = {
                id: objData.id,
                index,
                isTarget: objData.isTarget,
                animation: objData.animation
            };

            if (objData.isTarget) {
                this.addTargetIndicator(mesh);
            }

            this.scene.add(mesh);
            this.objects.push(mesh);
        });

        this.renderTargetPosition();
    }

    addTargetIndicator(mesh) {
        const geometry = new THREE.SphereGeometry(0.1, 16, 16);
        const material = new THREE.MeshBasicMaterial({ color: 0x00ff00 });
        const indicator = new THREE.Mesh(geometry, material);
        indicator.position.y = mesh.geometry.parameters.radius || 0.6;
        mesh.add(indicator);
    }

    renderTargetPosition() {
        if (!this.targetPosition) return;

        const geometry = new THREE.RingGeometry(0.3, 0.4, 32);
        const material = new THREE.MeshBasicMaterial({ 
            color: 0xff9800, 
            transparent: true, 
            opacity: 0.6,
            side: THREE.DoubleSide 
        });
        const ring = new THREE.Mesh(geometry, material);
        ring.rotation.x = -Math.PI / 2;
        ring.position.set(this.targetPosition.x, -1.9, this.targetPosition.z);
        this.scene.add(ring);

        const arrowHelper = new THREE.ArrowHelper(
            new THREE.Vector3(0, 1, 0),
            new THREE.Vector3(this.targetPosition.x, -2, this.targetPosition.z),
            0.5,
            0xff9800,
            0.2,
            0.1
        );
        this.scene.add(arrowHelper);
    }

    createGeometry(type) {
        switch (type) {
            case 'cube':
            case 'box':
                return new THREE.BoxGeometry(1, 1, 1);
            case 'sphere':
                return new THREE.SphereGeometry(0.6, 32, 32);
            case 'cylinder':
                return new THREE.CylinderGeometry(0.5, 0.5, 1, 32);
            case 'cone':
                return new THREE.ConeGeometry(0.5, 1, 32);
            case 'torus':
                return new THREE.TorusGeometry(0.4, 0.2, 16, 100);
            case 'pyramid':
                return new THREE.ConeGeometry(0.5, 1, 4);
            case 'diamond':
                return new THREE.OctahedronGeometry(0.6);
            case 'ring':
                return new THREE.TorusGeometry(0.4, 0.1, 16, 100);
            case 'star':
                return this.createStarGeometry();
            case 'heart':
                return this.createHeartGeometry();
            default:
                return new THREE.BoxGeometry(1, 1, 1);
        }
    }

    createStarGeometry() {
        const shape = new THREE.Shape();
        const spikes = 5;
        const outerRadius = 0.5;
        const innerRadius = 0.25;
        
        for (let i = 0; i < spikes * 2; i++) {
            const radius = i % 2 === 0 ? outerRadius : innerRadius;
            const angle = (Math.PI * i) / spikes - Math.PI / 2;
            const x = Math.cos(angle) * radius;
            const y = Math.sin(angle) * radius;
            
            if (i === 0) {
                shape.moveTo(x, y);
            } else {
                shape.lineTo(x, y);
            }
        }
        
        shape.closePath();
        const extrudeSettings = { depth: 0.2, bevelEnabled: true, bevelThickness: 0.05, bevelSize: 0.05 };
        return new THREE.ExtrudeGeometry(shape, extrudeSettings);
    }

    createHeartGeometry() {
        const x = 0, y = 0;
        const shape = new THREE.Shape();
        
        shape.moveTo(x + 5, y + 5);
        shape.bezierCurveTo(x + 5, y + 5, x + 4, y, x, y);
        shape.bezierCurveTo(x - 4, y, x - 5, y + 5, x - 5, y + 5);
        shape.bezierCurveTo(x - 5, y + 5, x - 3, y + 9, x, y + 11);
        shape.bezierCurveTo(x + 3, y + 9, x + 5, y + 5, x + 5, y + 5);
        
        const extrudeSettings = { depth: 0.3, bevelEnabled: true, bevelThickness: 0.1, bevelSize: 0.1 };
        const geometry = new THREE.ExtrudeGeometry(shape, extrudeSettings);
        geometry.scale(0.1, 0.1, 0.1);
        return geometry;
    }

    createMaterial(objData) {
        const materialParams = {
            color: objData.color,
            shininess: 100,
            transparent: objData.opacity !== undefined && objData.opacity < 1,
            opacity: objData.opacity !== undefined ? objData.opacity : 1,
            emissive: new THREE.Color(0x000000),
            emissiveIntensity: 0
        };

        if (objData.isTarget) {
            materialParams.emissive = new THREE.Color(objData.color);
            materialParams.emissiveIntensity = 0.2;
        }

        return new THREE.MeshPhongMaterial(materialParams);
    }

    onMouseDown(event) {
        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.objects);

        if (intersects.length > 0) {
            this.selectedObject = intersects[0].object;
            this.placedObjectID = this.selectedObject.userData.id;
            this.highlightObject(this.selectedObject, true);
            this.isDragging = true;
            this.previousMousePosition = { x: event.clientX, y: event.clientY };
            this.recordGesture('tap');
        }
    }

    onTap(touch) {
        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((touch.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((touch.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.objects);

        if (intersects.length > 0) {
            this.selectedObject = intersects[0].object;
            this.placedObjectID = this.selectedObject.userData.id;
            this.highlightObject(this.selectedObject, true);
        }
    }

    onMouseMove(event) {
        if (!this.isDragging || !this.selectedObject) return;

        const deltaX = event.clientX - this.previousMousePosition.x;
        const deltaY = event.clientY - this.previousMousePosition.y;

        this.selectedObject.rotation.y += deltaX * 0.01;
        this.selectedObject.rotation.x += deltaY * 0.01;

        this.previousMousePosition = { x: event.clientX, y: event.clientY };
    }

    onMouseUp() {
        this.isDragging = false;
    }

    highlightObject(obj, highlight) {
        if (!obj || !obj.material) return;
        
        if (highlight) {
            obj.material.emissive = new THREE.Color(0x666666);
            obj.material.emissiveIntensity = 0.5;
        } else {
            obj.material.emissive = new THREE.Color(0x000000);
            obj.material.emissiveIntensity = 0;
        }
    }

    onWindowResize() {
        if (!this.renderer || !this.camera) return;
        
        const container = document.getElementById('canvas-container');
        const width = container.clientWidth;
        const height = container.clientHeight;

        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    }

    animate() {
        this.animationFrameId = requestAnimationFrame(() => this.animate());
        
        this.autoAnimateObjects();
        this.renderer.render(this.scene, this.camera);
    }

    autoAnimateObjects() {
        this.objects.forEach(obj => {
            const animation = obj.userData.animation;
            if (animation && animation.enabled) {
                const speed = animation.speed || 0.005;
                switch (animation.type) {
                    case 'rotate':
                        obj.rotation.y += speed;
                        obj.rotation.z += speed * 0.5;
                        break;
                    case 'float':
                        obj.position.y = Math.sin(Date.now() * 0.001) * 0.1 + obj.userData.originalY || obj.position.y;
                        break;
                    case 'pulse':
                        const scale = 1 + Math.sin(Date.now() * 0.002) * 0.1;
                        obj.scale.set(scale, scale, scale);
                        break;
                    case 'bounce':
                        obj.position.y = Math.abs(Math.sin(Date.now() * 0.002)) * 0.2;
                        break;
                }
            }
        });
    }

    cleanup() {
        if (this.animationFrameId) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }
        
        if (this.objects) {
            this.objects.forEach(obj => {
                obj.geometry.dispose();
                obj.material.dispose();
            });
            this.objects = [];
        }
        
        if (this.renderer) {
            const container = document.getElementById('canvas-container');
            if (container && this.renderer.domElement) {
                container.removeChild(this.renderer.domElement);
            }
            this.renderer.dispose();
        }
        
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.selectedObject = null;
    }

    getCurrentSceneState() {
        return {
            sessionID: this.sessionId,
            targetShape: this.targetShape,
            targetColor: this.targetColor,
            objects: this.objects.map(obj => ({
                id: obj.userData.id,
                type: this.sceneData.objects[obj.userData.index]?.type,
                color: this.sceneData.objects[obj.userData.index]?.color,
                position: {
                    x: obj.position.x,
                    y: obj.position.y,
                    z: obj.position.z
                },
                rotation: {
                    x: THREE.MathUtils.radToDeg(obj.rotation.x),
                    y: THREE.MathUtils.radToDeg(obj.rotation.y),
                    z: THREE.MathUtils.radToDeg(obj.rotation.z)
                },
                scale: obj.scale.x,
                isTarget: obj.userData.isTarget
            })),
            targetPosition: this.targetPosition,
            gridSize: this.sceneData?.gridSize,
            difficulty: this.sceneData?.difficulty,
            requiredGesture: this.requiredGesture,
            environment: this.sceneData?.environment
        };
    }

    async submit() {
        if (!this.sessionId) {
            this.showResult(false, '请先生成验证码');
            return;
        }

        this.showLoading(true);

        try {
            const response = await fetch('/api/v1/captcha/ar/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    sessionID: this.sessionId,
                    scene: this.getCurrentSceneState(),
                    userGesture: this.requiredGesture,
                    placedObjectID: this.placedObjectID,
                    finalPosition: this.selectedObject ? {
                        x: this.selectedObject.position.x,
                        y: this.selectedObject.position.y,
                        z: this.selectedObject.position.z
                    } : this.targetPosition
                })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.showResult(result.data.success, result.data.message || (result.data.success ? '验证成功！' : '验证失败'));
                if (result.data.success) {
                    this.stopTimer();
                }
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

    startTimer() {
        this.seconds = 0;
        this.updateTimerDisplay();
        this.timerInterval = setInterval(() => {
            this.seconds++;
            this.updateTimerDisplay();
            
            const timeLimit = this.sceneData?.timeLimit || 45;
            if (this.seconds >= timeLimit) {
                this.showResult(false, '时间已用完');
                this.stopTimer();
            }
        }, 1000);
    }

    stopTimer() {
        if (this.timerInterval) {
            clearInterval(this.timerInterval);
            this.timerInterval = null;
        }
    }

    resetTimer() {
        this.stopTimer();
        this.seconds = 0;
        this.updateTimerDisplay();
    }

    updateTimerDisplay() {
        const minutes = Math.floor(this.seconds / 60).toString().padStart(2, '0');
        const seconds = (this.seconds % 60).toString().padStart(2, '0');
        document.getElementById('timer')?.textContent = `${minutes}:${seconds}`;
    }

    showResult(success, message) {
        const banner = document.getElementById('result-banner');
        banner.className = `result-banner show ${success ? 'success' : 'error'}`;
        banner.innerHTML = `
            <i class="fas fa-${success ? 'check-circle' : 'times-circle'} me-2"></i>
            <strong>${message}</strong>
        `;
    }

    hideResult() {
        const banner = document.getElementById('result-banner');
        banner?.classList.remove('show');
    }

    showLoading(show) {
        const overlay = document.getElementById('loading-overlay');
        if (overlay) {
            overlay.classList.toggle('show', show);
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new ARCaptcha();
});
