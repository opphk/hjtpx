class ARCaptcha {
    constructor() {
        this.sessionId = null;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.objects = [];
        this.gesturePath = [];
        this.userGesture = [];
        this.isDrawing = false;
        this.startTime = 0;
        this.animationFrameId = null;
        this.timerInterval = null;
        this.seconds = 0;
        this.webXRSupported = false;
        this.devicePixelRatio = Math.min(window.devicePixelRatio, 2);
        
        this.init();
    }

    init() {
        this.checkWebXRSupport();
        this.bindEvents();
        this.refresh();
    }

    async checkWebXRSupport() {
        if ('xr' in navigator) {
            try {
                this.webXRSupported = await navigator.xr.isSessionSupported('immersive-ar');
            } catch (e) {
                this.webXRSupported = false;
            }
        }
        
        const statusElement = document.getElementById('ar-status');
        if (statusElement) {
            statusElement.textContent = this.webXRSupported ? 'WebXR AR 支持' : 'WebXR 不可用，使用3D渲染';
        }
    }

    bindEvents() {
        const refreshBtn = document.getElementById('refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refresh());
        }

        const submitBtn = document.getElementById('submit-btn');
        if (submitBtn) {
            submitBtn.addEventListener('click', () => this.submit());
        }

        const canvas = document.getElementById('ar-canvas');
        if (canvas) {
            canvas.addEventListener('mousedown', (e) => this.onMouseDown(e));
            canvas.addEventListener('mousemove', (e) => this.onMouseMove(e));
            canvas.addEventListener('mouseup', () => this.onMouseUp());
            canvas.addEventListener('mouseleave', () => this.onMouseUp());
            
            canvas.addEventListener('touchstart', (e) => this.onTouchStart(e));
            canvas.addEventListener('touchmove', (e) => this.onTouchMove(e));
            canvas.addEventListener('touchend', () => this.onTouchEnd());
        }

        const difficultySelect = document.getElementById('difficulty-select');
        if (difficultySelect) {
            difficultySelect.addEventListener('change', (e) => {
                this.updateDifficultyDisplay(e.target.value);
            });
        }
    }

    updateDifficultyDisplay(difficulty) {
        const displayNames = {
            'easy': '简单',
            'medium': '中等',
            'hard': '困难',
            'expert': '专家'
        };
        const displayElement = document.getElementById('difficulty-display');
        if (displayElement) {
            displayElement.textContent = `难度: ${displayNames[difficulty] || difficulty}`;
        }
    }

    async refresh() {
        this.showLoading(true);
        this.hideResult();
        this.resetTimer();
        this.cleanup();

        const difficulty = document.getElementById('difficulty-select')?.value || 'medium';

        try {
            const response = await fetch('/api/v1/captcha/ar/generate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ difficulty })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.sessionId = result.data.sessionID;
                this.scene = result.data.scene;
                this.initThreeJS();
                this.renderScene();
                this.startTimer();
                this.updateInstructions();
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
        const container = document.getElementById('ar-canvas-container');
        if (!container) return;

        const width = container.clientWidth;
        const height = container.clientHeight;

        this.scene3D = new THREE.Scene();
        
        const bgColor = this.scene.backgroundColor || '#f5f5f5';
        this.scene3D.background = new THREE.Color(bgColor);

        const cameraConfig = this.scene.cameraConfig || {};
        const fov = cameraConfig.fov || 60;
        this.camera = new THREE.PerspectiveCamera(fov, width / height, 
            cameraConfig.nearClip || 0.1, 
            cameraConfig.farClip || 100);
        
        this.camera.position.set(
            cameraConfig.positionX || 0,
            cameraConfig.positionY || 0,
            cameraConfig.positionZ || 5
        );

        this.renderer = new THREE.WebGLRenderer({ 
            antialias: true,
            alpha: true,
            powerPreference: 'high-performance'
        });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(this.devicePixelRatio);
        
        container.appendChild(this.renderer.domElement);

        this.setupLighting();
        this.setupControls();

        window.addEventListener('resize', () => this.onWindowResize());

        this.animate();
    }

    setupLighting() {
        const lightingConfig = this.scene.lightingConfig || {};
        
        const ambientLight = new THREE.AmbientLight(
            lightingConfig.ambientColor || '#444444', 
            lightingConfig.ambientIntensity || 0.5
        );
        this.scene3D.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(
            0xffffff, 
            lightingConfig.directionalIntensity || 0.8
        );
        directionalLight.position.set(5, 5, 5);
        
        if (lightingConfig.shadowEnabled) {
            directionalLight.castShadow = true;
            directionalLight.shadow.mapSize.width = 2048;
            directionalLight.shadow.mapSize.height = 2048;
        }
        this.scene3D.add(directionalLight);

        const pointLight = new THREE.PointLight(
            0xffffff, 
            lightingConfig.pointIntensity || 0.3
        );
        pointLight.position.set(-5, -5, 5);
        this.scene3D.add(pointLight);
    }

    setupControls() {
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.isDragging = false;
        this.selectedObject = null;
        this.previousMousePosition = { x: 0, y: 0 };
    }

    renderScene() {
        this.objects = [];
        
        if (!this.scene.objects) return;

        const quality = this.scene.difficulty === 'expert' ? 'low' : 
                        this.scene.difficulty === 'hard' ? 'medium' : 'high';

        this.scene.objects.forEach((objData, index) => {
            if (objData.hidden) return;

            let geometry = this.createGeometry(objData.type, quality);
            let material = this.createMaterial(objData);

            const mesh = new THREE.Mesh(geometry, material);
            mesh.position.set(objData.positionX, objData.positionY, objData.positionZ);
            mesh.rotation.x = THREE.MathUtils.degToRad(objData.rotationX);
            mesh.rotation.y = THREE.MathUtils.degToRad(objData.rotationY);
            mesh.rotation.z = THREE.MathUtils.degToRad(objData.rotationZ);
            mesh.scale.setScalar(objData.scale || 1);
            
            mesh.userData = { 
                id: objData.id,
                index,
                type: objData.type,
                isTarget: objData.isTarget,
                animation: objData.animation,
                originalPosition: new THREE.Vector3(objData.positionX, objData.positionY, objData.positionZ)
            };

            this.scene3D.add(mesh);
            this.objects.push(mesh);
        });

        this.renderAnnotations();
        this.renderGesturePath();
    }

    createGeometry(type, quality) {
        const detail = this.getGeometryDetail(quality);
        
        switch (type) {
            case 'cube':
                return new THREE.BoxGeometry(1, 1, 1);
            case 'sphere':
                return new THREE.SphereGeometry(0.6, detail.sphere, detail.sphere);
            case 'cylinder':
                return new THREE.CylinderGeometry(0.5, 0.5, 1, detail.cylinder);
            case 'cone':
                return new THREE.ConeGeometry(0.5, 1, detail.cylinder);
            case 'pyramid':
                return new THREE.ConeGeometry(0.6, 1, 4);
            case 'torus':
                return new THREE.TorusGeometry(0.4, 0.2, detail.torus.radial, detail.torus.tubular);
            case 'star':
                return this.createStarGeometry(0.6, 5);
            case 'heart':
                return this.createHeartGeometry(0.6);
            case 'diamond':
                return new THREE.OctahedronGeometry(0.6);
            case 'ring':
                return new THREE.TorusGeometry(0.5, 0.15, 16, 32);
            default:
                return new THREE.BoxGeometry(1, 1, 1);
        }
    }

    createStarGeometry(radius, points) {
        const shape = new THREE.Shape();
        const outerRadius = radius;
        const innerRadius = radius * 0.5;
        
        for (let i = 0; i < points * 2; i++) {
            const r = i % 2 === 0 ? outerRadius : innerRadius;
            const angle = (i / (points * 2)) * Math.PI * 2 - Math.PI / 2;
            const x = Math.cos(angle) * r;
            const y = Math.sin(angle) * r;
            
            if (i === 0) {
                shape.moveTo(x, y);
            } else {
                shape.lineTo(x, y);
            }
        }
        shape.closePath();
        
        return new THREE.ExtrudeGeometry(shape, {
            depth: 0.2,
            bevelEnabled: true,
            bevelThickness: 0.05,
            bevelSize: 0.05,
            bevelSegments: 2
        });
    }

    createHeartGeometry(size) {
        const shape = new THREE.Shape();
        const x = 0, y = 0;
        
        shape.moveTo(x, y + size * 0.5);
        shape.bezierCurveTo(x, y + size * 0.8, x - size * 0.3, y + size * 0.7, x - size * 0.5, y + size * 0.5);
        shape.bezierCurveTo(x - size * 0.8, y + size * 0.2, x - size * 0.8, y - size * 0.3, x - size * 0.5, y - size * 0.5);
        shape.bezierCurveTo(x - size * 0.2, y - size * 0.8, x, y - size * 0.6, x, y - size * 0.4);
        shape.bezierCurveTo(x, y - size * 0.6, x + size * 0.2, y - size * 0.8, x + size * 0.5, y - size * 0.5);
        shape.bezierCurveTo(x + size * 0.8, y - size * 0.3, x + size * 0.8, y + size * 0.2, x + size * 0.5, y + size * 0.5);
        shape.bezierCurveTo(x + size * 0.3, y + size * 0.7, x, y + size * 0.8, x, y + size * 0.5);
        
        return new THREE.ExtrudeGeometry(shape, {
            depth: 0.2,
            bevelEnabled: true,
            bevelThickness: 0.05,
            bevelSize: 0.05,
            bevelSegments: 2
        });
    }

    createMaterial(objData) {
        const materialParams = {
            color: objData.color || '#3498db',
            shininess: 100,
            transparent: objData.opacity !== undefined && objData.opacity < 1,
            opacity: objData.opacity !== undefined ? objData.opacity : 1,
            emissive: new THREE.Color(objData.emissiveColor || '#000000'),
            emissiveIntensity: objData.emissiveColor ? 0.3 : 0
        };

        return new THREE.MeshPhongMaterial(materialParams);
    }

    getGeometryDetail(quality) {
        switch (quality) {
            case 'low':
                return { sphere: 16, cylinder: 16, torus: { radial: 8, tubular: 32 } };
            case 'medium':
                return { sphere: 24, cylinder: 24, torus: { radial: 12, tubular: 64 } };
            case 'high':
                return { sphere: 32, cylinder: 32, torus: { radial: 16, tubular: 100 } };
            default:
                return { sphere: 24, cylinder: 24, torus: { radial: 12, tubular: 64 } };
        }
    }

    renderAnnotations() {
        if (!this.scene.annotations) return;

        this.scene.annotations.forEach(annotation => {
            if (!annotation.visible) return;

            if (annotation.type === 'text') {
                const canvas = document.createElement('canvas');
                const ctx = canvas.getContext('2d');
                canvas.width = 256;
                canvas.height = 64;
                
                ctx.fillStyle = annotation.color || '#ffffff';
                ctx.font = '24px Arial';
                ctx.textAlign = 'center';
                ctx.fillText(annotation.text, 128, 40);
                
                const texture = new THREE.CanvasTexture(canvas);
                const spriteMaterial = new THREE.SpriteMaterial({ map: texture });
                const sprite = new THREE.Sprite(spriteMaterial);
                sprite.position.set(annotation.positionX * 10 - 5, annotation.positionY * 10 - 5, 0);
                sprite.scale.set(2, 0.5, 1);
                
                this.scene3D.add(sprite);
            }
        });
    }

    renderGesturePath() {
        if (!this.scene.gesturePath || this.scene.gesturePath.length === 0) return;

        const points = this.scene.gesturePath.map(p => 
            new THREE.Vector3(p.x * 10 - 5, p.y * 10 - 5, 0)
        );

        const lineGeometry = new THREE.BufferGeometry().setFromPoints(points);
        const lineMaterial = new THREE.LineDashedMaterial({
            color: 0x3498db,
            dashSize: 0.3,
            gapSize: 0.1,
            linewidth: 2
        });

        const line = new THREE.Line(lineGeometry, lineMaterial);
        line.computeLineDistances();
        this.scene3D.add(line);

        this.gesturePath = points;
    }

    onMouseDown(event) {
        if (!this.renderer) return;

        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.objects);

        if (intersects.length > 0) {
            this.selectedObject = intersects[0].object;
            this.highlightObject(this.selectedObject, true);
            this.isDragging = true;
            this.previousMousePosition = { x: event.clientX, y: event.clientY };
        } else {
            this.startDrawing(event);
        }
    }

    onMouseMove(event) {
        if (!this.isDragging || !this.selectedObject) return;

        const deltaX = event.clientX - this.previousMousePosition.x;
        const deltaY = event.clientY - this.previousMousePosition.y;

        this.selectedObject.position.x += deltaX * 0.01;
        this.selectedObject.position.y -= deltaY * 0.01;

        this.previousMousePosition = { x: event.clientX, y: event.clientY };
    }

    onMouseUp() {
        this.isDragging = false;
        if (this.selectedObject) {
            this.highlightObject(this.selectedObject, false);
            this.selectedObject = null;
        }
        this.stopDrawing();
    }

    onTouchStart(event) {
        if (event.touches.length === 1) {
            const touch = event.touches[0];
            this.onMouseDown({ clientX: touch.clientX, clientY: touch.clientY });
        }
    }

    onTouchMove(event) {
        if (event.touches.length === 1) {
            event.preventDefault();
            const touch = event.touches[0];
            this.onMouseMove({ clientX: touch.clientX, clientY: touch.clientY });
            
            if (this.isDrawing) {
                this.recordGesturePoint(touch.clientX, touch.clientY);
            }
        }
    }

    onTouchEnd() {
        this.onMouseUp();
    }

    startDrawing(event) {
        this.isDrawing = true;
        this.userGesture = [];
        this.startTime = Date.now();
        this.recordGesturePoint(event.clientX, event.clientY);
    }

    recordGesturePoint(clientX, clientY) {
        const canvas = document.getElementById('ar-canvas');
        if (!canvas) return;

        const rect = canvas.getBoundingClientRect();
        const x = (clientX - rect.left) / rect.width;
        const y = (clientY - rect.top) / rect.height;

        this.userGesture.push({
            x: x,
            y: y,
            timestamp: Date.now() - this.startTime,
            pressure: 0.5
        });
    }

    stopDrawing() {
        this.isDrawing = false;
    }

    highlightObject(object, highlight) {
        if (highlight) {
            object.material.emissive = new THREE.Color(0x333333);
        } else {
            object.material.emissive = new THREE.Color(object.userData.isTarget ? '#ff0000' : '#000000');
        }
    }

    onWindowResize() {
        if (!this.renderer || !this.camera) return;
        
        const container = document.getElementById('ar-canvas-container');
        const width = container.clientWidth;
        const height = container.clientHeight;

        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(width, height);
    }

    animate() {
        this.animationFrameId = requestAnimationFrame(() => this.animate());
        
        if (this.scene && this.scene.cameraConfig && this.scene.cameraConfig.autoRotate) {
            const rotationSpeed = this.scene.cameraConfig.rotationSpeed || 0.01;
            this.camera.position.x = Math.sin(Date.now() * rotationSpeed) * 5;
            this.camera.position.z = Math.cos(Date.now() * rotationSpeed) * 5;
            this.camera.lookAt(0, 0, 0);
        }

        this.objects.forEach(obj => {
            if (obj.userData.animation === 'rotate') {
                obj.rotation.y += 0.01;
            } else if (obj.userData.animation === 'pulse') {
                const scale = 1 + Math.sin(Date.now() * 0.005) * 0.1;
                obj.scale.setScalar(scale);
            } else if (obj.userData.animation === 'bounce') {
                obj.position.y = obj.userData.originalPosition.y + Math.abs(Math.sin(Date.now() * 0.003)) * 0.5;
            } else if (obj.userData.animation === 'float') {
                obj.position.y = obj.userData.originalPosition.y + Math.sin(Date.now() * 0.002) * 0.2;
            }
        });
        
        this.renderer.render(this.scene3D, this.camera);
    }

    updateInstructions() {
        const instructionsElement = document.getElementById('instructions');
        if (instructionsElement && this.scene) {
            const sceneType = this.scene.sceneType || 'object_placement';
            const instructions = this.getInstructionsBySceneType(sceneType);
            instructionsElement.textContent = instructions;
        }
    }

    getInstructionsBySceneType(sceneType) {
        const instructions = {
            'object_placement': '将物体拖动到指定位置',
            'gesture_recognition': '按照提示完成手势动作',
            'spatial_puzzle': '完成空间拼图',
            'object_tracking': '追踪目标物体',
            'depth_estimation': '判断物体深度'
        };
        return instructions[sceneType] || '按照提示完成验证';
    }

    getCurrentGestureState() {
        return {
            type: this.scene?.gestureType || 'unknown',
            points: this.userGesture.map(p => ({
                x: p.x,
                y: p.y,
                timestamp: p.timestamp,
                pressure: p.pressure,
                gesturePhase: p.timestamp < 200 ? 'start' : 
                              p.timestamp > this.userGesture[this.userGesture.length - 1].timestamp - 200 ? 'end' : 'middle'
            })),
            duration: this.userGesture.length > 0 ? 
                      this.userGesture[this.userGesture.length - 1].timestamp : 0,
            gestureType: this.scene?.gestureType || 'unknown'
        };
    }

    getSelectedObjectInfo() {
        if (!this.selectedObject) return null;
        
        return {
            id: this.selectedObject.userData.id,
            positionX: this.selectedObject.position.x,
            positionY: this.selectedObject.position.y,
            positionZ: this.selectedObject.position.z
        };
    }

    async submit() {
        if (!this.sessionId) {
            this.showResult(false, '请先生成验证码');
            return;
        }

        this.showLoading(true);

        try {
            const userGesture = this.getCurrentGestureState();
            const objectInfo = this.getSelectedObjectInfo();

            const response = await fetch('/api/v1/captcha/ar/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    sessionID: this.sessionId,
                    userGesture: userGesture,
                    objectID: objectInfo?.id || 0,
                    positionX: objectInfo?.positionX || 0,
                    positionY: objectInfo?.positionY || 0,
                    positionZ: objectInfo?.positionZ || 0
                })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.showResult(result.data.success, result.data.message || 
                    (result.data.success ? '验证成功！' : '验证失败'));
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
        
        const timeLimit = this.scene?.timeLimit || 30;
        
        this.timerInterval = setInterval(() => {
            this.seconds++;
            this.updateTimerDisplay();
            
            if (this.seconds >= timeLimit) {
                this.showResult(false, '时间到！验证失败');
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
        
        const timerElement = document.getElementById('timer');
        if (timerElement) {
            timerElement.textContent = `${minutes}:${seconds}`;
        }
    }

    showResult(success, message) {
        const banner = document.getElementById('result-banner');
        if (banner) {
            banner.className = `result-banner show ${success ? 'success' : 'error'}`;
            banner.innerHTML = `
                <i class="fas fa-${success ? 'check-circle' : 'times-circle'} me-2"></i>
                <strong>${message}</strong>
            `;
        }
    }

    hideResult() {
        const banner = document.getElementById('result-banner');
        if (banner) {
            banner.classList.remove('show');
        }
    }

    showLoading(show) {
        const overlay = document.getElementById('loading-overlay');
        if (overlay) {
            if (show) {
                overlay.classList.add('show');
            } else {
                overlay.classList.remove('show');
            }
        }
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
        
        if (this.scene3D) {
            while (this.scene3D.children.length > 0) { 
                const child = this.scene3D.children[0];
                if (child.geometry) child.geometry.dispose();
                if (child.material) {
                    if (Array.isArray(child.material)) {
                        child.material.forEach(m => m.dispose());
                    } else {
                        child.material.dispose();
                    }
                }
                this.scene3D.remove(child);
            }
        }
        
        if (this.renderer) {
            const container = document.getElementById('ar-canvas-container');
            if (container && this.renderer.domElement) {
                container.removeChild(this.renderer.domElement);
            }
            this.renderer.dispose();
        }
        
        this.scene3D = null;
        this.camera = null;
        this.renderer = null;
        this.selectedObject = null;
        this.userGesture = [];
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = ARCaptcha;
}
