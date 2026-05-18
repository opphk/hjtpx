const HybridCaptcha = {
  sessionId: null,
  phase: 'slider',
  currentStep: 0,
  clickResults: [],
  
  async init(options = {}) {
    this.options = options;
    this.container = document.getElementById(options.containerId || 'captcha-container');
    
    if (!this.container) {
      console.error('Container not found');
      return;
    }
    
    await this.generate();
  },
  
  async generate() {
    try {
      const response = await fetch('/api/v1/captcha/hybrid/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          width: this.options.width || 320,
          height: this.options.height || 160,
          slider_width: 40,
          slider_height: 40,
          click_count: this.options.clickCount || 3
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.sessionId = result.data.session_id;
        this.phase = result.data.phase;
        this.renderSliderPhase(result.data);
      } else {
        this.showError(result.message);
      }
    } catch (error) {
      console.error('Generate error:', error);
      this.showError('生成验证码失败');
    }
  },
  
  renderSliderPhase(data) {
    this.container.innerHTML = `
      <div class="hybrid-captcha">
        <div class="captcha-hint">${data.click_phase_hint || '请拖动滑块完成验证'}</div>
        <div class="slider-container">
          <div class="slider-background">
            <img src="${data.background_url}" alt="background" class="slider-bg-image" />
            <div class="slider-gap-mask" style="left: ${data.gap_x}px; top: ${data.gap_y}px;"></div>
          </div>
          <div class="slider-track">
            <div class="slider-handle" id="slider-handle"></div>
          </div>
          <div class="slider-slider" id="slider-element" style="background-image: url(${data.slider_url});"></div>
        </div>
        <div class="slider-trajectory" id="trajectory-data"></div>
      </div>
    `;
    
    this.initSliderInteraction();
  },
  
  initSliderInteraction() {
    const handle = document.getElementById('slider-handle');
    const slider = document.getElementById('slider-element');
    const container = this.container.querySelector('.slider-container');
    
    if (!handle || !slider || !container) return;
    
    let isDragging = false;
    let startX = 0;
    let currentX = 0;
    const maxX = container.offsetWidth - handle.offsetWidth;
    const trajectory = [];
    
    const onMouseDown = (e) => {
      isDragging = true;
      startX = e.clientX || e.touches?.[0]?.clientX;
      trajectory.length = 0;
      trajectory.push({ x: currentX, y: 0, timestamp: Date.now() });
      e.preventDefault();
    };
    
    const onMouseMove = (e) => {
      if (!isDragging) return;
      
      const clientX = e.clientX || e.touches?.[0]?.clientX;
      let newX = clientX - startX + currentX;
      
      if (newX < 0) newX = 0;
      if (newX > maxX) newX = maxX;
      
      handle.style.left = newX + 'px';
      slider.style.left = newX + 'px';
      
      trajectory.push({ 
        x: newX, 
        y: 0, 
        timestamp: Date.now() 
      });
    };
    
    const onMouseUp = async () => {
      if (!isDragging) return;
      isDragging = false;
      
      currentX = parseInt(handle.style.left) || 0;
      
      await this.verifySlider(currentX, trajectory);
    };
    
    handle.addEventListener('mousedown', onMouseDown);
    handle.addEventListener('touchstart', onMouseDown, { passive: false });
    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('touchmove', onMouseMove, { passive: false });
    document.addEventListener('mouseup', onMouseUp);
    document.addEventListener('touchend', onMouseUp);
  },
  
  async verifySlider(positionX, trajectory) {
    try {
      const response = await fetch('/api/v1/captcha/hybrid/slider/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: this.sessionId,
          position_x: positionX,
          position_y: 0,
          trajectory: trajectory,
          risk_score: 0.5
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        if (result.data.success && result.data.phase === 'click') {
          this.phase = 'click';
          this.renderClickPhase(result.data);
        } else if (!result.data.success) {
          this.showSliderError('位置不正确，请重试');
          this.resetSlider();
        }
      }
    } catch (error) {
      console.error('Verify error:', error);
      this.showSliderError('验证失败');
    }
  },
  
  renderClickPhase(data) {
    this.container.innerHTML = `
      <div class="hybrid-captcha click-phase">
        <div class="captcha-hint">${data.next_phase_hint || '请点击目标'}</div>
        <div class="click-progress">
          <span>已点击: ${data.correct_clicks || 0}/${data.total_clicks || 3}</span>
        </div>
        <div class="click-container" id="click-area">
          <div class="click-targets"></div>
        </div>
        <div class="click-feedback" id="click-feedback"></div>
      </div>
    `;
    
    this.loadClickTargets();
  },
  
  async loadClickTargets() {
    try {
      const response = await fetch(`/api/v1/captcha/hybrid/data/${this.sessionId}`);
      const result = await response.json();
      
      if (result.code === 0) {
        this.renderTargets(result.data);
      }
    } catch (error) {
      console.error('Load targets error:', error);
    }
  },
  
  renderTargets(data) {
    const container = document.getElementById('click-area');
    if (!container) return;
    
    const targetsContainer = container.querySelector('.click-targets');
    if (!targetsContainer) return;
    
    targetsContainer.innerHTML = '';
    
    data.click_targets?.forEach((target, index) => {
      const targetEl = document.createElement('div');
      targetEl.className = 'click-target';
      targetEl.dataset.index = index;
      targetEl.style.cssText = `
        left: ${target.x}px;
        top: ${target.y}px;
        width: ${target.width}px;
        height: ${target.height}px;
      `;
      
      if (index < this.clickResults.length) {
        targetEl.classList.add(this.clickResults[index] ? 'correct' : 'incorrect');
      }
      
      targetsContainer.appendChild(targetEl);
    });
    
    this.initClickInteraction();
  },
  
  initClickInteraction() {
    const container = document.getElementById('click-area');
    if (!container) return;
    
    container.addEventListener('click', async (e) => {
      const target = e.target.closest('.click-target');
      if (!target) return;
      
      const index = parseInt(target.dataset.index);
      if (index !== this.clickResults.length) {
        this.showClickFeedback('请按顺序点击');
        return;
      }
      
      const rect = target.getBoundingClientRect();
      const containerRect = container.getBoundingClientRect();
      
      await this.verifyClick(
        e.clientX - containerRect.left,
        e.clientY - containerRect.top,
        index
      );
    });
  },
  
  async verifyClick(clickX, clickY, clickIndex) {
    try {
      const response = await fetch('/api/v1/captcha/hybrid/click/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: this.sessionId,
          click_x: Math.round(clickX),
          click_y: Math.round(clickY),
          click_index: clickIndex,
          click_time: Date.now(),
          risk_score: 0.5
        })
      });
      
      const result = await response.json();
      if (result.code === 0) {
        this.clickResults.push(result.data.success);
        
        if (result.data.phase === 'completed' && result.data.success) {
          this.onSuccess(result.data);
        } else {
          this.showClickFeedback(result.data.message);
          this.loadClickTargets();
        }
      }
    } catch (error) {
      console.error('Click verify error:', error);
    }
  },
  
  showSliderError(message) {
    const hint = this.container.querySelector('.captcha-hint');
    if (hint) {
      hint.textContent = message;
      hint.classList.add('error');
      setTimeout(() => {
        hint.classList.remove('error');
        hint.textContent = '请拖动滑块完成验证';
      }, 2000);
    }
  },
  
  showClickFeedback(message) {
    const feedback = document.getElementById('click-feedback');
    if (feedback) {
      feedback.textContent = message;
      feedback.classList.add('show');
      setTimeout(() => feedback.classList.remove('show'), 1500);
    }
  },
  
  showError(message) {
    this.container.innerHTML = `
      <div class="captcha-error">
        <span>${message}</span>
        <button onclick="HybridCaptcha.generate()">重试</button>
      </div>
    `;
  },
  
  resetSlider() {
    const handle = document.getElementById('slider-handle');
    const slider = document.getElementById('slider-element');
    if (handle) handle.style.left = '0px';
    if (slider) slider.style.left = '0px';
  },
  
  onSuccess(data) {
    if (this.options.onSuccess) {
      this.options.onSuccess(data);
    }
    this.container.innerHTML = `
      <div class="captcha-success">
        <span>验证成功！</span>
      </div>
    `;
  }
};

window.HybridCaptcha = HybridCaptcha;
