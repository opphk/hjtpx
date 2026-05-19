(function() {
    'use strict';

    const ARCaptcha = {
        scene: null,
        camera: null,
        renderer: null,
        objects: [],
        targetZone: null,
        sessionID: null,
        sceneType: null,
        difficulty: 2,
        isVerified: false,
        startTime: null,
        moveCount: 0,
        config: {
            width: 640,
            height: 480,
            arButton: null,
            canvas: null,
            videoElement: null,
            onSuccess: null,
            onError: null,
            onProgress: null
        },

        async init(options = {}) {
            Object.assign(this.config, options);
            this.setupCanvas();
            this.setupEventListeners();
            this.checkWebXRSupport();
        },

        setupCanvas() {
            if (this.config.canvas) {
                this.canvas = this.config.canvas;
            } else {
                this.canvas = document.createElement('canvas');
                this.canvas.id = 'ar-captcha-canvas';
                this.canvas.style.cssText = `
                    width: ${this.config.width}px;
                    height: ${this.config.height}px;
                    border: 2px solid #007bff;
                    border-radius: 8px;
                    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                `;
            }

            this.setupRenderer();
            this.setupCamera();
        },

        setupRenderer() {
            if (typeof THREE === 'undefined') {
                console.error('Three.js is required for AR Captcha');
                return;
            }

            this.renderer = new THREE.WebGLRenderer({
                canvas: this.canvas,
                antialias: true,
                alpha: true
            });
            this.renderer.setSize(this.config.width, this.config.height);
            this.renderer.setPixelRatio(window.devicePixelRatio);
            this.renderer.shadowMap.enabled = true;
            this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        },

        setupCamera() {
            this.camera = new THREE.PerspectiveCamera(
                75,
                this.config.width / this.config.height,
                0.1,
                1000
            );
            this.camera.position.set(0, 2, 5);
            this.camera.lookAt(0, 0, 0);
        },

        setupEventListeners() {
            if (this.config.arButton) {
                this.config.arButton.addEventListener('click', () => this.startAR());
            }

            this.canvas.addEventListener('click', (e) => this.handleCanvasClick(e));
            this.canvas.addEventListener('touchstart', (e) => this.handleTouchStart(e));
            this.canvas.addEventListener('touchmove', (e) => this.handleTouchMove(e));
            this.canvas.addEventListener('touchend', (e) => this.handleTouchEnd(e));
        },

        checkWebXRSupport() {
            if (navigator.xr) {
                navigator.xr.isSessionSupported('immersive-ar')
                    .then(supported => {
                        this.config.webXRSupported = supported;
                        if (supported) {
                            console.log('WebXR AR is supported');
                        } else {
                            console.log('WebXR AR is not supported, using fallback mode');
                        }
                    })
                    .catch(err => {
                        console.error('Error checking WebXR support:', err);
                        this.config.webXRSupported = false;
                    });
            } else {
                this.config.webXRSupported = false;
            }
        },

        async startAR() {
            if (this.startTime) {
                return;
            }
            this.startTime = Date.now();

            if (this.config.webXRSupported) {
                await this.startWebXR();
            } else {
                this.startFallbackMode();
            }

            this.animate();
        },

        async startWebXR() {
            try {
                const session = await navigator.xr.requestSession('immersive-ar', {
                    requiredFeatures: ['hit-test', 'dom-overlay'],
                    domOverlay: { root: document.body }
                });

                this.renderer.xr.setReferenceSpaceType('local');
                await this.renderer.xr.setSession(session);

                this.createScene('room');
                this.setupHitTest();

                session.addEventListener('end', () => {
                    console.log('AR session ended');
                });

            } catch (err) {
                console.error('Failed to start WebXR session:', err);
                this.startFallbackMode();
            }
        },

        startFallbackMode() {
            this.createScene(this.sceneType || 'object_placement');
            this.animate();
        },

        createScene(type) {
            this.clearScene();

            const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
            this.scene.add(ambientLight);

            const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
            directionalLight.position.set(5, 10, 5);
            directionalLight.castShadow = true;
            this.scene.add(directionalLight);

            this.addGroundPlane();
            this.addEnvironment(type);
            this.addObjects(type);
        },

        clearScene() {
            if (!this.scene) {
                this.scene = new THREE.Scene();
            }

            while(this.scene.children.length > 0) {
                const object = this.scene.children[0];
                this.scene.remove(object);

                if (object.geometry) object.geometry.dispose();
                if (object.material) {
                    if (Array.isArray(object.material)) {
                        object.material.forEach(m => m.dispose());
                    } else {
                        object.material.dispose();
                    }
                }
            }

            this.objects = [];
            this.targetZone = null;
        },

        addGroundPlane() {
            const groundGeometry = new THREE.PlaneGeometry(20, 20);
            const groundMaterial = new THREE.MeshStandardMaterial({
                color: 0xcccccc,
                roughness: 0.8,
                metalness: 0.1
            });
            const ground = new THREE.Mesh(groundGeometry, groundMaterial);
            ground.rotation.x = -Math.PI / 2;
            ground.receiveShadow = true;
            this.scene.add(ground);
        },

        addEnvironment(type) {
            if (type === 'object_placement' || type === 'sequential_action') {
                this.targetZone = this.createTargetZone();
                this.scene.add(this.targetZone);
            }
        },

        createTargetZone() {
            const group = new THREE.Group();

            const zoneGeometry = new THREE.BoxGeometry(0.5, 0.1, 0.5);
            const zoneMaterial = new THREE.MeshStandardMaterial({
                color: 0x00ff00,
                transparent: true,
                opacity: 0.5,
                emissive: 0x00ff00,
                emissiveIntensity: 0.3
            });
            const zone = new THREE.Mesh(zoneGeometry, zoneMaterial);
            zone.position.y = 0.05;
            group.add(zone);

            const edgesGeometry = new THREE.EdgesGeometry(zoneGeometry);
            const edgesMaterial = new THREE.LineBasicMaterial({ color: 0x00ff00 });
            const edges = new THREE.LineSegments(edgesGeometry, edgesMaterial);
            edges.position.y = 0.05;
            group.add(edges);

            return group;
        },

        addObjects(type) {
            switch(type) {
                case 'object_placement':
                    this.addPlacementObjects();
                    break;
                case 'gesture_recognition':
                    this.addGestureIndicator();
                    break;
                case 'object_rotation':
                    this.addRotationObjects();
                    break;
                case 'sequential_action':
                    this.addSequentialObjects();
                    break;
            }
        },

        addPlacementObjects() {
            const shapes = ['cube', 'sphere', 'cylinder'];
            const colors = [0xff0000, 0x0000ff, 0x00ff00, 0xffff00, 0xff00ff];
            const objectCount = 1 + this.difficulty;

            for (let i = 0; i < objectCount; i++) {
                const shapeType = shapes[i % shapes.length];
                const color = colors[i % colors.length];
                const object = this.createObject(shapeType, color);

                object.position.set(
                    (Math.random() - 0.5) * 3,
                    0.2,
                    (Math.random() - 0.5) * 3
                );
                object.userData = {
                    id: `obj_${i}`,
                    type: 'placement',
                    originalPosition: object.position.clone(),
                    targetPosition: this.targetZone.position.clone()
                };

                object.castShadow = true;
                this.objects.push(object);
                this.scene.add(object);
            }
        },

        addRotationObjects() {
            const colors = [0xff0000, 0x0000ff, 0x00ff00];
            const objectCount = 1 + Math.floor(this.difficulty / 2);

            for (let i = 0; i < objectCount; i++) {
                const geometry = new THREE.BoxGeometry(0.4, 0.4, 0.4);
                const material = new THREE.MeshStandardMaterial({
                    color: colors[i % colors.length],
                    metalness: 0.3,
                    roughness: 0.4
                });
                const object = new THREE.Mesh(geometry, material);

                object.position.set(0, 0.2, -1);
                object.userData = {
                    id: `rot_obj_${i}`,
                    type: 'rotation',
                    targetRotation: Math.floor(Math.random() * 4) * 90,
                    isSelected: false
                };

                object.castShadow = true;
                this.objects.push(object);
                this.scene.add(object);
            }
        },

        addSequentialObjects() {
            const colors = [0xff0000, 0x0000ff, 0x00ff00, 0xffff00, 0xff00ff];
            const sequenceLength = 2 + this.difficulty;

            for (let i = 0; i < sequenceLength; i++) {
                const geometry = new THREE.SphereGeometry(0.15, 32, 32);
                const material = new THREE.MeshStandardMaterial({
                    color: colors[i % colors.length],
                    metalness: 0.2,
                    roughness: 0.5
                });
                const object = new THREE.Mesh(geometry, material);

                object.position.set(i * 0.5, 0.15, 0);
                object.userData = {
                    id: `seq_obj_${i}`,
                    type: 'sequential',
                    order: i,
                    isActivated: false
                };

                object.castShadow = true;
                this.objects.push(object);
                this.scene.add(object);
            }
        },

        addGestureIndicator() {
            const instructions = document.createElement('div');
            instructions.style.cssText = `
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                padding: 20px 40px;
                background: rgba(0, 0, 0, 0.8);
                color: white;
                border-radius: 10px;
                font-size: 24px;
                text-align: center;
                pointer-events: none;
            `;
            instructions.textContent = '请做出手势: 指向(point)';
            this.canvas.parentElement.appendChild(instructions);
            this.gestureInstructions = instructions;
        },

        createObject(shapeType, color) {
            let geometry;
            switch(shapeType) {
                case 'cube':
                    geometry = new THREE.BoxGeometry(0.3, 0.3, 0.3);
                    break;
                case 'sphere':
                    geometry = new THREE.SphereGeometry(0.2, 32, 32);
                    break;
                case 'cylinder':
                    geometry = new THREE.CylinderGeometry(0.15, 0.15, 0.3, 32);
                    break;
                default:
                    geometry = new THREE.BoxGeometry(0.3, 0.3, 0.3);
            }

            const material = new THREE.MeshStandardMaterial({
                color: color,
                metalness: 0.2,
                roughness: 0.5
            });

            const object = new THREE.Mesh(geometry, material);
            return object;
        },

        setupHitTest() {
            if (this.renderer.xr.getHitTestSource) {
                const hitTestSource = this.renderer.xr.getHitTestSource();
                const reticle = new THREE.Mesh(
                    new THREE.RingGeometry(0.1, 0.2, 32),
                    new THREE.MeshBasicMaterial({ color: 0x00ff00 })
                );
                reticle.rotation.x = -Math.PI / 2;
                reticle.visible = false;
                this.scene.add(reticle);
                this.hitTestReticle = reticle;
            }
        },

        handleCanvasClick(event) {
            const rect = this.canvas.getBoundingClientRect();
            const x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
            const y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

            const raycaster = new THREE.Raycaster();
            raycaster.setFromCamera({ x, y }, this.camera);

            const intersects = raycaster.intersectObjects(this.objects);

            if (intersects.length > 0) {
                this.handleObjectInteraction(intersects[0].object);
            }
        },

        handleTouchStart(event) {
            event.preventDefault();
            this.touchStartTime = Date.now();
            this.lastTouchPosition = {
                x: event.touches[0].clientX,
                y: event.touches[0].clientY
            };
        },

        handleTouchMove(event) {
            event.preventDefault();
            this.moveCount++;

            if (this.selectedObject) {
                const touch = event.touches[0];
                const deltaX = touch.clientX - this.lastTouchPosition.x;
                const deltaY = touch.clientY - this.lastTouchPosition.y;

                this.selectedObject.position.x += deltaX * 0.01;
                this.selectedObject.position.z -= deltaY * 0.01;

                this.lastTouchPosition = {
                    x: touch.clientX,
                    y: touch.clientY
                };
            }
        },

        handleTouchEnd(event) {
            event.preventDefault();
            const touchDuration = Date.now() - this.touchStartTime;

            if (touchDuration < 200 && !this.selectedObject) {
                const touch = event.changedTouches[0];
                const fakeEvent = {
                    clientX: touch.clientX,
                    clientY: touch.clientY
                };
                this.handleCanvasClick(fakeEvent);
            }
        },

        handleObjectInteraction(object) {
            this.moveCount++;

            if (object.userData.type === 'placement') {
                this.selectedObject = object;
                this.selectedObject.material.emissive = new THREE.Color(0xffffff);
                this.selectedObject.material.emissiveIntensity = 0.3;
            } else if (object.userData.type === 'rotation') {
                object.rotation.y += Math.PI / 4;
                this.checkRotationAccuracy(object);
            } else if (object.userData.type === 'sequential') {
                this.activateSequentialObject(object);
            }

            if (this.config.onProgress) {
                this.config.onProgress({
                    objectID: object.userData.id,
                    type: object.userData.type,
                    moveCount: this.moveCount
                });
            }
        },

        checkRotationAccuracy(object) {
            const currentAngle = (object.rotation.y * 180 / Math.PI) % 360;
            const targetAngle = object.userData.targetRotation;
            const diff = Math.abs(currentAngle - targetAngle);

            if (diff < 15 || diff > 345) {
                object.material.emissive = new THREE.Color(0x00ff00);
                object.material.emissiveIntensity = 0.5;
            }
        },

        activateSequentialObject(object) {
            const expectedOrder = this.objects.filter(o =>
                o.userData.type === 'sequential' && !o.userData.isActivated
            ).length;

            if (object.userData.order === expectedOrder) {
                object.userData.isActivated = true;
                object.material.emissive = new THREE.Color(0x00ff00);
                object.material.emissiveIntensity = 0.5;
            } else {
                object.material.emissive = new THREE.Color(0xff0000);
                object.material.emissiveIntensity = 0.5;

                setTimeout(() => {
                    this.objects.forEach(o => {
                        if (o.userData.type === 'sequential') {
                            o.userData.isActivated = false;
                            o.material.emissive = new THREE.Color(0x000000);
                            o.material.emissiveIntensity = 0;
                        }
                    });
                }, 1000);
            }
        },

        animate() {
            if (this.isVerified) return;

            requestAnimationFrame(() => this.animate());

            this.objects.forEach(obj => {
                if (obj.userData.type === 'placement' && obj !== this.selectedObject) {
                    obj.rotation.y += 0.01;
                }
            });

            if (this.gestureInstructions) {
                this.gestureInstructions.style.transform =
                    `translate(-50%, -50%) rotate(${Date.now() / 50}deg)`;
            }

            this.renderer.render(this.scene, this.camera);
        },

        async verify() {
            if (this.isVerified) {
                return { success: false, message: 'Already verified' };
            }

            const timeSpent = (Date.now() - this.startTime) / 1000;
            const result = await this.sendVerification(timeSpent);

            if (result.success) {
                this.isVerified = true;
                this.showSuccess();
            } else {
                this.showError(result.message);
            }

            return result;
        },

        async sendVerification(timeSpent) {
            const behaviorData = {
                move_count: this.moveCount,
                time_spent: timeSpent,
                accuracy: this.calculateAccuracy()
            };

            const objectPositions = this.objects
                .filter(o => o.userData.type === 'placement')
                .map(o => ({
                    id: o.userData.id,
                    position: [o.position.x, o.position.y, o.position.z]
                }));

            const requestData = {
                captcha_id: this.sessionID,
                scene_data: {
                    object_position: objectPositions.length > 0 ?
                        objectPositions[0].position : [0, 0, 0],
                    object_rotation: this.objects
                        .filter(o => o.userData.type === 'rotation')
                        .map(o => [o.rotation.x, o.rotation.y, o.rotation.z])
                },
                action_data: {
                    sequence: this.objects
                        .filter(o => o.userData.type === 'sequential')
                        .map(o => ({ id: o.userData.id, activated: o.userData.isActivated }))
                },
                behavior_data: behaviorData
            };

            try {
                const response = await fetch('/api/v1/captcha/ar/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(requestData)
                });

                return await response.json();
            } catch (error) {
                console.error('Verification request failed:', error);
                return {
                    success: false,
                    message: '验证请求失败',
                    score: 0
                };
            }
        },

        calculateAccuracy() {
            let correctCount = 0;
            let totalCount = 0;

            this.objects.forEach(obj => {
                if (obj.userData.type === 'placement' && obj.userData.targetPosition) {
                    totalCount++;
                    const distance = obj.position.distanceTo(obj.userData.targetPosition);
                    if (distance < 0.3) {
                        correctCount++;
                    }
                } else if (obj.userData.type === 'sequential' && obj.userData.isActivated) {
                    totalCount++;
                    correctCount++;
                }
            });

            return totalCount > 0 ? correctCount / totalCount : 0;
        },

        showSuccess() {
            this.objects.forEach(obj => {
                obj.material.emissive = new THREE.Color(0x00ff00);
                obj.material.emissiveIntensity = 0.5;
            });

            if (this.config.onSuccess) {
                this.config.onSuccess({
                    score: 1.0,
                    message: '验证成功'
                });
            }
        },

        showError(message) {
            if (this.config.onError) {
                this.config.onError({
                    message: message || '验证失败',
                    canRetry: true
                });
            }
        },

        reset() {
            this.isVerified = false;
            this.startTime = null;
            this.moveCount = 0;
            this.selectedObject = null;

            this.objects.forEach(obj => {
                if (obj.userData.originalPosition) {
                    obj.position.copy(obj.userData.originalPosition);
                }
                obj.material.emissive = new THREE.Color(0x000000);
                obj.material.emissiveIntensity = 0;

                if (obj.userData.type === 'sequential') {
                    obj.userData.isActivated = false;
                }
            });

            if (this.gestureInstructions) {
                this.gestureInstructions.remove();
                this.gestureInstructions = null;
            }

            this.createScene(this.sceneType || 'object_placement');
        },

        destroy() {
            this.isVerified = true;

            if (this.gestureInstructions) {
                this.gestureInstructions.remove();
            }

            this.clearScene();

            if (this.renderer) {
                this.renderer.dispose();
            }
        },

        async fetchSceneConfig() {
            try {
                const response = await fetch('/api/v1/captcha/ar/generate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        scene_type: this.sceneType,
                        difficulty: this.difficulty
                    })
                });

                const data = await response.json();

                if (data.code === 200 || data.success) {
                    this.sessionID = data.data.captcha_id;
                    this.applyServerConfig(data.data);
                    return data.data;
                }

                throw new Error('Failed to fetch scene config');
            } catch (error) {
                console.error('Error fetching scene config:', error);
                return null;
            }
        },

        applyServerConfig(config) {
            if (config.scene_config) {
                if (config.scene_config.objects) {
                    this.syncObjectsWithConfig(config.scene_config.objects);
                }
                if (config.scene_config.target_zone) {
                    this.syncTargetZone(config.scene_config.target_zone);
                }
            }
        },

        syncObjectsWithConfig(objectConfigs) {
            objectConfigs.forEach((objConfig, index) => {
                if (this.objects[index]) {
                    this.objects[index].userData.targetPosition = new THREE.Vector3(
                        objConfig.target_position[0],
                        objConfig.target_position[1],
                        objConfig.target_position[2]
                    );
                }
            });
        },

        syncTargetZone(zoneConfig) {
            if (this.targetZone) {
                this.targetZone.position.set(
                    zoneConfig.position[0],
                    zoneConfig.position[1],
                    zoneConfig.position[2]
                );
            }
        }
    };

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = ARCaptcha;
    } else {
        window.ARCaptcha = ARCaptcha;
    }
})();
