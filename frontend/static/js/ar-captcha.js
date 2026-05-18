const ARCaptcha = {
  sessionId: null,
  puzzle: null,
  gestureData: [],
  isSupported: false,
  lastRotation: { x: 0, y: 0, z: 0 },
  
  async init(options = {}) {
    this.options = options;
    this.container = document.getElementById(options.containerId || 'captcha-container');
    
    if (!this.container) {
      console.error('Container not found');
      return;
    }
    
    this.checkWebXRSupport();
    await this.generate();
  },
  
  checkWebXRSupport() {
    this.isSupported = !!(navigator.xr || navigator.getVRDisplays);
  },
  
  async generate() {
    try {
      const response = await fetch('/api/v1/captcha/ar/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          difficulty: this.options.difficulty || 'medium',
          object_type: this.options.objectType || '',
          gesture_type: this.options.gestureType || ''
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.sessionId = result.data.session_id;
        this.puzzle = result.data.puzzle;
        this.gestureData = [];
        this.render(result.data);
      } else {
        this.showError(result.message);
      }
    } catch (error) {
      console.error('Generate error:', error);
      this.showError('生成验证码失败');
    }
  },
  
  render(data) {
    const puzzle = data.puzzle;
    const obj = puzzle.object;
    
    this.container.innerHTML = `
      <div class="ar-captcha">
        <div class="captcha-hint">${puzzle.instructions || '旋转3D物体到目标位置'}</div>
        <div class="ar-viewport" id="ar-viewport">
          <div class="ar-3d-object" id="ar-3d-object" data-type="${obj.type}">
            ${this.render3DObject(obj)}
          </div>
          <div class="ar-guide-overlay" id="guide-overlay">
            <div class="guide-arrow ${this.getGestureDirection(puzzle.gesture_type)}"></div>
          </div>
        </div>
        <div class="ar-info">
          <div class="rotation-display">
            <span>X: <b id="rot-x">0</b>°</span>
            <span>Y: <b id="rot-y">0</b>°</span>
            <span>Z: <b id="rot-z">0</b>°</span>
          </div>
          <div class="target-display">
            目标角度: ${puzzle.target_angle.toFixed(0)}°
          </div>
        </div>
        <div class="ar-controls">
          <button id="reset-btn" class="ar-btn">重置</button>
          <button id="verify-btn" class="ar-btn primary">确认</button>
        </div>
      </div>
    `;
    
    this.initARInteraction();
    this.initDeviceMotion();
    this.initButtons();
  },
  
  render3DObject(obj) {
    switch (obj.type) {
      case 'cube':
        return `
          <div class="cube-face front" style="background: ${obj.color}"></div>
          <div class="cube-face back" style="background: ${obj.color}"></div>
          <div class="cube-face left" style="background: ${obj.color}"></div>
          <div class="cube-face right" style="background: ${obj.color}"></div>
          <div class="cube-face top" style="background: ${obj.color}"></div>
          <div class="cube-face bottom" style="background: ${obj.color}"></div>
        `;
      case 'sphere':
        return `<div class="sphere" style="background: radial-gradient(circle at 30% 30%, ${obj.color}, #000)"></div>`;
      case 'pyramid':
        return `<div class="pyramid" style="border-color: ${obj.color} transparent"></div>`;
      default:
        return `<div class="default-shape" style="background: ${obj.color}"></div>`;
    }
  },
  
  getGestureDirection(gestureType) {
    switch (gestureType) {
      case 'rotate_x': return 'direction-x';
      case 'rotate_y': return 'direction-y';
      case 'rotate_z': return 'direction-z';
      default: return 'direction-y';
    }
  },
  
  initARInteraction() {
    const viewport = document.getElementById('ar-viewport');
    const object3D = document.getElementById('ar-3d-object');
    
    if (!viewport || !object3D) return;
    
    let isDragging = false;
    let lastX = 0, lastY = 0;
    let rotationX = 0, rotationY = 0, rotationZ = 0;
    
    const updateRotation = () => {
      object3D.style.transform = `rotateX(${rotationX}deg) rotateY(${rotationY}deg) rotateZ(${rotationZ}deg)`;
      
      document.getElementById('rot-x').textContent = Math.round(rotationX % 360);
      document.getElementById('rot-y').textContent = Math.round(rotationY % 360);
      document.getElementById('rot-z').textContent = Math.round(rotationZ % 360);
      
      this.lastRotation = { x: rotationX, y: rotationY, z: rotationZ };
      
      this.gestureData.push({
        timestamp: Date.now(),
        rotation_x: rotationX,
        rotation_y: rotationY,
        rotation_z: rotationZ,
        scale: 1,
        gesture_type: this.puzzle?.gesture_type
      });
    };
    
    const onStart = (e) => {
      isDragging = true;
      const point = e.touches?.[0] || e;
      lastX = point.clientX;
      lastY = point.clientY;
      e.preventDefault();
    };
    
    const onMove = (e) => {
      if (!isDragging) return;
      
      const point = e.touches?.[0] || e;
      const deltaX = point.clientX - lastX;
      const deltaY = point.clientY - lastY;
      
      const sensitivity = 0.5;
      
      switch (this.puzzle?.gesture_type) {
        case 'rotate_x':
          rotationX += deltaY * sensitivity;
          break;
        case 'rotate_y':
          rotationY += deltaX * sensitivity;
          break;
        case 'rotate_z':
          rotationZ += deltaX * sensitivity;
          break;
        default:
          rotationY += deltaX * sensitivity;
          rotationX += deltaY * sensitivity;
      }
      
      lastX = point.clientX;
      lastY = point.clientY;
      
      updateRotation();
    };
    
    const onEnd = () => {
      isDragging = false;
    };
    
    viewport.addEventListener('mousedown', onStart);
    viewport.addEventListener('touchstart', onStart, { passive: false });
    document.addEventListener('mousemove', onMove);
    document.addEventListener('touchmove', onMove, { passive: false });
    document.addEventListener('mouseup', onEnd);
    document.addEventListener('touchend', onEnd);
  },
  
  initDeviceMotion() {
    if (!window.DeviceOrientationEvent) return;
    
    const handleOrientation = (e) => {
      if (e.alpha === null) return;
      
      this.gestureData.push({
        timestamp: Date.now(),
        rotation_x: e.beta || 0,
        rotation_y: e.gamma || 0,
        rotation_z: e.alpha || 0,
        scale: 1,
        gesture_type: 'device_motion'
      });
    };
    
    if (typeof DeviceOrientationEvent.requestPermission === 'function') {
      document.getElementById('ar-viewport')?.addEventListener('click', async () => {
        try {
          const permission = await DeviceOrientationEvent.requestPermission();
          if (permission === 'granted') {
            window.addEventListener('deviceorientation', handleOrientation);
          }
        } catch (err) {
          console.error('Device orientation permission error:', err);
        }
      }, { once: true });
    } else {
      window.addEventListener('deviceorientation', handleOrientation);
    }
  },
  
  initButtons() {
    document.getElementById('reset-btn')?.addEventListener('click', () => {
      this.gestureData = [];
      this.lastRotation = { x: 0, y: 0, z: 0 };
      
      const object3D = document.getElementById('ar-3d-object');
      if (object3D) {
        object3D.style.transform = 'rotateX(0deg) rotateY(0deg) rotateZ(0deg)';
      }
      
      document.getElementById('rot-x').textContent = '0';
      document.getElementById('rot-y').textContent = '0';
      document.getElementById('rot-z').textContent = '0';
    });
    
    document.getElementById('verify-btn')?.addEventListener('click', () => {
      this.verify();
    });
  },
  
  async verify() {
    try {
      const response = await fetch('/api/v1/captcha/ar/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: this.sessionId,
          rotation_x: this.lastRotation.x,
          rotation_y: this.lastRotation.y,
          rotation_z: this.lastRotation.z,
          scale: 1,
          gesture_data: this.gestureData,
          risk_score: 0.5
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.handleResult(result.data);
      }
    } catch (error) {
      console.error('Verify error:', error);
      this.showError('验证失败');
    }
  },
  
  handleResult(data) {
    if (data.success) {
      this.onSuccess(data);
    } else {
      this.showFeedback(data.message || '验证失败');
    }
  },
  
  showFeedback(message) {
    const hint = this.container.querySelector('.captcha-hint');
    if (hint) {
      hint.textContent = message;
      hint.classList.add('error');
      setTimeout(() => {
        hint.classList.remove('error');
        hint.textContent = this.puzzle?.instructions || '旋转3D物体到目标位置';
      }, 2000);
    }
  },
  
  showError(message) {
    this.container.innerHTML = `
      <div class="captcha-error">
        <span>${message}</span>
        <button onclick="ARCaptcha.generate()">重试</button>
      </div>
    `;
  },
  
  onSuccess(data) {
    if (this.options.onSuccess) {
      this.options.onSuccess(data);
    }
    this.container.querySelector('.captcha-hint').textContent = 'AR验证成功！';
    this.container.querySelector('.captcha-hint').classList.add('success');
  }
};

window.ARCaptcha = ARCaptcha;
