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
        this.scene.background = new THREE.Color(0xf0f0f0);

        this.camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
        this.camera.position.z = 8;

        this.renderer = new THREE.WebGLRenderer({ antialias: true });
        this.renderer.setSize(width, height);
        container.appendChild(this.renderer.domElement);

        const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
        this.scene.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
        directionalLight.position.set(5, 5, 5);
        this.scene.add(directionalLight);

        this.renderer.domElement.addEventListener('mousedown', (e) => this.onMouseDown(e));
        this.renderer.domElement.addEventListener('mousemove', (e) => this.onMouseMove(e));
        this.renderer.domElement.addEventListener('mouseup', () => this.onMouseUp());
        this.renderer.domElement.addEventListener('mouseleave', () => this.onMouseUp());

        window.addEventListener('resize', () => this.onWindowResize());

        this.animate();
    }

    renderPuzzle() {
        this.pieces = [];

        this.puzzle.pieces.forEach((pieceData, index) => {
            let geometry;
            switch (pieceData.type) {
                case 'cube':
                    geometry = new THREE.BoxGeometry(1, 1, 1);
                    break;
                case 'cylinder':
                    geometry = new THREE.CylinderGeometry(0.5, 0.5, 1, 32);
                    break;
                case 'sphere':
                    geometry = new THREE.SphereGeometry(0.6, 32, 32);
                    break;
                case 'cone':
                    geometry = new THREE.ConeGeometry(0.5, 1, 32);
                    break;
                case 'torus':
                    geometry = new THREE.TorusGeometry(0.4, 0.2, 16, 100);
                    break;
                default:
                    geometry = new THREE.BoxGeometry(1, 1, 1);
            }

            const material = new THREE.MeshPhongMaterial({
                color: pieceData.color,
                shininess: 100
            });

            const mesh = new THREE.Mesh(geometry, material);
            mesh.position.set(pieceData.positionX, pieceData.positionY, pieceData.positionZ);
            mesh.rotation.x = THREE.MathUtils.degToRad(pieceData.rotationX);
            mesh.rotation.y = THREE.MathUtils.degToRad(pieceData.rotationY);
            mesh.rotation.z = THREE.MathUtils.degToRad(pieceData.rotationZ);
            mesh.scale.set(pieceData.scale, pieceData.scale, pieceData.scale);
            mesh.userData = { id: pieceData.id, index };

            this.scene.add(mesh);
            this.pieces.push(mesh);
        });
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
        requestAnimationFrame(() => this.animate());
        this.renderer.render(this.scene, this.camera);
    }

    cleanup() {
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
        this.pieces = [];
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
