class ThreeDCaptcha {
    constructor() {
        this.sessionId = null;
        this.puzzle = null;
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.pieces = [];
        this.selectedPiece = null;
        this.isDragging = false;
        this.previousMousePosition = { x: 0, y: 0 };
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.timerInterval = null;
        this.seconds = 0;
        this.animationFrameId = null;
        this.isAnimating = false;
        this.qualitySettings = null;
        this.devicePixelRatio = Math.min(window.devicePixelRatio, 2);
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.refresh();
    }

    bindEvents() {
        document.getElementById('refresh-btn').addEventListener('click', () => this.refresh());
        document.getElementById('submit-btn').addEventListener('click', () => this.submit());
        document.getElementById('difficulty-select').addEventListener('change', (e) => {
            this.updateDifficultyDisplay(e.target.value);
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

    async refresh() {
        this.showLoading(true);
        this.hideResult();
        this.resetTimer();
        this.cleanup();

        const difficulty = document.getElementById('difficulty-select').value;

        try {
            const response = await fetch('/api/v1/captcha/3d/create', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ difficulty })
            });

            const result = await response.json();
            if (result.code === 0) {
                this.sessionId = result.data.sessionID;
                this.puzzle = result.data.puzzle;
                this.initThreeJS();
                this.renderPuzzle();
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
        
        const bgColor = this.puzzle.backgroundColor || '#f5f5f5';
        this.scene.background = new THREE.Color(bgColor);

        this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
        this.camera.position.z = 8;

        const antiAlias = this.puzzle.antiAlias !== false;
        this.renderer = new THREE.WebGLRenderer({ 
            antialias: antiAlias,
            alpha: true,
            powerPreference: 'high-performance'
        });
        this.renderer.setSize(width, height);
        this.renderer.setPixelRatio(this.devicePixelRatio);
        
        const shadowEnabled = this.puzzle.shadowEnabled === true;
        this.renderer.shadowMap.enabled = shadowEnabled;
        if (shadowEnabled) {
            this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        }
        
        container.appendChild(this.renderer.domElement);

        const lightIntensity = this.puzzle.lightIntensity || 0.8;
        const ambientColor = this.puzzle.ambientColor || '#444444';
        
        const ambientLight = new THREE.AmbientLight(ambientColor, lightIntensity * 0.5);
        this.scene.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(0xffffff, lightIntensity);
        directionalLight.position.set(5, 5, 5);
        directionalLight.castShadow = shadowEnabled;
        if (shadowEnabled) {
            directionalLight.shadow.mapSize.width = 2048;
            directionalLight.shadow.mapSize.height = 2048;
            directionalLight.shadow.camera.near = 0.5;
            directionalLight.shadow.camera.far = 50;
        }
        this.scene.add(directionalLight);

        const pointLight = new THREE.PointLight(0xffffff, lightIntensity * 0.3);
        pointLight.position.set(-5, -5, 5);
        this.scene.add(pointLight);

        this.renderer.domElement.addEventListener('mousedown', (e) => this.onMouseDown(e));
        this.renderer.domElement.addEventListener('mousemove', (e) => this.onMouseMove(e));
        this.renderer.domElement.addEventListener('mouseup', () => this.onMouseUp());
        this.renderer.domElement.addEventListener('mouseleave', () => this.onMouseUp());

        window.addEventListener('resize', () => this.onWindowResize());

        this.applyQualitySettings();
        this.animate();
    }

    applyQualitySettings() {
        const quality = this.puzzle.renderQuality || 'medium';
        
        switch (quality) {
            case 'low':
                this.renderer.setPixelRatio(1);
                break;
            case 'medium':
                this.renderer.setPixelRatio(Math.min(this.devicePixelRatio, 1.5));
                break;
            case 'high':
                this.renderer.setPixelRatio(this.devicePixelRatio);
                break;
        }
    }

    renderPuzzle() {
        this.pieces = [];

        const quality = this.puzzle.renderQuality || 'medium';
        const geometryDetail = this.getGeometryDetail(quality);

        this.puzzle.pieces.forEach((pieceData, index) => {
            let geometry = this.createGeometry(pieceData.type, geometryDetail);

            const material = this.createMaterial(pieceData);

            const mesh = new THREE.Mesh(geometry, material);
            mesh.position.set(pieceData.positionX, pieceData.positionY, pieceData.positionZ);
            mesh.rotation.x = THREE.MathUtils.degToRad(pieceData.rotationX);
            mesh.rotation.y = THREE.MathUtils.degToRad(pieceData.rotationY);
            mesh.rotation.z = THREE.MathUtils.degToRad(pieceData.rotationZ);
            mesh.scale.set(pieceData.scale, pieceData.scale, pieceData.scale);
            mesh.castShadow = this.puzzle.shadowEnabled === true;
            mesh.receiveShadow = this.puzzle.shadowEnabled === true;
            mesh.userData = { 
                id: pieceData.id, 
                index,
                originalRotX: pieceData.originalRotX,
                originalRotY: pieceData.originalRotY,
                originalRotZ: pieceData.originalRotZ,
                animationSpeed: pieceData.animationSpeed || 0
            };

            this.scene.add(mesh);
            this.pieces.push(mesh);
        });
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

    createGeometry(type, detail) {
        switch (type) {
            case 'cube':
            case 'box':
                return new THREE.BoxGeometry(1, 1, 1);
            case 'cylinder':
                return new THREE.CylinderGeometry(0.5, 0.5, 1, detail.cylinder);
            case 'sphere':
                return new THREE.SphereGeometry(0.6, detail.sphere, detail.sphere);
            case 'cone':
                return new THREE.ConeGeometry(0.5, 1, detail.cylinder);
            case 'torus':
                return new THREE.TorusGeometry(0.4, 0.2, detail.torus.radial, detail.torus.tubular);
            case 'octahedron':
                return new THREE.OctahedronGeometry(0.6);
            case 'tetrahedron':
                return new THREE.TetrahedronGeometry(0.6);
            case 'dodecahedron':
                return new THREE.DodecahedronGeometry(0.5);
            case 'icosahedron':
                return new THREE.IcosahedronGeometry(0.6);
            default:
                return new THREE.BoxGeometry(1, 1, 1);
        }
    }

    createMaterial(pieceData) {
        const materialParams = {
            color: pieceData.color,
            shininess: pieceData.shininess || 100,
            transparent: pieceData.opacity !== undefined && pieceData.opacity < 1,
            opacity: pieceData.opacity !== undefined ? pieceData.opacity : 1,
            wireframe: pieceData.wireframe === true,
            emissive: new THREE.Color(pieceData.emissiveColor || '#000000'),
            emissiveIntensity: pieceData.emissiveColor ? 0.3 : 0
        };

        if (pieceData.edgeType === 'smooth') {
            materialParams.side = THREE.DoubleSide;
        }

        return new THREE.MeshPhongMaterial(materialParams);
    }

    onMouseDown(event) {
        event.preventDefault();

        const rect = this.renderer.domElement.getBoundingClientRect();
        this.mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
        this.mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.pieces);

        if (intersects.length > 0) {
            this.selectedPiece = intersects[0].object;
            this.highlightPiece(this.selectedPiece, true);
            this.isDragging = true;
            this.previousMousePosition = { x: event.clientX, y: event.clientY };
        }
    }

    onMouseMove(event) {
        if (!this.isDragging || !this.selectedPiece) return;

        const deltaX = event.clientX - this.previousMousePosition.x;
        const deltaY = event.clientY - this.previousMousePosition.y;

        this.selectedPiece.rotation.y += deltaX * 0.01;
        this.selectedPiece.rotation.x += deltaY * 0.01;

        this.previousMousePosition = { x: event.clientX, y: event.clientY };
    }

    onMouseUp() {
        this.isDragging = false;
    }

    highlightPiece(piece, highlight) {
        if (highlight) {
            piece.material.emissive = new THREE.Color(0x333333);
        } else {
            piece.material.emissive = new THREE.Color(0x000000);
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
        
        if (!this.isDragging) {
            this.autoRotatePieces();
        }
        
        this.renderer.render(this.scene, this.camera);
    }

    autoRotatePieces() {
        this.pieces.forEach(piece => {
            const speed = piece.userData.animationSpeed || 0;
            if (speed > 0) {
                piece.rotation.y += speed;
                piece.rotation.z += speed * 0.5;
            }
        });
    }

    cleanup() {
        if (this.animationFrameId) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }
        
        if (this.pieces) {
            this.pieces.forEach(piece => {
                piece.geometry.dispose();
                piece.material.dispose();
            });
            this.pieces = [];
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
        this.selectedPiece = null;
    }

    getCurrentPuzzleState() {
        const pieces = this.pieces.map(piece => ({
            id: piece.userData.id,
            type: this.puzzle.pieces[piece.userData.index].type,
            color: this.puzzle.pieces[piece.userData.index].color,
            positionX: piece.position.x,
            positionY: piece.position.y,
            positionZ: piece.position.z,
            rotationX: THREE.MathUtils.radToDeg(piece.rotation.x),
            rotationY: THREE.MathUtils.radToDeg(piece.rotation.y),
            rotationZ: THREE.MathUtils.radToDeg(piece.rotation.z),
            scale: piece.scale.x
        }));

        return {
            pieces,
            gridSize: this.puzzle.gridSize,
            difficulty: this.puzzle.difficulty,
            targetRotX: this.puzzle.targetRotX,
            targetRotY: this.puzzle.targetRotY,
            targetRotZ: this.puzzle.targetRotZ
        };
    }

    async submit() {
        if (!this.sessionId) {
            this.showResult(false, '请先生成验证码');
            return;
        }

        this.showLoading(true);

        try {
            const response = await fetch('/api/v1/captcha/3d/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    sessionID: this.sessionId,
                    puzzle: this.getCurrentPuzzleState()
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
        document.getElementById('timer').textContent = `${minutes}:${seconds}`;
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
        banner.classList.remove('show');
    }

    showLoading(show) {
        const overlay = document.getElementById('loading-overlay');
        if (show) {
            overlay.classList.add('show');
        } else {
            overlay.classList.remove('show');
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new ThreeDCaptcha();
});
